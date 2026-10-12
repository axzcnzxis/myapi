package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/model"
)

// Self-update installs a published release over the running executable. It only
// ever talks to this project's own GitHub releases: the download URL is built
// from a validated tag, and the result must match the checksum the release
// publishes for the current platform before anything on disk is replaced.
const (
	systemUpdateRepository = "axzcnzxis/myapi"
	// Only this host and its asset CDN may serve the download, so a hijacked
	// redirect cannot point the update at an arbitrary server.
	systemUpdateHost            = "github.com"
	systemUpdateHostSuffix      = ".githubusercontent.com"
	systemUpdateChecksumAsset   = "checksums-linux.txt"
	systemUpdateMaxAssetBytes   = 512 << 20
	systemUpdateMaxChecksumSize = 1 << 20
	systemUpdateDownloadTimeout = 30 * time.Minute
	// The restart waits long enough for the dashboard to observe the restarting
	// state before the listening socket closes.
	systemUpdateRestartDelay = 1500 * time.Millisecond
)

// SystemUpdateState* report how far the current attempt has got. The dashboard
// polls them instead of holding one long request open, which a reverse proxy
// would cut off well before a release has finished downloading.
const (
	SystemUpdateStateIdle        = "idle"
	SystemUpdateStateDownloading = "downloading"
	SystemUpdateStateVerifying   = "verifying"
	SystemUpdateStateInstalling  = "installing"
	SystemUpdateStateRestarting  = "restarting"
	SystemUpdateStateFailed      = "failed"
)

// Stable error codes the dashboard maps to its own translations, so a failed
// attempt never surfaces an internal message.
const (
	SystemUpdateCodeUnsupported    = "SYSTEM_UPDATE_UNSUPPORTED"
	SystemUpdateCodeInvalidVersion = "SYSTEM_UPDATE_INVALID_VERSION"
	SystemUpdateCodeNotNewer       = "SYSTEM_UPDATE_NOT_NEWER"
	SystemUpdateCodeInProgress     = "SYSTEM_UPDATE_IN_PROGRESS"
	SystemUpdateCodeDownloadFailed = "SYSTEM_UPDATE_DOWNLOAD_FAILED"
	SystemUpdateCodeChecksumFailed = "SYSTEM_UPDATE_CHECKSUM_FAILED"
	SystemUpdateCodeInstallFailed  = "SYSTEM_UPDATE_INSTALL_FAILED"
	SystemUpdateCodeRestartFailed  = "SYSTEM_UPDATE_RESTART_FAILED"
	SystemUpdateCodeFailed         = "SYSTEM_UPDATE_FAILED"
)

// Sentinel failures. Every download and install error wraps one of these so
// callers never parse messages.
var (
	ErrSystemUpdateUnsupported    = errors.New("self-update is not supported on this platform")
	ErrSystemUpdateInvalidVersion = errors.New("invalid release version")
	ErrSystemUpdateNotNewer       = errors.New("the requested version is not newer than the running version")
	ErrSystemUpdateInProgress     = errors.New("another system update is already running")
	ErrSystemUpdateDownload       = errors.New("failed to download the release")
	ErrSystemUpdateChecksum       = errors.New("the downloaded release failed verification")
	ErrSystemUpdateInstall        = errors.New("failed to install the release")
	ErrSystemUpdateRestart        = errors.New("failed to restart the panel")
)

// Reasons reported by GetSystemUpdateStatus, translated by the dashboard.
const (
	SystemUpdateReasonPlatform   = "platform"
	SystemUpdateReasonExecutable = "executable"
	SystemUpdateReasonWritable   = "writable"
)

// SystemUpdateErrorCode maps a sentinel failure to the code the dashboard
// translates; an unrecognised failure reports the generic code.
func SystemUpdateErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrSystemUpdateUnsupported):
		return SystemUpdateCodeUnsupported
	case errors.Is(err, ErrSystemUpdateInvalidVersion):
		return SystemUpdateCodeInvalidVersion
	case errors.Is(err, ErrSystemUpdateNotNewer):
		return SystemUpdateCodeNotNewer
	case errors.Is(err, ErrSystemUpdateInProgress):
		return SystemUpdateCodeInProgress
	case errors.Is(err, ErrSystemUpdateDownload):
		return SystemUpdateCodeDownloadFailed
	case errors.Is(err, ErrSystemUpdateChecksum):
		return SystemUpdateCodeChecksumFailed
	case errors.Is(err, ErrSystemUpdateInstall):
		return SystemUpdateCodeInstallFailed
	case errors.Is(err, ErrSystemUpdateRestart):
		return SystemUpdateCodeRestartFailed
	default:
		return SystemUpdateCodeFailed
	}
}

