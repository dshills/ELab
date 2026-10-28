package workbench

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

const BlockModelID = "elab.circuit-block.v1"
const MaxBlockMembers = 32
const MaxBlockPorts = 16

var errBlockPosition = errors.New("block position unavailable")

type BlockPort struct {
	Name     string `json:"name"`
	Terminal string `json:"terminal"`
}
type BlockParameter struct {
	Name        string `json:"name"`
	ComponentID string `json:"component_id"`
}

// Members remain ordinary authored components. The block is only a named,
// single-level electrical boundary and a drawing projection, never a hidden model.
type CircuitBlock struct {
	Version    string           `json:"version"`
	Members    []string         `json:"members"`
	Ports      []BlockPort      `json:"ports"`
	Parameters []BlockParameter `json:"parameters"`
	Expanded   bool             `json:"expanded"`
}
type BlockCreateParams struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Position   Point            `json:"position"`
	Members    []string         `json:"members"`
	Ports      []BlockPort      `json:"ports"`
	Parameters []BlockParameter `json:"parameters"`
}
type BlockValueParams struct {
	ID        string `json:"id"`
	Parameter string `json:"parameter"`
	Value     string `json:"value"`
}
type BlockViewParams struct {
	ID       string `json:"id"`
	Expanded bool   `json:"expanded"`
}
type BlockPasteParams struct {
	ID       string `json:"id"`
	Position Point  `json:"position"`
}
type BlockTemplate struct {
	Name   string       `json:"name"`
	Graph  Graph        `json:"graph"`
	Block  CircuitBlock `json:"block"`
	Origin Point        `json:"origin"`
}

