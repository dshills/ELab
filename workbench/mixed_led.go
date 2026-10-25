package workbench

import "math"

// ledCurrent preserves the legacy branch except where strict Newton voltage
// correction requires resolving the declared LED law at leakage-scale current.
// At zero forward bias, use the forward one-sided Jacobian so a zero-current
// coil can initialize on that boundary. Current remains exactly zero; this
// neither adds reverse leakage nor changes the device equation.
func (n *mixedNetwork) ledCurrent(v float64) (i, g float64) {
	if !n.hasOptocoupler && !n.hasRegulator && len(n.inductors) == 0 && len(n.opamps) == 0 && !n.hasMOSFET && !n.hasGeneralBipolar && len(n.timers) == 0 && len(n.logic) == 0 && len(n.sequential) == 0 && len(n.displays) == 0 && len(n.controls) == 0 {
		return diodeCurrent(v)
	}
	if v < 0 {
		return 0, 0
	}
	m := Model()
	nv := m.EmissionCoefficient * 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	// Bisect junction voltage: a current-domain interval [0,V/Rs] cannot
	// resolve Is-scale startup currents in a fixed number of iterations.
	// Powered feedback can connect an LED to an off NPN's fA leakage, so
	// Op-amp and MOSFET networks need this precision for voltage-correction
	// convergence too, including cutoff points with tiny load currents.
	lo, hi := 0., v
	for k := 0; k < 65; k++ {
		junction := (lo + hi) / 2
		current := m.SaturationCurrent * math.Expm1(junction/nv)
		if junction+current*m.SeriesResistance > v {
			hi = junction
		} else {
			lo = junction
		}
	}
	junction := (lo + hi) / 2
	i = m.SaturationCurrent * math.Expm1(junction/nv)
	g = m.SaturationCurrent * math.Exp(junction/nv) / nv
	return i, g / (1 + m.SeriesResistance*g)
}
