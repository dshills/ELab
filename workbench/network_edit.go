package workbench

type networkTransition struct {
	state *mixedState
	trace *Transient
}

func (s *Service) applyNetworkEditLocked(graph Graph, event string) error {
	// A switch/wiper actuation changes branch parameters on an unchanged topology. Stage the
	// algebraic right-hand solve before committing graph, events or physical state.
	if s.transient != nil && !s.transient.Stale && s.mixedState != nil && IsNetworkAnalysis(s.mixedState.net.c.Analysis) {
		compiled, _, diagnostic := graph.Compile()
		if diagnostic != nil {
			return &actionError{"unavailable", diagnostic.Message}
		}
		trace := cloneTransient(s.transient)
		trace.ElectricalRevision = s.electrical + 1
		net := newMixedNetwork(compiled)
		state, err := net.solve(s.mixedState.x, s.mixedState.capV, 0, trace, trace.Sample.Time)
		if err != nil {
			return &actionError{"solve-failed", err.Error()}
		}
		appendTrace(trace, trace.Sample)
		trace.Sample = mixedSample(compiled, state.result, net.sourceVoltage(compiled.Source, trace, trace.Sample.Time))
		appendTrace(trace, trace.Sample)
		appendInputEvent(trace, InputEvent{Time: trace.Sample.Time, Mode: event})
		s.pendingNetwork = &networkTransition{state: state, trace: trace}
	}

	s.graph = graph
	return nil
}
