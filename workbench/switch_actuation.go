package workbench

import (
	"context"
	"fmt"
)

type SwitchState struct {
	ID            string `json:"id"`
	DesiredClosed bool   `json:"desired_closed"`
	AppliedClosed *bool  `json:"applied_closed,omitempty"`
	Pending       bool   `json:"pending"`
}

func cloneSwitchStates(states []SwitchState) []SwitchState {
	if states == nil {
		return nil
	}
	out := append([]SwitchState{}, states...)
	for i := range out {
		if out[i].AppliedClosed != nil {
			v := *out[i].AppliedClosed
			out[i].AppliedClosed = &v
		}
	}
	return out
}
func (s *Service) switchStatesLocked() []SwitchState {
	out := []SwitchState{}
	for _, part := range s.graph.Components {
		if part.Kind != "switch" {
			continue
		}
		state := SwitchState{ID: part.ID, DesiredClosed: *part.SwitchClosed}
		if s.mixed != nil && !s.mixed.Stale && s.mixedState != nil {
			applied, ok := s.mixedState.net.c.MixedGraph.Component(part.ID)
			if ok && applied.SwitchClosed != nil {
				v := *applied.SwitchClosed
				state.AppliedClosed = &v
				state.Pending = v != state.DesiredClosed && s.transient != nil && !s.transient.Stale
			}
		}
		out = append(out, state)
	}
	return out
}
func graphWithAppliedSwitches(desired, applied Graph) Graph {
	out := desired.Clone()
	for i, p := range out.Components {
		if p.Kind != "switch" {
			continue
		}
		before, ok := applied.Component(p.ID)
		if ok && before.SwitchClosed != nil {
			v := *before.SwitchClosed
			out.Components[i].SwitchClosed = &v
		}
	}
	return out
}

// Apply the final desired switch states only after a whole accepted sample.
// Advancing the old circuit and the right-hand event solve are both staged;
// failure cannot publish time, charge, history, or a partially applied group.
func advanceLiveNetwork(ctx context.Context, state *mixedState, t *Transient, end float64, desired Graph) (output *mixedState, result *Transient, err error) {
	budget := newIntegrationBudgetContext(ctx)
	budget.enter()
	defer budget.leave()
	defer func() {
		if err == nil {
			if failure := budget.checkTime(); failure != nil {
				output, result, err = nil, nil, failure
			}
		}
	}()
	next, trace, err := advanceMixedBudget(state, t, end, budget)
	if err != nil {
		return nil, nil, err
	}
	changes := []Component{}
	for _, part := range desired.Components {
		if part.Kind != "switch" {
			continue
		}
		applied, ok := next.net.c.MixedGraph.Component(part.ID)
		if ok && applied.SwitchClosed != nil && *applied.SwitchClosed != *part.SwitchClosed {
			changes = append(changes, part)
		}
	}
	if len(changes) == 0 {
		return next, trace, nil
	}
	compiled, _, diagnostic := desired.Compile()
	if diagnostic != nil {
		return nil, nil, fmt.Errorf("pending switch circuit: %s", diagnostic.Message)
	}
	net := newMixedNetwork(compiled)
	net.inheritTimers555(next.net)
	net.inheritSequential(next.net)
	net.inheritControls(next.net)
	if err := budget.take(); err != nil {
		return nil, nil, err
	}
	right, err := net.solveContext(budget.solverContext(), next.x, next.capV, 0, trace, end)
	if err != nil {
		return nil, nil, err
	}
	// advanceMixed already retained the left-hand output at end.
	trace.Sample = mixedSample(compiled, right.result, mixedSourceVoltage(compiled, right.result))
	appendNetworkTrace(trace, trace.Sample)
	for _, part := range changes {
		mode := "open"
		if *part.SwitchClosed {
			mode = "closed"
		}
		appendInputEvent(trace, InputEvent{Time: end, Mode: "switch:" + part.ID + ":" + mode})
	}
	return right, trace, nil
}
