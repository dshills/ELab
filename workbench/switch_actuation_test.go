package workbench

import (
	"context"
	"math"
	"reflect"
	"testing"
)

func TestQueuedSwitchCoalescingCancellationRestartAndVerifyProvenance(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration := .001
	_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	initial := s.Snapshot()
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
	if inspectOperation(t, s, record.Operation.ID).Operation.SourceUnchanged {
		t.Fatal("desired input change falsely kept Verify source association")
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "cancel")
	state := s.Snapshot()
	if state.SwitchStates[0].Pending || !reflect.DeepEqual(state.Mixed, initial.Mixed) || len(state.Transient.Events) != len(initial.Transient.Events) || state.ElectricalRevision != initial.ElectricalRevision {
		t.Fatal("canceled pending switch altered applied physics/history")
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open-again")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	if state = s.Snapshot(); state.SwitchStates[0].Pending || *state.SwitchStates[0].AppliedClosed || state.Transient.Sample.Time != 0 {
		t.Fatal("restart did not consume desired startup state")
	}
}
func TestPendingSwitchSurvivesWiperActuationAndBatchUsesStartupSemantics(t *testing.T) {
	g := potGraph(t, "capacitor")
	sw, err := MakeComponent("S1", "switch", Point{-100, -200}, "")
	if err != nil {
		t.Fatal(err)
	}
	closed := true
	sw.SwitchClosed = &closed
	g.Components = append(g.Components, sw)
	g.Wires[0].To = "S1.1"
	g.Wires = append(g.Wires, Wire{ID: "SW", From: "S1.2", To: "P1.1", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "queue")
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .75}, "wiper")
	state := s.Snapshot()
	if !state.SwitchStates[0].Pending || !*state.SwitchStates[0].AppliedClosed || state.Transient.Sample.Time != 0 || state.ElectricalRevision != 1 {
		t.Fatal("wiper prematurely applied switch or changed circuit revision")
	}
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	if state = s.Snapshot(); state.SwitchStates[0].Pending || *state.SwitchStates[0].AppliedClosed || math.Abs(state.Mixed.Potentiometers[0].WiperPosition-.75) > 1e-12 {
		t.Fatal("combined live actuations failed")
	}
	call(t, s, "edit.batch", BatchParams{Name: "Switch startup", Edits: []BatchEdit{edit("switch.set", SwitchParams{ID: "S1", Closed: true})}}, "batch")
	if state = s.Snapshot(); !state.Mixed.Stale || state.ElectricalRevision == 1 {
		t.Fatal("batch switch edit bypassed schematic invalidation")
	}
}

func TestQueuedSwitchEventFailureDoesNotPublishPhysicalAdvance(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "queue")
	s.mu.Lock()
	defer s.mu.Unlock()
	before := cloneMixed(s.mixedState.result)
	x := append([]float64{}, s.mixedState.x...)
	caps := append([]float64{}, s.mixedState.capV...)
	trace := cloneTransient(s.transient)
	invalid := s.graph.Clone()
	invalid.Wires = nil
	next, staged, err := advanceLiveNetwork(context.Background(), s.mixedState, s.transient, .001, invalid)
	if err == nil || next != nil || staged != nil || !reflect.DeepEqual(cloneMixed(s.mixedState.result), before) || !reflect.DeepEqual(s.transient, trace) || !reflect.DeepEqual(s.mixedState.x, x) || !reflect.DeepEqual(s.mixedState.capV, caps) {
		t.Fatalf("pending failure err=%v next=%v trace=%v result_equal=%v trace_equal=%v", err, next != nil, staged != nil, reflect.DeepEqual(cloneMixed(s.mixedState.result), before), reflect.DeepEqual(s.transient, trace))
	}
}
