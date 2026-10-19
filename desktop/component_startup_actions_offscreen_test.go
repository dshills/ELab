//go:build offscreen

package main

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedComponentStartup(t *testing.T, s *screen, kind string) *mountedControlWindow {
	t.Helper()
	field, reset := "inductor-initial", "inductor-initial-default"
	if kind == "capacitor" {
		field, reset = "cap-initial", "cap-initial-default"
	}
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: field, Sequence: 1}}
		for _, id := range []string{field, reset, "simulation.restart"} {
			node, ok := findNode(assembled, id)
			if !ok {
				t.Fatal("missing production startup control", id)
			}
			root.Children = append(root.Children, node)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func prepareComponentStartup(t *testing.T, s *screen, kind string) {
	t.Helper()
	example, part, command, value := "rl", "L1", "inductor.initial.set", "-20 mA"
	if kind == "capacitor" {
		example, part, command, value = "rc", "C1", "capacitor.initial.set", "2 V"
	}
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: example})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: part})
	agentAction(t, s.service, command, workbench.ValueParams{ID: part, Value: value})
	agentAction(t, s.service, "simulation.restart", struct{}{})
	uiAction(t, s, "transient-step", nil)
}

func TestMountedComponentStartupOneClick(t *testing.T) {
	for _, fixture := range []struct{ kind, action, draft string }{
		{"inductor", "inductor-initial-default", "30 mA"},
		{"inductor", "simulation.restart", "30 mA"},
		{"capacitor", "cap-initial-default", "3"},
		{"capacitor", "simulation.restart", "3"},
		{"inductor", "inductor-initial-default", "20 mV"},
		{"capacitor", "cap-initial-default", "-"},
	} {
		t.Run(fixture.kind+"-"+fixture.action, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareComponentStartup(t, s, fixture.kind)
			m := mountedComponentStartup(t, s, fixture.kind)
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
			for _, r := range fixture.draft {
				code := key.CodeUnknown
				if fixture.kind == "capacitor" {
					code = key.Code3
					if r == '-' {
						code = key.CodeHyphenMinus
					}
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r, Code: code})
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("typing committed before the action")
			}
			clickMountedControl(t, m, fixture.action)
			after := s.service.Snapshot()
			id := "L1"
			if fixture.kind == "capacitor" {
				id = "C1"
			}
			part, _ := (workbench.Graph{Components: after.Components}).Component(id)
			if fixture.action == "simulation.restart" {
				initial := part.InitialCurrent
				expected := .03
				if fixture.kind == "capacitor" {
					initial = part.InitialVoltage
					expected = 3
				}
				if s.errorMessage != "" || initial == nil || math.Abs(*initial-expected) > 1e-12 || after.Transient.Sample.Time != 0 || after.Transient.Stale {
					t.Fatalf("first Restart missed typed startup: part=%+v time=%g stale=%t error=%s", part, after.Transient.Sample.Time, after.Transient.Stale, s.errorMessage)
				}
			} else if s.errorMessage != "" || fixture.kind == "inductor" && part.InitialCurrent != nil || fixture.kind == "capacitor" && part.InitialVoltage != nil {
				t.Fatalf("first default click missed typed startup: part=%+v error=%s", part, s.errorMessage)
			}
		})
	}
}

