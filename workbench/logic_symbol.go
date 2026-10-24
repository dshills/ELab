package workbench

import "math"

// IEC functional rectangles distinguish all gates clearly even at browser size.
// Named pins describe a single gate, not a packaged IC's pin numbering.
func LogicTerminalLocal(kind, pin string) (Point, bool) {
	switch pin {
	case "a":
		if kind == "logic-not" {
			return Point{-65, 0}, true
		}
		return Point{-65, -20}, true
	case "b":
		return Point{-65, 20}, kind != "logic-not"
	case "out":
		return Point{65, 0}, true
	case "vcc":
		return Point{0, -65}, true
	case "gnd":
		return Point{0, 65}, true
	}
	return Point{}, false
}
func LogicSymbolText(kind string) string {
	switch kind {
	case "logic-and", "logic-nand":
		return "&"
	case "logic-or", "logic-nor":
		return ">=1"
	case "logic-xor", "logic-xnor":
		return "=1"
	case "logic-not":
		return "1"
	}
	return "?"
}
func LogicSymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{-40, -40}, {40, -40}, {40, 40}, {-40, 40}, {-40, -40}}}
	for _, pin := range logicPins(kind) {
		p, _ := LogicTerminalLocal(kind, pin)
		inside := p
		switch pin {
		case "a", "b":
			inside.X = -40
		case "out":
			inside.X = 40
		case "vcc":
			inside.Y = -40
		case "gnd":
			inside.Y = 40
		}
		if pin == "out" && (kind == "logic-not" || kind == "logic-nand" || kind == "logic-nor" || kind == "logic-xnor") {
			circle := []Point{}
			for k := 0; k <= 24; k++ {
				a := float64(k) * 2 * math.Pi / 24
				circle = append(circle, Point{46 + 6*math.Cos(a), 6 * math.Sin(a)})
			}
			paths = append(paths, circle, []Point{{52, 0}, p})
		} else {
			paths = append(paths, []Point{inside, p})
		}
	}
	return paths
}
func LogicSymbolLabels(p Component) []OpAmpSymbolLabel {
	text := LogicSymbolText(p.Kind)
	labels := []OpAmpSymbolLabel{{Point{-CanvasTextWidth(text, 1) / 2, 5}, text}, {Point{6, -48}, "V+"}, {Point{6, 59}, "GND"}}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
func LogicLabelPosition(p Component) Point { return Point{p.Position.X + 80, p.Position.Y - 90} }
