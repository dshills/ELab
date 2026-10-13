package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func switchLEDGraph(t *testing.T, closed bool) Graph {
	t.Helper()
	graph := ExampleGraph()
	part, err := MakeComponent("S1", "switch", Point{-100, -60}, "")
	if err != nil {
		t.Fatal(err)
	}
	part.SwitchClosed = &closed
	graph.Components = append(graph.Components, part)
	graph.Wires[0].To = "S1.1"
	graph.Wires = append(graph.Wires, Wire{ID: "WS", From: "S1.2", To: "R1.1", Connected: true})
	return graph
}
func switchRCGraph(t *testing.T, closed bool) Graph {
	t.Helper()
	graph := RCExampleGraph()
	for i := range graph.Components {
		if graph.Components[i].Kind == "pulse-source" {
			part, err := MakeComponent(graph.Components[i].ID, "dc-source", graph.Components[i].Position, "5V")
			if err != nil {
				t.Fatal(err)
			}
			graph.Components[i] = part
		}
	}
	part, err := MakeComponent("S1", "switch", Point{-100, -60}, "")
	if err != nil {
		t.Fatal(err)
	}
	part.SwitchClosed = &closed
	graph.Components = append(graph.Components, part)
	for i := range graph.Wires {
		if graph.Wires[i].From == "V1.positive" {
			graph.Wires[i].From = "S1.2"
		}
	}
	graph.Wires = append(graph.Wires, Wire{ID: "WS", From: "V1.positive", To: "S1.1", Connected: true})
	return graph
}
func TestSwitchLEDOperatingPointLeakageConservationAndProjectRecords(t *testing.T) {
	s := newGraphService(SolveLED, switchLEDGraph(t, true))
	defer s.Close()
	on := settled(t, s)
	if on.Analysis != NetworkDCAnalysis || on.Mixed == nil || len(on.Mixed.Switches) != 1 || on.SwitchModel == nil || on.Mixed.LEDs[0].Current < .0028 || on.Mixed.KCLResidual > 1e-9 || math.Abs(on.Mixed.PowerResidual) > 1e-8 {
		t.Fatalf("on switch result: %+v", on.Mixed)
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
	off := settled(t, s)
	if off.Mixed.LEDs[0].Current <= 0 || off.Mixed.LEDs[0].Current > 5e-9 || math.Abs(off.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("off switch was ideal zero or excessive leakage")
	}
	_, record := verify(t, s, VerifyParams{ProbeIDs: []string{"i:S1", "v:S1"}}, "verify")
	if record.Metadata.SwitchModel == nil || record.Metadata.Mixed == nil || record.Metadata.Engine != NetworkEngineID {
		t.Fatal("Verify omitted switch model/result")
	}
	path := filepath.Join(t.TempDir(), "switch.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.SwitchModel == nil || loaded.Document.RecordingCount != 1 || math.Abs(loaded.Mixed.Switches[0].Current-off.Mixed.Switches[0].Current) > 1e-12 {
		t.Fatal("switch project/record did not round trip")
	}
	data, _ := os.ReadFile(path)
	var document map[string]any
	_ = json.Unmarshal(data, &document)
	document["switch_model"].(map[string]any)["off_resistance_ohm"] = 1
	bad, _ := json.Marshal(document)
	badPath := filepath.Join(t.TempDir(), "bad.elab")
	_ = os.WriteFile(badPath, bad, 0600)
	refuse(t, fresh, "project.load", LoadParams{Path: badPath}, "invalid-document")
}
func TestSwitchRCActuationPreservesCapacitorTimeRunAndHistory(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .02}, "charge")
	before := s.Snapshot()
	voltage := before.Mixed.Capacitors[0].Voltage
	want := 5 * (1 - math.Exp(-.02/((10000+1)*1e-6)))
	// The bounded network solver is adaptive backward Euler, not the exact RC kernel.
	if math.Abs(voltage-want) > .003 {
		t.Fatalf("switched RC reference got %g want %g", voltage, want)
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open")
	opened := s.Snapshot()
	if opened.RunID != before.RunID || opened.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(opened.Mixed.Capacitors[0].Voltage-voltage) > 1e-10 || opened.Transient.Stale || opened.ElectricalRevision != before.ElectricalRevision+1 || opened.Transient.Events[len(opened.Transient.Events)-1].Mode != "switch:S1:open" {
		t.Fatal("live actuation restarted/lost charge/time/event")
	}
	call(t, s, "simulation.step", StepParams{Duration: .02}, "off-step")
	retained := s.Snapshot()
	hold := 5 + (voltage-5)*math.Exp(-.02/((1e9+10000)*1e-6))
	if math.Abs(retained.Mixed.Capacitors[0].Voltage-hold) > 2e-8 {
		t.Fatal("finite off-state capacitor hold was not modeled")
	}
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: true}, "close")
	if state := s.Snapshot(); state.Transient.Sample.Time != .04 || state.RunID != before.RunID || math.Abs(state.Mixed.Capacitors[0].Voltage-retained.Mixed.Capacitors[0].Voltage) > 1e-10 {
		t.Fatal("closing switch reset physical state")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "charge-again")
	if s.Snapshot().Mixed.Capacitors[0].Voltage <= voltage {
		t.Fatal("closed switch did not resume charge")
	}
	_, record := verify(t, s, VerifyParams{Duration: func() *float64 { v := .01; return &v }(), ProbeIDs: []string{"i:S1", "v:C1"}}, "verify")
	if record.Metadata.Analysis != NetworkTransientAnalysis || record.Metadata.SwitchModel == nil {
		t.Fatal("network transient record missing")
	}
}
func TestSwitchValidationNoOpAndSnapshotCopy(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, false))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "same")
	after := s.Snapshot()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("no-op switch actuation altered physics")
	}
	refuse(t, s, "switch.set", SwitchParams{ID: "R1", Closed: true}, "invalid-params")
	snapshot := s.Snapshot()
	for i := range snapshot.Components {
		if snapshot.Components[i].SwitchClosed != nil {
			*snapshot.Components[i].SwitchClosed = true
		}
	}
	for _, part := range s.Snapshot().Components {
		if part.Kind == "switch" && *part.SwitchClosed {
			t.Fatal("snapshot aliases switch state")
		}
	}
}
func TestGeneralNetworkVoltageDividerWithoutSwitch(t *testing.T) {
	graph := EmptyGraph()
	for _, p := range []PlaceParams{{ID: "V1", Kind: "dc-source", Position: Point{-200, 0}, Value: "5V"}, {ID: "R1", Kind: "resistor", Position: Point{0, -80}, Value: "10k"}, {ID: "R2", Kind: "resistor", Position: Point{0, 80}, Value: "10k"}, {ID: "GND", Kind: "ground", Position: Point{200, 150}}} {
		part, err := MakeComponent(p.ID, p.Kind, p.Position, p.Value)
		if err != nil {
			t.Fatal(err)
		}
		graph.Components = append(graph.Components, part)
	}
	for i, pair := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "R2.1"}, {"R2.2", "GND.reference"}, {"V1.negative", "GND.reference"}} {
		graph.Wires = append(graph.Wires, Wire{ID: fmtWire(i), From: pair[0], To: pair[1], Connected: true})
	}
	s := newGraphService(SolveLED, graph)
	defer s.Close()
	state := settled(t, s)
	if state.Analysis != NetworkDCAnalysis || state.Mixed == nil || math.Abs(state.Mixed.Resistors[0].Voltage-2.5) > 1e-9 || math.Abs(state.Mixed.Resistors[0].Current-.00025) > 1e-12 {
		t.Fatalf("divider unsupported/wrong: %+v", state.Diagnostics)
	}
}
func fmtWire(i int) string { return string(rune('A' + i)) }

func TestRunningSwitchActuationContinuesClock(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	before := s.Snapshot()
	call(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false}, "open-running")
	after := s.Snapshot()
	if after.Paused || after.RunID != before.RunID {
		t.Fatal("actuation stopped/restarted the clock")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.Transient.Sample.Time > after.Transient.Sample.Time {
			if state.RunID != before.RunID || state.Transient.Stale || state.Mixed.ElectricalRevision != state.ElectricalRevision {
				t.Fatal("continued clock published stale state")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("live clock did not continue")
}
