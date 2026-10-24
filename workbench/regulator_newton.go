package workbench

import "math"

// Keep Newton corrections from leaping across the narrow pass-current knee.
// Only regulators use this damping; existing device limits remain in force.
func (n *mixedNetwork) regulatorNewtonScale(x, delta []float64, stages int, scale float64) float64 {
	if !n.hasRegulator {
		return scale
	}
	size := len(x) / stages
	for stage := range stages {
		base := stage * size
		for _, p := range n.c.MixedGraph.Components {
			if !IsRegulatorKind(p.Kind) {
				continue
			}
			v := n.regulatorVoltages(p, x[base:])
			d := n.regulatorVoltages(p, delta[base:])
			target := func(vin float64) float64 { return math.Max(0, math.Min(p.Value, vin-regulatorDropout(p.Kind))) }
			errorV := target(v[0]-v[2]) - (v[1] - v[2])
			next := target(v[0]-v[2]+scale*(d[0]-d[2])) - (v[1] - v[2] + scale*(d[1]-d[2]))
			change := next - errorV
			if errorV*next < 0 {
				scale *= .75 * math.Abs(errorV/change)
				continue
			}
			if errorV*change < 0 || math.Abs(errorV) < .05 {
				limit := math.Max(.005, math.Abs(errorV)*.75)
				if math.Abs(change) > limit {
					scale *= limit / math.Abs(change)
				}
			}
		}
	}
	return scale
}
