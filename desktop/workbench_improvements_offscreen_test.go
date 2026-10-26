//go:build offscreen

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedProposalReviewFitsAndAppliesAcrossThemes(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "rc"})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			agentAction(t, s.service, "edit.proposal.set", resistanceProposal("20k"))
			view := func() ui.Node {
				rail, _ := findNode(s.view(), "inspector-rail")
				return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
			}
			theme := inspectorLayoutTestTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			m := newMountedControlWindowSize(t, s, view, 320, 1000, theme, func(c *cogent.ManualConfig) { c.GPUSceneRendering = true })
			group := mountedControlElement(t, m, "inspector-group-proposal").Bounds
			for _, id := range []string{"proposal-name", "proposal-change-0", "proposal-apply", "proposal-refresh", "proposal-dismiss"} {
				control := mountedControlElement(t, m, id)
				if !control.Visible || control.Bounds.Empty() || !control.Bounds.In(group) {
					t.Fatal("proposal text/action clipped", id, control.Bounds, group)
				}
			}
			if directory := os.Getenv("ELAB_WORKBENCH_ARTIFACTS"); directory != "" {
				im, err := m.manual.Capture(m.ctx, "main")
				if err != nil {
					t.Fatal(err)
				}
				f, err := os.OpenFile(filepath.Join(directory, mode+"-proposal.png"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					t.Fatal(err)
				}
				encodeErr, closeErr := png.Encode(f, im), f.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			at := mountedControlElement(t, m, "proposal-apply").Bounds.Min.Add(image.Pt(20, 15))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			after := s.service.Snapshot()
			part, _ := (workbench.Graph{Components: after.Components}).Component("R1")
			if s.errorMessage != "" || after.Proposal != nil || part.Value != 20000 || after.UndoName != "Tune the RC response" {
				t.Fatal("actual routed Apply did not commit one shared batch", s.errorMessage)
			}
		})
	}
}
