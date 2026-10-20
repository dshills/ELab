//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
)

func prepareGeometryPart(t *testing.T, service *workbench.Service, kind string) string {
	t.Helper()
	id := "PART"
	if kind == "resistor" {
		agentAction(t, service, "example.reset", workbench.ReplaceParams{})
		id = "R1"
	} else {
		agentAction(t, service, "component.place", workbench.PlaceParams{ID: id, Kind: kind})
		switch kind {
		case "net-label":
			agentAction(t, service, "component.set", workbench.ValueParams{ID: id, Value: "OUTPUT"})
		case "capacitor":
			agentAction(t, service, "capacitor.initial.set", workbench.ValueParams{ID: id, Value: "2.5 V"})
		case "inductor":
			agentAction(t, service, "inductor.initial.set", workbench.ValueParams{ID: id, Value: "-20 mA"})
		case "sine-source":
			agentAction(t, service, "source.sine.set", workbench.SineParams{ID: id, Waveform: workbench.SineWaveform{Amplitude: 1.25, Offset: -.5, Frequency: 17, Phase: 45}})
		case "finite-pulse-source":
			agentAction(t, service, "source.pulse.set", workbench.PulseParams{ID: id, Waveform: workbench.PulseWaveform{HighVoltage: -1, LowVoltage: -3, HighSeconds: .02, LowSeconds: .03, RiseSeconds: .001, FallSeconds: .002}})
		}
	}
	agentAction(t, service, "simulation.pause", struct{}{})
	agentAction(t, service, "component.select", workbench.IDParams{ID: id})
	return id
}

func geometryCommand(control, id string) (string, any) {
	if control == "rotate-part" {
		return "component.rotate", workbench.IDParams{ID: id}
	}
	return "component.duplicate", workbench.DuplicateParams{ID: id}
}

func TestMountedButtonOwnerGeometrySettlesValueDraft(t *testing.T) {
	for _, control := range []string{"rotate-part", "duplicate-part"} {
		for _, kind := range []string{"resistor", "net-label", "capacitor", "inductor", "sine-source", "finite-pulse-source"} {
			t.Run(control+"/"+kind, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				id := prepareGeometryPart(t, s.service, kind)
				reference := workbench.New()
				t.Cleanup(reference.Close)
				prepareGeometryPart(t, reference, kind)
				m := mountedValuePreset(t, s, control)
				before := s.service.Snapshot()
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
				// Drafts, including invalid values, are not configured component values.
				draft := "unfinished value"
				for _, r := range draft {
					m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
				}
				if f := mountedControlElement(t, m, "value-entry"); f.Value != draft || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("private fixture failed", f.Value)
				}
				clickMountedControl(t, m, control)
				after := s.service.Snapshot()
				if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || s.pendingValueApply != nil {
					t.Fatalf("one geometry click must settle only its draft and act once: %d→%d, %s", before.CommandRevision, after.CommandRevision, s.errorMessage)
				}
				agentAction(t, reference, "editor.value.set", workbench.DraftParams{Value: draft})
				command, params := geometryCommand(control, id)
				agentAction(t, reference, command, params)
				want := reference.Snapshot()
				if !reflect.DeepEqual(after.Components, want.Components) || !reflect.DeepEqual(after.Wires, before.Wires) || after.Selected != want.Selected || after.Drawing.Value != want.Drawing.Value {
					t.Fatal("geometry differs from public draft/geometry commands")
				}
				if control == "rotate-part" && (after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Result, before.Result) || !reflect.DeepEqual(after.Transient, before.Transient)) {
					t.Fatal("rotation changed accepted physics")
				}
				uiAction(t, s, "edit.undo", nil)
				if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) || !reflect.DeepEqual(before.Wires, s.service.Snapshot().Wires) {
					t.Fatal("one Undo did not restore the configured original graph")
				}
			})
		}
	}
}

func TestMountedButtonOwnerGeometryRefusesAgentAfterDraft(t *testing.T) {
	for _, control := range []string{"rotate-part", "duplicate-part"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareGeometryPart(t, s.service, "resistor")
			m := mountedValuePreset(t, s, control)
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
			for _, r := range "9k" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			button := mountedControlElement(t, m, control)
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
			before := s.service.Snapshot()
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("held geometry action adopted an intervening agent edit")
			}
			clickMountedControl(t, m, control)
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("fresh geometry action did not work", s.errorMessage)
			}
		})
	}
}