func cloneBlock(b *CircuitBlock) *CircuitBlock {
	if b == nil {
		return nil
	}
	c := *b
	c.Members = append([]string{}, b.Members...)
	c.Ports = append([]BlockPort{}, b.Ports...)
	c.Parameters = append([]BlockParameter{}, b.Parameters...)
	return &c
}
func blockComponent(id, name string, position Point, b *CircuitBlock) Component {
	terminals := make([]string, len(b.Ports))
	for k, p := range b.Ports {
		terminals[k] = p.Name
	}
	return Component{ID: id, Name: name, Kind: "circuit-block", Position: position, Terminals: terminals, Model: BlockModelID, Block: cloneBlock(b)}
}
func blockNameValid(s string) bool {
	if s != strings.TrimSpace(s) || len(s) < 1 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func (g Graph) blockOwner(id string) string {
	for _, p := range g.Components {
		if p.Block != nil && slices.Contains(p.Block.Members, id) {
			return p.ID
		}
	}
	return ""
}
func (g Graph) blockAliases() map[string]string {
	aliases := map[string]string{}
	for _, p := range g.Components {
		if p.Block != nil {
			for _, pin := range p.Block.Ports {
				aliases[p.ID+"."+pin.Name] = pin.Terminal
			}
		}
	}
	return aliases
}
func (g Graph) validateBlocks() error {
	owners := map[string]string{}
	for _, p := range g.Components {
		if p.Kind != "circuit-block" {
			if p.Block != nil {
				return fmt.Errorf("block metadata belongs only to circuit-block")
			}
			continue
		}
		b := p.Block
		if b == nil || b.Version != BlockModelID || !ValidID(p.ID) || len(p.ID) > 24 || !blockNameValid(p.Name) || p.Model != BlockModelID || p.Value != 0 || p.Unit != "" || len(b.Members) < 1 || len(b.Members) > MaxBlockMembers || len(b.Ports) < 1 || len(b.Ports) > MaxBlockPorts || len(b.Parameters) > MaxBlockMembers {
			return fmt.Errorf("invalid single-level circuit block %s", p.ID)
		}
		members := map[string]bool{}
		for _, id := range b.Members {
			c, ok := g.Component(id)
			if !ok || c.Kind == "circuit-block" || c.Kind == "net-label" || members[id] || owners[id] != "" {
				return fmt.Errorf("block members must be unique ordinary parts, without nested blocks or net labels")
			}
			members[id] = true
			owners[id] = p.ID
		}
		names := []string{}
		refs := map[string]bool{}
		for _, pin := range b.Ports {
			c, _, ok := g.Terminal(pin.Terminal)
			if !ValidID(pin.Name) || len(pin.Name) > 32 || slices.Contains(names, pin.Name) || !ok || !members[c.ID] || refs[pin.Terminal] {
				return fmt.Errorf("block pins need unique names and unique existing member terminals")
			}
			refs[pin.Terminal] = true
			names = append(names, pin.Name)
		}
		if !slices.Equal(names, p.Terminals) {
			return fmt.Errorf("block terminal metadata differs from its named pins")
		}
		exposed := map[string]bool{}
		targets := map[string]bool{}
		for _, param := range b.Parameters {
			c, ok := g.Component(param.ComponentID)
			if !ValidID(param.Name) || len(param.Name) > 32 || exposed[param.Name] || targets[c.ID] || !ok || !members[c.ID] || c.Unit == "" {
				return fmt.Errorf("block parameters need unique names and numeric member values")
			}
			exposed[param.Name] = true
			targets[c.ID] = true
		}
	}
	// Authored crossings must use declared boundary pins. Internal wires retain
	// their ordinary explicit connectivity, including disconnected wire state.
	for _, w := range g.Wires {
		a, _, _ := strings.Cut(w.From, ".")
		b, _, _ := strings.Cut(w.To, ".")
		if owners[a] != owners[b] && (owners[a] != "" || owners[b] != "") {
			return fmt.Errorf("wire %s crosses a block boundary; use a named block pin", w.ID)
		}
	}
	return nil
}
func (g Graph) hasBlocks() bool {
	for _, p := range g.Components {
		if p.Block != nil || p.Kind == "circuit-block" {
			return true
		}
	}
	return false
}

// PhysicalGraph resolves boundary aliases and removes only nonphysical symbols.
// All authored primitive identities, states, models and resource counts survive.
func (g Graph) PhysicalGraph() (Graph, error) {
	if err := g.validateBlocks(); err != nil {
		return Graph{}, err
	}
	out := g.Clone()
	components := out.Components
	out.Components = out.Components[:0]
	aliases := g.blockAliases()
	for _, p := range components {
		if p.Kind != "circuit-block" {
			out.Components = append(out.Components, p)
		}
	}
	for k, w := range out.Wires {
		if ref, ok := aliases[w.From]; ok {
			w.From = ref
		}
		if ref, ok := aliases[w.To]; ok {
			w.To = ref
		}
		out.Wires[k] = w
	}
	return out, nil
}

// VisibleGraph changes only the drawing projection. The service snapshot keeps
// the complete authored graph so agents and inspectors can address every member.
func (g Graph) VisibleGraph() Graph {
	hidden := map[string]bool{}
	aliases := map[string]string{}
	for _, p := range g.Components {
		if p.Block != nil {
			if p.Block.Expanded {
				hidden[p.ID] = true
				for _, pin := range p.Block.Ports {
					aliases[p.ID+"."+pin.Name] = pin.Terminal
				}
			} else {
				for _, id := range p.Block.Members {
					hidden[id] = true
				}
			}
		}
	}
	out := Graph{Components: []Component{}, Wires: []Wire{}}
	for _, p := range g.Components {
		if !hidden[p.ID] {
			out.Components = append(out.Components, p)
		}
	}
	for _, w := range g.Wires {
		if r, ok := aliases[w.From]; ok {
			w.From = r
		}
		if r, ok := aliases[w.To]; ok {
			w.To = r
		}
		a, _, _ := strings.Cut(w.From, ".")
		b, _, _ := strings.Cut(w.To, ".")
		if !hidden[a] && !hidden[b] {
			out.Wires = append(out.Wires, w)
		}
	}
	return out
}
func blockPinLocal(p Component, name string) (Point, bool) {
	if p.Block == nil {
		return Point{}, false
	}
	for k, pin := range p.Block.Ports {
		if pin.Name == name {
			rows := (len(p.Block.Ports) + 1) / 2
			x := -110.
			if k >= rows {
				x = 110.
			}
			row := k % rows
			return Point{X: x, Y: float64(row)*30 - float64(rows-1)*15}, true
		}
	}
	return Point{}, false
}
func BlockSymbolPaths(p Component) [][]Point {
	if p.Block == nil {
		return nil
	}
	_, h := BlockBodySize(p)
	paths := [][]Point{{{-85, -h}, {85, -h}, {85, h}, {-85, h}, {-85, -h}}}
	for _, pin := range p.Block.Ports {
		q, _ := blockPinLocal(p, pin.Name)
		inner := q
		inner.X *= 85. / 110
		paths = append(paths, []Point{q, inner})
	}
	return paths
}
func (s *Service) createBlockLocked(p BlockCreateParams) error {
	if !ValidID(p.ID) || len(p.ID) > 24 || s.usedIDs[p.ID] || s.graph.HasObject(p.ID) || len(s.graph.Components) >= MaxComponents {
		return fmt.Errorf("choose an unused block ID (1–24 characters) within drawing capacity")
	}
	position, err := snapPoint(p.Position)
	if err != nil {
		return err
	}

	g := s.graph.Clone()
	b := &CircuitBlock{Version: BlockModelID, Members: append([]string{}, p.Members...), Ports: append([]BlockPort{}, p.Ports...), Parameters: append([]BlockParameter{}, p.Parameters...)}
	g.Components = append(g.Components, blockComponent(p.ID, p.Name, position, b))
	aliases := map[string]string{}
	for _, pin := range b.Ports {
		aliases[pin.Terminal] = p.ID + "." + pin.Name
	}
	for k, w := range g.Wires {
		a, _, _ := strings.Cut(w.From, ".")
		z, _, _ := strings.Cut(w.To, ".")
		if slices.Contains(b.Members, a) != slices.Contains(b.Members, z) {
			if ref, ok := aliases[w.From]; ok {
				w.From = ref
			}
			if ref, ok := aliases[w.To]; ok {
				w.To = ref
			}
			g.Wires[k] = w
		}
	}
	if err := g.validateBlocks(); err != nil {
		return err
	}
	if err := g.validatePositions(); err != nil {
		return err
	}
	s.graph = g
	s.usedIDs[p.ID] = true
	s.selected = p.ID
	return nil
}
func (g Graph) BlockTemplate(id string) (*BlockTemplate, error) {
	p, ok := g.Component(id)
	if !ok || p.Block == nil {
		return nil, fmt.Errorf("choose a circuit block")
	}
	b := cloneBlock(p.Block)
	b.Expanded = false
	out := &BlockTemplate{Name: p.Name, Graph: EmptyGraph(), Block: *b, Origin: p.Position}
	for _, c := range g.Clone().Components {
		if slices.Contains(b.Members, c.ID) {
			out.Graph.Components = append(out.Graph.Components, c)
		}
	}
	for _, w := range g.Wires {
		a, _, _ := strings.Cut(w.From, ".")
		z, _, _ := strings.Cut(w.To, ".")
		if slices.Contains(b.Members, a) && slices.Contains(b.Members, z) {
			out.Graph.Wires = append(out.Graph.Wires, w)
		}
	}
	return out, nil
}
func (s *Service) pasteBlockLocked(p BlockPasteParams) error {
	if s.blockClipboard == nil {
		return fmt.Errorf("copy a circuit block first")
	}
	t := s.blockClipboard
	position, err := snapPoint(p.Position)
	if err != nil {
		return fmt.Errorf("%w: %v", errBlockPosition, err)
	}
	if !ValidID(p.ID) || len(p.ID) > 24 || s.usedIDs[p.ID] || s.graph.HasObject(p.ID) {
		return fmt.Errorf("choose an unused block ID (1–24 characters)")
	}
	if len(s.graph.Components)+len(t.Graph.Components)+1 > MaxComponents || len(s.graph.Wires)+len(t.Graph.Wires) > MaxWires {
		return fmt.Errorf("pasted block exceeds ordinary drawing limits")
	}
	g := s.graph.Clone()
	b := cloneBlock(&t.Block)
	ids := map[string]string{}
	delta := Point{X: position.X - t.Origin.X, Y: position.Y - t.Origin.Y}
	for k, c := range t.Graph.Clone().Components {
		id := fmt.Sprintf("%s_P%d", p.ID, k+1)
		if s.usedIDs[id] || g.HasObject(id) {
			return fmt.Errorf("pasted member ID %s already used", id)
		}
		ids[c.ID] = id
		c.ID = id
		c.Position = Point{X: c.Position.X + delta.X, Y: c.Position.Y + delta.Y}
		if _, err := snapPoint(c.Position); err != nil {
			return fmt.Errorf("%w: %v", errBlockPosition, err)
		}
		g.Components = append(g.Components, c)
	}
	rewrite := func(ref string) string { id, pin, _ := strings.Cut(ref, "."); return ids[id] + "." + pin }
	for k, w := range t.Graph.Wires {
		w.ID = fmt.Sprintf("%s_W%d", p.ID, k+1)
		if s.usedIDs[w.ID] || g.HasObject(w.ID) {
			return fmt.Errorf("pasted wire ID already used")
		}
		w.From = rewrite(w.From)
		w.To = rewrite(w.To)
		g.Wires = append(g.Wires, w)
	}
	for k, id := range b.Members {
		b.Members[k] = ids[id]
	}
	for k, pin := range b.Ports {
		b.Ports[k].Terminal = rewrite(pin.Terminal)
	}
	for k, q := range b.Parameters {
		b.Parameters[k].ComponentID = ids[q.ComponentID]
	}
	g.Components = append(g.Components, blockComponent(p.ID, t.Name, position, b))
	if err := g.validatePositions(); err != nil {
		return err
	}
	if err := g.validateBlocks(); err != nil {
		return err
	}
	s.graph = g
	for _, c := range g.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range g.Wires {
		s.usedIDs[w.ID] = true
	}
	s.selected = p.ID
	return nil
}
func (s *Service) blockCommandLocked(command string, raw json.RawMessage) error {
	switch command {
	case "block.create":
		var p BlockCreateParams
		if err := params(raw, &p, "id", "name", "position", "members", "ports", "parameters"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "position"); err != nil {
			return err
		}
		return s.createBlockLocked(p)
	case "block.paste":
		var p BlockPasteParams
		if err := params(raw, &p, "id", "position"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "position"); err != nil {
			return err
		}
		return s.pasteBlockLocked(p)
	case "block.parameter.set":
		var p BlockValueParams
		if err := params(raw, &p, "id", "parameter", "value"); err != nil {
			return err
		}
		part, ok := s.graph.Component(p.ID)
		if !ok || part.Block == nil {
			return fmt.Errorf("choose a block")
		}
		for _, v := range part.Block.Parameters {
			if v.Name == p.Parameter {
				for k, c := range s.graph.Components {
					if c.ID == v.ComponentID {
						next, err := SetValue(c, p.Value)
						if err != nil {
							return err
						}
						g := s.graph.Clone()
						g.Components[k] = next
						s.graph = g
						return nil
					}
				}
			}
		}
		return fmt.Errorf("choose an exposed block parameter")
	case "block.view.set":
		var p BlockViewParams
		if err := params(raw, &p, "id", "expanded"); err != nil {
			return err
		}
		for k, c := range s.graph.Components {
			if c.ID == p.ID && c.Block != nil {
				g := s.graph.Clone()
				g.Components[k].Block.Expanded = p.Expanded
				if err := g.validatePositions(); err != nil {
					return err
				}
				s.graph = g
				return nil
			}
		}
		return fmt.Errorf("choose a block")
	case "block.copy":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		t, err := s.graph.BlockTemplate(p.ID)
		if err != nil {
			return err
		}
		s.blockClipboard = t
		return nil
	case "block.release":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		part, ok := s.graph.Component(p.ID)
		if !ok || part.Block == nil {
			return fmt.Errorf("choose a block")
		}
		g := s.graph.Clone()
		g.Components = slices.DeleteFunc(g.Components, func(c Component) bool { return c.ID == p.ID })
		for k, w := range g.Wires {
			for _, pin := range part.Block.Ports {
				if w.From == p.ID+"."+pin.Name {
					w.From = pin.Terminal
				}
				if w.To == p.ID+"."+pin.Name {
					w.To = pin.Terminal
				}
			}
			g.Wires[k] = w
		}
		if err := g.validatePositions(); err != nil {
			return err
		}
		s.graph = g
		s.selected = ""
		return nil
	}
	return fmt.Errorf("unknown block command")
}

func (s *Service) moveBlockLocked(part Component, position Point) error {
	g := s.graph.Clone()
	delta := Point{X: position.X - part.Position.X, Y: position.Y - part.Position.Y}

	for k, c := range g.Components {
		if c.ID == part.ID || slices.Contains(part.Block.Members, c.ID) {
			c.Position = Point{X: c.Position.X + delta.X, Y: c.Position.Y + delta.Y}
			if _, err := snapPoint(c.Position); err != nil {
				return err
			}
			g.Components[k] = c
		}

	}
	if err := g.validatePositions(); err != nil {
		return err
	}
	s.graph = g
	return nil
}
func (s *Service) deleteBlockLocked(part Component) error {
	g := s.graph.Clone()
	g.Components = slices.DeleteFunc(g.Components, func(c Component) bool { return c.ID == part.ID || slices.Contains(part.Block.Members, c.ID) })
	g.Wires = slices.DeleteFunc(g.Wires, func(w Wire) bool {
		a, _, _ := strings.Cut(w.From, ".")
		b, _, _ := strings.Cut(w.To, ".")
		return a == part.ID || b == part.ID || slices.Contains(part.Block.Members, a) || slices.Contains(part.Block.Members, b)
	})
	s.graph = g
	return nil
}

func BlockSymbolLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{}
	if p.Block == nil {
		return labels
	}
	for _, pin := range p.Block.Ports {
		q, _ := blockPinLocal(p, pin.Name)
		if q.X < 0 {
			q.X = -79
		} else {
			q.X = 79 - float64(len(pin.Name))*7
		}
		q.Y += 4
		q = RotateLocal(q, p.Rotation)
		q.X += p.Position.X
		q.Y += p.Position.Y
		labels = append(labels, OpAmpSymbolLabel{At: q, Text: blockPinCaption(pin.Name)})
	}
	return labels
}

