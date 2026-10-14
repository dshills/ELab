package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func inspectMeasurement(t *testing.T, s *Service, p MeasurementQueryParams) MeasurementInspection {
	t.Helper()
	raw, _ := json.Marshal(p)
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "inspect", Command: "measurement.inspect", Params: raw}, "reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(MeasurementInspection)
}
func inspectionReading(t *testing.T, in MeasurementInspection, id string) MeasurementReading {
	t.Helper()
	for _, reading := range in.Readings {
		if reading.Probe.ID == id {
			return reading
		}
	}
	t.Fatalf("missing inspection reading %s", id)
	return MeasurementReading{}
}

func TestMeasurementSelectedDCObjectAndReadOnlyProbeQuery(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.select", IDParams{ID: "R1"}, "select")
	before := s.Snapshot()
	in := inspectMeasurement(t, s, MeasurementQueryParams{})
	if !in.Available || in.Target.Kind != "object" || in.Target.ID != "R1" || in.ResultKind != "dc" || len(in.Readings) != 4 || len(in.Objects) != 1 || len(in.Wires) != 0 {
		t.Fatal("selected component inspection is incomplete")
	}
	v := inspectionReading(t, in, "v:R1")
	i := inspectionReading(t, in, "i:R1")
	if v.State != "live" || i.State != "live" || v.Value == nil || i.Value == nil || math.Abs(*v.Value-1000*(*i.Value)) > 1e-12 || v.ReferenceDisplay != "Voltage: R1.1 - R1.2" || i.ReferenceDisplay != "Current: R1.1 -> R1.2" {
		t.Fatal("DC voltage/current sign convention or reference incorrect")
	}
	ground := inspectMeasurement(t, s, MeasurementQueryParams{Target: &MeasurementSelectParams{Kind: "terminal", ID: "GND.reference"}})
	if len(ground.Readings) != 1 || ground.Readings[0].Value == nil || *ground.Readings[0].Value != 0 {
		t.Fatal("accepted reference ground is not distinguished from missing data")
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("read-only inspection changes editor/physics/actor state")
	}
	*in.Readings[0].Value = 999
	in.Terminals[0] = "changed"
	if got := inspectMeasurement(t, s, MeasurementQueryParams{}); *got.Readings[0].Value == 999 || got.Terminals[0] == "changed" {
		t.Fatal("inspection exposes shared frames or associations")
	}
}

func TestMeasurementTerminalNetAndWireSelectionPreservePhysics(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step")
	before := s.Snapshot()
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "C1.1"}, "terminal")
	after := s.Snapshot()
	in := after.MeasurementInspection
	if !in.Available || in.ResultKind != "transient" || in.Target.Anchor != "C1.1" || len(in.Wires) != 1 || in.Wires[0] != "W2" || len(in.Terminals) != 2 || in.Readings[0].Value == nil || math.Abs(*in.Readings[0].Value-5*(1-math.Exp(-1))) > 1e-12 {
		t.Fatal("terminal does not expose its accepted voltage and exact net")
	}
	if after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) || after.Document != before.Document || after.Selected != before.Selected || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
		t.Fatal("presentation selection changes physics/document/history")
	}
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "net", ID: in.Nets[0].ID}, "net")
	net := s.Snapshot().MeasurementInspection
	if net.Target.Anchor != "C1.1" || !reflect.DeepEqual(net.Terminals, in.Terminals) || !reflect.DeepEqual(net.Wires, in.Wires) || *net.Readings[0].Value != *in.Readings[0].Value {
		t.Fatal("net identity differs from selected terminal")
	}
	call(t, s, "component.select", IDParams{ID: "W1"}, "wire")
	wire := s.Snapshot().MeasurementInspection
	if wire.Target.Kind != "object" || wire.Target.ID != "W1" || len(wire.Readings) != 1 || wire.Readings[0].Probe.Unit != "V" || *wire.Readings[0].Value != 5 {
		t.Fatal("ideal wire inspection invents current or loses its voltage")
	}
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "probe", ID: "i:C1"}, "probe")
	call(t, s, "measurement.select", MeasurementSelectParams{}, "clear")
	if s.Snapshot().MeasurementInspection.Target.ID != "W1" {
		t.Fatal("clearing measurement focus does not return to editing selection")
	}
}

