//go:build windows

package workbench

import "os"

// The current process is known. Other positive leases remain protected until
// Windows process-liveness qualification can distinguish active from inactive.
func recoveryProcessOwnerStatus(pid int) RecoveryOwnerStatus {
	if pid == os.Getpid() {
		return RecoveryOwnerActive
	}
	return RecoveryOwnerUnknown
}
