package workbench

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SearchParams struct {
	Query string `json:"query"`
}
type DuplicateParams struct {
	ID       string `json:"id"`
	Position *Point `json:"position,omitempty"`
}
type BatchEdit struct {
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params"`
}
type BatchParams struct {
	Name  string      `json:"name"`
	Edits []BatchEdit `json:"edits"`
}

func filteredParts(query string) []Part {
	out := []Part{}
	query = strings.ToLower(strings.TrimSpace(query))
	for _, part := range Library() {
		data, _ := json.Marshal(part)
		if strings.Contains(strings.ToLower(string(data)), query) {
			out = append(out, part)
		}
	}
	return out
}
func (s *Service) duplicateLocked(p DuplicateParams) error {
	part, ok := s.graph.Component(p.ID)
	if !ok {
		return fmt.Errorf("choose an existing component to duplicate")
	}
	position := p.Position

	if position == nil {
	candidates:
		for offset := 40.; offset <= 400; offset += 20 {
			for _, direction := range []Point{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				point := Point{part.Position.X + offset*direction.X, part.Position.Y + offset*direction.Y}
				if _, err := snapPoint(point); err != nil {
					continue
				}
				occupied := false
				for _, other := range s.graph.Components {
					if other.Position == point {
						occupied = true
						break
					}
				}
				if !occupied {
					position = &point
					break candidates
				}
			}
		}
	}
	if position == nil {
		return fmt.Errorf("choose an unoccupied duplicate position")
	}
	value := ""
	if part.Kind == "net-label" {
		value = part.Label
	}
	if part.Unit != "" {
		value = fmt.Sprintf("%.17g %s", part.Value, part.Unit)
	}
	raw, _ := json.Marshal(PlaceParams{Kind: part.Kind, Position: *position, Value: value})
	if err := s.applyLocked("component.place", raw); err != nil {
		return err
	}
	for i := range s.graph.Components {
		if s.graph.Components[i].ID == s.selected {
			s.graph.Components[i].Rotation = part.Rotation
			if part.SwitchClosed != nil {
				copy := *part.SwitchClosed
				s.graph.Components[i].SwitchClosed = &copy
			}
			if part.InitialVoltage != nil {
				v := *part.InitialVoltage
				s.graph.Components[i].InitialVoltage = &v
			}
			break
		}
	}
	return nil
}
func (s *Service) batchLocked(p BatchParams) error {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 80 || len(p.Edits) == 0 || len(p.Edits) > 100 {
		return fmt.Errorf("batch requires a name of 1–80 bytes and 1–100 edits")
	}
	staged := &Service{graph: s.graph.Clone(), drawing: s.drawing, selected: s.selected, usedIDs: map[string]bool{}, transientSettings: s.transientSettings}
	for id, used := range s.usedIDs {
		staged.usedIDs[id] = used
	}
	for i, edit := range p.Edits {
		switch edit.Command {
		case "component.place", "component.move", "component.rotate", "component.set", "component.duplicate", "capacitor.initial.set", "object.delete", "wire.connect", "wire.junction.insert", "switch.set":
		default:
			return fmt.Errorf("batch edit %d: command %q is not an atomic schematic edit", i+1, edit.Command)
		}
		if err := staged.applyLocked(edit.Command, edit.Params); err != nil {
			return fmt.Errorf("batch edit %d (%s): %w", i+1, edit.Command, err)
		}
	}
	s.graph = staged.graph
	s.usedIDs = staged.usedIDs
	s.selected = staged.selected
	s.drawing = staged.drawing
	s.nextHistoryName = p.Name
	return nil
}
func historyName(command string) string {
	names := map[string]string{"component.place": "Place component", "component.duplicate": "Duplicate component", "component.move": "Move component", "component.rotate": "Rotate component", "component.set": "Change value", "capacitor.initial.set": "Change capacitor startup", "wire.connect": "Connect wire", "wire.junction.insert": "Insert junction", "object.delete": "Delete object", "simulation.transient.set": "Change experiment settings", "project.clear": "New schematic", "project.load": "Open schematic"}
	if name := names[command]; name != "" {
		return name
	}
	return strings.ReplaceAll(command, ".", " ")
}
