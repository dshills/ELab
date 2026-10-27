package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCurrentDevicesInspectorAgentParityAndStaleGuards(t *testing.T) {
	for _, id := range []string{"current-led", "load-step", "constant-power-load"} {
		t.Run(id, func(t *testing.T) {
			u, done := testScreen(t)
			defer done()
			a := workbench.New()
			defer a.Close()
			for _, s := range []*workbench.Service{u.service, a} {
				agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: id})
				agentAction(t, s, "simulation.pause", struct{}{})
				agentAction(t, s, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, s)
			}
			field := "current-level"
			command := "current-source.configure"
			target := "I1"
			var params any
			state := a.Snapshot()
			if id == "current-led" {
				for _, p := range state.Components {
					if p.ID == target {
						c := *p.Device.Current
						c.Current = .02
						params = workbench.CurrentParams{ID: target, Parameters: c}
					}
				}
				uiAction(t, u, field, &ui.ChangeEvent{Number: 20})
			} else {
				field, command, target = "load-level", "load.configure", "LOAD1"
				for _, p := range state.Components {
					if p.ID == target {
						c := *p.Device.Load
						c.Level = .1
						params = workbench.ElectronicLoadParams{ID: target, Parameters: c}
					}
				}
				uiAction(t, u, field, &ui.ChangeEvent{Number: .1})
			}
			agentAction(t, a, command, params)
			if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
				t.Fatal("configuration parity")
			}
			for _, s := range []*workbench.Service{u.service, a} {
				agentAction(t, s, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, s)
			}
			uiAction(t, u, "transient-step", nil)
			agentAction(t, a, "simulation.step", workbench.StepParams{Duration: a.Snapshot().TransientSettings.StepSeconds})
			if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
				t.Fatal("electrical parity")
			}
			held, ok := findNode(u.view(), field)
			if !ok || held.Change == nil {
				t.Fatal("missing current field")
			}
			agentAction(t, u.service, "component.move", workbench.MoveParams{ID: target, Position: workbench.Point{X: 100, Y: 100}})
			before := u.service.Snapshot().Components
			u.app.Do(func() { held.Change(ui.ChangeEvent{Number: .03}) })
			if !reflect.DeepEqual(before, u.service.Snapshot().Components) {
				t.Fatal("stale current edit")
			}
			for _, theme := range []string{"light", "dark", "system", "apollo"} {
				agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
				if err := ui.Validate(u.view()); err != nil {
					t.Fatal(theme, err)
				}
			}
		})
	}
}

func TestCurrentMaximumComplianceInspectorParity(t *testing.T) {
	u, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.pause", struct{}{})
		for _, p := range []workbench.PlaceParams{{ID: "I1", Kind: "current-source", Position: workbench.Point{X: -200}}, {ID: "R1", Kind: "resistor", Position: workbench.Point{X: 200}, Value: "100k"}, {ID: "GND", Kind: "ground", Position: workbench.Point{Y: 200}}} {
			agentAction(t, s, "component.place", p)
		}
		for _, w := range []workbench.ConnectParams{{From: "I1.positive", To: "R1.1"}, {From: "I1.negative", To: "GND.reference"}, {From: "R1.2", To: "GND.reference"}} {
			agentAction(t, s, "wire.connect", w)
		}
		agentAction(t, s, "component.select", workbench.IDParams{ID: "I1"})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	uiAction(t, u, "current-compliance", &ui.ChangeEvent{Number: 24})
	cfg := workbench.DefaultCurrentConfig()
	cfg.Compliance = 24
	agentAction(t, a, "current-source.configure", workbench.CurrentParams{ID: "I1", Parameters: cfg})
	agentAction(t, u.service, "simulation.restart", struct{}{})
	agentAction(t, a, "simulation.restart", struct{}{})
	left, right := awaitInductorCircuit(t, u.service), awaitInductorCircuit(t, a)
	if left.Mixed == nil || len(left.Mixed.Sources) != 1 || left.Mixed.Sources[0].Voltage <= 23 || !reflect.DeepEqual(left.Mixed, right.Mixed) {
		t.Fatal("maximum compliance UI/agent mismatch", left.Diagnostics, right.Diagnostics)
	}
}
