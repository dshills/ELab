package main

import (
	"math"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestNativeCapacitorUprightLabelUsesSharedAnchorAtEveryRotation(t *testing.T) {
	for _, id := range []string{"C1", strings.Repeat("C", 32)} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: id, Kind: "capacitor", Value: "10uF", Position: workbench.Point{X: 100, Y: 40}})
			for rotation := 0; rotation < 360; rotation += 90 {
				if rotation > 0 {
					uiAction(t, s, "rotate-part", nil)
				}
				state := s.service.Snapshot()
				part := state.Components[0]
				at := workbench.CapacitorLabelPosition(part)
				if part.Rotation != rotation || nativePartLabelPosition(part, id) != at {
					t.Fatal("native capacitor label diverged from output anchor", rotation, part)
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
						t.Fatalf("rotation %d has no capacitor label glyphs at its shared anchor: %d", rotation, count)
					}
				})
			}
		})
	}
}

func TestNativeFlasherCapacitorCaptionsClearNeighboringLEDLabels(t *testing.T) {
	graph := workbench.FlasherExampleGraph()
	bounds := func(id, title string) [4]float64 {
		part, _ := graph.Component(id)
		at := nativePartLabelPosition(part, title)
		builder := &meshBuilder{}
		builder.text(at.X, at.Y, title, paletteForTheme("light").Ink)
		if len(builder.vertices) == 0 {
			t.Fatal("empty caption mesh", id)
		}
		box := [4]float64{math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)}
		for _, vertex := range builder.vertices {
			p := vertex.Position
			box[0], box[1] = min(box[0], p.X), max(box[1], p.X)
			box[2], box[3] = min(box[2], p.Y), max(box[3], p.Y)
		}
		return box
	}
	for _, capacitor := range []string{"C1", "C2"} {
		caption := bounds(capacitor, capacitor+" 10 uF")
		for _, led := range []string{"D1", "D2"} {
			neighbor := bounds(led, led)
			if caption[0] <= neighbor[1]+4 && caption[1]+4 >= neighbor[0] && caption[2] <= neighbor[3]+4 && caption[3]+4 >= neighbor[2] {
				t.Fatalf("%s caption crowds %s: %v / %v", capacitor, led, caption, neighbor)
			}
		}
	}
}
