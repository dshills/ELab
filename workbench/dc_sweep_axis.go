package workbench

import "math"

// logRatio retains nearby representable endpoints even when their separate
// logarithms round identically, and avoids overflowing a very large ratio.
func logRatio(start, stop float64) float64 {
	delta := (stop - start) / start
	if math.Abs(delta) < .5 {
		return math.Log1p(delta)
	}
	if ratio := stop / start; finite(ratio) && ratio > 0 {
		return math.Log(ratio)
	}
	return math.Log(stop) - math.Log(start)
}

func sweepAxisValue(start, stop, fraction float64, spacing string) float64 {
	if fraction == 0 {
		return start
	}
	if fraction == 1 {
		return stop
	}
	if spacing == "logarithmic" {
		// Work from the larger endpoint so exp's argument cannot overflow.
		if start < stop {
			return stop * math.Exp(logRatio(start, stop)*(fraction-1))
		}
		return start * math.Exp(-logRatio(stop, start)*fraction)
	}
	return start + (stop-start)*fraction
}

func sweepAxisPoint(start, stop float64, count, index int, spacing string) float64 {
	if spacing != "logarithmic" && index > 0 && index < count-1 {
		// Preserve the original linear record arithmetic exactly: saved axes
		// are validated bit-for-bit, including older version-1 captures.
		return start + (stop-start)*float64(index)/float64(count-1)
	}
	return sweepAxisValue(start, stop, float64(index)/float64(count-1), spacing)
}

// AxisValue returns a tick's parameter value along the validated result's
// ascending display axis. Stored points retain their requested sweep order.
func (r SweepResult) AxisValue(fraction float64) float64 {
	return sweepAxisValue(math.Min(r.Start, r.Stop), math.Max(r.Start, r.Stop), fraction, r.Spacing)
}

// AxisFraction maps a stored point to the same display axis used by ticks and
// cursor hit testing. Values and measurements remain in their original SI units.
func (r SweepResult) AxisFraction(value float64) float64 {
	start, stop := math.Min(r.Start, r.Stop), math.Max(r.Start, r.Stop)
	if r.Spacing == "logarithmic" {
		return logRatio(start, value) / logRatio(start, stop)
	}
	return (value - start) / (stop - start)
}
