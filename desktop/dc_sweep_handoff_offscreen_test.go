//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"

	"cogentcore.org/core/events"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedSweepStart(t *testing.T, s *screen, focus string) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: focus, Sequence: 1}}
		for _, id := range []string{"sweep-range-start", "sweep-range-stop", "sweep-points", "sweep-start"} {
			node, ok := findNode(assembled, id)
			if !ok {
				t.Fatalf("missing production sweep control %s", id)
			}
			root.Children = append(root.Children, node)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedDCSweepRefusesThenCorrectsInvalidSettings(t *testing.T) {
	for _, edit := range []struct{ id, bad, good string }{
		{"sweep-range-start", "nope", "1 V"}, {"sweep-range-stop", "13 V", "4 V"},
		{"sweep-points", "seven", "7"}, {"sweep-points", "999", "51"},
	} {
		t.Run(edit.id+"/"+edit.bad, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			m := mountedSweepStart(t, s, edit.id)
			typeMountedCursor(t, m, edit.id, edit.bad)
			clickMountedControl(t, m, "sweep-start")
			if s.errorMessage == "" || len(s.service.Snapshot().DCSweep.Operations) != 0 {
				t.Fatal("invalid draft started with previous settings")
			}
			typeMountedCursor(t, m, edit.id, edit.good)
			clickMountedControl(t, m, "sweep-start")
			if s.errorMessage != "" || len(s.service.Snapshot().DCSweep.Operations) != 1 {
				t.Fatal("one corrected Run failed", s.errorMessage)
			}
		})
	}
}

func TestMountedDCSweepHeldRunRefusesAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	m := mountedSweepStart(t, s, "sweep-points")
	typeMountedCursor(t, m, "sweep-points", "7")
	button := mountedControlElement(t, m, "sweep-start")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	m.frame()
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	after := s.service.Snapshot()
	if s.errorMessage == "" || len(after.DCSweep.Operations) != 0 || !reflect.DeepEqual(before.Components, after.Components) || before.ElectricalRevision != after.ElectricalRevision {
		t.Fatal("held Run captured changed source", s.errorMessage)
	}
	clickMountedControl(t, m, "sweep-start")
	if s.errorMessage != "" || len(s.service.Snapshot().DCSweep.Operations) != 1 {
		t.Fatal("fresh reviewed Run cannot start", s.errorMessage)
	}
}

func TestMountedDCSweepTargetChangeRetiresPrivatePoints(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	m := mountedSweepStart(t, s, "sweep-points")
	typeMountedCursor(t, m, "sweep-points", "7")
	original := mountedControlElement(t, m, "sweep-points")
	settings := s.service.Snapshot().DCSweep.Settings
	settings.ComponentID, settings.Parameter, settings.Start, settings.Stop = "R1", "resistance", "100 ohm", "10k ohm"
	agentAction(t, s.service, "sweep.settings.set", settings)
	before := s.service.Snapshot()
	m.frame()
	field := mountedControlElement(t, m, "sweep-points")
	if field.Incarnation == original.Incarnation || field.Value != "51" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("new target inherited private points", field.Value)
	}
	typeMountedCursor(t, m, "sweep-points", "9")
	clickMountedControl(t, m, "sweep-start")
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.DCSweep.Operations) != 1 || after.DCSweep.Settings.ComponentID != "R1" || after.DCSweep.Settings.Points != 9 {
		t.Fatal("fresh target edit failed", s.errorMessage)
	}
}

func TestMountedDCSweepRunUsesUnfinishedSettings(t *testing.T) {
	for _, activation := range []string{"pointer", "activation"} {
		for _, edit := range []struct{ id, text string }{
			{"sweep-range-start", "1 V"}, {"sweep-range-stop", "4 V"}, {"sweep-points", "7"},
		} {
			t.Run(activation+"/"+edit.id, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				uiAction(t, s, "example.reset", nil)
				uiAction(t, s, "simulation.pause", nil)
				m := mountedSweepStart(t, s, edit.id)
				before := s.service.Snapshot()
				typeMountedCursor(t, m, edit.id, edit.text)
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("private draft changed accepted sweep settings")
				}
				if activation == "pointer" {
					clickMountedControl(t, m, "sweep-start")
				} else {
					button := verificationControl(t, s, "sweep-start")
					s.app.Do(button.Activate)
					m.settle()
				}
				after := s.service.Snapshot()
				if s.errorMessage != "" || len(after.DCSweep.Operations) != 1 {
					t.Fatalf("one Run failed: operations=%d revision=%d→%d error=%s", len(after.DCSweep.Operations), before.CommandRevision, after.CommandRevision, s.errorMessage)
				}
				want := before.DCSweep.Settings
				switch edit.id {
				case "sweep-range-start":
					want.Start = edit.text
				case "sweep-range-stop":
					want.Stop = edit.text
				case "sweep-points":
					want.Points = 7
				}
				if !reflect.DeepEqual(after.DCSweep.Settings, want) {
					t.Fatal("Run used old settings", after.DCSweep.Settings, want)
				}
				curve := awaitNativeSweep(t, s)
				agent := workbench.NewExample()
				t.Cleanup(agent.Close)
				agentAction(t, agent, "sweep.start", workbench.SweepStartParams{Settings: &want})
				other := &screen{service: agent, app: s.app}
				expected := awaitNativeSweep(t, other)
				if !reflect.DeepEqual(curve.DCSweep.Result, expected.DCSweep.Result) {
					t.Fatal("completed native curve differs from public agent")
				}
				if before.RunID != curve.RunID || before.ElectricalRevision != curve.ElectricalRevision || !reflect.DeepEqual(before.Result, curve.Result) || !reflect.DeepEqual(before.Components, curve.Components) || !reflect.DeepEqual(before.Wires, curve.Wires) {
					t.Fatal("independent sweep changed the live circuit")
				}
			})
		}
	}
}
