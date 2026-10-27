package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestJFETInspectorAgentElectricalParity(t *testing.T) {
	for _, id := range []string{"jfet-current", "pjfet-current", "jfet-preamp"} {
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
			var cfg workbench.JFETConfig
			for _, p := range a.Snapshot().Components {
				if p.ID == "J1" {
					cfg = *p.Device.JFET
				}
			}
			cfg.IDSS = .008
			uiAction(t, u, "jfet-idss", &ui.ChangeEvent{Number: 8})
			agentAction(t, a, "jfet.configure", workbench.JFETParams{ID: "J1", Parameters: cfg})
			if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
				t.Fatal("JFET configure parity")
			}
			for _, s := range []*workbench.Service{u.service, a} {
				agentAction(t, s, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, s)
			}
			if id == "jfet-preamp" {
				uiAction(t, u, "transient-step", nil)
				agentAction(t, a, "simulation.step", workbench.StepParams{Duration: a.Snapshot().TransientSettings.StepSeconds})
			}
			if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
				t.Fatal("JFET electrical parity")
			}
			held, ok := findNode(u.view(), "jfet-cutoff")
			if !ok || held.Change == nil {
				t.Fatal("missing JFET field")
			}
			cfg.Lambda = .02
			agentAction(t, u.service, "jfet.configure", workbench.JFETParams{ID: "J1", Parameters: cfg})
			before := u.service.Snapshot().Components
			u.app.Do(func() { held.Change(ui.ChangeEvent{Number: 4}) })
			if !reflect.DeepEqual(before, u.service.Snapshot().Components) {
				t.Fatal("stale JFET config overwrote agent")
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
