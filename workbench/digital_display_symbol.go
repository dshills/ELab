package workbench

func DisplayTerminalLocal(kind, pin string) (Point, bool) {
	if pin == "vcc" && kind == "bcd-decoder" {
		return Point{0, -230}, true
	}
	if pin == "gnd" && kind == "bcd-decoder" || pin == "common" && kind == "seven-segment" {
		return Point{0, 230}, true
	}
	if kind == "bcd-decoder" {
		for k, p := range []string{"b0", "b1", "b2", "b3"} {
			if p == pin {
				return Point{-120, -120 + float64(k)*80}, true
			}
		}
	}
	for k, p := range segmentNames {
		if pin == p {
			x := 120.
			if kind == "seven-segment" {
				x = -120
			}
			return Point{x, -150 + float64(k)*50}, true
		}
	}
	return Point{}, false
}
func DisplaySegmentPaths() [][]Point {
	return [][]Point{{{-15, -95}, {35, -95}}, {{40, -90}, {40, -15}}, {{40, 5}, {40, 80}}, {{-15, 85}, {35, 85}}, {{-20, 5}, {-20, 80}}, {{-20, -90}, {-20, -15}}, {{-15, -5}, {35, -5}}}
}
func DisplaySymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{-90, -200}, {90, -200}, {90, 200}, {-90, 200}, {-90, -200}}}
	for _, pin := range displayPins(kind) {
		at, _ := DisplayTerminalLocal(kind, pin)
		in := at
		switch pin {
		case "vcc":
			in.Y = -200
		case "gnd", "common":
			in.Y = 200
		default:
			if at.X > 0 {
				in.X = 90
			} else {
				in.X = -90
			}
		}
		paths = append(paths, []Point{at, in})
	}
	return paths
}
func DisplaySymbolLabels(p Component) []OpAmpSymbolLabel {
	title := "BCD-7"
	if p.Kind == "seven-segment" {
		title = "7 SEG CC"
	}
	out := []OpAmpSymbolLabel{{Point{-45, -180}, title}}
	for _, pin := range displayPins(p.Kind) {
		at, _ := DisplayTerminalLocal(p.Kind, pin)
		switch pin {
		case "vcc":
			at = Point{7, -205}
		case "gnd", "common":
			at = Point{7, 214}
		default:
			if at.X > 0 {
				at.X = 68
			} else {
				at.X = -80
			}
			at.Y += 4
		}
		out = append(out, OpAmpSymbolLabel{at, pin})
	}
	for k := range out {
		at := RotateLocal(out[k].At, p.Rotation)
		out[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return out
}
