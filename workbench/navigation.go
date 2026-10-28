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
	drawing := s.graph.VisibleGraph()
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	include := func(a, b Point) {
		minX = min(minX, a.X)
		minY = min(minY, a.Y)
		maxX = max(maxX, b.X)
		maxY = max(maxY, b.Y)
	}
	annotations := s.sourceAnnotationsLocked()
	annotated := map[string]bool{}
	for _, a := range annotations {
		annotated[a.ID] = true
	}
	type caption struct {
		part Component
		text string
	}
	captions := []caption{}
	for _, c := range drawing.Components {
		left, right, top, bottom := BodyBounds(c)
		include(Point{left - 4, top - 4}, Point{right + 4, bottom + 4})
		if !annotated[c.ID] {
			x, y, label := schematicPartLabel(c)
			include(Point{x - 10, y - 25}, Point{x + schematicLabelWidth(label) + 10, y + 10})
			_, title := CanvasPartLabel(c)
			captions = append(captions, caption{c, title})
		}
		for _, terminal := range c.Terminals {
			if pos, ok := TerminalPosition(c, terminal); ok {
				include(Point{pos.X - 10, pos.Y - 10}, Point{pos.X + 10, pos.Y + 10})
			}
		}
	}
	for _, w := range drawing.Wires {
		// Fit the deterministic route used by both drawing and output, rather
		// than its much larger path-finder search envelope. WirePath owns its
		// result and caches by the complete graph/route input.
		for _, at := range WirePath(drawing, w) {
			include(Point{at.X - 4, at.Y - 4}, Point{at.X + 4, at.Y + 4})
		}
	}
	for _, r := range s.pinnedReadingsLocked() {
		a, b := PinnedReadingBounds(r)
		include(a, b)
	}
	for _, a := range annotations {
		if _, ok := drawing.Component(a.ID); !ok {
			continue
		}
		lo, hi := SourceAnnotationBounds(a)
		include(lo, hi)
	}
	if math.IsInf(minX, 1) {
		return View{Zoom: 1}, nil
	}
	baseMinX, baseMinY, baseMaxX, baseMaxY := minX, minY, maxX, maxY
	fitAt := func(zoom float64) (View, bool) {
		minX, minY, maxX, maxY = baseMinX, baseMinY, baseMaxX, baseMaxY
		for _, c := range captions {
			at := CanvasPartLabelPositionAtZoom(c.part, c.text, zoom)
			lo, hi := CanvasTextBounds(at, c.text, CanvasFontScale(zoom))
			include(lo, hi)
		}
		for _, a := range annotations {
			part, visible := drawing.Component(a.ID)
			if !visible {
				continue
			}
			layout := LayoutCanvasAnnotation(a, part.Rotation, zoom)
			for i, line := range a.Lines {
				lo, hi := CanvasTextBounds(layout.Position(line, i), line, layout.Scale)
				include(lo, hi)
			}
		}
		center := boundedCenter(Point{(minX + maxX) / 2, (minY + maxY) / 2})
		width := 2 * (max(center.X-minX, maxX-center.X) + 20)
		height := 2 * (max(center.Y-minY, maxY-center.Y) + 20)
		// Reserve screen-space clearance for the minimum visible stroke too.
		return View{Center: center, Zoom: zoom}, width*zoom <= p.Width-4 && height*zoom <= p.Height-4
	}
	view, fits := fitAt(.01)
	if !fits {
		return View{}, &actionError{"view-limit", "Schematic exceeds the minimum zoom; use object navigation, move objects closer, or enlarge the viewport"}
	}
	if largest, ok := fitAt(2); ok {
		return largest, nil
	}
	// Enlarged captions depend on zoom. Keep a known-fitting lower endpoint
	// and check every accepted candidate against its own caption geometry.
	// Route finding is outside this bounded search.
	low, high := .01, 2.
	for range 40 {
		mid := (low + high) / 2
		if candidate, ok := fitAt(mid); ok {
			low, view = mid, candidate
		} else {
			high = mid
		}
	}
	return view, nil
}
