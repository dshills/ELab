package workbench

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const MaxChildSheets = 24
const SheetDocumentVersion = 6

// The implicit root uses the empty ID. Components retain project-wide IDs so
// existing probes, recordings and wires survive a move between sheets.
type Sheet struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Parent string `json:"parent,omitempty"`
}

type SheetRenameParams struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SheetMoveParams struct {
	IDs   []string `json:"ids"`
	Sheet string   `json:"sheet"`
}

type SheetPortParams struct {
	From string `json:"from"`
	To   string `json:"to"`
	Name string `json:"name"`
}

type SheetPort struct {
	ID       string
	Name     string
	Terminal string
	Target   string
	Active   bool
}

type scopedLabel struct{ sheet, name string }

func validSheetName(name string) bool {
	return name != "" && len(name) <= 64 && strings.TrimSpace(name) == name && strings.IndexFunc(name, unicode.IsControl) < 0
}

func (g Graph) sheet(id string) (Sheet, bool) {
	if id == "" {
		return Sheet{Name: "Main"}, true
	}
	for _, sheet := range g.Sheets {
		if sheet.ID == id {
			return sheet, true
		}
	}
	return Sheet{}, false
}

func (g Graph) childAcross(a, b string) (string, bool) {
	if a == b {
		return "", false
	}
	if s, ok := g.sheet(a); ok && a != "" && s.Parent == b {
		return a, true
	}
	if s, ok := g.sheet(b); ok && b != "" && s.Parent == a {
		return b, true
	}
	return "", false
}

func (g Graph) validateSheets() error {
	if len(g.Sheets) > MaxChildSheets {
		return fmt.Errorf("a project can contain at most %d child sheets", MaxChildSheets)
	}
	if len(g.Sheets) == 0 {
		for _, part := range g.Components {
			if part.Sheet != "" {
				return fmt.Errorf("component %s is on a missing sheet", part.ID)
			}
		}
		for _, wire := range g.Wires {
			if wire.Port != "" {
				return fmt.Errorf("sheet port %s requires a child sheet", wire.Port)
			}
		}
		return nil
	}
	parents := map[string]string{"": ""}
	names := map[scopedLabel]bool{}
	for _, sheet := range g.Sheets {
		_, duplicate := parents[sheet.ID]
		_, parentExists := parents[sheet.Parent]
		key := scopedLabel{sheet.Parent, strings.ToLower(sheet.Name)}
		if !ValidID(sheet.ID) || len(sheet.ID) > 32 || duplicate || !parentExists || !validSheetName(sheet.Name) || names[key] {
			return fmt.Errorf("choose a unique sheet ID and name under an existing parent")
		}
		parents[sheet.ID], names[key] = sheet.Parent, true
	}
	parts := make(map[string]string, len(g.Components))
	for _, part := range g.Components {
		if _, ok := parents[part.Sheet]; !ok {
			return fmt.Errorf("component %s is on a missing sheet", part.ID)
		}
		parts[part.ID] = part.Sheet
	}
	for _, part := range g.Components {
		if part.Block != nil {
			for _, member := range part.Block.Members {
				if parts[member] != part.Sheet {
					return fmt.Errorf("block %s and its members must be on the same sheet", part.ID)
				}
			}
		}
	}
	ports := map[scopedLabel]bool{}
	for _, wire := range g.Wires {
		from, _, _ := strings.Cut(wire.From, ".")
		to, _, _ := strings.Cut(wire.To, ".")
		a, aOK := parts[from]
		b, bOK := parts[to]
		if !aOK || !bOK {
			return fmt.Errorf("wire %s has a missing endpoint", wire.ID)
		}
		if a == b && wire.Port == "" {
			continue
		}
		child, adjacent := g.childAcross(a, b)
		key := scopedLabel{child, wire.Port}
		if !adjacent || !validNetLabel(wire.Port) || ports[key] {
			return fmt.Errorf("wire %s needs a unique named port between a parent and its child sheet", wire.ID)
		}
		ports[key] = true
	}
	return nil
}

// SheetDrawing keeps drawing coordinates local. The authored graph and all
// electrical results remain project-wide.
func (g Graph) SheetDrawing(id string) Graph {
	if len(g.Sheets) == 0 {
		return g
	}
	out := EmptyGraph()
	visible := map[string]bool{}
	for _, part := range g.Components {
		if part.Sheet == id {
			out.Components = append(out.Components, part)
			visible[part.ID] = true
		}
	}
	for _, wire := range g.Wires {
		from, _, _ := strings.Cut(wire.From, ".")
		to, _, _ := strings.Cut(wire.To, ".")
		if visible[from] && visible[to] {
			out.Wires = append(out.Wires, wire)
		}
	}
	return out
}

