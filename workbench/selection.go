package workbench

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SelectionIDs struct {
	IDs []string `json:"ids"`
}
type SelectionOffset struct {
	IDs    []string `json:"ids"`
	Offset Point    `json:"offset"`
}
type SelectionPaste struct {
	ClipboardID uint64 `json:"clipboard_id"`
	Offset      Point  `json:"offset"`
}

// Selection is explicit command input, not another document authority. Browser
// selection stays local; every committed operation is one ordinary history edit.
func (s *Service) selectedGraphLocked(ids []string) (Graph, error) {
	if len(ids) == 0 || len(ids) > MaxComponents {
		return Graph{}, fmt.Errorf("select 1–%d components", MaxComponents)
	}
	chosen := map[string]bool{}
	for _, id := range ids {
		if chosen[id] {
			return Graph{}, fmt.Errorf("selection contains duplicate IDs")
		}
		if part, ok := s.graph.Component(id); !ok || part.Sheet != s.activeSheet {
			return Graph{}, fmt.Errorf("unknown selected component %q", id)
		}
		if s.graph.blockOwner(id) != "" {
			return Graph{}, fmt.Errorf("select the owning block instead of a hidden member")
		}
		chosen[id] = true
	}
	var include func(string)
	include = func(id string) {
		chosen[id] = true
		p, _ := s.graph.Component(id)
		if p.Block != nil {
			for _, member := range p.Block.Members {
				if !chosen[member] {
					include(member)
				}
			}
		}
	}
	for _, id := range ids {
		include(id)
	}
	graph := EmptyGraph()
	for _, p := range s.graph.Clone().Components {
		if chosen[p.ID] {
			graph.Components = append(graph.Components, p)
		}
	}
	for _, w := range s.graph.Wires {
		a, _, _ := strings.Cut(w.From, ".")
		b, _, _ := strings.Cut(w.To, ".")
		if chosen[a] && chosen[b] {
			graph.Wires = append(graph.Wires, w)
		}
	}
	return graph, nil
}
func (s *Service) moveSelectionLocked(p SelectionOffset) error {
	selected, err := s.selectedGraphLocked(p.IDs)
	if err != nil {
		return err
	}
	delta, err := snapPoint(p.Offset)
	if err != nil {
		return err
	}
	chosen := map[string]bool{}
	for _, c := range selected.Components {
		chosen[c.ID] = true
	}
	graph := s.graph.Clone()
	for i, c := range graph.Components {
		if chosen[c.ID] {
			position, err := snapPoint(Point{c.Position.X + delta.X, c.Position.Y + delta.Y})
			if err != nil {
				return err
			}
			graph.Components[i].Position = position
		}
	}
	if err = graph.validatePositions(); err != nil {
		return err
	}
	s.graph = graph
	s.selected = p.IDs[0]
	s.nextHistoryName = "Move selection"
	return nil
}
func copyPrefix(id string) string {
	prefix := ""
	for _, c := range id {
		if c < 'A' || c > 'Z' && c < 'a' || c > 'z' {
			break
		}
		prefix += string(c)
		if len(prefix) >= 40 {
			break
		}
	}
	if prefix == "" {
		return "COPY"
	}
	return prefix
}
func (s *Service) pasteSelectionLocked(source Graph, offset Point) error {
	delta, err := snapPoint(offset)
	if err != nil {
		return err
	}
	if len(s.graph.Components)+len(source.Components) > MaxComponents || len(s.graph.Wires)+len(source.Wires) > MaxWires {
		return fmt.Errorf("pasted selection exceeds drawing limits")
	}
	graph := s.graph.Clone()
	used := map[string]bool{}
	for id, v := range s.usedIDs {
		used[id] = v
	}
	staged := &Service{graph: graph, usedIDs: used}
	mapping := map[string]string{}
	labels := map[string]string{}
	usedLabels := map[string]bool{}
	for _, c := range s.graph.Components {
		if c.Kind == "net-label" && c.Sheet == s.activeSheet {
			usedLabels[c.Label] = true
		}
	}
	for _, p := range source.Components {
		id := staged.nextID(copyPrefix(p.ID))
		mapping[p.ID] = id
		staged.usedIDs[id] = true
	}
	rewrite := func(ref string) string { id, pin, _ := strings.Cut(ref, "."); return mapping[id] + "." + pin }
	for _, p := range source.Clone().Components {
		if p.Kind == "net-label" && source.blockOwner(p.ID) == "" {
			old := p.Label
			if labels[old] == "" {
				base := old
				if len(base) > 20 {
					base = base[:20]
				}
				for n := 1; ; n++ {
					candidate := fmt.Sprintf("%s_copy%d", base, n)
					if !usedLabels[candidate] {
						labels[old] = candidate
						usedLabels[candidate] = true
						break
					}
				}
			}
			p.Label = labels[old]
		}
		p.ID = mapping[p.ID]
		p.Sheet = s.activeSheet
		p.TemplatePart, p.TemplateHash = "", ""
		p.Position = Point{p.Position.X + delta.X, p.Position.Y + delta.Y}
		if _, err = snapPoint(p.Position); err != nil {
			return err
		}
		if p.Block != nil {
			for i, id := range p.Block.Members {
				p.Block.Members[i] = mapping[id]
			}
			for i, pin := range p.Block.Ports {
				p.Block.Ports[i].Terminal = rewrite(pin.Terminal)
			}
			for i, param := range p.Block.Parameters {
				p.Block.Parameters[i].ComponentID = mapping[param.ComponentID]
			}
		}
		staged.graph.Components = append(staged.graph.Components, p)
	}
	for _, w := range source.Wires {
		w.TemplateWire = ""
		w.ID = staged.nextID("W")
		staged.usedIDs[w.ID] = true
		w.From = rewrite(w.From)
		w.To = rewrite(w.To)
		staged.graph.Wires = append(staged.graph.Wires, w)
	}
	if err = staged.graph.validatePositions(); err != nil {
		return err
	}
	if err = staged.graph.validateBlocks(); err != nil {
		return err
	}
	s.graph = staged.graph
	s.usedIDs = staged.usedIDs
	for _, part := range source.Components {
		if source.blockOwner(part.ID) == "" {
			s.selected = mapping[part.ID]
			break
		}
	}
	s.nextHistoryName = "Duplicate selection"
	return nil
}
func (s *Service) selectionCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "selection.move", "selection.duplicate":
		var p SelectionOffset
		if err := params(raw, &p, "ids", "offset"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "offset"); err != nil {
			return err
		}
		if command == "selection.move" {
			return s.moveSelectionLocked(p)
		}
		graph, err := s.selectedGraphLocked(p.IDs)
		if err != nil {
			return err
		}
		return s.pasteSelectionLocked(graph, p.Offset)
	case "selection.copy", "selection.delete":
		var p SelectionIDs
		if err := params(raw, &p, "ids"); err != nil {
			return err
		}
		graph, err := s.selectedGraphLocked(p.IDs)
		if err != nil {
			return err
		}
		if command == "selection.copy" {
			s.selectionClipboard = &graph
			s.selectionClipboardID++
			return nil
		}
		remove := map[string]bool{}
		for _, c := range graph.Components {
			remove[c.ID] = true
		}
		kept := EmptyGraph()
		for _, c := range s.graph.Components {
			if !remove[c.ID] {
				kept.Components = append(kept.Components, c)
			}
		}
		for _, w := range s.graph.Wires {
			a, _, _ := strings.Cut(w.From, ".")
			b, _, _ := strings.Cut(w.To, ".")
			if !remove[a] && !remove[b] {
				kept.Wires = append(kept.Wires, w)
			}
		}
		s.graph = kept
		s.nextHistoryName = "Delete selection"
		return nil
	case "selection.paste":
		var p SelectionPaste
		if err := params(raw, &p, "clipboard_id", "offset"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "offset"); err != nil {
			return err
		}
		if s.selectionClipboard == nil || p.ClipboardID == 0 || p.ClipboardID != s.selectionClipboardID {
			return &actionError{"conflict", "clipboard changed or is unavailable; copy the selection again"}
		}
		if err := s.pasteSelectionLocked(*s.selectionClipboard, p.Offset); err != nil {
			return err
		}
		s.nextHistoryName = "Paste selection"
		return nil
	}
	return fmt.Errorf("unknown selection command")
}