func TestMeasurementNetAnchorSurvivesRenamingAndRemoval(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "net", ID: "net:C1.1"}, "net")
	call(t, s, "component.place", PlaceParams{ID: "A1", Kind: "resistor", Position: Point{X: -500, Y: -400}}, "place")
	call(t, s, "wire.connect", ConnectParams{From: "A1.1", To: "C1.1"}, "join")
	in := s.Snapshot().MeasurementInspection
	if !in.Available || in.Target.ID != "net:A1.1" || in.Target.Anchor != "C1.1" || in.Readings[0].Probe.ID != "v:C1.1" || in.Readings[0].State != "stale" {
		t.Fatal("renamed net selection jumps to a new source or relabels old result live")
	}
	call(t, s, "object.delete", IDParams{ID: "W2"}, "split")
	for _, ref := range s.Snapshot().MeasurementInspection.Terminals {
		if ref == "R1.2" {
			t.Fatal("split net retains disconnected members")
		}
	}
	call(t, s, "object.delete", IDParams{ID: "C1"}, "delete-anchor")
	missing := s.Snapshot().MeasurementInspection
	if missing.Available || len(missing.Readings) != 0 || !strings.Contains(missing.Reason, "Source removed") {
		t.Fatal("missing net anchor selects an unrelated remaining member")
	}
	call(t, s, "edit.undo", HistoryParams{}, "restore-anchor")
	if !s.Snapshot().MeasurementInspection.Available {
		t.Fatal("undo does not restore anchored selection")
	}
}

func TestMeasurementStrictInputsCacheOwnershipAndRecordedSeparation(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	refuse(t, s, "measurement.select", MeasurementSelectParams{Kind: "current", ID: "Q1"}, "invalid-params")
	refuse(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "Q1.missing"}, "not-found")
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "probe", ID: "i:Q1.emitter"}, "select")
	before := s.Snapshot()
	r := inspectionReading(t, before.MeasurementInspection, "i:Q1.emitter")
	if r.ReferenceDisplay != "Current: Q1.emitter -> out of device" || r.Value == nil || *r.Value < 0 {
		t.Fatal("NPN emitter sign/reference lost")
	}
	first := call(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "Q1.collector"}, "cached")
	copy := first.Result.(Snapshot)
	copy.MeasurementInspection.Target.ID = "mutated"
	*copy.MeasurementInspection.Readings[0].Value = 999
	copy.MeasurementInspection.Nets[0].Terminals[0] = "mutated"
	second := call(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "Q1.collector"}, "cached").Result.(Snapshot)
	if second.MeasurementInspection.Target.ID == "mutated" || *second.MeasurementInspection.Readings[0].Value == 999 || second.MeasurementInspection.Nets[0].Terminals[0] == "mutated" {
		t.Fatal("retry cache aliases inspection")
	}
	duration := .01
	_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "operation.select", IDParams{ID: record.Operation.ID}, "record")
	live := s.Snapshot().MeasurementInspection
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "recorded")
	if !reflect.DeepEqual(live, s.Snapshot().MeasurementInspection) {
		t.Fatal("recorded plot masquerades as live pin/net inspection")
	}
	for _, raw := range []string{`{"target":{"kind":"terminal"}}`, `{"target":{"kind":"net","id":"net:missing"}}`, `{"target":{"kind":"probe","id":"i:D1","anchor":"Q2.collector"}}`} {
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad", Command: "measurement.inspect", Params: json.RawMessage(raw)}, "reader")
		if response.Error == nil {
			t.Fatal("unvalidated measurement query accepted", raw)
		}
	}
}

func TestMeasurementDisconnectedWireDoesNotInventNetVoltage(t *testing.T) {
	g := ExampleGraph()
	g.Wires[0].Connected = false
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "component.select", IDParams{ID: "W1"}, "wire")
	in := s.Snapshot().MeasurementInspection
	if in.Available || len(in.Readings) != 0 || !strings.Contains(in.Reason, "not electrically connected") {
		t.Fatal("disconnected drawing wire masquerades as a physical net")
	}
}
