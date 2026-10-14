package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
)

type DisplayPreferences struct {
	VoltageColor  bool    `json:"voltage_color"`
	VoltageRange  float64 `json:"voltage_range_v"`
	CurrentCues   bool    `json:"current_cues"`
	CurrentMotion bool    `json:"current_motion"`
	ReducedMotion bool    `json:"reduced_motion"`
	Direction     string  `json:"direction"`
}

func DefaultDisplayPreferences() DisplayPreferences {
	return DisplayPreferences{VoltageRange: 5, Direction: "conventional"}
}
func decodeDisplayPreferences(raw json.RawMessage) (DisplayPreferences, error) {
	var p DisplayPreferences
	if err := params(raw, &p, "voltage_color", "voltage_range_v", "current_cues", "current_motion", "reduced_motion", "direction"); err != nil {
		return p, err
	}
	if !finite(p.VoltageRange) || p.VoltageRange < .01 || p.VoltageRange > 30 {
		return p, fmt.Errorf("voltage color range must be 0.01–30 V")
	}
	if p.Direction != "conventional" && p.Direction != "electron" {
		return p, fmt.Errorf("direction must be conventional or electron; signed measurements always use conventional current")
	}
	if p.CurrentMotion && !p.CurrentCues {
		return p, fmt.Errorf("enable current cues before requesting current motion")
	}
	return p, nil
}
func preferencesHash(pins []PinSpec, display DisplayPreferences, exampleID string) [32]byte {
	data, _ := json.Marshal(struct {
		ExampleID string
		Pins      []PinSpec
		Display   DisplayPreferences
	}{exampleID, clonePins(pins), display})
	return sha256.Sum256(data)
}

type WireDisplay struct {
	ID            string   `json:"id"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	Voltage       *float64 `json:"voltage_v,omitempty"`
	Current       *float64 `json:"current_a,omitempty"`
	State         string   `json:"state"`
	Reason        string   `json:"reason,omitempty"`
	CurrentReason string   `json:"current_reason,omitempty"`
	Color         string   `json:"color"`
}
type CircuitDisplay struct {
	Preferences        DisplayPreferences `json:"preferences"`
	RunID              uint64             `json:"run_id"`
	ElectricalRevision uint64             `json:"electrical_revision"`
	Time               float64            `json:"time_s"`
	Animate            bool               `json:"animate"`
	Wires              []WireDisplay      `json:"wires"`
}

func cloneCircuitDisplay(in CircuitDisplay) CircuitDisplay {
	out := in
	out.Wires = append([]WireDisplay{}, in.Wires...)
	for i := range out.Wires {
		for _, value := range []**float64{&out.Wires[i].Voltage, &out.Wires[i].Current} {
			if *value != nil {
				v := **value
				*value = &v
			}
		}
	}
	return out
}
func voltageColor(value *float64, state string, p DisplayPreferences) string {
	if !p.VoltageColor {
		return "#387d93"
	}
	if value == nil || state != "live" {
		return "#86969f"
	}
	neutral := [3]float64{112, 132, 142}
	target := [3]float64{213, 106, 57}
	if *value < 0 {
		target = [3]float64{74, 109, 199}
	}
	u := math.Min(1, math.Abs(*value)/p.VoltageRange)
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(neutral[0]+u*(target[0]-neutral[0]))), int(math.Round(neutral[1]+u*(target[1]-neutral[1]))), int(math.Round(neutral[2]+u*(target[2]-neutral[2]))))
}
func (s *Service) circuitDisplayLocked() CircuitDisplay {
	p := s.displayPreferences
	out := CircuitDisplay{Preferences: p, Wires: []WireDisplay{}}
	if len(s.graph.Wires) == 0 {
		return out
	}
	context := s.readingContextLocked()
	out.RunID, out.ElectricalRevision, out.Time = context.run, context.revision, context.sample.Time
	flows := uniqueWireCurrents(s.graph, context)
	for _, w := range s.graph.Wires {
		r := context.reading("v:" + w.From)
		v := WireDisplay{ID: w.ID, From: w.From, To: w.To, Voltage: r.Value, State: r.State, Reason: r.Reason}
		if !w.Connected {
			v.Voltage = nil
			v.State = "unavailable"
			v.Reason = "Wire is not electrically connected"
		}
		if v.State == "live" {
			if current, ok := flows[w.ID]; ok {
				v.Current = &current
			} else {
				v.CurrentReason = "Ideal-wire current is not uniquely determined or accepted terminal currents do not balance"
			}
		} else {
			v.CurrentReason = "Current direction requires a fresh accepted circuit"
		}
		v.Color = voltageColor(v.Voltage, v.State, p)
		if p.CurrentCues && p.CurrentMotion && !p.ReducedMotion && !s.paused && s.status == "running" && v.Current != nil && math.Abs(*v.Current) > 1e-12 {
			out.Animate = true
		}
		out.Wires = append(out.Wires, v)
	}
	return out
}
