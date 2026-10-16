package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) inductorControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	initial := 0.
	if part.InitialCurrent != nil {
		initial = *part.InitialCurrent
	}
	nodes := []ui.Node{quiet("inductor-initial-label", "Startup current · ±1 A"), {
		ID: "inductor-initial", Kind: ui.TextField, Text: workbench.FormatValue(initial, "A"),
		Accessibility: ui.Accessibility{Name: "Inductor initial current", Help: "Positive current flows from pin 1 toward pin 2. Signed current within ±1 A. Enter A, mA or µA, for example -20 mA. Commits at Return or focus change."},
		Change: func(e ui.ChangeEvent) {
			s.invokeAtCommandRevision("inductor.initial.set", workbench.ValueParams{ID: part.ID, Value: e.Text}, state.CommandRevision)
		},
	}, s.action("inductor-initial-default", "Reset startup to zero", "inductor.initial.set", workbench.ValueParams{ID: part.ID, Value: "default"}), quiet("inductor-startup-help", "Positive current flows from pin 1 toward pin 2. After a paused startup edit, choose Restart to apply it. Live switching preserves coil current and needs a current return path.")}
	if result := state.Mixed; result != nil {
		for _, reading := range result.Inductors {
			if reading.ID == part.ID {
				status := "Live"
				if result.Stale {
					status = "Stale"
				}
				nodes = append(nodes, quiet("inductor-reading", fmt.Sprintf("%s · %+.6g mA · %+.6g V · energy %.6g µJ", status, reading.Current*1000, reading.Voltage, reading.Energy*1e6)))
			}
		}
	}
	return nodes
}
