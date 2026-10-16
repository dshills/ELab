package workbench

import (
	"fmt"
	"math"
)

type NavigateParams struct {
	Direction string `json:"direction"`
	Kind      string `json:"kind,omitempty"`
	Center    bool   `json:"center,omitempty"`
}
type CenterSelectionParams struct {
	ID string `json:"id,omitempty"`
}

func boundedCenter(p Point) Point {
	return Point{max(-1e6, min(1e6, p.X)), max(-1e6, min(1e6, p.Y))}
}

func (s *Service) objectCenterLocked(id string) (Point, error) {
	if c, ok := s.graph.Component(id); ok {
		return boundedCenter(c.Position), nil
	}
	for _, w := range s.graph.Wires {
		if w.ID != id {
			continue
		}
		path := WirePath(s.graph, w)
		length := 0.
		for i := 1; i < len(path); i++ {
			length += math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)
		}
		remaining := length / 2
		for i := 1; i < len(path); i++ {
			a, b := path[i-1], path[i]
			d := math.Hypot(b.X-a.X, b.Y-a.Y)
			if d > 0 && remaining <= d {
				return boundedCenter(Point{a.X + (b.X-a.X)*remaining/d, a.Y + (b.Y-a.Y)*remaining/d}), nil
			}
			remaining -= d
		}
		if len(path) > 0 {
			return boundedCenter(path[0]), nil
		}
	}
	return Point{}, &actionError{"unavailable", "Select an existing schematic object first"}
}

func (s *Service) navigateLocked(p NavigateParams) error {
	switch p.Direction {
	case "next", "previous", "first", "last":
	default:
		return fmt.Errorf("direction must be next, previous, first or last")
	}
	switch p.Kind {
	case "", "all", "components", "wires":
	default:
		return fmt.Errorf("kind must be all, components or wires")
	}
	ids := []string{}
	if p.Kind != "wires" {
		for _, c := range s.graph.Components {
			ids = append(ids, c.ID)
		}
	}
	if p.Kind != "components" {
		for _, w := range s.graph.Wires {
			ids = append(ids, w.ID)
		}
	}
	if len(ids) == 0 {
		return &actionError{"unavailable", "No matching schematic objects"}
	}
	index := -1
	for i, id := range ids {
		if id == s.selected {
			index = i
			break
		}
	}
	switch p.Direction {
	case "first":
		index = 0
	case "last":
		index = len(ids) - 1
	case "next":
		index = (index + 1) % len(ids)
	case "previous":
		if index < 0 {
			index = 0
		}
		index = (index + len(ids) - 1) % len(ids)
	}
	center := s.view.Center
	if p.Center {
		var err error
		center, err = s.objectCenterLocked(ids[index])
		if err != nil {
			return err
		}
	}
	s.selected = ids[index]
	s.measurementTarget = nil
	s.drawing.Tool = "select"
	s.drawing.WireStart = ""
	s.normalizeDrawingLocked()
	s.view.Center = center
	if p.Center {
		s.stopViewportFollowingLocked()
	}
	return nil
}

func (s *Service) fitViewLocked(p FitParams) (View, error) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	include := func(a, b Point) {
		minX = min(minX, a.X)
		minY = min(minY, a.Y)
		maxX = max(maxX, b.X)
		maxY = max(maxY, b.Y)
	}
	for _, c := range s.graph.Components {
		include(Point{c.Position.X - 90, c.Position.Y - 80}, Point{c.Position.X + 140, c.Position.Y + 95})
		x, y, label := schematicPartLabel(c)
		include(Point{x - 10, y - 25}, Point{x + schematicLabelWidth(label) + 10, y + 10})
		for _, terminal := range c.Terminals {
			if pos, ok := TerminalPosition(c, terminal); ok {
				include(Point{pos.X - 10, pos.Y - 10}, Point{pos.X + 10, pos.Y + 10})
			}
		}
	}
	for _, w := range s.graph.Wires {
		if a, b, ok := WireRoutingBounds(s.graph, w); ok {
			include(a, b)
		}
	}
	for _, r := range s.pinnedReadingsLocked() {
		a, b := PinnedReadingBounds(r)
		include(a, b)
	}
	for _, a := range s.sourceAnnotationsLocked() {
		lo, hi := SourceAnnotationBounds(a)
		include(lo, hi)
	}
	if math.IsInf(minX, 1) {
		return View{Zoom: 1}, nil
	}
	center := boundedCenter(Point{(minX + maxX) / 2, (minY + maxY) / 2})
	width := 2 * (max(center.X-minX, maxX-center.X) + 20)
	height := 2 * (max(center.Y-minY, maxY-center.Y) + 20)
	zoom := min(2, p.Width/width, p.Height/height)
	if zoom < .01 {
		return View{}, &actionError{"view-limit", "Schematic exceeds the minimum zoom; use object navigation, move objects closer, or enlarge the viewport"}
	}
	return View{Center: center, Zoom: zoom}, nil
}