// SystemUpdateStatus describes whether this deployment can update itself, which
// file an update would replace, and the state of the attempt in progress.
type SystemUpdateStatus struct {
	CurrentVersion string `json:"current_version"`
	Executable     string `json:"executable"`
	Supported      bool   `json:"supported"`
	Writable       bool   `json:"writable"`
	// Container reports that the process runs inside a container, where the
	// replaced binary lives in the writable layer and is lost when the
	// container is recreated from its image.
	Container bool   `json:"container"`
	Reason    string `json:"reason"`

	State         string `json:"state"`
	TargetVersion string `json:"target_version"`
	ErrorCode     string `json:"error_code"`
}

// systemUpdateRunning is the single-flight guard: only one attempt may download
// and replace the executable at a time.
var systemUpdateRunning atomic.Bool

var systemUpdateJob = struct {
	mu            sync.RWMutex
	state         string
	targetVersion string
	errorCode     string
}{state: SystemUpdateStateIdle}

func setSystemUpdateProgress(state, version, code string) {
	systemUpdateJob.mu.Lock()
	systemUpdateJob.state = state
	systemUpdateJob.targetVersion = version
	systemUpdateJob.errorCode = code
	systemUpdateJob.mu.Unlock()
}

func systemUpdateProgress() (state, version, code string) {
	systemUpdateJob.mu.RLock()
	defer systemUpdateJob.mu.RUnlock()
	return systemUpdateJob.state, systemUpdateJob.targetVersion, systemUpdateJob.errorCode
}

// systemUpdateHTTPClient follows the release download redirects to the asset
// CDN and nothing else. It deliberately ignores the relay transport settings:
// a self-update is an operator action against a fixed host, and relay timeouts
// are far too short for a multi-megabyte download.
var systemUpdateHTTPClient = &http.Client{
	Timeout:       systemUpdateDownloadTimeout,
	CheckRedirect: checkSystemUpdateRedirect,
}

func checkSystemUpdateRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refusing insecure redirect to %s", req.URL.Redacted())
	}
	if !isSystemUpdateHost(req.URL.Hostname()) {
		return fmt.Errorf("refusing redirect to unexpected host %s", req.URL.Hostname())
	}
	return nil
}

func isSystemUpdateHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == systemUpdateHost || strings.HasSuffix(host, systemUpdateHostSuffix)
}

// releaseVersion is the comparable form of a project release tag. The grammar
// mirrors the dashboard's parser (web/src/features/system-update/releases.ts)
// so both sides rank the same tags in the same order.
type releaseVersion struct {
	core     []int64
	stage    int
	sequence int64
	revision int64
}

var releaseVersionPattern = regexp.MustCompile(
	`^v?(\d+(?:\.\d+){2,})(?:-(alpha|beta|rc|patch)(?:\.(\d+))?(?:-i18nfix\.(\d+))?)?(?:\+[\da-zA-Z.-]+)?$`)

// Release stages, ordered so a pre-release never outranks the stable tag of the
// same core version.
var releaseVersionStages = map[string]int{
	"alpha":  0,
	"beta":   1,
	"rc":     2,
	"":       3,
	"stable": 3,
	"patch":  4,
}

func parseReleaseVersion(raw string) (releaseVersion, bool) {
	match := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return releaseVersion{}, false
	}
	// The i18n revision suffix only exists on release candidates.
	if match[4] != "" && match[2] != "rc" {
		return releaseVersion{}, false
	}
	parts := strings.Split(match[1], ".")
	core := make([]int64, 0, len(parts))
	allZero := true
	for _, part := range parts {
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return releaseVersion{}, false
		}
		if value != 0 {
			allZero = false
		}
		core = append(core, value)
	}
	// The development build reports v0.0.0; treating it as a version would make
	// every release look like a downgrade.
	if allZero {
		return releaseVersion{}, false
	}
	return releaseVersion{
		core:     core,
		stage:    releaseVersionStages[match[2]],
		sequence: parseReleaseNumber(match[3]),
		revision: parseReleaseNumber(match[4]),
	}, true
}

