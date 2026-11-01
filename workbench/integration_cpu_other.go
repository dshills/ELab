//go:build !darwin && !linux

package workbench

import (
	"errors"
	"time"
)

const integrationThreadClock = false
const integrationClockBasis = "wall-fallback"

func integrationCPUTime() (time.Duration, error) { return 0, errors.ErrUnsupported }
