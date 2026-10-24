package workbench

import "math"

const DiodeModelID = "elab.generic-silicon-diode.v1"

// DiodeModel describes an educational junction, without a physical part identity.
type DiodeModel LEDModel

func SiliconDiodeModel() DiodeModel {
	return DiodeModel{DiodeModelID, 1e-9, 1.7, 1, 27, "Generic silicon junction with Shockley forward/reverse leakage and finite series resistance; no reverse breakdown, capacitance, charge storage, thermal dynamics, ratings or physical part identity."}
}

// siliconDiodeCurrent solves I=Is*expm1((V-I*Rs)/(n*Vt)). The forward
// relation is monotone. In reverse bias, solving for junction voltage avoids
// log1p rounding to -Inf as current approaches -Is.
func siliconDiodeCurrent(v float64) (current, conductance float64) {
	return junctionDiodeCurrent(SiliconDiodeModel(), v)
}

func junctionDiodeCurrent(m DiodeModel, v float64) (current, conductance float64) {
	nv := m.EmissionCoefficient * 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	if v <= 0 {
		lo, hi := v, v+m.SaturationCurrent*m.SeriesResistance
		for k := 0; k < 60; k++ {
			junction := (lo + hi) / 2
			i := m.SaturationCurrent * math.Expm1(junction/nv)
			if junction+i*m.SeriesResistance > v {
				hi = junction
			} else {
				lo = junction
			}
		}
		junction := (lo + hi) / 2
		current = m.SaturationCurrent * math.Expm1(junction/nv)
		g := m.SaturationCurrent * math.Exp(junction/nv) / nv
		return current, g / (1 + m.SeriesResistance*g)
	}
	lo, hi := 0., v/m.SeriesResistance
	for k := 0; k < 65; k++ {
		i := (lo + hi) / 2
		drop := nv*math.Log1p(i/m.SaturationCurrent) + i*m.SeriesResistance
		if drop > v {
			hi = i
		} else {
			lo = i
		}
	}
	current = (lo + hi) / 2
	return current, 1 / (m.SeriesResistance + nv/(current+m.SaturationCurrent))
}
