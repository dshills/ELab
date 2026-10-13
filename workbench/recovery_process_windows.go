//go:build windows

package workbench

// Process-liveness qualification is part of the future Windows pilot. Preserve
// leases conservatively there; recovery copies remain explicitly restorable.
func recoveryProcessAlive(pid int) bool { return pid > 0 }
