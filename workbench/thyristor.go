package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

const ThyristorModelID = "elab.generic-thyristor.v1"
const thyristorDescription = ThyristorModelID + ": generic low-voltage SCR/TRIAC. Finite on resistance and forward drop, 1 Gohm blocking; gate threshold 0.6 V plus configured resistance and 1 Gohm bleed. SCR has positive gate/unidirectional main path; TRIAC has symmetric four-quadrant gate and bipolar main path. Gate-supported conduction latches only after actual solved main current reaches Il, then holds above Ih in the same direction; reversal commutates. No breakover, dv/dt triggering, recovery time, quadrant sensitivity, snubber design, heating, manufacturer fit or mains/hardware rating. Main differential +/-24 V, gate +/-12 V educational envelope. Existing hybrid bracket/accuracy/work limits unchanged."

type ThyristorConfig struct {
	Version        string  `json:"version"`
	Trigger        float64 `json:"gate_trigger_current_a"`
	Latching       float64 `json:"latching_current_a"`
	Holding        float64 `json:"holding_current_a"`
	Resistance     float64 `json:"on_resistance_ohm"`
	Drop           float64 `json:"forward_drop_v"`
	GateResistance float64 `json:"gate_resistance_ohm"`
}
type ThyristorParams struct {
	ID         string          `json:"id"`
	Parameters ThyristorConfig `json:"parameters"`
}

func DefaultThyristorConfig() ThyristorConfig {
	return ThyristorConfig{ThyristorModelID, .0005, .005, .002, 1, .8, 100}
}
func IsThyristorKind(kind string) bool { return kind == "scr" || kind == "triac" }
func hasThyristors(g Graph) bool {
	for _, p := range g.Components {
		if IsThyristorKind(p.Kind) {
			return true
		}
	}
	return false
}
func thyristorFields() []string {
	return []string{"version", "gate_trigger_current_a", "latching_current_a", "holding_current_a", "on_resistance_ohm", "forward_drop_v", "gate_resistance_ohm"}
}
func validateThyristor(c ThyristorConfig) error {
	if c.Version != ThyristorModelID || !finite(c.Trigger) || c.Trigger < 1e-5 || c.Trigger > .01 || !finite(c.Latching) || c.Latching < .0001 || c.Latching > 1 || !finite(c.Holding) || c.Holding < 1e-5 || c.Holding > .05 || c.Holding >= c.Latching || !finite(c.Resistance) || c.Resistance < .1 || c.Resistance > 100 || !finite(c.Drop) || c.Drop < 0 || c.Drop > 2 || !finite(c.GateResistance) || c.GateResistance < 10 || c.GateResistance > 10000 {
		return fmt.Errorf("thyristor requires supported version, Igt10µA..10mA, Il0.1mA..1A, Ih10µA..50mA strictly below Il, Ron0.1..100Ω, drop0..2V and gate resistance10..10000Ω")
	}
	return nil
}
func thyristorPins(kind string) []string {
	if kind == "scr" {
		return []string{"anode", "cathode", "gate"}
	}
	return []string{"mt2", "mt1", "gate"}
}

