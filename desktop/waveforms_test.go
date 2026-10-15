package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeWaveformControlsAndAgentParity(t *testing.T) {
	for _, kind := range []string{"sine-source", "finite-pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			user, cleanup := testScreen(t)
			defer cleanup()
			if err := smokeConstructWaveform(user, kind); err != nil {
				t.Fatal(err)
			}
			agent := workbench.New()
			defer agent.Close()
			agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
			agentAction(t, agent, "object.delete", workbench.IDParams{ID: "V1"})
			agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: kind, Position: workbench.Point{X: -200, Y: 0}})
			id := agent.Snapshot().Selected
			for _, pair := range [][2]string{{id + ".positive", "R1.1"}, {id + ".negative", "GND.reference"}} {
				agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]})
			}
			uiAction(t, user, "simulation.pause", nil)
			agentAction(t, agent, "simulation.pause", struct{}{})
			if kind == "sine-source" {
				for _, field := range []struct {
					id    string
					value string
				}{{"sine-amplitude", "2 V"}, {"sine-offset", "1 V"}, {"sine-phase", "30 deg"}, {"sine-frequency", "50 Hz"}} {
					uiAction(t, user, field.id, &ui.ChangeEvent{Text: field.value})
				}
				cfg := workbench.SineWaveform{Amplitude: 2, Offset: 1, Phase: 30, Frequency: 50}
				agentAction(t, agent, "source.sine.set", workbench.SineParams{ID: id, Waveform: cfg})
			} else {
				uiAction(t, user, "pulse-rise", &ui.ChangeEvent{Text: "2 ms"})
				cfg := workbench.DefaultPulseWaveform()
				cfg.RiseSeconds = .002
				agentAction(t, agent, "source.pulse.set", workbench.PulseParams{ID: id, Waveform: cfg})
			}
			uiAction(t, user, "fit-waveform-sampling", nil)
			agentAction(t, agent, "simulation.sampling.fit", struct{}{})
			uiAction(t, user, "simulation.restart", nil)
			agentAction(t, agent, "simulation.restart", struct{}{})
			uiAction(t, user, "transient-step", nil)
			agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: agent.Snapshot().TransientSettings.StepSeconds})
			u, a := user.service.Snapshot(), agent.Snapshot()
			// User field edits create separate history entries; the final numerical
			// state and circuit remain identical despite different run counters.
			if !reflect.DeepEqual(u.Components, a.Components) || u.TransientSettings != a.TransientSettings || u.WaveformModel == nil || !reflect.DeepEqual(u.WaveformModel, a.WaveformModel) || math.Abs(u.Transient.Sample.Input-a.Transient.Sample.Input) > 1e-10 || math.Abs(u.Mixed.Capacitors[0].Voltage-a.Mixed.Capacitors[0].Voltage) > 1e-10 {
				t.Fatal("native waveform controls/agent result differ")
			}
			if err := ui.Validate(user.view()); err != nil {
				t.Fatal(err)
			}
			if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
				t.Fatal("waveform symbol absent")
			}
		})
	}
}
