package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestPassiveControlsUseSharedCommandsAndKeepIdealDefaults(t *testing.T) {
	for _, kind := range []string{"capacitor", "inductor"} {
		t.Run(kind, func(t *testing.T) {
			user, done := testScreen(t)
			defer done()
			agent := workbench.New()
			defer agent.Close()
			name, id := "rc", "C1"
			if kind == "inductor" {
				name, id = "rlc", "L1"
			}
			for _, s := range []*workbench.Service{user.service, agent} {
				agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: name})
				agentAction(t, s, "simulation.pause", struct{}{})
				agentAction(t, s, "component.select", workbench.IDParams{ID: id})
				agentAction(t, s, "simulation.restart", struct{}{})
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, err := s.Wait(ctx)
				cancel()
				if err != nil || len(s.Snapshot().Diagnostics) > 0 {
					t.Fatal("initial ideal fixture", err, s.Snapshot().Diagnostics)
				}
			}
			cfg := workbench.PassiveConfig{Version: workbench.PassiveLossModelID}
			if kind == "capacitor" {
				cfg.ESR, cfg.Leakage = 1, 1e6
			} else {
				cfg.Winding = 1
			}
			uiAction(t, user, "passive-enabled", &ui.ChangeEvent{Checked: true})
			agentAction(t, agent, "passive.configure", workbench.PassiveParams{ID: id, Enabled: true, Parameters: cfg})
			field := "passive-esr"
			if kind == "inductor" {
				field = "passive-winding"
			}
			uiAction(t, user, field, &ui.ChangeEvent{Number: 20})
			if kind == "capacitor" {
				cfg.ESR = 20
			} else {
				cfg.Winding = 20
			}
			agentAction(t, agent, "passive.configure", workbench.PassiveParams{ID: id, Enabled: true, Parameters: cfg})
			if !reflect.DeepEqual(user.service.Snapshot().Components, agent.Snapshot().Components) {
				t.Fatal("passive user/agent divergence")
			}
			for _, s := range []*workbench.Service{user.service, agent} {
				agentAction(t, s, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, s)
			}
			uiAction(t, user, "transient-step", nil)
			agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: agent.Snapshot().TransientSettings.StepSeconds})
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
				t.Fatal("lossy step differs")
			}
			held, ok := findNode(user.view(), field)
			if !ok || held.Change == nil {
				t.Fatal("missing passive field", field)
			}
			agentAction(t, user.service, "passive.configure", workbench.PassiveParams{ID: id, Parameters: cfg})
			before := user.service.Snapshot().Components
			user.app.Do(func() { held.Change(ui.ChangeEvent{Number: 40}) })
			if !reflect.DeepEqual(before, user.service.Snapshot().Components) {
				t.Fatal("stale passive field overwrote agent disable")
			}
			for _, theme := range []string{"light", "dark", "system", "apollo"} {
				agentAction(t, user.service, "appearance.theme.set", map[string]string{"theme": theme})
				if err := ui.Validate(user.view()); err != nil {
					t.Fatal(theme, err)
				}
			}
		})
	}
}

func TestPassiveNumericSIEndpointsRemainExact(t *testing.T) {
	s, done := testScreen(t)
	defer done()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "capacitor-loss"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	awaitInductorCircuit(t, s.service)
	uiAction(t, s, "passive-esr", &ui.ChangeEvent{Number: .001})
	state := s.service.Snapshot()
	var cfg *workbench.PassiveConfig
	for _, p := range state.Components {
		if p.ID == "C1" {
			cfg = p.Device.Passive
		}
	}
	if cfg == nil || cfg.ESR != .001 {
		t.Fatal("ESR minimum rounded across domain", cfg)
	}
	uiAction(t, s, "passive-leak", &ui.ChangeEvent{Number: .001})
	for _, p := range s.service.Snapshot().Components {
		if p.ID == "C1" && (p.Device == nil || p.Device.Passive == nil || p.Device.Passive.Leakage != 1000) {
			t.Fatal("MΩ minimum conversion", p.Device.Passive)
		}
	}
	uiAction(t, s, "passive-leak", &ui.ChangeEvent{Number: 1e6})
	for _, p := range s.service.Snapshot().Components {
		if p.ID == "C1" && (p.Device == nil || p.Device.Passive == nil || p.Device.Passive.Leakage != 1e12) {
			t.Fatal("MΩ maximum conversion", p.Device.Passive)
		}
	}
}
