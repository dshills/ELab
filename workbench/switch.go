package workbench

import "fmt"

const SwitchModelID = "elab.spst-resistive.v1"
const NetworkEngineID = "elab.network-mna-be.v1"
const NetworkDCAnalysis = "Network DC operating point"
const NetworkTransientAnalysis = "Network transient"

type SwitchModel struct {
	ID            string  `json:"id"`
	OnResistance  float64 `json:"on_resistance_ohm"`
	OffResistance float64 `json:"off_resistance_ohm"`
	Limitations   string  `json:"limitations"`
}

func SwitchDeviceModel() SwitchModel {
	return SwitchModel{SwitchModelID, 1, 1e9, "Generic SPST with finite on/off resistance; no contact bounce, arcing, thermal behavior, ratings or named physical part. Actuation is instantaneous; capacitor voltage remains continuous."}
}
func switchResistance(p Component) float64 {
	model := SwitchDeviceModel()
	if p.SwitchClosed != nil && *p.SwitchClosed {
		return model.OnResistance
	}
	return model.OffResistance
}
func IsNetworkAnalysis(analysis string) bool {
	return analysis == MixedAnalysis || analysis == NetworkDCAnalysis || analysis == NetworkTransientAnalysis
}
func networkEngine(analysis string) string {
	if analysis == MixedAnalysis {
		return MixedEngineID
	}
	return NetworkEngineID
}

type SwitchParams struct {
	ID     string `json:"id"`
	Closed bool   `json:"closed"`
}
type switchTransition struct {
	state *mixedState
	trace *Transient
}

func (s *Service) setSwitchLocked(p SwitchParams) error {
	part, ok := s.graph.Component(p.ID)
	if !ok || part.Kind != "switch" {
		return fmt.Errorf("choose a switch")
	}
	if part.SwitchClosed != nil && *part.SwitchClosed == p.Closed {
		return nil
	}
	graph := s.graph.Clone()
	for i := range graph.Components {
		if graph.Components[i].ID == p.ID {
			closed := p.Closed
			graph.Components[i].SwitchClosed = &closed
		}
	}
	// Live actuation is a source event on an unchanged topology. Stage the
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
		mode := "open"
		if p.Closed {
			mode = "closed"
		}
		appendInputEvent(trace, InputEvent{Time: trace.Sample.Time, Mode: "switch:" + p.ID + ":" + mode})
		s.pendingSwitch = &switchTransition{state: state, trace: trace}
	}
	s.graph = graph
	return nil
}
