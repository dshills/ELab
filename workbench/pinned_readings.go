package workbench

import (
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

func (s *Service) pinnedReadingsLocked() []PinnedReading {
	out := []PinnedReading{}
	if len(s.pins) == 0 {
		return out
	}
	context := s.readingContextLocked()
	for _, pin := range s.pins {
		out = append(out, context.reading(pin.ID).pinned(pin))
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
	// Keep missing-source preferences portable using the same physical pin
	// names as the library. Present owners still require an exact probe match.
	for _, part := range libraryParts() {
		for _, terminal := range part.Terminals {
			if parts[1] == terminal {
				return true
			}
		}
	}
	// These voltage differences are probes rather than physical terminals.
	return strings.HasPrefix(id, "v:") && (parts[1] == "gate-source" || parts[1] == "input-difference" || parts[1] == "supply")
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
