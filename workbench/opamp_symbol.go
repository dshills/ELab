package workbench

// OpAmpSymbolPaths returns owned local-coordinate paths shared by the native
// canvas and schematic exports. Signal signs and explicit power leads are part
// of the symbol; all five terminals remain independently wireable.
func OpAmpSymbolPaths() [][]Point {
	return [][]Point{
		{{-30, -36}, {-30, 36}, {30, 0}, {-30, -36}},
		{{-55, -20}, {-30, -20}}, {{-55, 20}, {-30, 20}},
		{{30, 0}, {55, 0}}, {{0, -45}, {0, -18}}, {{0, 18}, {0, 45}},
		{{-23, -20}, {-13, -20}}, {{-18, -25}, {-18, -15}},
		{{-23, 20}, {-13, 20}},
	}
}

type OpAmpSymbolLabel struct {
	At   Point
	Text string
}

// OpAmpSupplyLabels returns world-coordinate power-pin labels with upright
// text at every component rotation.
func OpAmpSupplyLabels(p Component) []OpAmpSymbolLabel {
	labels := []OpAmpSymbolLabel{{Point{7, -38}, "V+"}, {Point{7, 42}, "V-"}}
	for k := range labels {
		at := RotateLocal(labels[k].At, p.Rotation)
		labels[k].At = Point{p.Position.X + at.X, p.Position.Y + at.Y}
	}
	return labels
}

func OpAmpLabelPosition(p Component) Point {
	return Point{p.Position.X - 30 - schematicLabelWidth(p.ID), p.Position.Y - 55}
}
