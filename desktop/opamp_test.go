package main

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func opAmpDrawingFixture(t *testing.T) workbench.Graph {
	t.Helper()
	g := workbench.EmptyGraph()
	for _, spec := range []struct {
		id, kind, value string
		point           workbench.Point
	}{
		{"VIN", "pulse-source", "1 V", workbench.Point{X: -300}},
		{"VP", "dc-source", "5 V", workbench.Point{X: -120, Y: -180}},
		{"VN", "dc-source", "5 V", workbench.Point{X: -120, Y: 180}},
		{"RL", "resistor", "1k", workbench.Point{X: 220, Y: 80}},
		{"U1", "opamp", "", workbench.Point{}},
		{"GND", "ground", "", workbench.Point{X: -300, Y: 180}},
	} {
		part, err := workbench.MakeComponent(spec.id, spec.kind, spec.point, spec.value)
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, part)
	}
	for _, ends := range [][2]string{
		{"VIN.positive", "U1.noninverting"}, {"VIN.negative", "GND.reference"},
		{"VP.positive", "U1.vplus"}, {"VP.negative", "GND.reference"},
		{"VN.positive", "GND.reference"}, {"VN.negative", "U1.vminus"},
		{"U1.out", "U1.inverting"}, {"U1.out", "RL.1"}, {"RL.2", "GND.reference"},
	} {
		g.Wires = append(g.Wires, workbench.Wire{ID: "W" + string(rune('a'+len(g.Wires))), From: ends[0], To: ends[1], Connected: true})
	}
	return g
}

func TestOpAmpBlankNativeConstructionInspectorAndAgentParity(t *testing.T) {
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, opAmpDrawingFixture(t))
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["U1"]})
	left, right := user.service.Snapshot(), agent.Snapshot()
	if left.OpAmpModel == nil || !reflect.DeepEqual(left.Mixed.OpAmps, right.Mixed.OpAmps) || math.Abs(left.Mixed.OpAmps[0].OutputVoltage-1) > .0001 {
		t.Fatal("blank amplifier UI/agent physics differs", left.Diagnostics, right.Diagnostics)
	}
	if reading, ok := findNode(user.view(), "opamp-selected-reading"); !ok || !strings.Contains(reading.Text, "delivered") || !strings.Contains(reading.Text, "Into V+") {
		t.Fatal("selected amplifier lost signed supply/output readings")
	}
	for _, probe := range []string{"v:" + ids["U1"], "v:" + ids["U1"] + ".input-difference", "v:" + ids["U1"] + ".supply", "i:" + ids["U1"]} {
		uiAction(t, user, "plot-"+probe, nil)
		agentAction(t, agent, "scope.probes.set", workbench.ProbeParams{IDs: withProbe(agent.Snapshot().Scope.Selected, probe)})
	}
	if !reflect.DeepEqual(user.service.Snapshot().Scope.Selected, agent.Snapshot().Scope.Selected) {
		t.Fatal("op-amp plots bypass shared probe commands")
	}
	for range 50 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right = user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed.OpAmps, right.Mixed.OpAmps) || math.Abs(left.Mixed.OpAmps[0].OutputVoltage) > .0001 {
		t.Fatal("pulse edge amplifier parity", left.Mixed, right.Mixed)
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["VIN"]})
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: ".5 V"})
	uiAction(t, user, "apply-value", nil)
	agentAction(t, agent, "component.set", workbench.ValueParams{ID: ids["VIN"], Value: ".5 V"})
	if !user.service.Snapshot().Mixed.Stale {
		t.Fatal("paused edit did not retain stale result")
	}
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := user.service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	left, right = user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed.OpAmps, right.Mixed.OpAmps) || math.Abs(left.Mixed.OpAmps[0].OutputVoltage-.5) > .0001 {
		t.Fatal("edited/restarted amplifier parity", left.Diagnostics, right.Diagnostics)
	}
	if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil || len(mesh.Vertices()) == 0 {
		t.Fatal("amplifier mesh failed", err)
	}
}

func TestOpAmpGuidedMenuEditsPlotsAndProtectedReload(t *testing.T) {
	for _, example := range []string{"opamp-follower", "opamp-inverting"} {
		t.Run(example, func(t *testing.T) {
			user, cleanup := testScreen(t)
			defer cleanup()
			agent := workbench.New()
			defer agent.Close()
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
			agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: example})
			for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
				uiAction(t, user, cmd, nil)
				agentAction(t, agent, cmd, struct{}{})
			}
			left, right := user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(left.Components, right.Components) || !reflect.DeepEqual(left.Mixed.OpAmps, right.Mixed.OpAmps) || left.Selected != "U1" {
				t.Fatal("guided menu differs from public example.load")
			}
			nodes := user.componentInspector(left, workbench.Graph{Components: left.Components, Wires: left.Wires}, left.Components[2])
			positions := map[string]int{}
			for i, n := range nodes {
				positions[n.ID] = i
			}
			if positions["opamp-selected-reading"] >= positions["opamp-plot-row-0"] || positions["opamp-plot-row-1"] >= positions["opamp-model"] || positions["rotate-part"] >= positions["part-assumptions"] {
				t.Fatal("live measurements and actions must precede model explanation", positions)
			}
			for _, probe := range []struct{ id, name string }{
				{"v:U1", "Plot output voltage"}, {"i:U1", "Plot delivered output current"},
				{"v:U1.input-difference", "Plot input difference"}, {"v:U1.supply", "Plot supply voltage"},
			} {
				n, ok := findNode(user.view(), "plot-"+probe.id)
				if !ok || n.Accessibility.Name != probe.name {
					t.Fatal("compact plot action lost full accessible name", probe, n)
				}
				uiAction(t, user, n.ID, nil)
				agentAction(t, agent, "scope.probes.set", workbench.ProbeParams{IDs: withProbe(agent.Snapshot().Scope.Selected, probe.id)})
			}
			if !reflect.DeepEqual(user.service.Snapshot().Scope.Selected, agent.Snapshot().Scope.Selected) {
				t.Fatal("guided plot actions differ")
			}
			part, value := "RL", "10 Ω"
			if example == "opamp-inverting" {
				part, value = "RF", "10k"
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: part})
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: value})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: part, Value: value})
			uiAction(t, user, "simulation.restart", nil)
			agentAction(t, agent, "simulation.restart", struct{}{})
			for range 25 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
			}
			left, right = user.service.Snapshot(), agent.Snapshot()
			want := -.5
			if example == "opamp-follower" {
				want = .2
			}
			if !reflect.DeepEqual(left.Mixed.OpAmps, right.Mixed.OpAmps) || math.Abs(left.Mixed.OpAmps[0].OutputVoltage-want) > .01 {
				t.Fatal("guided electrical edit/step differs", left.Diagnostics, right.Diagnostics)
			}
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
			if user.pendingCommand != "example.load" {
				t.Fatal("guided reload bypasses unsaved changes")
			}
			uiAction(t, user, "discard-continue", nil)
			if user.service.Snapshot().Example.Modified {
				t.Fatal("protected reload failed to restore template")
			}
		})
	}
}