func parseReleaseNumber(raw string) int64 {
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// compareReleaseVersions returns -1, 0 or 1 for left against right.
func compareReleaseVersions(left, right releaseVersion) int {
	for index := 0; index < max(len(left.core), len(right.core)); index++ {
		a, b := int64(0), int64(0)
		if index < len(left.core) {
			a = left.core[index]
		}
		if index < len(right.core) {
			b = right.core[index]
		}
		if a != b {
			if a < b {
				return -1
			}
			return 1
		}
	}
	for _, pair := range [][2]int64{
		{int64(left.stage), int64(right.stage)},
		{left.sequence, right.sequence},
		{left.revision, right.revision},
	} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// systemUpdateAssetName maps a release tag to the asset the release workflow
// publishes for the given platform (see .github/workflows/release.yml).
func systemUpdateAssetName(version, goos, goarch string) (string, bool) {
	if goos != "linux" {
		return "", false
	}
	switch goarch {
	case "amd64":
		return "new-api-" + version, true
	case "arm64":
		return "new-api-arm64-" + version, true
	default:
		return "", false
	}
}

func systemUpdatePlatformSupported() bool {
	if !systemUpdateRestartSupported {
		return false
	}
	_, ok := systemUpdateAssetName("v0.0.0", runtime.GOOS, runtime.GOARCH)
	return ok
}

func systemUpdateDownloadURL(version, asset string) string {
	return "https://" + systemUpdateHost + "/" + systemUpdateRepository + "/releases/download/" +
		url.PathEscape(version) + "/" + url.PathEscape(asset)
}

// systemUpdateCapability is the part of the status that cannot change while the
// process runs.
type systemUpdateCapability struct {
	executable string
	supported  bool
	writable   bool
	container  bool
	reason     string
}

var (
	systemUpdateCapabilityOnce  sync.Once
	systemUpdateCapabilityValue systemUpdateCapability
)

// systemUpdateCapabilityOf resolves the file an update would replace and whether
// this process may replace it. Probing writability writes to the program
// directory, so the answer is resolved once instead of on every status poll.
func systemUpdateCapabilityOf() systemUpdateCapability {
	systemUpdateCapabilityOnce.Do(func() {
		capability := systemUpdateCapability{container: runningInContainer()}
		defer func() { systemUpdateCapabilityValue = capability }()
		if !systemUpdatePlatformSupported() {
			capability.reason = SystemUpdateReasonPlatform
			return
		}
		executable, err := systemUpdateExecutable()
		if err != nil || executable == "" {
			capability.reason = SystemUpdateReasonExecutable
			return
		}
		capability.executable = executable
		capability.supported = true
		if !directoryAcceptsNewFiles(filepath.Dir(executable)) {
			capability.reason = SystemUpdateReasonWritable
			return
		}
		capability.writable = true
	})
	return systemUpdateCapabilityValue
}

// systemUpdateExecutable is the path an update replaces. It is resolved once and
// kept, because after the rename the running process can no longer name the file
// it was started from.
func systemUpdateExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return executable, nil
}

// directoryAcceptsNewFiles proves the update can stage its download next to the
// executable, which is required for the atomic replace.
func directoryAcceptsNewFiles(dir string) bool {
	probe, err := os.CreateTemp(dir, ".myapi-update-probe-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return true
}

// runningInContainer detects a containerised process, where replacing the
// binary only survives restarts of the same container.
func runningInContainer() bool {
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}
	content, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	for _, marker := range []string{"docker", "containerd", "kubepods", "podman", "lxc"} {
		if bytes.Contains(content, []byte(marker)) {
			return true
		}
	}
	return false
}

// GetSystemUpdateStatus reports whether this process can replace its own
// executable, and how far the current attempt has got.
func GetSystemUpdateStatus() SystemUpdateStatus {
	capability := systemUpdateCapabilityOf()
	state, targetVersion, errorCode := systemUpdateProgress()
	return SystemUpdateStatus{
		CurrentVersion: common.Version,
		Executable:     capability.executable,
		Supported:      capability.supported,
		Writable:       capability.writable,
		Container:      capability.container,
		Reason:         capability.reason,
		State:          state,
		TargetVersion:  targetVersion,
		ErrorCode:      errorCode,
	}
}

// StartSystemUpdate validates the request and hands the download to a background
// worker. Installing a release takes minutes, so the call returns as soon as the
// attempt is accepted and the dashboard follows GetSystemUpdateStatus for
// progress. Every rejection is returned synchronously.
func StartSystemUpdate(version string) error {
	if !systemUpdatePlatformSupported() {
		return ErrSystemUpdateUnsupported
	}
	version = strings.TrimSpace(version)
	target, ok := parseReleaseVersion(version)
	if !ok {
		return fmt.Errorf("%w: %q", ErrSystemUpdateInvalidVersion, version)
	}
	if _, ok := systemUpdateAssetName(version, runtime.GOOS, runtime.GOARCH); !ok {
		return ErrSystemUpdateUnsupported
	}
	// A running version that does not parse (a development build) cannot be
	// compared, so only a comparable current version may reject the target.
	if current, ok := parseReleaseVersion(common.Version); ok {
		if compareReleaseVersions(target, current) <= 0 {
			return fmt.Errorf("%w: %s", ErrSystemUpdateNotNewer, common.Version)
		}
	}
	if !systemUpdateRunning.CompareAndSwap(false, true) {
		return ErrSystemUpdateInProgress
	}
	setSystemUpdateProgress(SystemUpdateStateDownloading, version, "")
	go runSystemUpdate(version)
	return nil
}

// runSystemUpdate downloads the release, verifies it against the checksums the
// release publishes, replaces the executable, and restarts the panel. The
// running process keeps executing the previous binary until the restart.
func runSystemUpdate(version string) {
	// The attempt outlives the request that started it, so it carries its own
	// deadline rather than the caller's.
	ctx, cancel := context.WithTimeout(context.Background(), systemUpdateDownloadTimeout)
	defer cancel()

	executable, err := systemUpdateExecutable()
	if err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateInstall, err))
		return
	}
	staged, err := os.CreateTemp(filepath.Dir(executable), ".myapi-update-*")
	if err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateInstall, err))
		return
	}
	stagedPath := staged.Name()
	defer func() { _ = os.Remove(stagedPath) }()

	asset, _ := systemUpdateAssetName(version, runtime.GOOS, runtime.GOARCH)
	// Fetch the checksums first: a missing release or asset then fails before a
	// large download starts.
	checksums, err := fetchSystemUpdateChecksums(ctx, version)
	if err != nil {
		failSystemUpdate(version, err)
		return
	}
	expected, ok := checksums[asset]
	if !ok {
		failSystemUpdate(version, fmt.Errorf("%w: the release publishes no checksum for %s", ErrSystemUpdateChecksum, asset))
		return
	}
	actual, err := downloadSystemUpdateAsset(ctx, systemUpdateDownloadURL(version, asset), staged)
	if err != nil {
		failSystemUpdate(version, err)
		return
	}
	if err := staged.Close(); err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateInstall, err))
		return
	}
	setSystemUpdateProgress(SystemUpdateStateVerifying, version, "")
	if actual != expected {
		failSystemUpdate(version, fmt.Errorf("%w: checksum mismatch for %s", ErrSystemUpdateChecksum, asset))
		return
	}
	if err := validateSystemUpdateBinary(stagedPath, runtime.GOARCH); err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateChecksum, err))
		return
	}
	setSystemUpdateProgress(SystemUpdateStateInstalling, version, "")
	if err := os.Chmod(stagedPath, 0o755); err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateInstall, err))
		return
	}
	backupSystemUpdateBinary(executable)
	// Renaming over the running executable keeps the old inode mapped for this
	// process while the path starts resolving to the new binary.
	if err := os.Rename(stagedPath, executable); err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateInstall, err))
		return
	}
	common.SysLog(fmt.Sprintf("system update: installed %s to %s", version, executable))
	setSystemUpdateProgress(SystemUpdateStateRestarting, version, "")
	restartAfterSystemUpdate(version, executable)
}

