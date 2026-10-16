package workbench

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func publicOpAmpCircuit(t *testing.T) (*Service, Graph) {
	t.Helper()
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	g := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000).wiredGraph(t)
	for k, part := range g.Components {
		value := ""
		if part.Unit != "" {
			value = fmt.Sprint(part.Value)
		}
		id := part.ID
		if part.Kind == "opamp" {
			id = "" // Check the shared automatic U1 identity.
		}
		call(t, s, "component.place", PlaceParams{ID: id, Kind: part.Kind, Value: value, Position: Point{X: float64(k) * 150}}, "part-"+part.ID)
	}
	for _, wire := range g.Wires {
		call(t, s, "wire.connect", ConnectParams{ID: wire.ID, From: wire.From, To: wire.To}, wire.ID)
	}
	call(t, s, "simulation.restart", struct{}{}, "restart")
	state := settled(t, s)
	if state.Mixed == nil || len(state.Mixed.OpAmps) != 1 || state.OpAmpModel == nil || state.Limits.OpAmps != 2 || math.Abs(state.Mixed.OpAmps[0].OutputVoltage-1) > .0001 {
		t.Fatal("public placement/wiring failed", state.Diagnostics)
	}
	return s, Graph{Components: state.Components, Wires: state.Wires}
}

func TestOpAmpPublicConstructionProbeSignsAndGeometry(t *testing.T) {
	s, g := publicOpAmpCircuit(t)
	call(t, s, "parts.search.set", SearchParams{Query: "op-amp"}, "search")
	if parts := s.Snapshot().MatchingParts; len(parts) != 1 || parts[0].Kind != "opamp" || parts[0].ValueType != "none" {
		t.Fatal("op-amp discovery", parts)
	}
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	r := s.Snapshot().Mixed
	u := r.OpAmps[0]
	sample := mixedSample(c, r, 1)
	values := map[string]float64{
		"v:U1": u.OutputVoltage, "v:U1.out": u.OutputVoltage,
		"v:U1.input-difference": u.InputDifference, "v:U1.supply": u.SupplyVoltage,
		"i:U1": u.OutputCurrent, "i:U1.out": u.OutputCurrent,
		"i:U1.noninverting": u.NoninvertingCurrent, "i:U1.inverting": u.InvertingCurrent,
		"i:U1.vplus": u.PositiveSupplyCurrent, "i:U1.vminus": u.NegativeSupplyCurrent,
	}
	for _, probe := range probeInventory(g) {
		want, requested := values[probe.ID]
		if !requested {
			continue
		}
		got, ok := frameValue(sample, probe)
		if !ok || math.Abs(got-want) > 1e-12 {
			t.Fatal("probe lost its value or sign", probe, got, want)
		}
		delete(values, probe.ID)
	}
	if len(values) > 0 {
		t.Fatal("missing op-amp probes", values)
	}
	wireCurrents := uniqueWireCurrents(g, readingContext{sample: sample})
	for _, w := range g.Wires {
		if w.From == "U1.vplus" || w.To == "U1.vplus" {
			want := u.PositiveSupplyCurrent
			if w.From == "U1.vplus" {
				want = -want
			}
			if got, ok := wireCurrents[w.ID]; !ok || math.Abs(got-want) > 1e-12 {
				t.Fatal("supply-wire current lost terminal injection", w, got, ok, want)
			}
		}
	}
	part, _ := g.Component("U1")
	for _, rotation := range []int{0, 90, 180, 270} {
		part.Rotation = rotation
		seen := map[Point]bool{}
		for _, pin := range part.Terminals {
			point, ok := TerminalPosition(part, pin)
			if !ok || seen[point] {
				t.Fatal("rotated power/signal terminals overlap or disappear", rotation, pin)
			}
			seen[point] = true
		}
	}
	before := s.Snapshot()
	call(t, s, "component.rotate", IDParams{ID: "U1"}, "rotate")
	call(t, s, "component.move", MoveParams{ID: "U1", Position: Point{X: 750, Y: 300}}, "move")
	after := s.Snapshot()
	if before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Mixed, after.Mixed) {
		t.Fatal("geometry changed accepted amplifier physics")
	}
	after.OpAmpModel.OpenLoopGain = 1
	after.Mixed.OpAmps[0].Model.OpenLoopGain = 2
	if s.Snapshot().OpAmpModel.OpenLoopGain != 1e5 || s.Snapshot().Mixed.OpAmps[0].Model.OpenLoopGain != 1e5 {
		t.Fatal("public snapshots alias fixed model/result")
	}
}

