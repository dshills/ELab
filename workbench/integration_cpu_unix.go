//go:build darwin || linux

package workbench

import (
	"time"

	"golang.org/x/sys/unix"
)

const integrationThreadClock = true
const integrationClockBasis = "thread-cpu"

// The caller pins its goroutine for the complete staged sample. A process-wide
// clock would incorrectly charge other simulations, rendering and GC workers.
func integrationCPUTime() (time.Duration, error) {
	var value unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_THREAD_CPUTIME_ID, &value); err != nil {
		return 0, err
	}
	return time.Duration(value.Sec)*time.Second + time.Duration(value.Nsec), nil
}
