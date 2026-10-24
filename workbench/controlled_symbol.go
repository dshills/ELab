package workbench

func ControlSymbolPaths(kind string, m ControlMemory) [][]Point {
	if IsButtonKind(kind) {
		y := -16.
		if m.ContactClosed {
			y = 0
		}
		paths := [][]Point{{{-55, 0}, {-20, 0}}, {{20, 0}, {55, 0}}, {{-22, y}, {22, y}}, {{0, y}, {0, y - 20}}, {{-12, y - 20}, {12, y - 20}}}
		if kind == "bouncing-pushbutton" {
			paths = append(paths, []Point{{25, -30}, {31, -36}, {37, -30}, {43, -36}})
		}
		return paths
	}
	if kind == "relay" {
		paths := [][]Point{{{-80, -90}, {100, -90}, {100, 90}, {-80, 90}, {-80, -90}}, {{-120, -60}, {-40, -60}, {-40, -55}}, {{-120, 60}, {-40, 60}, {-40, 55}}, {{45, 120}, {45, 0}}, {{85, -60}, {120, -60}}, {{85, 60}, {120, 60}}}
		coil := []Point{}
		for _, point := range InductorPolyline() {
			q := RotateLocal(point, 90)
			coil = append(coil, Point{q.X - 40, q.Y})
		}
		paths = append(paths, coil)
		y := 60.
		if m.High {
			y = -60
		}
		paths = append(paths, []Point{{45, 0}, {82, y}})
		for x := -5.; x < 30; x += 10 {
			paths = append(paths, []Point{{x, 0}, {x + 5, 0}})
		}
		return paths
	}
	if kind == "comparator" {
		return append(OpAmpSymbolPaths(), []Point{{-7, 8}, {-2, 8}, {-2, -7}, {5, -7}}, []Point{{-7, -7}, {-4, -7}, {-4, 8}, {5, 8}})
	}
	return nil
}

func ControlTerminalLocal(kind, pin string) (Point, bool) {
	if kind == "relay" {
		switch pin {
		case "coil1":
			return Point{-120, -60}, true
		case "coil2":
			return Point{-120, 60}, true
		case "common":
			return Point{45, 120}, true
		case "no":
			return Point{120, -60}, true
		case "nc":
			return Point{120, 60}, true
		}
	}
	if kind == "comparator" {
		switch pin {
		case "noninverting":
			return Point{-55, -20}, true
		case "inverting":
			return Point{-55, 20}, true
		case "out":
			return Point{55, 0}, true
		case "vcc":
			return Point{0, -45}, true
		case "gnd":
			return Point{0, 45}, true
		}
	}
	return Point{}, false
}
func ControlSymbolLabels(p Component) []OpAmpSymbolLabel {
	if p.Kind == "relay" {
		labels := []OpAmpSymbolLabel{{Point{-115, -73}, "coil1"}, {Point{-115, 78}, "coil2"}, {Point{90, -65}, "NO"}, {Point{90, 78}, "NC"}, {Point{52, 110}, "C"}}
		for k := range labels {
			at := RotateLocal(labels[k].At, p.Rotation)
			labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
		}
		return labels
	}
	if p.Kind != "comparator" {
		return nil
	}
	labels := []OpAmpSymbolLabel{{Point{7, -38}, "V+"}, {Point{7, 50}, "GND"}}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
