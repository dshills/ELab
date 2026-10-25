package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestExtendedNativeExamplesConfigurationAndAgentParity(t *testing.T) {
	for _, id := range []string{"buck", "opamp-slew", "opamp-bandwidth", "shift-register", "555-tone", "light-alarm"} {
		t.Run(id, func(t *testing.T) {
			u, cleanup := testScreen(t)
			defer cleanup()
			a := workbench.New()
			defer a.Close()
			uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: id})
			for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
				uiAction(t, u, cmd, nil)
				agentAction(t, a, cmd, struct{}{})
			}
			left, right := awaitInductorCircuit(t, u.service), awaitInductorCircuit(t, a)
			if !reflect.DeepEqual(left.Mixed, right.Mixed) || left.DeviceModel == nil {
				t.Fatal("example parity", left.Diagnostics, right.Diagnostics)
			}
			switch id {
			case "buck":
				uiAction(t, u, "pwm-duty", &ui.ChangeEvent{Number: 60})
				agentAction(t, a, "pwm.configure", workbench.PWMParams{ID: "PWM", Parameters: workbench.PWMConfig{Frequency: 500, Duty: .6}})
			case "opamp-slew", "opamp-bandwidth":
				uiAction(t, u, "opamp-gbw", &ui.ChangeEvent{Number: 2000})
				agentAction(t, a, "opamp.dynamics.set", workbench.DynamicsParams{ID: "U1", Enabled: true, Parameters: workbench.OpAmpDynamics{GBW: 2000, Slew: 1000}})
			case "555-tone":
				uiAction(t, u, "sound-capacitance", &ui.ChangeEvent{Number: 30})
				agentAction(t, a, "sound.configure", workbench.SoundParams{ID: "BZ1", Parameters: workbench.SoundConfig{Capacitance: 30e-9, Frequency: 1000}})
			case "light-alarm":
				uiAction(t, u, "sound-frequency", &ui.ChangeEvent{Number: 2000})
				agentAction(t, a, "sound.configure", workbench.SoundParams{ID: "BZ1", Parameters: workbench.SoundConfig{Capacitance: 20e-9, Frequency: 2000}})
			case "shift-register":
				if _, ok := findNode(u.view(), "shift-help"); !ok {
					t.Fatal("missing register help")
				}
			}
			if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
				t.Fatal("configuration parity")
			}
			for _, theme := range []string{"light", "dark", "apollo"} {
				agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
				if err := ui.Validate(u.view()); err != nil {
					t.Fatal(theme, err)
				}
			}
		})
	}
}
func TestExtendedInspectorRejectsStaleConfiguration(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "buck"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	awaitInductorCircuit(t, s.service)
	node, _ := findNode(s.view(), "pwm-duty")
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "buck"})
	node.Change(ui.ChangeEvent{Number: 75})
	for _, p := range s.service.Snapshot().Components {
		if p.ID == "PWM" && p.Device.PWM.Duty != .45 {
			t.Fatal("old field changed replacement")
		}
	}
}
