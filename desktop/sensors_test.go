package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestSensorNativeEnvironmentControlsAndAgentParity(t *testing.T) {
	for _, example := range []string{"night-light", "thermostat"} {
		t.Run(example, func(t *testing.T) {
			u, cleanup := testScreen(t)
			defer cleanup()
			a := workbench.New()
			defer a.Close()
			uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
			agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: example})
			for _, command := range []string{"simulation.pause", "simulation.restart"} {
				uiAction(t, u, command, nil)
				agentAction(t, a, command, struct{}{})
			}
			awaitInductorCircuit(t, u.service)
			awaitInductorCircuit(t, a)
			command := "sensor.light.set"
			param := any(workbench.LightParams{ID: "LDR1", Illuminance: 1})
			value := float32(1)
			if example == "thermostat" {
				command = "sensor.temperature.set"
				param = workbench.TemperatureParams{ID: "RT1", Temperature: 30}
				value = 30
			}
			uiAction(t, u, "sensor-exact", &ui.ChangeEvent{Number: value})
			agentAction(t, a, command, param)
			for _, s := range []*workbench.Service{u.service, a} {
				if _, err := s.Wait(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
				t.Fatal("sensor environment user/agent results differ")
			}
			if _, ok := findNode(u.view(), "sensor-reading"); !ok {
				t.Fatal("missing sensor resistance reading")
			}
			// The slider goes through the same command, including exact bounds at light endpoints.
			slider := float32(0)
			if example == "thermostat" {
				slider = 20
				param = workbench.TemperatureParams{ID: "RT1", Temperature: 20}
			} else {
				param = workbench.LightParams{ID: "LDR1", Illuminance: 0}
			}
			uiAction(t, u, "sensor-slider", &ui.ChangeEvent{Number: slider})
			agentAction(t, a, command, param)
			for _, s := range []*workbench.Service{u.service, a} {
				s.Wait(t.Context())
			}
			if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
				t.Fatal("sensor slider parity")
			}
			if example == "night-light" {
				uiAction(t, u, "sensor-slider", &ui.ChangeEvent{Number: 100})
				if _, err := u.service.Wait(t.Context()); err != nil {
					t.Fatal(err)
				}
				if u.service.Snapshot().Mixed.Sensors[0].Environment != 10000 {
					t.Fatal("light slider maximum")
				}
			}
		})
	}
}
func TestSensorLightSliderScaleAndUnchangedDisplayPrecision(t *testing.T) {
	for _, lux := range []float64{0, .01, 1, 10, 100, 1000, 10000} {
		if got := lightAtPosition(lightPosition(lux)); math.Abs(got-lux) > 1e-8*math.Max(1, lux) {
			t.Fatal("light slider inverse", lux, got)
		}
	}
	u, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.night-light"})
	uiAction(t, u, "simulation.pause", nil)
	awaitInductorCircuit(t, u.service)
	agentAction(t, u.service, "sensor.light.set", workbench.LightParams{ID: "LDR1", Illuminance: 1.234567890123})
	u.service.Wait(t.Context())
	before := u.service.Snapshot()
	uiAction(t, u, "sensor-exact", &ui.ChangeEvent{Number: float32(1.234567890123)})
	after := u.service.Snapshot()
	if after.CommandRevision != before.CommandRevision || !reflect.DeepEqual(before.Mixed, after.Mixed) {
		t.Fatal("unchanged displayed environment quantized precise value")
	}
}

func TestSensorRetainedFieldRefusesInterveningAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{Kind: "thermistor", ID: "RT1"})
	var old ui.Node
	s.app.Do(func() { old, _ = findNode(s.view(), "sensor-slider") })
	agentAction(t, s.service, "sensor.temperature.set", workbench.TemperatureParams{ID: "RT1", Temperature: 40})
	before := s.service.Snapshot()
	s.app.Do(func() { old.Change(ui.ChangeEvent{Number: 10}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil {
		t.Fatal("stale temperature field overwrote agent input")
	}
}
