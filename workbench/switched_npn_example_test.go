package workbench

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func TestManualTransistorExampleMatchesTheveninReferenceAndUndo(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.switch-npn", ReplaceParams{}, "load")
	for index, closed := range []bool{false, true, false} {
		call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: closed}, fmt.Sprint("switch-", index))
		state := settled(t, s)
		if state.Analysis != NetworkDCAnalysis || state.Mixed == nil || state.Mixed.Stale || state.Example.Info.ID != "switch-npn" {
			t.Fatal("manual example lacks a fresh network result")
		}
		// Replace the linear switch/base/pull-down network with its exact
		// Thevenin source, then solve with the separate nested-bisection kernel.
		series := 1e9 + 10000.
		if closed {
			series = 1 + 10000.
		}
		g := NPNExampleGraph()
		for i := range g.Components {
			switch g.Components[i].ID {
			case "VIN":
				g.Components[i].Value = 5 * 100000 / (series + 100000)
			case "RB":
				g.Components[i].Value = series * 100000 / (series + 100000)
			}
		}
		c, _, diagnostic := g.Compile()
		if diagnostic != nil {
			t.Fatal(diagnostic)
		}
		ref, err := SolveNPN(context.Background(), c)
		if err != nil {
			t.Fatal(err)
		}
		q := state.Mixed.Transistors[0]
		// Use the release suite's max(1 mV, 1%) voltage bound. Near
		// cutoff, femtoamp leakage makes the LED-side voltage ill-conditioned.
		if math.Abs(q.VBE-ref.VBE) > math.Max(.001, .01*math.Abs(ref.VBE)) || math.Abs(q.VCE-ref.VCE) > math.Max(.001, .01*math.Abs(ref.VCE)) || math.Abs(q.BaseCurrent-ref.BaseCurrent) > 1e-9 || math.Abs(q.CollectorCurrent-ref.CollectorCurrent) > 1e-9 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatalf("manual fixture differs from reference: %+v / %+v", q, ref)
		}
		if closed && state.Mixed.LEDs[0].Current < .002 || !closed && math.Abs(state.Mixed.LEDs[0].Current) > 1e-8 {
			t.Fatal("switch fails to control LED current")
		}
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo-open")
	if state := settled(t, s); !state.SwitchStates[0].DesiredClosed || state.Mixed.LEDs[0].Current < .002 {
		t.Fatal("undo does not restore the lit circuit")
	}
}
