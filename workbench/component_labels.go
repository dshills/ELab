package workbench

// GroundLabelPosition keeps the upright caption clear of the bars and the
// terminal lead at every rotation. Native and exported schematics share the
// baseline; arbitrary neighboring objects still need layout consideration.
func GroundLabelPosition(p Component) Point {
	y := p.Position.Y - 42
	if p.Rotation == 0 {
		y = p.Position.Y + 42
	}
	return Point{p.Position.X - float64(len([]rune(p.ID)))*4.375, y}
}

// LEDLabelPosition is the upright label baseline shared by the native canvas
// and schematic outputs. It clears the larger native body/emission arrows and
// terminal routing stubs at every supported rotation. Global label/wire layout
// remains separate; this is presentation geometry, not electrical topology.
func LEDLabelPosition(p Component) Point {
	switch p.Rotation {
	case 90:
		return Point{p.Position.X, p.Position.Y - 36}
	case 180:
		return Point{p.Position.X + 35, p.Position.Y + 3}
	case 270:
		return Point{p.Position.X, p.Position.Y + 46}
	default:
		return Point{p.Position.X + 64, p.Position.Y + 3}
	}
}

// CapacitorLabelPosition is the upright baseline shared by native and exported
// capacitors. Horizontal symbols put the caption above the plates; the left
// offset also keeps the bundled flasher's adjacent base route visible. Vertical
// symbols put it beside the plates, away from both terminal routing stubs.
// Arbitrary neighboring objects still require separate layout consideration.
func CapacitorLabelPosition(p Component) Point {
	if p.Rotation == 90 || p.Rotation == 270 {
		return Point{p.Position.X - 90, p.Position.Y - 36}
	}
	return Point{p.Position.X + 34, p.Position.Y + 3}
}
