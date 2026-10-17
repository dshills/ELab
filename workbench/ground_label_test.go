package workbench

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestGroundLabelClearsRotatedBarsAndTerminalStub(t *testing.T) {
	for _, id := range []string{"GND", strings.Repeat("G", 32)} {
		for _, rotation := range []int{0, 90, 180, 270} {
			part, err := MakeComponent(id, "ground", Point{100, -40}, "")
			if err != nil {
				t.Fatal(err)
			}
			part.Rotation = rotation
			x, y, text := schematicPartLabel(part)
			if (Point{x, y}) != GroundLabelPosition(part) || text != id {
				t.Fatal("ground export diverges from shared caption", part, x, y, text)
			}
			label := routeRect{x - 3, x + schematicLabelWidth(text), y - 18, y + 5}
			for name, local := range map[string]routeRect{
				"bars and lead": {-24, 24, -20, 16}, "terminal stub": {-2, 2, -44, -20},
			} {
				shape := routeRect{math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)}
				for _, corner := range []Point{{local.left, local.top}, {local.left, local.bottom}, {local.right, local.top}, {local.right, local.bottom}} {
					p := RotateLocal(corner, rotation)
					shape.left, shape.right = min(shape.left, p.X+part.Position.X), max(shape.right, p.X+part.Position.X)
					shape.top, shape.bottom = min(shape.top, p.Y+part.Position.Y), max(shape.bottom, p.Y+part.Position.Y)
				}
				if label.left <= shape.right && label.right >= shape.left && label.top <= shape.bottom && label.bottom >= shape.top {
					t.Errorf("%s rotation %d caption overlaps %s: %+v / %+v", id, rotation, name, label, shape)
				}
			}
		}
	}
}

func TestGroundLabelFitPreservesCircuitAtEveryRotation(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		s := New()
		defer s.Close()
		id := strings.Repeat("G", 32)
		call(t, s, "component.place", PlaceParams{ID: id, Kind: "ground", Position: Point{X: 100, Y: 40}}, "ground")
		for step := 0; step < rotation; step += 90 {
			call(t, s, "component.rotate", IDParams{ID: id}, fmt.Sprintf("rotate-%d", step))
		}
		before := s.Snapshot()
		call(t, s, "view.fit", FitParams{Width: 500, Height: 300}, "fit")
		after := s.Snapshot()
		x, y, text := schematicPartLabel(after.Components[0])
		for _, corner := range []Point{{x - 3, y - 18}, {x + schematicLabelWidth(text), y + 5}} {
			px, py := (corner.X-after.View.Center.X)*after.View.Zoom+250, (corner.Y-after.View.Center.Y)*after.View.Zoom+150
			if px < 0 || px > 500 || py < 0 || py > 300 {
				t.Fatal("Fit clips ground caption", rotation, corner, after.View)
			}
		}
		if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
			t.Fatal("caption Fit changed circuit/document/history", rotation)
		}
	}
}