func (g Graph) SheetPorts(id string) []SheetPort {
	if len(g.Sheets) == 0 {
		return nil
	}
	var ports []SheetPort
	for _, wire := range g.Wires {
		if wire.Port == "" {
			continue
		}
		from, _, _ := g.Terminal(wire.From)
		to, _, _ := g.Terminal(wire.To)
		local, remote := wire.From, to.Sheet
		if to.Sheet == id {
			local, remote = wire.To, from.Sheet
		} else if from.Sheet != id {
			continue
		}
		if part, pin, ok := g.Terminal(local); ok && part.Block != nil && part.Block.Expanded {
			for _, port := range part.Block.Ports {
				if port.Name == pin {
					local = port.Terminal
					break
				}
			}
		}
		ports = append(ports, SheetPort{wire.ID, wire.Port, local, remote, wire.Connected})
	}
	return ports
}

// Export keeps the current sheet readable and labels each off-sheet lead.
// Synthetic labels are drawing geometry and are never put into the project.
func (s *Service) sheetExportGraphLocked() Graph {
	g := s.graph.SheetDrawing(s.activeSheet).Clone()
	offsets := map[string]float64{}
	for i, port := range s.graph.SheetPorts(s.activeSheet) {
		part, pin, ok := g.Terminal(port.Terminal)
		if !ok {
			continue
		}
		at, ok := TerminalPosition(part, pin)
		if !ok {
			continue
		}
		x := at.X + 100
		if at.X < part.Position.X {
			x = at.X - 100
		}
		id := fmt.Sprintf("SheetPort%d", i+1)
		for g.HasObject(id) {
			id += "X"
		}
		label, _ := MakeComponent(id, "net-label", Point{}, port.Name)
		label.Position = Point{X: x, Y: at.Y + offsets[port.Terminal]}
		offsets[port.Terminal] += 40
		target, _ := s.graph.sheet(port.Target)
		label.Label = port.Name + " → " + target.Name
		g.Components = append(g.Components, label)
		g.Wires = append(g.Wires, Wire{ID: port.ID, From: port.Terminal, To: id + ".node", Connected: port.Active})
	}
	return g
}

func (s *Service) sheetAnnotationsLocked() []SourceAnnotation {
	annotations := s.sourceAnnotationsLocked()
	if len(s.graph.Sheets) == 0 {
		return annotations
	}
	active := annotations[:0]
	for _, annotation := range annotations {
		if part, ok := s.graph.Component(annotation.ID); ok && part.Sheet == s.activeSheet {
			active = append(active, annotation)
		}
	}
	return active
}

func (s *Service) sheetReadingsLocked() []PinnedReading {
	readings := s.pinnedReadingsLocked()
	if len(s.graph.Sheets) == 0 {
		return readings
	}
	active := readings[:0]
	for _, reading := range readings {
		id, _, _ := strings.Cut(reading.Probe.Positive, ".")
		if part, ok := s.graph.Component(id); ok && part.Sheet == s.activeSheet {
			active = append(active, reading)
		}
	}
	return active
}

func (s *Service) openSheetLocked(id string) error {
	if _, ok := s.graph.sheet(id); !ok {
		return fmt.Errorf("choose an existing sheet")
	}
	if id != s.activeSheet {
		s.activeSheet, s.selected = id, ""
		s.measurementTarget = nil
		s.drawing = DefaultDrawing()
		s.view = View{Zoom: 1}
		s.stopViewportFollowingLocked()
	}
	return nil
}

func (s *Service) sheetCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "sheet.open", "sheet.delete":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if command == "sheet.open" {
			return s.openSheetLocked(p.ID)
		}
		return s.deleteSheetLocked(p.ID)
	case "sheet.create":
		var p Sheet
		if err := params(raw, &p, "id", "name"); err != nil {
			return err
		}
		g := s.graph.Clone()
		g.Sheets = append(g.Sheets, p)
		if err := g.validateSheets(); err != nil {
			return err
		}
		s.graph = g
		s.nextHistoryName = "Create sheet"
		return s.openSheetLocked(p.ID)
	case "sheet.rename":
		var p SheetRenameParams
		if err := params(raw, &p, "id", "name"); err != nil {
			return err
		}
		g := s.graph.Clone()
		for i, sheet := range g.Sheets {
			if sheet.ID == p.ID {
				g.Sheets[i].Name = p.Name
				if err := g.validateSheets(); err != nil {
					return err
				}
				s.graph = g
				s.nextHistoryName = "Rename sheet"
				return nil
			}
		}
		return fmt.Errorf("choose an existing child sheet")
	case "sheet.move":
		var p SheetMoveParams
		if err := params(raw, &p, "ids", "sheet"); err != nil {
			return err
		}
		return s.moveToSheetLocked(p)
	case "sheet.port.connect":
		var p SheetPortParams
		if err := params(raw, &p, "from", "to", "name"); err != nil {
			return err
		}
		if !validNetLabel(p.Name) {
			return fmt.Errorf("choose a valid port name of 1–32 characters")
		}
		s.nextHistoryName = "Connect sheet port"
		return s.connectPortLocked(ConnectParams{From: p.From, To: p.To}, p.Name)
	}
	return fmt.Errorf("unknown sheet command")
}

