//go:build !linux

package service

import (
	"fmt"
	"runtime"
)

// systemUpdateRestartSupported is false where replacing the running process
// image is unavailable; those builds keep the manual download flow.
const systemUpdateRestartSupported = false

func restartSelf(string, []string, []string) error {
	return fmt.Errorf("restarting in place is not supported on %s", runtime.GOOS)
}
