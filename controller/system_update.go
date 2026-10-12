package controller

import (
	"net/http"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/i18n"
	"github.com/axzcnzxis/myapi/service"

	"github.com/gin-gonic/gin"
)

type systemUpdateRequest struct {
	Version string `json:"version"`
}

// GetSystemUpdateStatus reports whether this deployment can replace its own
// executable, and how far the attempt already running has got.
func GetSystemUpdateStatus(c *gin.Context) {
	// The progress changes between polls, so no intermediary may keep a copy.
	c.Header("Cache-Control", "private, no-store")
	common.ApiSuccess(c, service.GetSystemUpdateStatus())
}

// ApplySystemUpdate starts installing a published release over the running
// executable. The download runs in the background: it outlives any proxy read
// timeout, and the dashboard follows GetSystemUpdateStatus for progress.
func ApplySystemUpdate(c *gin.Context) {
	var request systemUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := service.StartSystemUpdate(request.Version); err != nil {
		writeSystemUpdateError(c, err)
		return
	}
	common.ApiSuccess(c, service.GetSystemUpdateStatus())
}

// writeSystemUpdateError reports the stable code the dashboard translates; the
// internal detail stays in the server log.
func writeSystemUpdateError(c *gin.Context, err error) {
	code := service.SystemUpdateErrorCode(err)
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"code":    code,
		"message": common.TranslateMessage(c, systemUpdateMessageKey(code)),
	})
}

// systemUpdateMessageKey maps a failure code to the message returned to clients
// that do not translate the code themselves.
func systemUpdateMessageKey(code string) string {
	switch code {
	case service.SystemUpdateCodeUnsupported:
		return i18n.MsgSystemUpdateUnsupported
	case service.SystemUpdateCodeInvalidVersion:
		return i18n.MsgSystemUpdateInvalidVersion
	case service.SystemUpdateCodeNotNewer:
		return i18n.MsgSystemUpdateNotNewer
	case service.SystemUpdateCodeInProgress:
		return i18n.MsgSystemUpdateInProgress
	case service.SystemUpdateCodeDownloadFailed:
		return i18n.MsgSystemUpdateDownloadFailed
	case service.SystemUpdateCodeChecksumFailed:
		return i18n.MsgSystemUpdateChecksumFailed
	case service.SystemUpdateCodeInstallFailed:
		return i18n.MsgSystemUpdateInstallFailed
	case service.SystemUpdateCodeRestartFailed:
		return i18n.MsgSystemUpdateRestartFailed
	default:
		return i18n.MsgSystemUpdateFailed
	}
}
