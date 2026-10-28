package workbench

import (
	"fmt"
	"math"
	"sort"
)

func IsConnectivityPrimitive(kind string) bool { return kind == "junction" || kind == "net-label" }
func validNetLabel(label string) bool {
	if len(label) < 1 || len(label) > 32 {
		return false
	}
	for _, r := range label {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '+' || r == '-' || r == '.' || r == '/') {
			return false
		}
	}
	return true
}
func physicalTerminals(g Graph, refs []string) []string {
	out := []string{}
	for _, ref := range refs {
		part, _, ok := g.Terminal(ref)
		if ok && !IsConnectivityPrimitive(part.Kind) {
			out = append(out, ref)
		}
	}
	return out
}

// Match a displayed route only when a command explicitly names the wire.
// Visual crossings and near misses never invoke electrical connectivity here.
func WireDistance(g Graph, w Wire, p Point) float64 {
	path := WirePath(g, w)
	distance := math.Inf(1)
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := dx*dx + dy*dy
		if length == 0 {
			continue
		}
		u := math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/length))
		distance = math.Min(distance, math.Hypot(p.X-a.X-u*dx, p.Y-a.Y-u*dy))
	}
	return distance
}

// Default insertion tries interior route segments and preserves exact command
// validation. It is useful for keyboard/agent users without a pointer gesture.
func WireJunctionPosition(g Graph, w Wire) (Point, error) {
	path := WirePath(g, w)
	type segment struct {
		a, b   Point
		length float64
	}
	segments := []segment{}
	for i := 1; i < len(path); i++ {
		segments = append(segments, segment{path[i-1], path[i], math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)})
	}
	sort.SliceStable(segments, func(i, j int) bool { return segments[i].length > segments[j].length })
	for _, segment := range segments {
		for _, fraction := range []float64{.5, .25, .75} {
			p, err := snapPoint(Point{segment.a.X + (segment.b.X-segment.a.X)*fraction, segment.a.Y + (segment.b.Y-segment.a.Y)*fraction})
			if err == nil && validJunctionPosition(g, []Wire{w}, p) == nil {
				return p, nil
			}
		}
	}
	return Point{}, fmt.Errorf("choose a free grid point along the interior of the wire")
}
func validJunctionPosition(g Graph, wires []Wire, p Point) error {
	for _, part := range g.VisibleGraph().Components {
		if part.Position == p {
			return fmt.Errorf("junction position is occupied")
		}
	}
	for _, wire := range wires {
		if !wire.Connected {
			return fmt.Errorf("only connected wires can be split; the disconnected wire is preserved")
		}
		if WireDistance(g, wire, p) > GridSize/2+1e-9 {
			return fmt.Errorf("junction must lie on the named wire route (within half a grid step)")
		}
		a, _ := g.TerminalPoint(wire.From)
		b, _ := g.TerminalPoint(wire.To)
		if math.Hypot(p.X-a.X, p.Y-a.Y) < GridSize || math.Hypot(p.X-b.X, p.Y-b.Y) < GridSize {
			return fmt.Errorf("connect to the existing endpoint instead of inserting a nearby junction")
		}
	}
	return nil
}

type JunctionInsertParams struct {
	ID       string   `json:"id,omitempty"`
	WireIDs  []string `json:"wire_ids"`
	Position *Point   `json:"position,omitempty"`
}

func (s *Service) insertJunctionLocked(p JunctionInsertParams) error {
	if len(p.WireIDs) < 1 || len(p.WireIDs) > 8 || len(s.graph.Components) >= MaxComponents || len(s.graph.Wires)+len(p.WireIDs) > MaxWires {
		return fmt.Errorf("junction requires 1–8 wires within component/wire limits")
	}
	wires := []Wire{}
	indices := []int{}
	seen := map[string]bool{}
	for _, id := range p.WireIDs {
		if seen[id] {
			return fmt.Errorf("junction wire IDs must be unique")
		}
		seen[id] = true
		found := false
		for i, wire := range s.graph.Wires {
			if wire.ID == id {
				found = true
				from, _, _ := s.graph.Terminal(wire.From)
				to, _, _ := s.graph.Terminal(wire.To)
				if s.graph.blockOwner(from.ID) != "" || s.graph.blockOwner(to.ID) != "" {
					return fmt.Errorf("release the block before inserting new members into an internal wire")
				}
				wires = append(wires, wire)
				indices = append(indices, i)
				break
			}
		}
		if !found {
			return fmt.Errorf("junction references an unknown wire")
		}
	}
	position := p.Position
	if position == nil {
		if len(wires) != 1 {
			return fmt.Errorf("multi-wire junction requires an explicit position")
		}
		point, err := WireJunctionPosition(s.graph, wires[0])
		if err != nil {
			return err
		}
		position = &point
	}
	point, err := snapPoint(*position)
	if err != nil {
		return err
	}
	if err := validJunctionPosition(s.graph, wires, point); err != nil {
		return err
	}
	id := p.ID
	if id == "" {
		id = s.nextID("J")
	}
	if s.usedIDs[id] || s.graph.HasObject(id) {
		return fmt.Errorf("junction ID is already used")
	}
	part, err := MakeComponent(id, "junction", point, "")
	if err != nil {
		return err
	}
	// All failure conditions precede mutation; ID reservations and the complete
	// graph update are one history operation, including multi-wire joins.
	s.graph.Components = append(s.graph.Components, part)
	s.usedIDs[id] = true
	for i, wire := range wires {
		nextID := s.nextID("W")
		s.usedIDs[nextID] = true
		s.graph.Wires[indices[i]].To = id + ".node"
		s.graph.Wires = append(s.graph.Wires, Wire{ID: nextID, From: id + ".node", To: wire.To, Connected: true})
	}
	s.selected = id
	s.drawing.Tool = "select"
	return nil
}
