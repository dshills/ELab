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
var routeCache = struct {
	sync.Mutex
	paths map[[32]byte][]Point
	order [][32]byte
}{paths: map[[32]byte][]Point{}}

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
	case "resistor", "switch":
		w, h = 58, 24
	case "led", "diode":
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
	routeCache.Lock()
	if len(routeCache.order) >= 512 {
		delete(routeCache.paths, routeCache.order[0])
		routeCache.order = routeCache.order[1:]
	}
	routeCache.paths[key] = path
	routeCache.order = append(routeCache.order, key)
	routeCache.Unlock()
	return append([]Point{}, path...)
}
func computeWireRoute(g Graph, w Wire) []Point {
	a, ok := g.TerminalPoint(w.From)
	if !ok {
		return nil
	}
	b, ok := g.TerminalPoint(w.To)
	if !ok {
		return nil
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
	sa, sb := stub(w.From, a), stub(w.To, b)
	cell := func(p Point) routeCell {
		return routeCell{int(math.Round(p.X / 10)), int(math.Round(p.Y / 10))}
	}
	start, end := cell(sa), cell(sb)
	rects := []routeRect{}
	for _, p := range g.Components {
		if !IsConnectivityPrimitive(p.Kind) {
			rects = append(rects, routeBody(p))
		}
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
	minX, maxX := int(math.Min(float64(start.x), float64(end.x)))-12, int(math.Max(float64(start.x), float64(end.x)))+12
	minY, maxY := int(math.Min(float64(start.y), float64(end.y)))-12, int(math.Max(float64(start.y), float64(end.y)))+12
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
