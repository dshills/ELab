package workbench

import "strings"

func ExtendedTerminalLocal(kind, pin string) (Point, bool) {
	if kind == "pwm-controller" {
		switch pin {
		case "out":
			return Point{90, 0}, true
		case "vcc":
			return Point{0, -100}, true
		case "gnd":
			return Point{0, 100}, true
		}
	}
	if isSoundKind(kind) {
		if pin == "positive" {
			return Point{0, -65}, true
		}
		if pin == "negative" {
			return Point{0, 65}, true
		}
	}
	if kind == "shift-register" {
		for k, p := range shiftPins {
			if p != pin {
				continue
			}
			if k < 5 {
				return Point{-120, float64(k)*60 - 120}, true
			}
			if k < 14 {
				return Point{120, float64(k-5)*35 - 140}, true
			}
			if pin == "vcc" {
				return Point{0, -210}, true
			}
			return Point{0, 210}, true
		}
	}
	return Point{}, false
}
func PWMTerminalLocal(pin string) (Point, bool) { return ExtendedTerminalLocal("pwm-controller", pin) }
func ExtendedSymbolPaths(kind string) [][]Point {
	w, h := 65., 75.
	pins := pwmPins
	if kind == "shift-register" {
		w, h = 95, 185
		pins = shiftPins
	}
	if isSoundKind(kind) {
		paths := [][]Point{{{0, -65}, {0, -25}}, {{-22, -25}, {22, -25}, {22, 25}, {-22, 25}, {-22, -25}}, {{0, 25}, {0, 65}}, {{30, -20}, {40, -10}, {40, 10}, {30, 20}}, {{46, -30}, {58, -15}, {58, 15}, {46, 30}}}
		if kind == "active-buzzer" {
			paths = append(paths, []Point{{-12, 6}, {-12, -6}, {0, -6}, {0, 6}, {12, 6}, {12, -6}})
		} else {
			paths = append(paths, []Point{{-14, -7}, {14, -7}}, []Point{{-14, 7}, {14, 7}})
		}
		return paths
	}
	paths := [][]Point{{{-w, -h}, {w, -h}, {w, h}, {-w, h}, {-w, -h}}}
	for _, pin := range pins {
		p, _ := ExtendedTerminalLocal(kind, pin)
		q := p
		if p.X < 0 {
			q.X = -w
		}
		if p.X > 0 {
			q.X = w
		}
		if p.X == 0 {
			if p.Y < 0 {
				q.Y = -h
			} else {
				q.Y = h
			}
		}
		paths = append(paths, []Point{p, q})
	}
	return paths
}
func ExtendedSymbolLabels(p Component) []OpAmpSymbolLabel {
	pins := pwmPins
	if p.Kind == "shift-register" {
		pins = shiftPins
	}
	if isSoundKind(p.Kind) {
		return nil
	}
	labels := []OpAmpSymbolLabel{}
	for _, pin := range pins {
		at, _ := ExtendedTerminalLocal(p.Kind, pin)
		text := strings.ToUpper(pin)
		if at.X < 0 {
			at.X += 30
			at.Y += 5
		} else if at.X > 0 {
			at.X -= 70
			at.Y += 5
		} else if at.Y < 0 {
			at.X = 5
			at.Y += 35
		} else {
			at.X = 5
			at.Y -= 23
		}
		at = RotateLocal(at, p.Rotation)
		labels = append(labels, OpAmpSymbolLabel{Point{p.Position.X + at.X, p.Position.Y + at.Y}, text})
	}
	return labels
}
