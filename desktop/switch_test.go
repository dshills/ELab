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

func TestNativeSwitchControlPreservesPhysicalStateLikeAgent(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "example.rc", nil)
	agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
	for _, service := range []*workbench.Service{user.service, agent} {
		agentAction(t, service, "component.place", workbench.PlaceParams{ID: "S1", Kind: "switch", Position: workbench.Point{X: -100, Y: -60}})
		agentAction(t, service, "object.delete", workbench.IDParams{ID: "W1"})
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: "V1.positive", To: "S1.1"})
		agentAction(t, service, "wire.connect", workbench.ConnectParams{From: "S1.2", To: "R1.1"})
		agentAction(t, service, "simulation.pause", struct{}{})
	}
	uiAction(t, user, "switch-actuate", nil)
	agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: "S1", Closed: true})
	// Restart from the same closed-switch startup, then freeze both live clocks.
	for _, service := range []*workbench.Service{user.service, agent} {
		agentAction(t, service, "simulation.restart", struct{}{})
		agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .01})
	}
	before := user.service.Snapshot()
	uiAction(t, user, "switch-actuate", nil)
	agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: "S1", Closed: false})
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Transient, a.Transient) || !reflect.DeepEqual(u.Mixed, a.Mixed) {
		t.Fatal("native/agent switch actuation differs")
	}
	if u.RunID != before.RunID || u.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(u.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-10 {
		t.Fatal("native switch reset charge/time")
	}
	if _, ok := findNode(user.view(), "switch-reading"); !ok {
		t.Fatal("switch voltage/current/power readout absent")
	}
	stateNode, ok := findNode(user.view(), "switch-state")
	if !ok || !strings.Contains(stateNode.Text, "pending") || !u.SwitchStates[0].Pending || !*u.SwitchStates[0].AppliedClosed || u.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(u.Mixed, before.Mixed) {
		t.Fatal("native paused switch did not expose unchanged applied physics and pending state")
	}
	for _, service := range []*workbench.Service{user.service, agent} {
		agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .001})
	}
	u, a = user.service.Snapshot(), agent.Snapshot()
	if u.SwitchStates[0].Pending || *u.SwitchStates[0].AppliedClosed || !reflect.DeepEqual(u.Mixed, a.Mixed) || u.Transient.Events[len(u.Transient.Events)-1].Mode != "switch:S1:open" {
		t.Fatal("native/agent next-step switch application differs")
	}
	root := user.view()
	if err := ui.Validate(root); err != nil {
		t.Fatal(err)
	}
}
func TestNativeSwitchFromScratchDrawsAndRunsLED(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, user, "example.reset", nil)
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "W1"})
	uiAction(t, user, "delete-object", nil)
	uiAction(t, user, "part-list", &ui.ChangeEvent{SelectedID: "switch"})
	uiAction(t, user, "position-x", &ui.ChangeEvent{Number: -100})
	uiAction(t, user, "position-y", &ui.ChangeEvent{Number: -60})
	uiAction(t, user, "place-at-position", nil)
	for _, pair := range [][2]string{{"V1.positive", "S1.1"}, {"S1.2", "R1.1"}} {
		uiAction(t, user, "tool-wire", nil)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: pair[0]})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: pair[1]})
		uiAction(t, user, "connect-terminals", nil)
	}
	uiAction(t, user, "switch-actuate", nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := user.service.Wait(ctx)
	if err != nil || state.Mixed == nil || state.Analysis != workbench.NetworkDCAnalysis || state.Mixed.LEDs[0].Current < .0028 {
		t.Fatal("native switched LED did not simulate")
	}
	mesh, err := user.mesh(*user.drawing.View)
	if err != nil || mesh == nil {
		t.Fatal("native switch symbol did not render")
	}
}

func TestNativeUndoButtonRefusesAnInterveningActorTip(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	button, ok := findNode(s.view(), "edit.undo")
	if !ok || button.Disabled {
		t.Fatal("undo missing")
	}
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	s.app.Do(button.Activate)
	if !strings.Contains(s.errorMessage, "history tip changed") || !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
		t.Fatal("stale native button reversed intervening agent work")
	}
	s.errorMessage = ""
	fresh, ok := findNode(s.view(), "edit.undo")
	if !ok || !strings.Contains(fresh.Text, "agent") || !strings.Contains(fresh.Accessibility.Help, "agent") {
		t.Fatal("history actor hidden")
	}
	s.app.Do(fresh.Activate)
	if s.errorMessage != "" || s.service.Snapshot().RedoEntry.Actor != "agent" {
		t.Fatal("explicitly labeled actor undo failed")
	}
}
