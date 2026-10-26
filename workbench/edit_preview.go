package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type EditChange struct {
	Target    string          `json:"target"`
	ID        string          `json:"id"`
	Operation string          `json:"operation"`
	Summary   string          `json:"summary"`
	Details   []string        `json:"details"`
	Before    json.RawMessage `json:"before,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
}

type BatchPreview struct {
	Name               string       `json:"name"`
	CommandRevision    uint64       `json:"command_revision"`
	DocumentGeneration uint64       `json:"document_generation"`
	Changes            []EditChange `json:"changes"`
	Notice             string       `json:"notice"`
}

func cloneEditChanges(in []EditChange) []EditChange {
	out := append([]EditChange{}, in...)
	for i := range out {
		out[i].Before = append(json.RawMessage(nil), in[i].Before...)
		out[i].After = append(json.RawMessage(nil), in[i].After...)
		out[i].Details = append([]string{}, in[i].Details...)
	}
	return out
}

func schematicChanges(before, after Graph, oldSettings, newSettings TransientSettings) []EditChange {
	changes := []EditChange{}
	appendChange := func(target, id, summary string, a, b any) {
		if reflect.DeepEqual(a, b) {
			return
		}
		change := EditChange{Target: target, ID: id, Operation: "changed", Summary: summary, Details: []string{}}
		if a != nil {
			change.Before, _ = json.Marshal(a)
		} else {
			change.Operation = "added"
		}
		if b != nil {
			change.After, _ = json.Marshal(b)
		} else {
			change.Operation = "removed"
		}
		if a != nil && b != nil {
			change.Details = changedParameterDetails(change.Before, change.After, target)
		} else if added, ok := b.(Component); ok {
			if factory, err := MakeComponent(added.ID, added.Kind, added.Position, ""); err == nil {
				defaults, _ := json.Marshal(factory)
				change.Details = changedParameterDetails(defaults, change.After, target)
			}
		}
		changes = append(changes, change)
	}
	oldParts, newParts := map[string]Component{}, map[string]Component{}
	ids := []string{}
	for _, part := range before.Components {
		oldParts[part.ID] = part
		ids = append(ids, part.ID)
	}
	for _, part := range after.Components {
		newParts[part.ID] = part
		if _, exists := oldParts[part.ID]; !exists {
			ids = append(ids, part.ID)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		a, had := oldParts[id]
		b, has := newParts[id]
		switch {
		case !had:
			summary := fmt.Sprintf("Add %s · %s at %.0f, %.0f", id, b.Name, b.Position.X, b.Position.Y)
			if b.Unit != "" {
				summary += " · " + FormatValue(b.Value, b.Unit)
			}
			if b.Rotation != 0 {
				summary += fmt.Sprintf(" · rotation %d°", b.Rotation)
			}
			appendChange("component", id, summary, nil, b)
		case !has:
			appendChange("component", id, "Remove "+id+" · "+a.Name, a, nil)
		default:
			details := []string{}
			if a.Value != b.Value {
				details = append(details, "from "+FormatValue(a.Value, a.Unit)+" to "+FormatValue(b.Value, b.Unit))
			}
			if a.Position != b.Position {
				details = append(details, fmt.Sprintf("position from %.0f, %.0f to %.0f, %.0f", a.Position.X, a.Position.Y, b.Position.X, b.Position.Y))
			}
			if a.Rotation != b.Rotation {
				details = append(details, fmt.Sprintf("rotation from %d° to %d°", a.Rotation, b.Rotation))
			}
			if len(details) == 0 {
				details = append(details, "configuration changed")
			}
			appendChange("component", id, id+": "+strings.Join(details, "; "), a, b)
		}
	}
	oldWires, newWires := map[string]Wire{}, map[string]Wire{}
	ids = nil
	for _, wire := range before.Wires {
		oldWires[wire.ID] = wire
		ids = append(ids, wire.ID)
	}
	for _, wire := range after.Wires {
		newWires[wire.ID] = wire
		if _, exists := oldWires[wire.ID]; !exists {
			ids = append(ids, wire.ID)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		a, had := oldWires[id]
		b, has := newWires[id]
		switch {
		case !had:
			appendChange("wire", id, "Connect "+b.From+" to "+b.To, nil, b)
		case !has:
			appendChange("wire", id, "Remove wire from "+a.From+" to "+a.To, a, nil)
		default:
			appendChange("wire", id, "Change wire "+id, a, b)
		}
	}
	appendChange("settings", "transient", "Change transient simulation settings", oldSettings, newSettings)
	return changes
}

// Surface nested device settings as actual before/after values, rather than
// asking people to approve an unexplained "configuration changed" summary.
func changedParameterDetails(before, after json.RawMessage, target string) []string {
	var a, b any
	_ = json.Unmarshal(before, &a)
	_ = json.Unmarshal(after, &b)
	out := []string{}
	var visit func(string, any, any)
	visit = func(path string, a, b any) {
		if reflect.DeepEqual(a, b) {
			return
		}
		am, aMap := a.(map[string]any)
		bm, bMap := b.(map[string]any)
		if aMap || bMap {
			keys := []string{}
			for key := range am {
				keys = append(keys, key)
			}
			for key := range bm {
				if _, ok := am[key]; !ok {
					keys = append(keys, key)
				}
			}
			slices.Sort(keys)
			for _, key := range keys {
				next := key
				if path != "" {
					next = path + " / " + key
				}
				visit(next, am[key], bm[key])
			}
			return
		}
		// These component changes already have concise unit-aware summaries.
		if target == "component" && (path == "value" || path == "rotation" || strings.HasPrefix(path, "position / ")) {
			return
		}
		oldValue, _ := json.Marshal(a)
		newValue, _ := json.Marshal(b)
		out = append(out, strings.ReplaceAll(path, "_", " ")+": from "+string(oldValue)+" to "+string(newValue))
	}
	visit("", a, b)
	return out
}

func (s *Service) previewBatchLocked(p BatchParams) (*BatchPreview, error) {
	staged, err := s.stageBatchLocked(p)
	if err != nil {
		return nil, err
	}
	return &BatchPreview{Name: p.Name, CommandRevision: s.commandRevision, DocumentGeneration: s.documentGeneration, Changes: schematicChanges(s.graph, staged.graph, s.transientSettings, staged.transientSettings), Notice: "Preview only; no simulation was run. Applying may restart the circuit or mark paused readings stale."}, nil
}
