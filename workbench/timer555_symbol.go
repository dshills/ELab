package workbench

// The functional arrangement keeps timing pins together. Numbers identify the
// standard eight-pin package; coordinates are schematic geometry, not a footprint.
func Timer555TerminalLocal(pin string) (Point, bool) {
	switch pin {
	case "gnd":
		return Point{40, 110}, true
	case "vcc":
		return Point{40, -110}, true
	case "reset":
		return Point{-40, -110}, true
	case "control":
		return Point{-40, 110}, true
	case "discharge":
		return Point{-100, -70}, true
	case "trigger":
		return Point{-100, 0}, true
	case "threshold":
		return Point{-100, 60}, true
	case "out":
		return Point{100, 0}, true
	}
	return Point{}, false
}
func Timer555SymbolPaths() [][]Point {
	paths := [][]Point{{{-70, -90}, {70, -90}, {70, 90}, {-70, 90}, {-70, -90}}}
	for _, pin := range timer555Pins {
		at, _ := Timer555TerminalLocal(pin)
		inside := at
		switch pin {
		case "vcc", "reset":
			inside.Y = -90
		case "gnd", "control":
			inside.Y = 90
		case "out":
			inside.X = 70
		default:
			inside.X = -70
		}
		paths = append(paths, []Point{at, inside})
	}
	return paths
}
func Timer555SymbolLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{{Point{-63, -62}, "7 DIS"}, {Point{-63, 4}, "2 TRIG"}, {Point{-63, 64}, "6 THR"}, {Point{14, 4}, "3 OUT"}, {Point{-62, -80}, "4 RST"}, {Point{12, -80}, "8 VCC"}, {Point{-62, 84}, "5 CTRL"}, {Point{12, 84}, "1 GND"}}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
func Timer555LabelPosition(p Component) Point { return Point{p.Position.X - 110, p.Position.Y - 160} }
