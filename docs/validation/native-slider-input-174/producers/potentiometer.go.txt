package workbench

import "fmt"

const PotentiometerModelID = "elab.linear-potentiometer.v1"

type PotentiometerModel struct {
	ID            string  `json:"id"`
	EndResistance float64 `json:"end_resistance_ohm"`
	Limitations   string  `json:"limitations"`
}

func LinearPotentiometerModel() PotentiometerModel {
	return PotentiometerModel{PotentiometerModelID, 1, "Generic linear three-pin resistive track. Total resistance is split into two segments, each with 1 ohm endpoint resistance. Position 0 is toward pin 2; position 1 is toward pin 1. No contact noise, nonlinear taper, ratings, heating or physical part identity."}
}
func potentiometerSegments(p Component) (upper, lower float64) {
	w := .5
	if p.WiperPosition != nil {
		w = *p.WiperPosition
	}
	end := LinearPotentiometerModel().EndResistance
	return end + (p.Value-2*end)*(1-w), end + (p.Value-2*end)*w
}

type WiperParams struct {
	ID       string  `json:"id"`
	Position float64 `json:"position"`
}
type PotentiometerResult struct {
	ID               string  `json:"id"`
	Resistance       float64 `json:"resistance_ohm"`
	WiperPosition    float64 `json:"wiper_position"`
	UpperResistance  float64 `json:"upper_resistance_ohm"`
	LowerResistance  float64 `json:"lower_resistance_ohm"`
	Terminal1Voltage float64 `json:"terminal_1_voltage_v"`
	Terminal2Voltage float64 `json:"terminal_2_voltage_v"`
	WiperVoltage     float64 `json:"wiper_voltage_v"`
	Terminal1Current float64 `json:"terminal_1_current_a"`
	Terminal2Current float64 `json:"terminal_2_current_a"`
	WiperCurrent     float64 `json:"wiper_current_a"`
	Power            float64 `json:"power_w"`
}

func (s *Service) setWiperLocked(p WiperParams) error {
	if !finite(p.Position) || p.Position < 0 || p.Position > 1 {
		return fmt.Errorf("wiper position must be finite and 0–1")
	}
	part, ok := s.graph.Component(p.ID)
	if !ok || part.Kind != "potentiometer" {
		return fmt.Errorf("choose a potentiometer")
	}
	if part.WiperPosition != nil && *part.WiperPosition == p.Position {
		if s.activeInputWorkLocked() && s.inputJob.work.Wiper != nil && s.inputJob.work.Wiper.ID == p.ID {
			s.cancelInputWorkLocked("superseded")
		}
		return nil
	}
	graph := s.graph.Clone()
	for i := range graph.Components {
		if graph.Components[i].ID == p.ID {
			v := p.Position
			graph.Components[i].WiperPosition = &v
		}
	}
	if s.transient != nil && !s.transient.Stale && s.mixedState != nil && IsNetworkAnalysis(s.mixedState.net.c.Analysis) {
		return s.startInputWorkLocked("", &p, graph, fmt.Sprintf("wiper:%s:%.17g", p.ID, p.Position))
	}
	s.graph = graph
	return nil
}
