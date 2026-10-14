package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

const MaxPinnedReadings = 8

// PinSpec is document presentation state; values always come from accepted live
// solver frames and are never trusted from a file.
type PinSpec struct {
	ID       string `json:"id"`
	Position Point  `json:"position"`
}
type PinPreferences struct {
	Items []PinSpec `json:"items"`
}
type PinnedReading struct {
	Definition         PinSpec  `json:"definition"`
	Probe              Probe    `json:"probe"`
	Value              *float64 `json:"value,omitempty"`
	Time               float64  `json:"time_s"`
	RunID              uint64   `json:"run_id"`
	ElectricalRevision uint64   `json:"electrical_revision"`
	State              string   `json:"state"`
	Reason             string   `json:"reason,omitempty"`
	Display            string   `json:"display"`
	ReferenceDisplay   string   `json:"reference_display"`
}

func clonePins(in []PinSpec) []PinSpec { return append([]PinSpec{}, in...) }
func pinHash(in []PinSpec) [32]byte {
	data, _ := json.Marshal(clonePins(in))
	return sha256.Sum256(data)
}
func clonePinnedReadings(in []PinnedReading) []PinnedReading {
	out := append([]PinnedReading{}, in...)
	for i := range out {
		if out[i].Value != nil {
			v := *out[i].Value
			out[i].Value = &v
		}
	}
	return out
}
func (s *Service) pinCommandLocked(command string, raw json.RawMessage) error {
	if command == "measurement.pin.remove" {
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		for i, pin := range s.pins {
			if pin.ID == p.ID {
				s.pins = append(clonePins(s.pins[:i]), s.pins[i+1:]...)
				return nil
			}
		}
		return &actionError{"not-found", "Pinned reading not found"}
	}
	var p PinSpec
	var previous *Point
	if command == "measurement.pin.move" {
		var move MoveParams
		if err := params(raw, &move, "id", "position"); err != nil {
			return err
		}
		p = PinSpec{ID: move.ID, Position: move.Position}
		previous = move.PreviousPosition
		if previous != nil {
			if err := validatePointParam(raw, "previous_position"); err != nil {
				return err
			}
		}
	} else if err := params(raw, &p, "id", "position"); err != nil {
		return err
	}
	if err := validatePointParam(raw, "position"); err != nil {
		return err
	}
	position, err := snapPoint(p.Position)
	if err != nil {
		return err
	}
	p.Position = position
	for i, pin := range s.pins {
		if pin.ID != p.ID {
			continue
		}
		if command == "measurement.pin.add" {
			return fmt.Errorf("this probe is already pinned; move its reading instead")
		}
		if previous != nil && *previous != pin.Position {
			return &actionError{"conflict", "Pinned reading moved during this gesture; inspect and try again"}
		}
		s.pins[i] = p
		return nil
	}
	if command == "measurement.pin.move" {
		return &actionError{"not-found", "Pinned reading not found"}
	}
	if len(s.pins) >= MaxPinnedReadings {
		return fmt.Errorf("at most %d readings can be pinned", MaxPinnedReadings)
	}
	for _, probe := range probeInventory(s.graph) {
		if probe.ID == p.ID {
			s.pins = append(s.pins, p)
			return nil
		}
	}
	return &actionError{"not-found", "Choose a current probe from scope.available"}
}

