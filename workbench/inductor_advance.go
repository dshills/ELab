package workbench

import (
	"fmt"
	"math"
)

func (state *mixedState) advanceInductorBudget(trace *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	if end <= state.result.Time {
		return state, nil
	}
	n := state.net
	current := &inductorStage{x: state.x, capV: state.capV}
	at, h := state.result.Time, end-state.result.Time
	lastStep := h
	failure := ""
	for at < end {
		if err := budget.take(); err != nil {
			return nil, fmt.Errorf("%w at %.9g s, step %.3g: %s", err, at, h, failure)
		}
		h = math.Min(h, end-at)
		h = math.Min(h, n.maximumWaveformStep(trace, at))
		h = math.Min(h, n.maximumInductorStep())
		if h < minimumInductorStep || at+h <= at {
			return nil, fmt.Errorf("RLC timestep too small at %.9g, h %.3g: %s", at, h, failure)
		}
		full, one := n.radauStep(current, trace, h, at+h, budget)
		half, two := n.radauStep(current, trace, h/2, at+h/2, budget)
		var refined *inductorStage
		var three error
		if two == nil {
			refined, three = n.radauStep(half, trace, h/2, at+h, budget)
		}
		ratio := math.Inf(1)
		if one == nil && two == nil && three == nil {
			ratio = n.inductorError(current, full, refined)
		}
		failure = fmt.Sprintf("ratio %.3g, errors %v / %v / %v", ratio, one, two, three)
		if ratio > 1 {
			h /= 2
			continue
		}
		current = refined
		at += h
		lastStep = h / 2
		if end-at < 1e-13 {
			at = end
		}
		if ratio == 0 {
			h *= 2
		} else {
			// Conservative growth for algebraic as well as differential
			// variables; every proposed step still gets its own comparison.
			h *= math.Min(2, math.Cbrt(.75/ratio))
		}
	}
	if err := budget.checkTime(); err != nil {
		return nil, err
	}
	accepted := n.result(current.x, nil, lastStep, trace, end)
	if !finite(accepted.result.PowerResidual) || !finite(accepted.result.KCLResidual) || accepted.result.KCLResidual > 1e-9 {
		return nil, fmt.Errorf("RLC accepted output exceeds current/power range")
	}
	if err := budget.checkTime(); err != nil {
		return nil, err
	}
	return accepted, nil
}
