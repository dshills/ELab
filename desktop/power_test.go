package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestBatteryNativeStartupControlsAndAgentParity(t *testing.T) {
	for _, example := range []string{"battery-led", "battery-regulator"} {
		t.Run(example, func(t *testing.T) {
			u, cleanup := testScreen(t)
			defer cleanup()
			a := workbench.New()
			defer a.Close()
			uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
			agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: example})
			for _, c := range []string{"simulation.pause", "simulation.restart"} {
				uiAction(t, u, c, nil)
				agentAction(t, a, c, struct{}{})
			}
			awaitInductorCircuit(t, u.service)
			awaitInductorCircuit(t, a)
			before := u.service.Snapshot()
			var b workbench.BatteryConfig
			for _, p := range before.Components {
				if p.Kind == "battery" {
					b = *p.Power.Battery
				}
			}
			uiAction(t, u, "battery-charge-slider", &ui.ChangeEvent{Number: 25})
			agentAction(t, a, "battery.configure", workbench.BatteryParams{ID: "B1", CapacityAh: b.CapacityAh, Resistance: b.Resistance, Charge: .25})
			for _, s := range []*workbench.Service{u.service, a} {
				agentAction(t, s, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, s)
			}
			if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
				t.Fatal("battery user/agent startup differs")
			}
			b.Charge = .25
			for _, entry := range []struct {
				id     string
				number float32
			}{{"battery-esr", .01}, {"battery-capacity", 1}} {
				if entry.id == "battery-esr" {
					b.Resistance = .01
				} else {
					b.CapacityAh = .001
				}
				uiAction(t, u, entry.id, &ui.ChangeEvent{Number: entry.number})
				agentAction(t, a, "battery.configure", workbench.BatteryParams{ID: "B1", CapacityAh: b.CapacityAh, Resistance: b.Resistance, Charge: b.Charge})
				for _, s := range []*workbench.Service{u.service, a} {
					agentAction(t, s, "simulation.restart", struct{}{})
					awaitInductorCircuit(t, s)
				}
				if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
					t.Fatal("battery numeric control differs from agent", entry.id)
				}
			}
			if len(u.service.Snapshot().LiveInputModes) != 0 {
				t.Fatal("battery exposed no-op supply mode controls")
			}
			for _, id := range []string{"battery-state", "battery-voltage", "battery-current", "battery-energy"} {
				if _, ok := findNode(u.view(), id); !ok {
					t.Fatal("missing battery reading", id)
				}
			}
		})
	}
}
