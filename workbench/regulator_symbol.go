package workbench

func RegulatorTerminalLocal(pin string) (Point, bool) {
	switch pin {
	case "in":
		return Point{-100, 0}, true
	case "out":
		return Point{100, 0}, true
	case "gnd":
		return Point{0, 90}, true
	}
	return Point{}, false
}
func RegulatorSymbolPaths() [][]Point {
	return [][]Point{{{-65, -45}, {65, -45}, {65, 45}, {-65, 45}, {-65, -45}}, {{-100, 0}, {-65, 0}}, {{65, 0}, {100, 0}}, {{0, 45}, {0, 90}}}
}
func RegulatorSymbolLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{{Point{-56, -8}, "IN"}, {Point{24, -8}, "OUT"}, {Point{8, 67}, "GND"}}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}
func regulatorLabels(p Component) []OpAmpSymbolLabel {
	if IsRegulatorKind(p.Kind) {
		return RegulatorSymbolLabels(p)
	}
	return nil
}