func TestOpAmpProjectCaptureRecoveryAndModelGuards(t *testing.T) {
	s, _ := publicOpAmpCircuit(t)
	before := s.Snapshot()
	_, view := verify(t, s, VerifyParams{ProbeIDs: []string{"v:U1", "v:U1.input-difference", "i:U1", "i:U1.noninverting", "i:U1.vplus", "i:U1.vminus"}}, "verify")
	if view.Metadata.OpAmpModel == nil || view.Metadata.Mixed.OpAmps[0].Model != OpAmpDeviceModel() {
		t.Fatal("captured model missing")
	}
	view.Metadata.OpAmpModel.OpenLoopGain = 99
	if inspectOperation(t, s, view.Operation.ID).Metadata.OpAmpModel.OpenLoopGain != 1e5 {
		t.Fatal("capture metadata aliases client")
	}
	dir := t.TempDir()
	project := filepath.Join(dir, "opamp.elab")
	call(t, s, "project.save", SaveParams{Path: project}, "save")
	data, err := os.ReadFile(project)
	if err != nil {
		t.Fatal(err)
	}
	document, err := decodeDocument(data)
	if err != nil || document.OpAmpModel == nil {
		t.Fatal("model-bearing project failed decode", err)
	}
	for _, mode := range []string{"missing", "changed"} {
		var raw map[string]any
		_ = json.Unmarshal(data, &raw)
		if mode == "missing" {
			delete(raw, "opamp_model")
		} else {
			raw["opamp_model"].(map[string]any)["open_loop_gain_v_per_v"] = 42
		}
		encoded, _ := json.Marshal(raw)
		if _, err := decodeDocument(encoded); err == nil {
			t.Fatal("accepted invalid saved model", mode)
		}
	}
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: project}, "load")
	loaded := settled(t, fresh)
	if loaded.Document.RecordingCount != 1 || !reflect.DeepEqual(loaded.Mixed.OpAmps, before.Mixed.OpAmps) || *loaded.OpAmpModel != OpAmpDeviceModel() {
		t.Fatal("project lost model, capture or amplifier result", loaded.Diagnostics)
	}
	jsonPath := filepath.Join(dir, "verify.json")
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: jsonPath, Format: "json"}, "export")
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(jsonData); err != nil {
		t.Fatal("saved op-amp verification refused", err)
	}
	for _, mode := range []string{"missing-model", "result-model", "result-id", "result-current", "missing-result", "missing-ground"} {
		var raw map[string]any
		_ = json.Unmarshal(jsonData, &raw)
		metadata := raw["metadata"].(map[string]any)
		mixed := metadata["mixed"].(map[string]any)
		device := mixed["opamps"].([]any)[0].(map[string]any)
		switch mode {
		case "missing-model":
			delete(metadata, "opamp_model")
		case "result-model":
			device["model"].(map[string]any)["id"] = "unknown"
		case "result-id":
			device["id"] = "unknown"
		case "result-current":
			device["positive_supply_current_a"] = 1
		case "missing-result":
			delete(mixed, "opamps")
		case "missing-ground":
			compiled, _, _ := document.Graph.Compile()
			delete(mixed["nodes_v"].(map[string]any), compiled.Ground)
		}
		encoded, _ := json.Marshal(raw)
		if _, err := decodeRecording(encoded); err == nil {
			t.Fatal("accepted corrupted op-amp capture", mode)
		}
	}
	csvPath := filepath.Join(dir, "verify.csv")
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: csvPath, Format: "csv"}, "csv")
	csv, err := os.ReadFile(csvPath)
	if err != nil || !strings.Contains(string(csv), "opamp_model") || !strings.Contains(string(csv), OpAmpModelID) {
		t.Fatal("CSV lost model provenance", err)
	}
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "component.set", ValueParams{ID: "V1", Value: ".5 V"}, "recovery-edit")
	call(t, s, "simulation.restart", struct{}{}, "recovery-restart")
	before = settled(t, s)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	choice.DiscardChanges = true
	call(t, s, "recovery.restore", choice, "restore")
	restored := settled(t, s)
	if restored.OpAmpModel == nil || *restored.OpAmpModel != OpAmpDeviceModel() || !reflect.DeepEqual(restored.Mixed.OpAmps, before.Mixed.OpAmps) || restored.Document.RecordingCount != 1 {
		t.Fatal("recovery lost amplifier/model/capture", restored.Diagnostics)
	}
}

func TestOpAmpBiasSweepCapturesModelAndIndependentFollowerEnvelope(t *testing.T) {
	g := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000).wiredGraph(t)
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	cfg := SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "2 V", Points: 5, ProbeIDs: []string{"v:U1", "i:U1"}}
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "sweep")
	state := awaitSweep(t, s, "completed")
	inspection := call(t, s, "sweep.inspect", IDParams{ID: state.DCSweep.Selected}, "inspect").Result.(SweepInspection)
	if inspection.Metadata.Source.OpAmpModel == nil || *inspection.Metadata.Source.OpAmpModel != OpAmpDeviceModel() {
		t.Fatal("sweep omitted fixed op-amp model")
	}
	for _, point := range state.DCSweep.Result.Points {
		if math.Abs(point.Values[0]-point.Input) > .0001 || math.Abs(point.Values[1]-point.Values[0]/1000) > 1e-10 {
			t.Fatal("bias sweep missed follower/load-current envelope", point)
		}
	}
	path := filepath.Join(t.TempDir(), "sweep.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.DCSweep == nil || !reflect.DeepEqual(loaded.DCSweep.Result, state.DCSweep.Result) || loaded.OpAmpModel == nil {
		t.Fatal("project lost model-bearing sweep", loaded.Diagnostics)
	}
}