func (s *Service) deleteSheetLocked(id string) error {
	sheet, ok := s.graph.sheet(id)
	if id == "" || !ok {
		return fmt.Errorf("choose an existing child sheet")
	}
	for _, child := range s.graph.Sheets {
		if child.Parent == id {
			return fmt.Errorf("move or remove child sheets first")
		}
	}
	for _, part := range s.graph.Components {
		if part.Sheet == id {
			return fmt.Errorf("move the components before removing this sheet")
		}
	}
	g := s.graph.Clone()
	g.Sheets = nil
	for _, child := range s.graph.Sheets {
		if child.ID != id {
			g.Sheets = append(g.Sheets, child)
		}
	}
	s.graph = g
	s.nextHistoryName = "Remove empty sheet"
	if s.activeSheet == id {
		return s.openSheetLocked(sheet.Parent)
	}
	return nil
}

func (s *Service) moveToSheetLocked(p SheetMoveParams) error {
	if _, ok := s.graph.childAcross(s.activeSheet, p.Sheet); !ok {
		return fmt.Errorf("move parts between a parent and its child sheet")
	}
	selection, err := s.selectedGraphLocked(p.IDs)
	if err != nil {
		return err
	}
	g := s.graph.Clone()
	moved := map[string]bool{}
	for _, part := range selection.Components {
		moved[part.ID] = true
	}
	remainingLabels, movedLabels, destinationLabels := map[string]string{}, map[string]string{}, map[string]string{}
	for i, part := range g.Components {
		if part.Kind == "net-label" {
			switch {
			case moved[part.ID]:
				movedLabels[part.Label] = part.ID + ".node"
			case part.Sheet == s.activeSheet:
				remainingLabels[part.Label] = part.ID + ".node"
			case part.Sheet == p.Sheet:
				destinationLabels[part.Label] = part.ID + ".node"
			}
		}
		if moved[part.ID] {
			g.Components[i].Sheet = p.Sheet
		}
	}
	if err := g.validatePositions(); err != nil {
		return err
	}

	usedNames := map[string]bool{}
	for _, wire := range g.Wires {
		usedNames[wire.Port] = true
	}
	portNumber := 0
	portName := func(preferred string) string {
		if validNetLabel(preferred) && !usedNames[preferred] {
			usedNames[preferred] = true
			return preferred
		}
		for {
			portNumber++
			name := fmt.Sprintf("Port%d", portNumber)
			if !usedNames[name] {
				usedNames[name] = true
				return name
			}
		}
	}
	for i := range g.Wires {
		wire := &g.Wires[i]
		from, _, _ := g.Terminal(wire.From)
		to, _, _ := g.Terminal(wire.To)
		if from.Sheet == to.Sheet {
			wire.Port = ""
		} else if wire.Port == "" {
			preferred := wire.To
			if moved[from.ID] {
				preferred = wire.From
			}
			wire.Port = portName(preferred)
		}
	}
	// A label that used to join two parts invisibly gets an explicit port if
	// the move separates them. A same-name label already on the destination
	// sheet would introduce a new electrical connection, so refuse that move.
	newWires := []string{}
	var oldNets map[string]string
	for _, part := range g.Components {
		if part.Kind != "net-label" || movedLabels[part.Label] != part.ID+".node" {
			continue
		}
		if destination := destinationLabels[part.Label]; destination != "" {
			if oldNets == nil {
				before, _, _ := s.compileGraphLocked()
				oldNets = before.TerminalNet
			}
			origin := oldNets[part.ID+".node"]
			if origin == "" || origin != oldNets[destination] {
				return fmt.Errorf("label %s already exists on the destination sheet; rename it before moving", part.Label)
			}
		}
		if ref := remainingLabels[part.Label]; ref != "" {
			if len(g.Wires) >= MaxWires {
				return fmt.Errorf("there is no wire capacity for the new sheet ports")
			}
			id := (&Service{graph: g, usedIDs: s.usedIDs}).nextID("W")
			g.Wires = append(g.Wires, Wire{ID: id, From: ref, To: movedLabels[part.Label], Connected: true, Port: portName(part.Label)})
			newWires = append(newWires, id)
		}
	}
	if err := g.validateSheets(); err != nil {
		return err
	}
	for _, id := range newWires {
		s.usedIDs[id] = true
	}
	s.graph = g
	s.nextHistoryName = "Move parts to sheet"
	return s.openSheetLocked(p.Sheet)
}
