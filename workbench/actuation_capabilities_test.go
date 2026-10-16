package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestActuationContractsArePureOwnedAndKeepOrdinaryCatalogCompact(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	registryBefore, _ := json.Marshal(SchemaRegistry())
	for _, item := range []struct{ command, kind string }{{"switch.set", "switch"}, {"potentiometer.wiper.set", "potentiometer"}} {
		raw, _ := json.Marshal(CapabilityQuery{Command: item.command})
		response := s.Dispatch(t.Context(), Request{Version: 1, ID: "discovery", Command: "capabilities.list", Params: raw}, "agent")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		var schema map[string]any
		if err := json.Unmarshal(response.Result.(map[string]any)["capabilities"].([]Capability)[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract := schema["x-command-contract"].(map[string]any)
		if contract["scope"] != "interactive-part-actuation" || schema["x-interaction-summary"] != PartActuationHint(item.kind) || contract["completion"] == nil {
			t.Fatal("missing actuator semantics", item.command)
		}
		input := inputProperty(schema, "id")
		if input["maxLength"] != float64(64) || !strings.Contains(input["description"].(string), item.kind) {
			t.Fatal("target kind/identity missing")
		}
		// A caller may alter its decoded result without mutating later discovery.
		contract["scope"] = "corrupt"
		fresh := sweepDetailSchema(t, item.command)
		if fresh["x-command-contract"].(map[string]any)["scope"] != "interactive-part-actuation" {
			t.Fatal("discovery aliases client data")
		}
		var ordinary map[string]any
		json.Unmarshal(catalogInputForLiveTest(t, item.command), &ordinary)
		if ordinary["x-command-contract"] != nil {
			t.Fatal("ordinary catalog inflated")
		}
	}
	registryAfter, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(registryBefore, registryAfter) || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("filtered details changed output schemas or editor state")
	}
}

func TestPublishedWiperBoundsAndMalformedActuationPreserveState(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, ""))
	defer s.Close()
	settled(t, s)
	schema := sweepDetailSchema(t, "potentiometer.wiper.set")
	position := inputProperty(schema, "position")
	lo, hi := position["minimum"].(float64), position["maximum"].(float64)
	for _, value := range []float64{lo, .5, hi} {
		call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: value}, fmt.Sprintf("endpoint-%g", value))
		state := settled(t, s)
		want := 5 * (1 + (10000-2)*value) / 10000
		if math.Abs(state.Mixed.Potentiometers[0].WiperVoltage-want) > 1e-10 {
			t.Fatal("published fraction differs from physical divider", value, state.Mixed)
		}
	}
	for _, value := range []float64{math.Nextafter(lo, math.Inf(-1)), math.Nextafter(hi, math.Inf(1))} {
		before := s.Snapshot()
		refuse(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: value}, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("refused wiper changes state")
		}
	}
	call(t, s, "component.place", PlaceParams{Kind: "switch", ID: "S1", Position: Point{X: 300, Y: 200}}, "switch-part")
	for _, closed := range []bool{false, true} {
		call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: closed}, fmt.Sprintf("switch-%t", closed))
		part, _ := s.graph.Component("S1")
		if *part.SwitchClosed != closed {
			t.Fatal("published Boolean differs from saved desired state")
		}
	}
	for _, item := range []struct{ command, body string }{
		{"potentiometer.wiper.set", `{"id":"P1"}`}, {"potentiometer.wiper.set", `{"id":"P1","position":null}`},
		{"potentiometer.wiper.set", `{"id":"P1","position":"50%"}`}, {"potentiometer.wiper.set", `{"id":"V1","position":0.5}`},
		{"switch.set", `{"id":"P1","closed":true}`}, {"switch.set", `{"id":"S1"}`}, {"switch.set", `{"id":"S1","closed":null}`},
		{"switch.set", `{"id":"S1","closed":1}`}, {"switch.set", `{"id":"S1","closed":true,"closed":false}`},
	} {
		before := s.Snapshot()
		revision := before.CommandRevision
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: "invalid", Command: item.command, Params: json.RawMessage(item.body), ExpectedCommandRevision: &revision, Key: "invalid"}, "test")
		if response.Error == nil || response.Error.Code != "invalid-params" || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("malformed/wrong-kind actuator changes state", item, response.Error)
		}
	}
}

func TestSwitchWaitDoesNotApplyPendingAndNextSampleAdvancesOldCircuit(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "queue")
	queued := s.Snapshot()
	waited := settled(t, s)
	if !waited.SwitchStates[0].Pending || !*waited.SwitchStates[0].AppliedClosed || !reflect.DeepEqual(queued.Mixed, waited.Mixed) || queued.Transient.Sample.Time != waited.Transient.Sample.Time {
		t.Fatal("wait applied a queued switch")
	}
	call(t, s, "simulation.step", StepParams{Duration: .001}, "advance")
	applied := s.Snapshot()
	want := 5 * -math.Expm1(-.001/(10001*1e-6))
	if applied.SwitchStates[0].Pending || *applied.SwitchStates[0].AppliedClosed || math.Abs(applied.Mixed.Capacitors[0].Voltage-want) > .002 || applied.Transient.Sample.Time != .001 || applied.RunID != queued.RunID || applied.ElectricalRevision != queued.ElectricalRevision {
		t.Fatal("pending switch did not apply after old-circuit advance", applied.Mixed)
	}
}

func TestCommittedWiperRequestSupersedesItsPendingMovementWithoutPublication(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	entered, release := blockInputSolver(t, s, false)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .9})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .5})
	if s.Snapshot().InputWork.Status != "canceling" {
		t.Fatal("committed position does not supersede in-flight movement")
	}
	release()
	after := awaitInputStatus(t, s, "superseded")
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("superseded wiper partially publishes source/history/physics")
	}
}
