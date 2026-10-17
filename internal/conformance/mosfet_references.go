package conformance

import "math"

func pmosDriverLEDReference(gate float64) float64 { return mosfetLEDLoadReference(5 - gate - 1.5) }
func nmosDriverLEDReference(gate float64) float64 { return mosfetLEDLoadReference(gate - 1.5) }

// Independent 5 V /1k/red-LED load line for either illustrative MOS polarity.
// This uses scalar bisection of the declared v1 laws, without the MNA/device
// evaluator. Replacing reverse body leakage by 1 nA stays below the 2 nA bound.
func mosfetLEDLoadReference(overdrive float64) float64 {
	if overdrive <= 0 {
		return 1e-9
	}
	lo, hi := 0., .005
	for range 100 {
		current := (lo + hi) / 2
		led := 2*1.380649e-23*300.15/1.602176634e-19*math.Log1p(current/1e-20) + current
		vds := 5 - 1000*current - led
		channel := 0.
		if vds > 0 {
			if vds < overdrive {
				channel = .1 * (overdrive*vds - .5*vds*vds) * (1 + .02*vds)
			} else {
				channel = .05 * overdrive * overdrive * (1 + .02*vds)
			}
		}
		if channel+1e-9 > current {
			lo = current
		} else {
			hi = current
		}
	}
	return (lo + hi) / 2
}
