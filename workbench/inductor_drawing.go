package workbench

import (
	"fmt"
	"math"
)

// InductorPolyline is the same owned local coil/lead geometry for native and
// vector output; callers apply the component's position and rotation.
func InductorPolyline() []Point {
	points := []Point{{X: -55}, {X: -32}}
	for coil := range 4 {
		center := -24. + 16*float64(coil)
		for step := 1; step <= 16; step++ {
			angle := math.Pi * float64(step) / 16
			points = append(points, Point{X: center - 8*math.Cos(angle), Y: -8 * math.Sin(angle)})
		}
	}
	return append(points, Point{X: 55})
}

func InductorLabelPosition(part Component) Point {
	if part.Rotation == 90 || part.Rotation == 270 {
		return Point{X: part.Position.X + 18, Y: part.Position.Y + 3}
	}
	return Point{X: part.Position.X - 30, Y: part.Position.Y + 24}
}

func InductorValueLabel(value float64) string {
	scale, suffix := 1., "H"
	if value < .001 {
		scale, suffix = 1e6, "µH"
	} else if value < 1 {
		scale, suffix = 1e3, "mH"
	}
	return fmt.Sprintf("%.4g %s", value*scale, suffix)
}
