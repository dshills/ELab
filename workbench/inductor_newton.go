package workbench

import "math"

// Limit forward junction excursions, allowing a reverse-biased junction to
// approach conduction in one update. Linear/common-mode node motion need not
// crawl in 0.2 V increments. These are guesses only: the same equations,
// convergence tests and ±30 V range guard still decide acceptance.
func (n *mixedNetwork) inductorNewtonScale(x, delta []float64, stages int) float64 {
	scale, size := 1., len(x)/stages
	forward := func(value, change float64) {
		allowed := maxMixedNewtonDelta + math.Max(0, -value)
		if change > allowed {
			scale = math.Min(scale, allowed/change)
		}
	}
	for stage := range stages {
		values, changes := x[stage*size:(stage+1)*size], delta[stage*size:(stage+1)*size]
		junction := func(p Component, one, two string) (float64, float64) {
			a, b := n.pin(p, one), n.pin(p, two)
			return voltage(values, a) - voltage(values, b), voltage(changes, a) - voltage(changes, b)
		}
		for _, p := range n.c.MixedGraph.Components {
			switch p.Kind {
			case "optocoupler":
				forward(junction(p, "anode", "cathode"))
			case "led", "diode", "zener", "schottky":
				forward(junction(p, "anode", "cathode"))
			case "npn", "pnp":
				for _, other := range []string{"emitter", "collector"} {
					v, d := junction(p, "base", other)
					sign := bipolarPolarity(p.Kind)
					forward(sign*v, sign*d)
				}
			case "nmos", "pmos":
				_, gateChange := junction(p, "gate", "source")
				if math.Abs(gateChange) > maxMixedNewtonDelta {
					scale = math.Min(scale, maxMixedNewtonDelta/math.Abs(gateChange))
				}
				if p.Kind == "nmos" {
					forward(junction(p, "source", "drain"))
				} else {
					forward(junction(p, "drain", "source"))
				}
			case "hbridge":
				for _, out := range []string{"out1", "out2"} {
					forward(junction(p, out, "vm"))
					forward(junction(p, "gnd", out))
				}
			case "speaker", "shift-register", "piezo", "active-buzzer", "pwm-controller", "relay", "comparator", "pushbutton", "bouncing-pushbutton", "timer555", "thermistor", "ldr", "resistor", "capacitor", "inductor", "switch", "potentiometer",
				"dc-source", "pulse-source", "sine-source", "finite-pulse-source",
				"ground", "junction", "net-label":
			default:
				// Future nonlinear parts retain the legacy conservative bound
				// until their controlling junctions are explicitly described.
				for node := range n.nodes {
					if change := math.Abs(changes[node]); change > maxMixedNewtonDelta {
						scale = math.Min(scale, maxMixedNewtonDelta/change)
					}
				}
			}
		}
	}
	return n.regulatorNewtonScale(x, delta, stages, n.opAmpNewtonScale(x, delta, stages, scale))
}