// failSystemUpdate records the failure for the dashboard and releases the
// single-flight guard so the operator can retry. The detail stays in the log.
func failSystemUpdate(version string, err error) {
	common.SysError("system update: " + err.Error())
	systemUpdateRunning.Store(false)
	setSystemUpdateProgress(SystemUpdateStateFailed, version, SystemUpdateErrorCode(err))
}

// restartAfterSystemUpdate replaces the process image with the installed binary.
func restartAfterSystemUpdate(version, executable string) {
	time.Sleep(systemUpdateRestartDelay)
	common.SysLog("system update: restarting into " + version)
	// syscall.Exec never runs deferred cleanups, so persist what the normal
	// shutdown path would have saved before the image is replaced.
	if common.DataExportEnabled {
		model.SaveQuotaDataCache()
	}
	if err := restartSelf(executable, os.Args, os.Environ()); err != nil {
		failSystemUpdate(version, fmt.Errorf("%w: %v", ErrSystemUpdateRestart, err))
	}
}

func fetchSystemUpdateChecksums(ctx context.Context, version string) (map[string]string, error) {
	body, err := getSystemUpdateAsset(ctx, systemUpdateDownloadURL(version, systemUpdateChecksumAsset), systemUpdateMaxChecksumSize)
	if err != nil {
		return nil, err
	}
	checksums := parseSystemUpdateChecksums(string(body))
	if len(checksums) == 0 {
		return nil, fmt.Errorf("%w: the release checksums are unreadable", ErrSystemUpdateChecksum)
	}
	return checksums, nil
}

