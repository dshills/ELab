package workbench

import (
	"fmt"
	"strings"
	"testing"
)

func TestDCSourceCaptionClearsSymbolAtFittedZooms(t *testing.T) {
	for _, id := range []string{"V1", strings.Repeat("SUPPLY", 5)} {
		for _, rotation := range []int{0, 90, 180, 270} {
			for _, zoom := range []float64{.25, .3328, .4724, .6, 1, 2} {
				t.Run(fmt.Sprintf("%s/%d/%g", id, rotation, zoom), func(t *testing.T) {
					part := Component{ID: id, Kind: "dc-source", Unit: "V", Value: 5, Rotation: rotation, Position: Point{X: 100, Y: 200}}
					_, text := CanvasPartLabel(part)
					at := CanvasPartLabelPositionAtZoom(part, text, zoom)
					_, hi := CanvasTextBounds(at, text, CanvasFontScale(zoom))
					// The native source has a 34-unit radius. Leave clearance for
					// its visible stroke; horizontal leads need the text above them.
					if rotation == 90 || rotation == 270 {
						if hi.Y > part.Position.Y-38 {
							t.Fatalf("caption reaches horizontal symbol/lead row: %+v", hi)
						}
					} else if hi.X > part.Position.X-38 {
						t.Fatalf("caption reaches source circle: %+v", hi)
					}
				})
			}
		}
	}
}

func TestLongDCSourceCaptionFitsAtEveryRotation(t *testing.T) {
	s := New()
	defer s.Close()
	id := strings.Repeat("SUPPLY", 5)
	call(t, s, "component.place", PlaceParams{ID: id, Kind: "dc-source", Position: Point{X: 100, Y: 200}, Value: "12"}, "place")
	call(t, s, "view.viewport.set", ViewportParams{Width: 456, Height: 209}, "viewport")
	call(t, s, "view.fit", FitParams{FollowViewport: true}, "fit")
	for rotation := 0; rotation < 360; rotation += 90 {
		if rotation > 0 {
			call(t, s, "component.rotate", IDParams{ID: id}, fmt.Sprint(rotation))
		}
		state := s.Snapshot()
		if state.Components[0].Rotation != rotation || state.Viewport.FitFailure != nil {
			t.Fatal("source rotation lost Fit", state.Viewport)
		}
		assertViewportContainsSchematic(t, state)
	}
}