func TestOpAmpPulseBufferedRCChargeDischarge(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	f.graph.Components[0], _ = MakeComponent("V1", "pulse-source", Point{}, "1 V")
	f.pins["U1.noninverting"] = "rc"
	f.add(t, "RI", "resistor", "10k", "input", "rc")
	f.add(t, "C1", "capacitor", "1uF", "rc", "0")
	s := newGraphService(SolveLED, f.wiredGraph(t))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	initial := settled(t, s)
	if initial.Analysis != NetworkTransientAnalysis || math.Abs(initial.Mixed.Capacitors[0].Voltage) > 1e-9 {
		t.Fatal("buffered RC startup", initial.Analysis, initial.Mixed, initial.Diagnostics)
	}
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:U1", "v:U1.noninverting", "i:U1", "v:C1", "i:U1.vplus", "i:U1.vminus"}}, "probes")
	call(t, s, "simulation.step", StepParams{Duration: .05}, "charge")
	charged := s.Snapshot()
	want := -math.Expm1(-5)
	if math.Abs(charged.Mixed.Capacitors[0].Voltage-want) > .001 || math.Abs(charged.Mixed.OpAmps[0].OutputVoltage-want) > .001 {
		t.Fatal("buffered charge missed exponential envelope", charged.Mixed, want)
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "discharge")
	discharged := s.Snapshot()
	want *= math.Exp(-1)
	if math.Abs(discharged.Mixed.Capacitors[0].Voltage-want) > .001 || math.Abs(discharged.Mixed.OpAmps[0].OutputVoltage-want) > .001 || discharged.Scope.Stale || len(discharged.Scope.Traces) != 6 {
		t.Fatal("buffered discharge/probes missed envelope", discharged.Mixed, discharged.Scope, want)
	}
	for _, trace := range discharged.Scope.Traces {
		if len(trace.Points) < 2 {
			t.Fatal("new signed probe lost transient history", trace.Probe)
		}
	}
	duration := .06
	_, recorded := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: discharged.Scope.Selected}, "verify")
	if recorded.Metadata.OpAmpModel == nil || math.Abs(recorded.Metadata.Mixed.OpAmps[0].OutputVoltage-want) > .001 {
		t.Fatal("independent Verify lost powered RC discharge", recorded.Metadata)
	}
}

func TestOpAmpSVGPowerPinsModelsAndDependencyRefusal(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	g := f.wiredGraph(t)
	for _, rotation := range []int{0, 90, 180, 270} {
		for k := range g.Components {
			if g.Components[k].Kind == "opamp" {
				g.Components[k].Rotation = rotation
			}
		}
		decoder := xml.NewDecoder(bytes.NewReader(schematicSVG(g, 1, 1)))
		pins := map[string]bool{}
		var metadata map[string]json.RawMessage
		labels := map[string]bool{}
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal("invalid schematic XML", rotation, err)
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			switch start.Name.Local {
			case "metadata":
				var text string
				if err := decoder.DecodeElement(&text, &start); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(text), &metadata); err != nil {
					t.Fatal(err)
				}
			case "title", "text":
				var text string
				if err := decoder.DecodeElement(&text, &start); err != nil {
					t.Fatal(err)
				}
				if part, _, ok := g.Terminal(text); ok && part.ID == "U1" {
					pins[text] = true
				}
				labels[text] = true
			}
		}
		if len(pins) != 5 || !labels["V+"] || !labels["V-"] {
			t.Fatal("export lost independently named power/signal pins", rotation, pins, labels)
		}
		var model OpAmpModel
		if err := json.Unmarshal(metadata["opamp_model"], &model); err != nil || model != OpAmpDeviceModel() {
			t.Fatal("export omitted exact op-amp assumptions", err)
		}
	}
	captured := dcCircuitSnapshot(g, DefaultTransientSettings())
	for _, mode := range []string{"valid", "missing", "changed"} {
		doc := captured.document()
		if mode == "missing" {
			doc.OpAmpModel = nil
		} else if mode == "changed" {
			model := *doc.OpAmpModel
			model.ActiveCurrentLimit = 1
			doc.OpAmpModel = &model
		}
		data, _ := json.Marshal(doc)
		_, err := decodeDocument(data)
		if mode == "valid" && err != nil || mode != "valid" && err == nil {
			t.Fatal("sweep source did not enforce model dependency", mode, err)
		}
	}
	legacy := dcCircuitSnapshot(ExampleGraph(), DefaultTransientSettings())
	data, _ := json.Marshal(legacy)
	if bytes.Contains(data, []byte("opamp_model")) {
		t.Fatal("legacy source gained an unrelated model dependency")
	}
	paths := OpAmpSymbolPaths()
	paths[0][0].X = 99
	if OpAmpSymbolPaths()[0][0].X == 99 {
		t.Fatal("symbol caller aliases shared geometry")
	}
}
