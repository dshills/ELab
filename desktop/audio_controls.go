package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) audioControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{s.action("audio-render", "Render 250 ms preview", "audio.render", workbench.AudioRenderParams{ID: p.ID, Start: .02, Duration: .25, Gain: .1}), quiet("audio-level-help", "From circuit startup · 20 ms offset\nGain 10% · no autoplay")}
	if a := state.Audio; a != nil && a.Component == p.ID {
		nodes = append(nodes, quiet("audio-status", fmt.Sprintf("%s · %d bytes · current %t", a.Status, a.Bytes, a.Current)))
		if a.Error != nil {
			nodes = append(nodes, quiet("audio-error", a.Error.Message))
		}
		if a.Status == "completed" && a.Current {
			nodes = append(nodes, s.action("audio-play", "Play preview", "audio.play", workbench.IDParams{ID: a.ID}), s.action("audio-stop", "Stop", "audio.stop", workbench.IDParams{ID: a.ID}))
			export := buttonStyle(ui.Node{ID: "audio-export", Kind: ui.Button, Text: "Export WAV…", Activate: func() {
				s.beginFile("export.start", true, workbench.ExportStartParams{ID: a.ID, Kind: "audio-wav"}, nil)
			}}, false)
			nodes = append(nodes, export)
		} else if a.Status == "running" {
			nodes = append(nodes, s.action("audio-stop", "Cancel rendering", "audio.stop", workbench.IDParams{ID: a.ID}))
		}
	}
	return nodes
}
