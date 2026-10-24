package workbench

import (
	"container/heap"
	"crypto/sha256"
	"encoding/json"
	"math"
	"sync"
)

// Routing is presentation only. Connectivity remains explicit named terminals.
// Cache by graph geometry/content so live waveform frames reuse the same paths.
type wireRouteCache struct {
	sync.Mutex
	paths map[[32]byte][]Point
	order [][32]byte
}

var routeCache wireRouteCache

func (c *wireRouteCache) retain(key [32]byte, path []Point) []Point {
	c.Lock()
	defer c.Unlock()
	// Routing runs outside this mutex. Another caller may have published the
	// same miss while we computed; retain its slot rather than duplicating FIFO
	// entries or evicting an unrelated route when the cache is full.
	if cached, ok := c.paths[key]; ok {
		return cached
	}
	if c.paths == nil {
		c.paths = make(map[[32]byte][]Point)
	}
	if len(c.order) >= 512 {
		delete(c.paths, c.order[0])
		c.order = c.order[1:]
	}
	c.paths[key] = path
	c.order = append(c.order, key)
	return path
}

type routeCell struct{ x, y int }
type routeNode struct {
	cell routeCell
	g, f int
}
type routeQueue []routeNode

func (q routeQueue) Len() int { return len(q) }
func (q routeQueue) Less(i, j int) bool {
	if q[i].f == q[j].f {
		return q[i].g > q[j].g
	}
	return q[i].f < q[j].f
}
func (q routeQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *routeQueue) Push(v any)   { *q = append(*q, v.(routeNode)) }
func (q *routeQueue) Pop() any     { old := *q; v := old[len(old)-1]; *q = old[:len(old)-1]; return v }

type routeRect struct{ left, right, top, bottom float64 }

func routeBody(p Component) routeRect {
	w, h := 42., 42.
	switch p.Kind {
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		w, h = 42, 42
	case "bcd-decoder", "seven-segment":
		w, h = 92, 202
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter", "timer555":
		w, h = 72, 92
	case "potentiometer":
		w, h = 58, 38
	case "pushbutton", "bouncing-pushbutton", "resistor", "inductor", "switch":
		w, h = 58, 24
	case "led", "diode", "zener", "schottky":
		w = 56
	case "junction", "net-label":
		w, h = 4, 4
	case "ground":
		w, h = 28, 28
	}
	if p.Rotation == 90 || p.Rotation == 270 {
		w, h = h, w
	}
	return routeRect{p.Position.X - w - 4, p.Position.X + w + 4, p.Position.Y - h - 4, p.Position.Y + h + 4}
}
func WirePath(g Graph, w Wire) []Point {
	data, _ := json.Marshal(struct {
		Graph Graph
		Wire  Wire
	}{g, w})
	key := sha256.Sum256(data)
	routeCache.Lock()
	cached, ok := routeCache.paths[key]
	routeCache.Unlock()
	if ok {
		return append([]Point{}, cached...)
	}
	path := computeWireRoute(g, w)
	path = routeCache.retain(key, path)
	return append([]Point{}, path...)
}
func wireRouteEndpoints(g Graph, w Wire) (a, b, sa, sb Point, ok bool) {
	a, ok = g.TerminalPoint(w.From)
	if !ok {
		return a, b, sa, sb, false
	}
	b, ok = g.TerminalPoint(w.To)
	if !ok {
		return a, b, sa, sb, false
	}
	stub := func(ref string, p Point) Point {
		c, _, _ := g.Terminal(ref)
		if IsConnectivityPrimitive(c.Kind) {
			return p
		}
		dx, dy := p.X-c.Position.X, p.Y-c.Position.Y
		if math.Abs(dx) > math.Abs(dy) {
			p.X += math.Copysign(24, dx)
		} else {
			p.Y += math.Copysign(24, dy)
		}
		return p
	}
	sa, sb = stub(w.From, a), stub(w.To, b)
	return a, b, sa, sb, true
}

func routeSearchBounds(sa, sb Point) (start, end routeCell, minX, maxX, minY, maxY int) {
	cell := func(p Point) routeCell { return routeCell{int(math.Round(p.X / 10)), int(math.Round(p.Y / 10))} }
	start, end = cell(sa), cell(sb)
	minX, maxX = min(start.x, end.x)-12, max(start.x, end.x)+12
	minY, maxY = min(start.y, end.y)-12, max(start.y, end.y)+12
	return
}