// All currents enter physical pins; gate current returns through cathode/MT1.
func thyristorCurrents(p Component, v [5]float64, m ControlMemory) controlEvaluation {
	c := p.Device.Thyristor
	e := controlEvaluation{}
	branch := func(a, b int, i, g float64) {
		e.Current[a] += i
		e.Current[b] -= i
		e.Jacobian[a][a] += g
		e.Jacobian[b][b] += g
		e.Jacobian[a][b] -= g
		e.Jacobian[b][a] -= g
	}
	vm, vg := v[0]-v[1], v[2]-v[1]
	main, gmain := vm*1e-9, 1e-9
	if m.High {
		sign := 1.
		if p.Kind == "triac" && vm < 0 {
			sign = -1
		}
		if sign*vm > c.Drop {
			main += sign * (sign*vm - c.Drop) / c.Resistance
			gmain += 1 / c.Resistance
		}
	}
	gate, ggate := vg*1e-9, 1e-9
	sign := 1.
	if p.Kind == "triac" && vg < 0 {
		sign = -1
	}
	if sign*vg > .6 {
		gate += sign * (sign*vg - .6) / c.GateResistance
		ggate += 1 / c.GateResistance
	}
	branch(0, 1, main, gmain)
	branch(2, 1, gate, ggate)
	return e
}
func thyristorDecision(p Component, v [5]float64, m ControlMemory) (ControlMemory, error) {
	vm, vg := v[0]-v[1], v[2]-v[1]
	if !finite(vm) || !finite(vg) || math.Abs(vm) > 24+1e-9 || math.Abs(vg) > 12+1e-9 {
		return m, fmt.Errorf("thyristor main +/-24 V and gate +/-12 V model range exceeded")
	}
	c := p.Device.Thyristor
	e := thyristorCurrents(p, v, m)
	direction := 1
	if vm < 0 {
		direction = -1
	}
	gate := e.Current[2] >= c.Trigger
	if p.Kind == "triac" {
		gate = math.Abs(e.Current[2]) >= c.Trigger
	}
	forward := math.Abs(vm) > c.Drop && (p.Kind == "triac" || vm > 0)
	if m.High {
		current := float64(m.Direction) * e.Current[0]
		// A signed guard detects commutation even across an unsampled zero window.
		if direction == m.Direction && current >= c.Holding && (m.Latched || current >= c.Latching) {
			return ControlMemory{High: true, Latched: true, Direction: m.Direction}, nil
		}
	}
	if gate && forward {
		return ControlMemory{High: true, Direction: direction}, nil
	}
	return ControlMemory{}, nil
}
func componentControlModel(p Component) ControlModel {
	m := ControlledDeviceModel()
	if IsThyristorKind(p.Kind) {
		m.Thyristor = thyristorDescription
	}
	return m
}
func (s *Service) configureThyristorLocked(raw json.RawMessage) error {
	var v ThyristorParams
	if err := params(raw, &v, "id", "parameters"); err != nil {
		return err
	}
	if err := paramsNested(raw, "parameters", &ThyristorConfig{}, thyristorFields()...); err != nil {
		return err
	}
	if err := validateThyristor(v.Parameters); err != nil {
		return err
	}
	for k, p := range s.graph.Components {
		if p.ID == v.ID && IsThyristorKind(p.Kind) {
			s.graph.Components[k].Device = &DeviceParameters{Thyristor: &v.Parameters}
			return nil
		}
	}
	return fmt.Errorf("choose an SCR or TRIAC")
}
func ThyristorSymbolPaths(kind string) [][]Point {
	if kind == "scr" {
		return [][]Point{{{0, -40}, {0, -18}}, {{-20, -18}, {20, -18}, {0, 15}, {-20, -18}}, {{-22, 16}, {22, 16}}, {{0, 16}, {0, 40}}, {{-55, 25}, {-25, 25}, {-10, 16}}}
	}
	return [][]Point{{{0, -40}, {0, -24}, {-12, -24}, {-12, -18}}, {{0, -24}, {12, -24}, {12, -16}}, {{-24, -18}, {0, -18}, {-12, 16}, {-24, -18}}, {{0, 16}, {24, 16}, {12, -18}, {0, 16}}, {{-24, 17}, {0, 17}}, {{0, -19}, {24, -19}}, {{-12, 17}, {-12, 27}, {0, 27}, {12, 27}, {12, 16}}, {{0, 27}, {0, 40}}, {{-55, 25}, {-30, 25}, {-20, 14}}}
}
func ThyristorTerminalLocal(kind, pin string) (Point, bool) {
	pins := thyristorPins(kind)
	switch pin {
	case pins[0]:
		return Point{0, -40}, true
	case pins[1]:
		return Point{0, 40}, true
	case "gate":
		return Point{-55, 25}, true
	}
	return Point{}, false
}
