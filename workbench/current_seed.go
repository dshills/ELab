package workbench

import "math"

// A single Norton source in an otherwise purely resistive referenced network
// has one monotone scalar compliance equation. Use it only as a Newton guess:
// the ordinary equations/residuals, iteration/time and node limits still decide
// acceptance. No storage, semiconductors, controlled loads or future parts enter
// this reduction. Both source terminals may float relative to ground.
func (n *mixedNetwork) currentResistiveGuess(t *Transient, at float64) []float64 {
	if len(n.sources)+len(n.caps)+len(n.inductors) > 0 || n.startupSourceScale != nil {
		return nil
	}
	var source *Component
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "current-source":
			if source != nil || p.Device == nil || p.Device.Current == nil {
				return nil
			}
			v := p
			source = &v
		case "resistor", "ground", "junction", "net-label":
		default:
			return nil
		}
	}
	if source == nil {
		return nil
	}
	zero := make([]float64, n.matrixSize(0))
	level := currentLevel(*source, at)
	if level == 0 {
		return zero
	}
	// At zero terminal voltage, admitted compliance >=1 V has zero control
	// derivative. The matrix is exactly the resistors plus declared 1 GΩ bleed.
	a, _ := n.equations(zero, nil, 0, t, at)
	rhs := make([]float64, len(a))
	positive, negative := n.pin(*source, "positive"), n.pin(*source, "negative")
	if positive >= 0 {
		rhs[positive]++
	}
	if negative >= 0 {
		rhs[negative]--
	}
	response, err := linearSolve(a, rhs)
	if err != nil {
		return nil
	}
	resistance := voltage(response, positive) - voltage(response, negative)
	if !finite(resistance) || resistance <= 0 {
		return nil
	}
	compliance := source.Device.Current.Compliance
	lo, hi := 0., compliance
	for range 64 {
		v := (lo + hi) / 2
		q, _ := smoothLevel(v, compliance-1, compliance)
		if v/resistance < level*(1-q) {
			lo = v
		} else {
			hi = v
		}
	}
	scale := ((lo + hi) / 2) / resistance
	for k, v := range response {
		response[k] = v * scale
		if !finite(response[k]) || math.Abs(response[k]) > maxMixedNodeVoltage {
			return nil
		}
	}
	return response
}
