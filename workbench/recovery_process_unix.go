//go:build !windows

package workbench

import (
	"errors"
	"os"
	"syscall"
)

func recoveryProcessOwnerStatus(pid int) RecoveryOwnerStatus {
	process, err := os.FindProcess(pid)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
			return RecoveryOwnerInactive
		}
		return RecoveryOwnerUnknown
	}
	err = process.Signal(syscall.Signal(0))
	if err == nil || errors.Is(err, syscall.EPERM) {
		return RecoveryOwnerActive
	}
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
		return RecoveryOwnerInactive
	}
	return RecoveryOwnerUnknown
}