func blockPinCaption(name string) string {
	if len(name) > 10 {
		return name[:7] + "..."
	}
	return name
}

// BlockBodySize returns local half extents; leads extend a further 25 units.
func BlockBodySize(p Component) (float64, float64) {
	h := 45.
	if p.Block != nil {
		h = max(h, float64((len(p.Block.Ports)+1)/2)*15+15)
	}
	return 85, h
}

// Collapsed internals occupy their own drawing domain. Visible objects share
// the sheet. Expanding/releasing into an occupied visible point refuses instead
// of introducing an invisible placement obstacle or overlapping selectable parts.
func (g Graph) positionScopes() map[string]string {
	scopes := map[string]string{}
	for _, p := range g.Components {
		if p.Block != nil {
			if p.Block.Expanded {
				scopes[p.ID] = "hidden-symbol:" + p.ID
			} else {
				for _, id := range p.Block.Members {
					scopes[id] = "inside:" + p.ID
				}
			}
		}
	}
	return scopes
}
func (g Graph) positionScope(id string) string { return g.positionScopes()[id] }
func (g Graph) validatePositions() error {
	type key struct {
		scope string
		point Point
	}
	seen := map[key]string{}
	scopes := g.positionScopes()
	for _, c := range g.Components {
		k := key{scopes[c.ID], c.Position}
		if previous := seen[k]; previous != "" {
			return fmt.Errorf("%w: %s and %s share a grid point in the same drawing; choose a free position", errBlockPosition, previous, c.ID)
		}
		seen[k] = c.ID
	}
	return nil
}
