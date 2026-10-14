// Package workbench owns ELab behavior independently of native UI and GPU state.
package workbench

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const ModelID = "elab.generic-red-led.v1"
const EngineID = "elab.series-led-dc.v1"

type Circuit struct {
	Voltage    float64 `json:"voltage_v"`
	Resistance float64 `json:"resistance_ohm"`
	Connected  bool    `json:"connected"`
}

func DefaultCircuit() Circuit { return Circuit{5, 1000, true} }

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type View struct {
	Center Point   `json:"center"`
	Zoom   float64 `json:"zoom"`
}
type Component struct {
	Sine           *SineWaveform  `json:"sine,omitempty"`
	Pulse          *PulseWaveform `json:"pulse,omitempty"`
	WiperPosition  *float64       `json:"wiper_position,omitempty"`
	SwitchClosed   *bool          `json:"switch_closed,omitempty"`
	Label          string         `json:"label,omitempty"`
	InitialVoltage *float64       `json:"initial_voltage_v,omitempty"`
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Name           string         `json:"name"`
	Terminals      []string       `json:"terminals"`
	Position       Point          `json:"position"`
	Value          float64        `json:"value"`
	Unit           string         `json:"unit,omitempty"`
	Model          string         `json:"model,omitempty"`
	Rotation       int            `json:"rotation"`
}
type Wire struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Connected bool   `json:"connected"`
}
type LEDModel struct {
	ID                  string  `json:"id"`
	SaturationCurrent   float64 `json:"is_a"`
	EmissionCoefficient float64 `json:"n"`
	SeriesResistance    float64 `json:"rs_ohm"`
	Temperature         float64 `json:"temperature_c"`
	Limitations         string  `json:"limitations"`
}

func Model() LEDModel {
	return LEDModel{ModelID, 1e-20, 2, 1, 27, "Educational generic diode; no physical part identity, reverse breakdown, thermal dynamics, ratings, or calibrated brightness."}
}
func Components(c Circuit) []Component {
	return []Component{
		{ID: "V1", Kind: "dc-source", Name: "DC source", Terminals: []string{"positive", "negative"}, Position: Point{-200, 0}, Value: c.Voltage, Unit: "V", Model: "ideal-dc.v1"},
		{ID: "R1", Kind: "resistor", Name: "Resistor", Terminals: []string{"1", "2"}, Position: Point{0, -110}, Value: c.Resistance, Unit: "Ω", Model: "ideal-resistor.v1"},
		{ID: "D1", Kind: "led", Name: "Red LED", Terminals: []string{"anode", "cathode"}, Position: Point{200, 0}, Model: ModelID},
		{ID: "GND", Kind: "ground", Name: "Ground", Terminals: []string{"reference"}, Position: Point{0, 110}, Model: "ground.v1"},
	}
}
func Wires(c Circuit) []Wire {
	return []Wire{
		{"W1", "V1.positive", "R1.1", true}, {"W2", "R1.2", "D1.anode", c.Connected},
		{"W3", "D1.cathode", "GND.reference", true}, {"W4", "GND.reference", "V1.negative", true},
	}
}

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	ObjectID string `json:"object_id"`
}
type Result struct {
	RunID              uint64             `json:"run_id"`
	ElectricalRevision uint64             `json:"electrical_revision"`
	Engine             string             `json:"engine"`
	Model              LEDModel           `json:"model"`
	Current            float64            `json:"current_a"`
	LEDVoltage         float64            `json:"led_voltage_v"`
	ResistorVoltage    float64            `json:"resistor_voltage_v"`
	LEDPower           float64            `json:"led_power_w"`
	ResistorPower      float64            `json:"resistor_power_w"`
	KVLResidual        float64            `json:"kvl_residual_v"`
	Nodes              map[string]float64 `json:"node_voltage_v"`
	Stale              bool               `json:"stale"`
	SourceID           string             `json:"source_id"`
	ResistorID         string             `json:"resistor_id"`
	LEDID              string             `json:"led_id"`
}

// SolveLED solves V=I*(R+Rs)+N*Vt*log1p(I/Is). The positive branch is
// monotonic, so bisection is bounded and cannot diverge like unguarded Newton.
// This engine deliberately supports only the curated series LED DC circuit.
func SolveLED(ctx context.Context, c Circuit) (Result, error) {
	if !c.Connected {
		return Result{}, fmt.Errorf("connect W2 before solving")
	}
	if err := validateCircuit(c); err != nil {
		return Result{}, err
	}
	m := Model()
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	lo, hi := 0., c.Voltage/(c.Resistance+m.SeriesResistance)
	for i := 0; i < 100; i++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		current := (lo + hi) / 2
		total := current*(c.Resistance+m.SeriesResistance) + m.EmissionCoefficient*vt*math.Log1p(current/m.SaturationCurrent)
		if total > c.Voltage {
			hi = current
		} else {
			lo = current
		}
	}
	current := (lo + hi) / 2
	led := m.EmissionCoefficient*vt*math.Log1p(current/m.SaturationCurrent) + current*m.SeriesResistance
	resistor := current * c.Resistance
	return Result{Engine: EngineID, Model: m, Current: current, LEDVoltage: led, ResistorVoltage: resistor, LEDPower: current * led, ResistorPower: current * resistor, KVLResidual: c.Voltage - led - resistor, Nodes: map[string]float64{"ground": 0, "supply": c.Voltage, "led-anode": led}}, nil
}
func validateCircuit(c Circuit) error {
	if !finite(c.Voltage) || c.Voltage < 0 || c.Voltage > 12 {
		return fmt.Errorf("source voltage must be 0–12 V")
	}
	if !finite(c.Resistance) || c.Resistance < 10 || c.Resistance > 100000 {
		return fmt.Errorf("resistance must be 10–100000 Ω")
	}
	return nil
}
func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// ParseValue accepts explicit units and common maker resistance suffixes.
func ParseValue(text, unit string) (float64, error) {
	s := strings.TrimSpace(text)
	if unit == "V" {
		s = strings.TrimSpace(strings.TrimSuffix(s, "V"))
	} else if unit == "F" {
		s = strings.TrimSpace(strings.TrimSuffix(s, "F"))
	} else {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(s, "Ω"), "ohm"))
	}
	scale := 1.
	if unit == "Ω" && len(s) > 0 {
		switch s[len(s)-1] {
		case 'k', 'K':
			scale = 1e3
			s = s[:len(s)-1]
		case 'M':
			scale = 1e6
			s = s[:len(s)-1]
		}
	}
	if unit == "F" {
		for suffix, factor := range map[string]float64{"p": 1e-12, "n": 1e-9, "u": 1e-6, "µ": 1e-6, "μ": 1e-6, "m": 1e-3} {
			if strings.HasSuffix(s, suffix) {
				s = strings.TrimSpace(strings.TrimSuffix(s, suffix))
				scale = factor
				break
			}
		}
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || !finite(n) {
		return 0, fmt.Errorf("enter a finite value in %s", unit)
	}
	return n * scale, nil
}
