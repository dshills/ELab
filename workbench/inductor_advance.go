package workbench

import (
	"errors"
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
	roundoffRetried := false
	for at < end {
		if err := budget.take(); err != nil {
			return nil, fmt.Errorf("%w at %.9g s, step %.3g: %s", err, at, h, failure)
		}
		remaining := end - at
		h = math.Min(h, remaining)
		h = math.Min(h, n.maximumWaveformStep(trace, at))
		h = math.Min(h, n.maximumInductorStep())
		// Split the last two intervals evenly rather than leave a tiny tail.
		// Both proposals stay below the original step bounds and undergo
		// the complete solves/error check. Never advance reported time by
		// discarding a fixed number of seconds of real storage evolution.
		if h < remaining && remaining-h < h/4 {
			h = remaining / 2
		}
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
		var roundoff *inductorRoundoffError
		if !roundoffRetried && (errors.As(one, &roundoff) || errors.As(two, &roundoff) || errors.As(three, &roundoff)) {
			// Smaller intervals can amplify algebraic cancellation. Retry once
			// from the last accepted state, inside the same source/LC bounds.
			// This is only a proposal: all three solves and the error estimate
			// must succeed, with the original convergence and work limits.
			larger := math.Min(end-at, math.Min(n.maximumWaveformStep(trace, at), n.maximumInductorStep()))
			if larger > 2*h {
				h, roundoffRetried = larger, true
				budget.roundoffRetries++
				continue
			}
		}
		if ratio > 1 {
			h /= 2
			continue
		}
		if len(n.timers)+len(n.sequential) > 0 {
			for _, point := range []struct {
				state  *inductorStage
				at, dt float64
			}{{half, at + h/2, h / 2}, {refined, at + h, h / 2}} {
				candidate := n.result(point.state.x, nil, point.dt, trace, point.at)
				changed, err := candidate.hybridGuardChanged()
				if err != nil {
					return nil, err
				}
				if changed {
					return candidate, nil
				}
			}
		}
		current = refined
		roundoffRetried = false
		at += h
		lastStep = h / 2
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