// The compiled graph paired with a DC result owns its node associations. Never
// interpret old node voltages using a new/incomplete graph after an edit.
func (s *Service) readingFrameLocked() (TraceSample, uint64, uint64, bool) {
	if s.transient != nil {
		t := s.transient
		return t.Sample, t.RunID, t.ElectricalRevision, t.Stale
	}
	if s.mixed != nil && s.mixedState != nil {
		r := s.mixed
		return mixedSample(s.mixedState.net.c, r, 0), r.RunID, r.ElectricalRevision, r.Stale
	}
	if s.dcCompiled != nil {
		c := *s.dcCompiled
		if s.npn != nil {
			r := s.npn
			return npnTraceSample(c, r, r, 0, c.Input.Value), r.RunID, r.ElectricalRevision, r.Stale
		}
		if s.result != nil {
			r := s.result
			return ledTraceSample(c, *r), r.RunID, r.ElectricalRevision, r.Stale
		}
	}
	return TraceSample{}, 0, 0, true
}
func ledTraceSample(c Compiled, r Result) TraceSample {
	currents := map[string]float64{c.Source.ID: r.Current, c.LED.ID: r.Current, c.Resistor.ID: (r.Nodes[c.TerminalNet[c.Resistor.ID+".1"]] - r.Nodes[c.TerminalNet[c.Resistor.ID+".2"]]) / c.Resistor.Value}
	return captureFrame(c, TraceSample{Input: c.Source.Value, Voltage: r.LEDVoltage, Current: r.Current}, r.Nodes, currents)
}
func (s *Service) pinnedReadingsLocked() []PinnedReading {
	out := []PinnedReading{}
	if len(s.pins) == 0 {
		return out
	}
	available := map[string]Probe{}
	for _, p := range probeInventory(s.graph) {
		available[p.ID] = p
	}
	sample, run, revision, stale := s.readingFrameLocked()
	stale = stale || run != s.run || revision != s.electrical
	for _, pin := range s.pins {
		r := PinnedReading{Definition: pin, Time: sample.Time, RunID: run, ElectricalRevision: revision, State: "unavailable"}
		p, exists := available[pin.ID]
		if !exists {
			r.Probe = Probe{ID: pin.ID, Label: pin.ID, Unit: "V"}
			if strings.HasPrefix(pin.ID, "i:") {
				r.Probe.Unit = "A"
			}
			r.Reason = "Source removed; undo restores it or remove this pin"
		} else {
			r.Probe = p
			if v, ok := frameValue(sample, p); ok {
				r.Value = &v
				r.State = "live"
				if stale {
					r.State = "stale"
					r.Reason = "Previous accepted live result; resume or restart a valid circuit"
				}
			} else {
				r.Reason = "No accepted live result for this probe"
			}
		}
		r.ReferenceDisplay = "Source unavailable"
		if exists {
			r.ReferenceDisplay = "Voltage: " + p.Positive + " - " + p.Reference
			if p.Unit == "A" {
				r.ReferenceDisplay = "Current: " + p.Positive + " -> " + p.Reference
			}
		}
		r.Display = "Live " + pin.ID + " · unavailable"
		if r.Value != nil {
			value, unit := *r.Value, p.Unit
			if unit == "A" {
				value *= 1000
				unit = "mA"
			}
			r.Display = fmt.Sprintf("Live %s · %.5g %s", pin.ID, value, unit)
			if r.State == "stale" {
				r.Display += " · stale"
			}
		}
		out = append(out, r)
	}
	return out
}

func validatePinPreferences(raw json.RawMessage, g Graph) (*PinPreferences, error) {
	var p PinPreferences
	if err := params(raw, &p, "items"); err != nil {
		return nil, err
	}
	if p.Items == nil || len(p.Items) > MaxPinnedReadings {
		return nil, fmt.Errorf("pinned_readings requires an array of at most eight items")
	}
	var fields struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(raw, &fields)
	available := map[string]bool{}
	for _, probe := range probeInventory(g) {
		available[probe.ID] = true
	}
	seen := map[string]bool{}
	for i, pin := range p.Items {
		if err := params(fields.Items[i], &PinSpec{}, "id", "position"); err != nil {
			return nil, err
		}
		if err := validatePointParam(fields.Items[i], "position"); err != nil {
			return nil, err
		}
		position, err := snapPoint(pin.Position)
		if err != nil || position != pin.Position || seen[pin.ID] || !validPinnedProbeID(pin.ID) {
			return nil, fmt.Errorf("invalid or duplicate pinned reading %q", pin.ID)
		}
		owner := strings.Split(pin.ID[2:], ".")[0]
		if _, exists := g.Component(owner); exists && !available[pin.ID] {
			return nil, fmt.Errorf("pinned reading references an unknown probe %q", pin.ID)
		}
		seen[pin.ID] = true
	}
	return &p, nil
}
func validPinnedProbeID(id string) bool {
	if !strings.HasPrefix(id, "v:") && !strings.HasPrefix(id, "i:") {
		return false
	}
	parts := strings.Split(id[2:], ".")
	if !ValidID(parts[0]) || len(parts) > 2 {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	switch parts[1] {
	case "1", "2", "positive", "negative", "anode", "cathode", "base", "collector", "emitter", "wiper", "reference", "node":
		return true
	}
	return false
}

// PinnedReadingBounds is shared by native drawing/hit testing, Fit and export.
// Position is the upper-left of a card; metrics match the retained bitmap font.
func PinnedReadingBounds(r PinnedReading) (Point, Point) {
	length := max(len([]rune(r.Display)), len([]rune(r.ReferenceDisplay)))
	width := 24 + float64(length)*8.75
	if width < 180 {
		width = 180
	}
	return r.Definition.Position, Point{X: r.Definition.Position.X + width, Y: r.Definition.Position.Y + 54}
}
