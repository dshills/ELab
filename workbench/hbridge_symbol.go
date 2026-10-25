package workbench

func HBridgeTerminalLocal(pin string) (Point, bool) {
	switch pin {
	case "in1":
		return Point{-90, -40}, true
	case "in2":
		return Point{-90, 0}, true
	case "enable":
		return Point{-90, 40}, true
	case "out1":
		return Point{90, -30}, true
	case "out2":
		return Point{90, 30}, true
	case "vm":
		return Point{0, -100}, true
	case "gnd":
		return Point{0, 100}, true
	}
	return Point{}, false
}
func HBridgeSymbolPaths() [][]Point {
	paths := [][]Point{{{-65, -75}, {65, -75}, {65, 75}, {-65, 75}, {-65, -75}}}
	for _, pin := range hbridgePins {
		at, _ := HBridgeTerminalLocal(pin)
		inside := at
		switch pin {
		case "in1", "in2", "enable":
			inside.X = -65
		case "out1", "out2":
			inside.X = 65
		case "vm":
			inside.Y = -75
		case "gnd":
			inside.Y = 75
		}
		paths = append(paths, []Point{inside, at})
	}
	return paths
}
func HBridgeSymbolLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{{Point{-55, -35}, "IN1"}, {Point{-55, 5}, "IN2"}, {Point{-55, 45}, "EN"}, {Point{20, -25}, "OUT1"}, {Point{20, 35}, "OUT2"}, {Point{5, -57}, "VM"}, {Point{5, 68}, "GND"}}
	for i := range labels {
		at := RotateLocal(labels[i].At, p.Rotation)
		labels[i].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
