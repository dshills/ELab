package main

import (
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestNativeGroundCaptionUsesSharedAnchorAtEveryRotation(t *testing.T) {
	for _, id := range []string{"GND", strings.Repeat("G", 32)} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: id, Kind: "ground", Position: workbench.Point{X: 100, Y: 40}})
			for rotation := 0; rotation < 360; rotation += 90 {
				if rotation > 0 {
					uiAction(t, s, "rotate-part", nil)
				}
				part := s.service.Snapshot().Components[0]
				at := workbench.GroundLabelPosition(part)
				if part.Rotation != rotation || nativePartLabelPosition(part, id) != at {
					t.Fatal("native ground diverges from shared caption", rotation, part)
				}
				s.app.Do(func() {
					mesh, err := s.mesh(*s.drawing.View)
					if err != nil {
						t.Fatal(err)
					}
					ink, count := paletteForTheme(s.service.Snapshot().Theme).Ink, 0
					for _, vertex := range mesh.Vertices() {
						p := vertex.Position
						if vertex.Color.R == ink.R && vertex.Color.G == ink.G && vertex.Color.B == ink.B && vertex.Color.A > 96 && p.X >= at.X && p.X <= at.X+float64(len(id))*8.75+2 && p.Y >= at.Y-18 && p.Y <= at.Y+5 {
							count++
						}
					}
					if count < 40 {
						t.Fatalf("rotation %d has no ground caption ink at shared anchor: %d", rotation, count)
					}
				})
			}
		})
	}
}