func TestMountedComponentStartupDescriptorAndDiscard(t *testing.T) {
	for _, kind := range []string{"inductor", "capacitor"} {
		for _, reset := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-reset-%t", kind, reset), func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				prepareComponentStartup(t, s, kind)
				m := mountedComponentStartup(t, s, kind)
				field, action := "inductor-initial", "simulation.restart"
				if kind == "capacitor" {
					field = "cap-initial"
				}
				if reset {
					action = "inductor-initial-default"
					if kind == "capacitor" {
						action = "cap-initial-default"
					}
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
				text := "30 mA"
				if kind == "capacitor" {
					text = "3"
				}
				for _, r := range text {
					code := key.CodeUnknown
					if kind == "capacitor" {
						code = key.Code3
					}
					m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r, Code: code})
				}
				node := verificationControl(t, s, action)
				s.app.Do(node.Activate)
				m.settle()
				state := s.service.Snapshot()
				if s.errorMessage != "" {
					t.Fatal("descriptor action failed", s.errorMessage)
				}
				id := "L1"
				if kind == "capacitor" {
					id = "C1"
				}
				part, _ := (workbench.Graph{Components: state.Components}).Component(id)
				value := part.InitialCurrent
				expected := .03
				if kind == "capacitor" {
					value = part.InitialVoltage
					expected = 3
				}
				if reset {
					if value != nil {
						t.Fatal("descriptor reset kept private startup")
					}
				} else if value == nil || math.Abs(*value-expected) > 1e-12 || state.Transient.Sample.Time != 0 || state.Transient.Stale {
					t.Fatal("descriptor Restart missed private startup", s.errorMessage)
				}
				// Reset must discard even an invalid private edit when the model is
				// already default. The field's old native incarnation must retire.
				if !reset {
					return
				}
				clickMountedControl(t, m, field)
				original := mountedControlElement(t, m, field)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
				invalid := "20 mV"
				if kind == "capacitor" {
					invalid = "-"
				}
				for _, r := range invalid {
					code := key.CodeUnknown
					if kind == "capacitor" {
						code = key.CodeHyphenMinus
					}
					m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r, Code: code})
				}
				restart := verificationControl(t, s, "simulation.restart")
				before := s.service.Snapshot()
				s.app.Do(restart.Activate)
				m.settle()
				if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("invalid startup restarted using old accepted state", s.errorMessage)
				}
				node = verificationControl(t, s, action)
				s.app.Do(node.Activate)
				m.settle()
				if s.errorMessage != "" || mountedControlElement(t, m, field).Incarnation == original.Incarnation {
					t.Fatal("reset failed to retire invalid private text", s.errorMessage)
				}
			})
		}
	}
}

func TestMountedComponentStartupHeldActionRefusesAgent(t *testing.T) {
	for _, action := range []string{"inductor-initial-default", "simulation.restart"} {
		t.Run(action, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareComponentStartup(t, s, "inductor")
			m := mountedComponentStartup(t, s, "inductor")
			typeMountedCursor(t, m, "inductor-initial", "30 mA")
			button := mountedControlElement(t, m, action)
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			agentAction(t, s.service, "inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "40 mA"})
			m.frame()
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("held action overwrote an agent's newer startup", s.errorMessage)
			}
			clickMountedControl(t, m, action)
			if s.errorMessage != "" {
				t.Fatal("fresh action failed", s.errorMessage)
			}
		})
	}
}

func TestMountedComponentStartupDeferredGuards(t *testing.T) {
	for _, reset := range []bool{false, true} {
		for _, scenario := range []string{"none", "agent-before", "agent-after", "selection", "document", "other-user-field", "settlement-error", "admission-error"} {
			t.Run(fmt.Sprintf("reset-%t-%s", reset, scenario), func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				prepareComponentStartup(t, s, "inductor")
				field := verificationControl(t, s, "inductor-initial")
				var finish func(error)
				s.settleField = func(_ string, done func(error)) error {
					finish = done
					if scenario == "admission-error" {
						return errors.New("field retired")
					}
					return nil
				}
				action := "simulation.restart"
				if reset {
					action = "inductor-initial-default"
				}
				button := verificationControl(t, s, action)
				s.app.Do(button.Activate)
				if finish == nil {
					t.Fatal("startup action did not await field")
				}
				if scenario == "agent-before" {
					agentAction(t, s.service, "inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "40 mA"})
				}
				if scenario != "admission-error" {
					s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "30 mA"}) })
				}
				switch scenario {
				case "agent-after":
					agentAction(t, s.service, "inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "40 mA"})
				case "selection":
					agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
				case "document":
					agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "rl", DiscardChanges: true})
				case "other-user-field":
					uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
				}
				before := s.service.Snapshot()
				s.app.Do(func() {
					if scenario == "settlement-error" {
						finish(errors.New("composition incomplete"))
					} else {
						finish(nil)
					}
					finish(nil)
				})
				if scenario == "none" {
					after := s.service.Snapshot()
					if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
						t.Fatal("deferred action did not finish exactly once", s.errorMessage)
					}
				} else if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("startup action changed intervening work", scenario, s.errorMessage)
				}
			})
		}
	}
}

