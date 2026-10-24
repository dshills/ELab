package workbench

func SequentialTerminalLocal(kind, pin string) (Point, bool) {
	if pin == "vcc" {
		return Point{0, -110}, true
	}
	if pin == "gnd" {
		return Point{0, 110}, true
	}
	if IsCounterKind(kind) {
		switch pin {
		case "clk":
			return Point{-100, -40}, true
		case "enable":
			return Point{-100, 0}, true
		case "reset":
			return Point{-100, 40}, true
		case "q0":
			return Point{100, -60}, true
		case "q1":
			return Point{100, -20}, true
		case "q2":
			return Point{100, 20}, true
		case "q3":
			return Point{100, 60}, true
		}
		return Point{}, false
	}
	if pin == "q" {
		return Point{100, -30}, true
	}
	if pin == "nq" {
		return Point{100, 30}, true
	}
	if kind == "jk-flipflop" {
		switch pin {
		case "j":
			return Point{-100, -60}, true
		case "k":
			return Point{-100, -20}, true
		case "clk":
			return Point{-100, 20}, true
		case "reset":
			return Point{-100, 60}, true
		}
		return Point{}, false
	}
	switch pin {
	case "d":
		return Point{-100, -45}, kind == "d-flipflop"
	case "t":
		return Point{-100, -45}, kind == "t-flipflop"
	case "clk":
		return Point{-100, 0}, true
	case "reset":
		return Point{-100, 45}, true
	}
	return Point{}, false
}
func SequentialSymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{-70, -90}, {70, -90}, {70, 90}, {-70, 90}, {-70, -90}}}
	for _, pin := range sequentialPins(kind) {
		at, _ := SequentialTerminalLocal(kind, pin)
		inside := at
		switch pin {
		case "vcc":
			inside.Y = -90
		case "gnd":
			inside.Y = 90
		default:
			if sequentialOutput(pin) {
				inside.X = 70
			} else {
				inside.X = -70
			}
		}
		paths = append(paths, []Point{at, inside})
		if pin == "clk" {
			paths = append(paths, []Point{{-70, at.Y - 6}, {-60, at.Y}, {-70, at.Y + 6}})
		}
	}
	return paths
}
func SequentialShortName(kind string) string {
	switch kind {
	case "d-flipflop":
		return "D FF"
	case "jk-flipflop":
		return "JK FF"
	case "t-flipflop":
		return "T FF"
	case "binary-counter":
		return "BIN4"
	case "decade-counter":
		return "BCD"
	}
	return "?"
}
func SequentialSymbolLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{{Point{-22, -70}, SequentialShortName(p.Kind)}}
	for _, pin := range p.Terminals {
		at, _ := SequentialTerminalLocal(p.Kind, pin)
		text := pin
		switch pin {
		case "vcc":
			at = Point{7, -78}
			text = "V+"
		case "gnd":
			at = Point{7, 83}
			text = "GND"
		default:
			if sequentialOutput(pin) {
				at.X = 32
			} else {
				at.X = -57
			}
			at.Y += 4
		}
		labels = append(labels, OpAmpSymbolLabel{at, text})
	}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
func SequentialLabelPosition(p Component) Point { return Point{p.Position.X + 110, p.Position.Y - 125} }
