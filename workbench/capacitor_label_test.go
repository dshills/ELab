package workbench

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestCapacitorExportLabelClearsPlatesAndTerminalStubs(t *testing.T) {
	for _, identity := range []string{"C1", strings.Repeat("C", 32)} {
		for _, rotation := range []int{0, 90, 180, 270} {
			part, err := MakeComponent(identity, "capacitor", Point{100, -40}, "10uF")
			if err != nil {
				t.Fatal(err)
			}
			part.Rotation = rotation
			x, y, text := schematicPartLabel(part)
			label := routeRect{x - 3, x + schematicLabelWidth(text), y - 18, y + 5}
			// Enclose both native plates (larger than export plates) and
			// the 24-unit routing stubs outside ±32-unit pins.
			for name, local := range map[string]routeRect{
				"plates":     {-24, 24, -10.5, 10.5},
				"first stub": {-1, 1, -56, -32}, "second stub": {-1, 1, 32, 56},
			} {
				shape := routeRect{math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)}
				for _, corner := range []Point{{local.left, local.top}, {local.left, local.bottom}, {local.right, local.top}, {local.right, local.bottom}} {
					p := RotateLocal(corner, rotation)
					shape.left, shape.right = min(shape.left, p.X+part.Position.X-2), max(shape.right, p.X+part.Position.X+2)
					shape.top, shape.bottom = min(shape.top, p.Y+part.Position.Y-2), max(shape.bottom, p.Y+part.Position.Y+2)
				}
				if label.left <= shape.right && label.right >= shape.left && label.top <= shape.bottom && label.bottom >= shape.top {
					t.Errorf("%s rotation %d label overlaps %s: label %+v shape %+v", identity, rotation, name, label, shape)
				}
			}
		}
	}
}

func TestFitIncludesCapacitorLabelAtEveryRotation(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		s := New()
		defer s.Close()
		call(t, s, "component.place", PlaceParams{ID: strings.Repeat("C", 32), Kind: "capacitor", Value: "10uF", Position: Point{X: 100, Y: 40}}, "capacitor")
		for step := 0; step < rotation; step += 90 {
			call(t, s, "component.rotate", IDParams{ID: strings.Repeat("C", 32)}, fmt.Sprintf("rotate-%d", step))
		}
		before := s.Snapshot()
		call(t, s, "view.fit", FitParams{Width: 500, Height: 300}, "fit")
		after := s.Snapshot()
		x, y, text := schematicPartLabel(after.Components[0])
		for _, corner := range []Point{{x - 3, y - 18}, {x + schematicLabelWidth(text), y + 5}} {
			px := (corner.X-after.View.Center.X)*after.View.Zoom + 250
			py := (corner.Y-after.View.Center.Y)*after.View.Zoom + 150
			if px < 0 || px > 500 || py < 0 || py > 300 {
				t.Fatal("Fit clips capacitor label", rotation, corner, after.View)
			}
		}
		if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) {
			t.Fatal("label-aware Fit changed circuit/document", rotation)
		}
	}
}
