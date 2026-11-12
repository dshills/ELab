package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

type SheetInstanceParams struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Parent   string `json:"parent,omitempty"`
	Template string `json:"template"`
}

func validTemplateHash(hash string) bool {
	if len(hash) != 2*sha256.Size {
		return false
	}
	for _, digit := range hash {
		if !(digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f') {
			return false
		}
	}
	return true
}

// Keep an instance's changed electrical settings when the source is applied.
// Positions, rotation and block membership belong to the shared drawing.
func templateSettingsHash(part Component) string {
	part.ID, part.Sheet, part.TemplatePart, part.TemplateHash = "", "", "", ""
	part.Position, part.Rotation, part.Block = Point{}, 0, nil
	raw, _ := json.Marshal(part)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func (s *Service) sheetInstanceCommandLocked(command string, raw json.RawMessage) error {
	staged := &Service{graph: s.graph.Clone(), usedIDs: maps.Clone(s.usedIDs)}
	active, history := s.activeSheet, "Apply source sheet changes"
	var instances []Sheet
	if command == "sheet.instance.create" {
		var p SheetInstanceParams
		if err := params(raw, &p, "id", "name", "template"); err != nil {
			return err
		}
		if p.Template == "" {
			return fmt.Errorf("choose an existing child sheet as the source")
		}
		instance := Sheet(p)
		staged.graph.Sheets = append(staged.graph.Sheets, instance)
		if err := staged.graph.validateSheets(); err != nil {
			return err
		}
		instances = []Sheet{instance}
		active, history = p.ID, "Place linked sheet"
	} else {
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		for _, instance := range staged.graph.Sheets {
			if instance.Template == p.ID && p.ID != "" {
				instances = append(instances, instance)
			}
		}
		if len(instances) == 0 {
			return fmt.Errorf("this sheet has no linked instances")
		}
	}
	if err := staged.graph.checkInstanceCapacity(instances); err != nil {
		return err
	}
	for _, instance := range instances {
		if err := staged.applySheetInstance(instance); err != nil {
			return err
		}
	}
	if err := staged.graph.validateSheets(); err != nil {
		return err
	}
	if err := staged.graph.validateBlocks(); err != nil {
		return err
	}
	if err := staged.graph.validatePositions(); err != nil {
		return err
	}
	pairs := map[[2]string]bool{}
	for _, wire := range staged.graph.Wires {
		if _, _, ok := staged.graph.Terminal(wire.From); !ok {
			return fmt.Errorf("disconnect wire %s from the changed terminal before applying the source", wire.ID)
		}
		if _, _, ok := staged.graph.Terminal(wire.To); !ok {
			return fmt.Errorf("disconnect wire %s from the changed terminal before applying the source", wire.ID)
		}
		pair := [2]string{wire.From, wire.To}
		if pair[0] > pair[1] {
			pair[0], pair[1] = pair[1], pair[0]
		}
		if pairs[pair] {
			return fmt.Errorf("remove the duplicate instance connection at %s before applying the source", wire.ID)
		}
		pairs[pair] = true
	}
	s.graph, s.usedIDs, s.nextHistoryName = staged.graph, staged.usedIDs, history
	return s.openSheetLocked(active)
}

func (g Graph) checkInstanceCapacity(instances []Sheet) error {
	targets := make(map[string]bool, len(instances))
	for _, instance := range instances {
		targets[instance.ID] = true
	}
	source := g.SheetDrawing(instances[0].Template)
	parts := len(g.Components) + len(instances)*len(source.Components)
	wires := len(g.Wires) + len(instances)*len(source.Wires)
	for _, part := range g.Components {
		if targets[part.Sheet] && part.TemplatePart != "" {
			parts--
		}
	}
	for _, wire := range g.Wires {
		if wire.TemplateWire != "" {
			from, _, _ := g.Terminal(wire.From)
			if targets[from.Sheet] {
				wires--
			}
		}
	}
	if parts > MaxComponents || wires > MaxWires {
		return fmt.Errorf("linked sheets exceed the project drawing limits")
	}
	return nil
}

func (s *Service) applySheetInstance(instance Sheet) error {
	g := &s.graph
	original := g.SheetDrawing(instance.Template).Clone()
	oldParts := map[string]Component{}
	oldWires := map[string]Wire{}
	partIDs := map[string]string{}
	removed := map[string]bool{}
	for _, part := range g.Components {
		if part.Sheet == instance.ID && part.TemplatePart != "" {
			oldParts[part.TemplatePart] = part
			removed[part.ID] = true
		}
	}
	for _, wire := range g.Wires {
		from, _, _ := g.Terminal(wire.From)
		if from.Sheet == instance.ID && wire.TemplateWire != "" {
			oldWires[wire.TemplateWire] = wire
			removed[wire.ID] = true
		}
	}
	for _, part := range original.Components {
		id := oldParts[part.ID].ID
		if id == "" {
			id = s.nextID("SP")
			s.usedIDs[id] = true
		}
		partIDs[part.ID] = id
	}
	rewrite := func(ref string) string {
		id, terminal, _ := strings.Cut(ref, ".")
		return partIDs[id] + "." + terminal
	}
	partChanges := map[string]Component{}
	partOrder := make([]string, 0, len(original.Components))
	for _, source := range original.Components {
		id := partIDs[source.ID]
		old := oldParts[source.ID]
		part, hash := source, templateSettingsHash(source)
		if old.ID != "" && old.Kind == source.Kind && templateSettingsHash(old) != old.TemplateHash {
			part = old
		}
		part.ID, part.Sheet, part.TemplatePart, part.TemplateHash = id, instance.ID, source.ID, hash
		part.Position, part.Rotation, part.Block = source.Position, source.Rotation, source.Block
		if part.Block != nil {
			for i, member := range part.Block.Members {
				part.Block.Members[i] = partIDs[member]
			}
			for i, port := range part.Block.Ports {
				part.Block.Ports[i].Terminal = rewrite(port.Terminal)
			}
			for i, parameter := range part.Block.Parameters {
				part.Block.Parameters[i].ComponentID = partIDs[parameter.ComponentID]
			}
		}
		partChanges[id] = part
		partOrder = append(partOrder, id)
		delete(removed, id)
	}
	wireChanges := map[string]Wire{}
	wireOrder := make([]string, 0, len(original.Wires))
	for _, source := range original.Wires {
		id := oldWires[source.ID].ID
		if id == "" {
			id = s.nextID("W")
			s.usedIDs[id] = true
		}
		wireChanges[id] = Wire{ID: id, From: rewrite(source.From), To: rewrite(source.To), Connected: source.Connected, TemplateWire: source.ID}
		wireOrder = append(wireOrder, id)
		delete(removed, id)
	}
	parts := make([]Component, 0, len(g.Components))
	for _, part := range g.Components {
		if replacement, ok := partChanges[part.ID]; ok {
			part = replacement
			delete(partChanges, part.ID)
		}
		if !removed[part.ID] {
			parts = append(parts, part)
		}
	}
	for _, id := range partOrder {
		if part, ok := partChanges[id]; ok {
			parts = append(parts, part)
		}
	}
	wires := make([]Wire, 0, len(g.Wires))
	for _, wire := range g.Wires {
		if replacement, ok := wireChanges[wire.ID]; ok {
			wire = replacement
			delete(wireChanges, wire.ID)
		}
		from, _, _ := strings.Cut(wire.From, ".")
		to, _, _ := strings.Cut(wire.To, ".")
		if removed[wire.ID] {
			continue
		}
		if removed[from] || removed[to] {
			return fmt.Errorf("disconnect %s on %s before applying its source change", wire.ID, instance.Name)
		}
		wires = append(wires, wire)
	}
	for _, id := range wireOrder {
		if wire, ok := wireChanges[id]; ok {
			wires = append(wires, wire)
		}
	}
	g.Components, g.Wires = parts, wires
	return nil
}
