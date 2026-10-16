package main

import (
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestNativeLEDUprightLabelUsesSharedAnchorAtEveryRotation(t *testing.T) {
	for _, id := range []string{"D1", strings.Repeat("D", 32)} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: id, Kind: "led", Position: workbench.Point{X: 100, Y: 40}})
			for rotation := 0; rotation < 360; rotation += 90 {
				if rotation > 0 {
					uiAction(t, s, "rotate-part", nil)
				}
				state := s.service.Snapshot()
				part := state.Components[0]
				at := workbench.LEDLabelPosition(part)
				if part.Rotation != rotation || nativePartLabelPosition(part, id) != at {
					t.Fatal("native LED label diverged from output anchor", rotation, part)
				}
				s.app.Do(func() {
					mesh, err := s.mesh(*s.drawing.View)
					if err != nil {
						t.Fatal(err)
					}
					ink, count := paletteForTheme(state.Theme).Ink, 0
					for _, vertex := range mesh.Vertices() {
						p := vertex.Position
						if vertex.Color == ink && p.X >= at.X && p.X <= at.X+float64(len(id))*8.75+2 && p.Y >= at.Y-18 && p.Y <= at.Y+5 {
							count++
						}
					}
					if count < 40 {
						t.Fatalf("rotation %d has no LED label glyphs at its shared anchor: %d", rotation, count)
					}
				})
			}
		})
	}
}
