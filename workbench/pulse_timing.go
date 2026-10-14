package workbench

import "math"

// Reconcile arithmetic representations of the same source corner with an
// output boundary. This is a few floating-point ULPs, not a fraction of the
// pulse period: a genuinely earlier sample must retain the earlier input.
func sameSampleTime(a, b float64) bool {
	if !finite(a) || !finite(b) {
		return false
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	spacing := math.Nextafter(scale, math.Inf(1)) - scale
	return finite(spacing) && math.Abs(a-b) <= 4*spacing
}
func sourceCornerReached(corner, end float64) bool {
	return corner <= end || sameSampleTime(corner, end)
}
func legacyPulseCycle(cfg TransientSettings, at float64) float64 {
	period := cfg.HighSeconds + cfg.LowSeconds
	cycle := math.Floor(at / period)
	if sameSampleTime(at, (cycle+1)*period) {
		cycle++
	}
	return cycle
}
func pulseHigh(cfg TransientSettings, at float64) bool {
	period := cfg.HighSeconds + cfg.LowSeconds
	edge := legacyPulseCycle(cfg, at)*period + cfg.HighSeconds
	return at < edge && !sameSampleTime(at, edge)
}
func nextPulseEdge(cfg TransientSettings, at float64) float64 {
	period := cfg.HighSeconds + cfg.LowSeconds
	cycle := legacyPulseCycle(cfg, at)
	edge := cycle*period + cfg.HighSeconds
	if edge <= at || sameSampleTime(edge, at) {
		edge = (cycle + 1) * period
	}
	return edge
}
