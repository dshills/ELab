package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestThermalInspectorUserAgentParity(t *testing.T) {
	u, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "self-heating"})
		agentAction(t, s, "simulation.pause", struct{}{})
		awaitBlockRC(t, s)
	}
	uiAction(t, u, "thermal-ambient", &ui.ChangeEvent{Number: 40})
	cfg := workbench.DefaultThermalConfig("resistor")
	cfg.Ambient = 40
	cfg.Alpha = 0
	cfg.SelfHeating = true
	cfg.Resistance = 100
	cfg.Capacitance = .001
	agentAction(t, a, "thermal.configure", workbench.ThermalParams{ID: "R1", Enabled: true, Parameters: cfg})
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("ambient parity")
	}
	held, ok := findNode(u.view(), "thermal-ambient")
	if !ok {
		t.Fatal("thermal field absent")
	}
	cfg.Ambient = 45
	agentAction(t, u.service, "thermal.configure", workbench.ThermalParams{ID: "R1", Enabled: true, Parameters: cfg})
	before := u.service.Snapshot().Components
	u.app.Do(func() { held.Change(ui.ChangeEvent{Number: 50}) })
	if !reflect.DeepEqual(before, u.service.Snapshot().Components) {
		t.Fatal("stale temperature input accepted")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
