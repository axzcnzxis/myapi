//go:build linux

package service

import "syscall"

// systemUpdateRestartSupported marks the platforms where the process can
// replace its own image, so an update takes effect without a supervisor.
const systemUpdateRestartSupported = true

// restartSelf replaces the current process image with the binary at path. It
// never returns on success: the same PID continues as the new build, which also
// covers containers where this process is PID 1 and no restart policy applies.
func restartSelf(path string, argv, env []string) error {
	return syscall.Exec(path, argv, env)
}