// parseSystemUpdateChecksums reads sha256sum output: "<hex>  <file>".
func parseSystemUpdateChecksums(content string) map[string]string {
	checksums := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != sha256.Size*2 {
			continue
		}
		if _, err := hex.DecodeString(sum); err != nil {
			continue
		}
		checksums[strings.TrimPrefix(fields[1], "*")] = sum
	}
	return checksums
}

func getSystemUpdateAsset(ctx context.Context, assetURL string, limit int64) ([]byte, error) {
	response, err := doSystemUpdateRequest(ctx, assetURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSystemUpdateDownload, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: the release asset is larger than expected", ErrSystemUpdateDownload)
	}
	return body, nil
}

// downloadSystemUpdateAsset streams the asset into dest and returns its sha256.
func downloadSystemUpdateAsset(ctx context.Context, assetURL string, dest *os.File) (string, error) {
	response, err := doSystemUpdateRequest(ctx, assetURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()

	digest := sha256.New()
	written, err := io.Copy(io.MultiWriter(dest, digest), io.LimitReader(response.Body, systemUpdateMaxAssetBytes+1))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSystemUpdateDownload, err)
	}
	if written > systemUpdateMaxAssetBytes {
		return "", fmt.Errorf("%w: the release asset is larger than expected", ErrSystemUpdateDownload)
	}
	if written == 0 {
		return "", fmt.Errorf("%w: the release asset is empty", ErrSystemUpdateDownload)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func doSystemUpdateRequest(ctx context.Context, assetURL string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSystemUpdateDownload, err)
	}
	response, err := systemUpdateHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSystemUpdateDownload, err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("%w: %s returned HTTP %d", ErrSystemUpdateDownload, assetURL, response.StatusCode)
	}
	return response, nil
}

// validateSystemUpdateBinary rejects a download that is not a 64-bit
// little-endian ELF executable for this machine, which catches truncated
// responses and wrong-architecture assets before they replace a working binary.
func validateSystemUpdateBinary(path, goarch string) error {
	expected, ok := map[string]uint16{"amd64": 0x3e, "arm64": 0xb7}[goarch]
	if !ok {
		return fmt.Errorf("unsupported architecture %s", goarch)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	header := make([]byte, 20)
	if _, err := io.ReadFull(file, header); err != nil {
		return errors.New("the downloaded file is too short to be an executable")
	}
	if !bytes.Equal(header[0:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return errors.New("the downloaded file is not an ELF executable")
	}
	if header[4] != 2 || header[5] != 1 {
		return errors.New("the downloaded executable is not 64-bit little-endian")
	}
	if machine := binary.LittleEndian.Uint16(header[18:20]); machine != expected {
		return fmt.Errorf("the downloaded executable targets machine 0x%x, expected 0x%x", machine, expected)
	}
	return nil
}

// backupSystemUpdateBinary keeps the replaced binary next to the executable so
// an operator can restore it after an update that fails to start. A missing
// backup never blocks the update itself.
func backupSystemUpdateBinary(executable string) {
	source, err := os.Open(executable)
	if err != nil {
		common.SysError("system update: cannot read the current binary for backup: " + err.Error())
		return
	}
	defer func() { _ = source.Close() }()

	backupPath := executable + ".bak"
	backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		common.SysError("system update: cannot write the binary backup: " + err.Error())
		return
	}
	if _, err := io.Copy(backup, source); err != nil {
		common.SysError("system update: cannot complete the binary backup: " + err.Error())
	}
	if err := backup.Close(); err != nil {
		common.SysError("system update: cannot close the binary backup: " + err.Error())
		return
	}
	common.SysLog("system update: kept the previous binary at " + backupPath)
}
