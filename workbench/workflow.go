package workbench

import (
	"encoding/json"
	"fmt"
	"slices"
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

// One allowlist serves staged validation and on-demand agent discovery.
var atomicBatchCommands = [...]string{
	"component.place", "component.move", "component.rotate", "component.set", "component.duplicate",
	"capacitor.initial.set", "inductor.initial.set", "object.delete", "wire.connect", "wire.junction.insert",
	"transformer.configure", "optocoupler.configure", "pwm.configure", "opamp.dynamics.set", "sound.configure", "motor.configure", "motor.load.set", "battery.configure", "sensor.temperature.set", "sensor.light.set", "switch.set", "potentiometer.wiper.set", "source.sine.set", "source.pulse.set",
	"source.pulse.timing.set", "simulation.pulse.timing.set", "simulation.sampling.fit",
}

func filteredParts(query string) []Part {
	if strings.TrimSpace(query) == "" {
		return Library()
	}
	out := []Part{}
	query = strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(query)
	for _, part := range Library() {
		searchable := strings.ToLower(strings.Join(append([]string{part.Kind, part.Name, part.Model, part.Unit, part.DefaultValue, strings.Join(part.Terminals, " ")}, part.Aliases...), " "))
		matches := true
		for _, token := range tokens {
			if !strings.Contains(searchable, token) {
				matches = false
				break
			}
		}
		if matches {
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
	if part.Unit != "" && !IsWaveformSource(part.Kind) {
		value = fmt.Sprintf("%.17g %s", part.Value, part.Unit)
	}
	raw, _ := json.Marshal(PlaceParams{Kind: part.Kind, Position: *position, Value: value})
	if err := s.applyLocked("component.place", raw); err != nil {
		return err
	}
	for i := range s.graph.Components {
		if s.graph.Components[i].ID == s.selected {
			s.graph.Components[i].Rotation = part.Rotation
			s.graph.Components[i].Power = clonePower(part.Power)
			s.graph.Components[i].Device = cloneDeviceParameters(part.Device)
			if IsSensorKind(part.Kind) {
				setSensorEnvironment(&s.graph.Components[i], sensorEnvironment(part))
			}
			s.graph.Components[i].Value = part.Value
			if part.Sine != nil {
				w := *part.Sine
				s.graph.Components[i].Sine = &w
			}
			if part.Pulse != nil {
				w := *part.Pulse
				s.graph.Components[i].Pulse = &w
			}
			if part.WiperPosition != nil {
				v := *part.WiperPosition
				s.graph.Components[i].WiperPosition = &v
			}
			if part.SwitchClosed != nil {
				copy := *part.SwitchClosed
				s.graph.Components[i].SwitchClosed = &copy
			}
			if part.InitialCurrent != nil {
				v := *part.InitialCurrent
				s.graph.Components[i].InitialCurrent = &v
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
func (s *Service) stageBatchLocked(p BatchParams) (*Service, error) {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 80 || len(p.Edits) == 0 || len(p.Edits) > 100 {
		return nil, fmt.Errorf("batch requires a name of 1–80 bytes and 1–100 edits")
	}
	staged := &Service{graph: s.graph.Clone(), drawing: s.drawing, selected: s.selected, usedIDs: map[string]bool{}, transientSettings: s.transientSettings}
	for id, used := range s.usedIDs {
		staged.usedIDs[id] = used
	}
	for i, edit := range p.Edits {
		if !slices.Contains(atomicBatchCommands[:], edit.Command) {
			return nil, fmt.Errorf("batch edit %d: command %q is not an atomic schematic edit", i+1, edit.Command)
		}
		if err := staged.applyLocked(edit.Command, edit.Params); err != nil {
			return nil, fmt.Errorf("batch edit %d (%s): %w", i+1, edit.Command, err)
		}
	}
	// Final graph equality can suppress outer document-change normalization.
	// Still retire selection/gesture references to staged objects now deleted.
	staged.normalizeDrawingLocked()
	return staged, nil
}

func (s *Service) batchLocked(p BatchParams) error {
	staged, err := s.stageBatchLocked(p)
	if err != nil {
		return err
	}
	s.graph = staged.graph
	s.transientSettings = staged.transientSettings
	s.usedIDs = staged.usedIDs
	s.selected = staged.selected
	s.drawing = staged.drawing
	s.nextHistoryName = p.Name
	return nil
}
func historyName(command string) string {
	if command == "simulation.sampling.fit" {
		return "Fit sampling"
	}
	if command == "simulation.pulse.timing.set" {
		return "Change shared pulse timing"
	}
	if command == "source.pulse.timing.set" {
		return "Change pulse timing"
	}
	names := map[string]string{"example.load": "Load example", "example.current.reset": "Reset current example", "display.preferences.set": "Change display preferences", "measurement.pin.add": "Pin reading", "measurement.pin.move": "Move pinned reading", "measurement.pin.remove": "Remove pinned reading", "component.place": "Place component", "component.duplicate": "Duplicate component", "component.move": "Move component", "component.rotate": "Rotate component", "component.set": "Change value", "capacitor.initial.set": "Change capacitor startup", "inductor.initial.set": "Change inductor startup", "wire.connect": "Connect wire", "wire.junction.insert": "Insert junction", "object.delete": "Delete object", "simulation.transient.set": "Change experiment settings", "project.clear": "New schematic", "project.load": "Open schematic"}
	if name := names[command]; name != "" {
		return name
	}
	return strings.ReplaceAll(command, ".", " ")
}