// WireRoutingBounds encloses all possible routes without running the path finder.
func WireRoutingBounds(g Graph, w Wire) (Point, Point, bool) {
	a, b, sa, sb, ok := wireRouteEndpoints(g, w)
	if !ok {
		return Point{}, Point{}, false
	}
	_, _, x0, x1, y0, y1 := routeSearchBounds(sa, sb)
	return Point{min(float64(x0*10), a.X, b.X), min(float64(y0*10), a.Y, b.Y)}, Point{max(float64(x1*10), a.X, b.X), max(float64(y1*10), a.Y, b.Y)}, true
}

func computeWireRoute(g Graph, w Wire) []Point {
	a, b, sa, sb, ok := wireRouteEndpoints(g, w)
	if !ok {
		return nil
	}
	start, end, minX, maxX, minY, maxY := routeSearchBounds(sa, sb)
	rects := []routeRect{}
	for _, p := range g.Components {
		if !IsConnectivityPrimitive(p.Kind) {
			rects = append(rects, routeBody(p))
		}
		switch p.Kind {
		case "pulse-source", "finite-pulse-source", "sine-source":
			// Waveform annotations depend on experiment settings, outside
			// this graph-only route cache. Do not substitute a false label.
			continue
		}
		// Exported captions have opaque masks. Avoid those masks as well
		// as symbols: a connected route must not appear broken beneath a
		// caption after moving or rotating a component. Include clearance
		// for the ordinary wire stroke, using the shared caption geometry.
		x, y, text := schematicPartLabel(p)
		rects = append(rects, routeRect{x - 4.5, x - 3 + schematicLabelWidth(text) + 1.5, y - 16.5, y + 5.5})
	}
	blocked := func(c routeCell) bool {
		x, y := float64(c.x*10), float64(c.y*10)
		for _, r := range rects {
			if x >= r.left && x <= r.right && y >= r.top && y <= r.bottom {
				return true
			}
		}
		return false
	}
	distance := func(c routeCell) int { return int(math.Abs(float64(c.x-end.x)) + math.Abs(float64(c.y-end.y))) }
	open := &routeQueue{{start, 0, distance(start)}}
	heap.Init(open)
	best := map[routeCell]int{start: 0}
	parents := map[routeCell]routeCell{}
	found := false
	for count := 0; open.Len() > 0 && count < 12000; count++ {
		n := heap.Pop(open).(routeNode)
		if n.g != best[n.cell] {
			continue
		}
		if n.cell == end {
			found = true
			break
		}
		for _, d := range []routeCell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := routeCell{n.cell.x + d.x, n.cell.y + d.y}
			if next.x < minX || next.x > maxX || next.y < minY || next.y > maxY || blocked(next) {
				continue
			}
			cost := n.g + 1
			if previous, ok := best[next]; ok && previous <= cost {
				continue
			}
			best[next] = cost
			parents[next] = n.cell
			heap.Push(open, routeNode{next, cost, cost + distance(next)})
		}
	}
	point := func(p Point) Point { return Point{X: p.X, Y: p.Y} }
	if !found {
		return []Point{point(a), point(sa), {X: sa.X, Y: sb.Y}, point(sb), point(b)}
	}
	cells := []routeCell{end}
	for c := end; c != start; {
		c = parents[c]
		cells = append(cells, c)
	}
	path := []Point{point(a), point(sa)}
	for i := len(cells) - 1; i >= 0; i-- {
		p := Point{X: float64(cells[i].x * 10), Y: float64(cells[i].y * 10)}
		if len(path) >= 2 {
			u, v := path[len(path)-2], path[len(path)-1]
			if u.X == v.X && v.X == p.X || u.Y == v.Y && v.Y == p.Y {
				path[len(path)-1] = p
				continue
			}
		}
		path = append(path, p)
	}
	return append(path, point(sb), point(b))
}

// BodyBounds exposes presentation bounds for rendering/route validation only.
func BodyBounds(p Component) (left, right, top, bottom float64) {
	r := routeBody(p)
	return r.left, r.right, r.top, r.bottom
}
