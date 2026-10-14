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
	return analysis == WaveformAnalysis || analysis == MixedAnalysis || analysis == NetworkDCAnalysis || analysis == NetworkTransientAnalysis
}
func networkEngine(analysis string) string {
	if analysis == WaveformAnalysis {
		return WaveformEngineID
	}
	if analysis == MixedAnalysis {
		return MixedEngineID
	}
	return NetworkEngineID
}

type SwitchParams struct {
	ID     string `json:"id"`
	Closed bool   `json:"closed"`
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
	mode := "open"
	if p.Closed {
		mode = "closed"
	}
	return s.applyNetworkEditLocked(graph, "switch:"+p.ID+":"+mode)
}