func TestMountedComponentStartupViewportOwnership(t *testing.T) {
	for _, gap := range []bool{false, true} {
		t.Run(fmt.Sprint("agent-gap-", gap), func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareComponentStartup(t, s, "inductor")
			button := verificationControl(t, s, "inductor-initial-default")
			field := verificationControl(t, s, "inductor-initial")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "30 mA"}) })
			if gap {
				agentAction(t, s.service, "inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "40 mA"})
			}
			s.drawing.widget = &core.WidgetBase{}
			beforeViewport := s.service.Snapshot()
			s.app.Do(func() { s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 138, Zoom: 1}, beforeViewport) })
			drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Registered })
			before := s.service.Snapshot()
			s.app.Do(button.Activate)
			after := s.service.Snapshot()
			if gap {
				if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
					t.Fatal("viewport hid agent's newer startup")
				}
			} else {
				part, _ := (workbench.Graph{Components: after.Components}).Component("L1")
				if s.errorMessage != "" || part.InitialCurrent != nil {
					t.Fatal("owner viewport broke own startup handoff", s.errorMessage)
				}
			}
		})
	}
}

func TestMountedComponentStartupMatchesPublicAgent(t *testing.T) {
	for _, kind := range []string{"inductor", "capacitor"} {
		for _, reset := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-reset-%t", kind, reset), func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				prepareComponentStartup(t, s, kind)
				reference := workbench.New()
				t.Cleanup(reference.Close)
				example, id, command, initial, draft, field := "rl", "L1", "inductor.initial.set", "-20 mA", "30 mA", "inductor-initial"
				if kind == "capacitor" {
					example, id, command, initial, draft, field = "rc", "C1", "capacitor.initial.set", "2 V", "3 V", "cap-initial"
				}
				for _, request := range []struct {
					command string
					params  any
				}{
					{"example.load", workbench.ExampleLoadParams{ID: example}},
					{"simulation.pause", struct{}{}}, {"component.select", workbench.IDParams{ID: id}},
					{command, workbench.ValueParams{ID: id, Value: initial}}, {"simulation.restart", struct{}{}}, {"simulation.step", workbench.StepParams{Duration: .001}},
				} {
					agentAction(t, reference, request.command, request.params)
				}
				m := mountedComponentStartup(t, s, kind)
				if kind == "inductor" {
					typeMountedCursor(t, m, field, "30 mA")
				} else {
					clickMountedControl(t, m, field)
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
					m.input(cogent.ManualInput{Type: events.KeyChord, Rune: '3', Code: key.Code3})
				}
				agentAction(t, reference, command, workbench.ValueParams{ID: id, Value: draft})
				action := "simulation.restart"
				if reset {
					action = "inductor-initial-default"
					if kind == "capacitor" {
						action = "cap-initial-default"
					}
					agentAction(t, reference, command, workbench.ValueParams{ID: id, Value: "default"})
				} else {
					agentAction(t, reference, "simulation.restart", struct{}{})
				}
				clickMountedControl(t, m, action)
				compare := func() {
					a, b := s.service.Snapshot(), reference.Snapshot()
					if s.errorMessage != "" || !reflect.DeepEqual(a.Components, b.Components) || !reflect.DeepEqual(a.Wires, b.Wires) || !reflect.DeepEqual(a.Transient, b.Transient) || !reflect.DeepEqual(a.Mixed, b.Mixed) {
						t.Fatal("startup action differs from independent public commands", s.errorMessage)
					}
				}
				compare()
				// Paused startup edits remain stale until an explicit restart. Compare
				// the resulting trajectory too, rather than only the component setting.
				if reset {
					uiAction(t, s, "simulation.restart", nil)
					m.settle()
					agentAction(t, reference, "simulation.restart", struct{}{})
				}
				for step := 0; step < 6; step++ {
					compare()
					state := s.service.Snapshot()
					at := float64(step) * .001
					if kind == "inductor" {
						start := .03
						if reset {
							start = 0
						}
						want := .05 + (start-.05)*math.Exp(-at/.01)
						if math.Abs(state.Mixed.Inductors[0].Current-want) > 1e-5 {
							t.Fatal("RL startup trajectory differs from equation")
						}
					} else {
						start := 3.
						if reset {
							start = 0
						}
						want := 5 + (start-5)*math.Exp(-at/.01)
						if math.Abs(state.Transient.Sample.Voltage-want) > 1e-12 {
							t.Fatal("RC startup trajectory differs from equation")
						}
					}
					uiAction(t, s, "transient-step", nil)
					agentAction(t, reference, "simulation.step", workbench.StepParams{Duration: .001})
				}
			})
		}
	}
}
