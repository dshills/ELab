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

func potGraph(t *testing.T, loadKind string) Graph {
	t.Helper()
	g := EmptyGraph()
	parts := []PlaceParams{{ID: "V1", Kind: "dc-source", Position: Point{-200, 0}, Value: "5V"}, {ID: "P1", Kind: "potentiometer", Position: Point{0, -100}, Value: "10k"}, {ID: "GND", Kind: "ground", Position: Point{0, 120}}}
	if loadKind != "" {
		value := "10k"
		id := "R1"
		if loadKind == "capacitor" {
			value = "1uF"
			id = "C1"
		}
		parts = append(parts, PlaceParams{ID: id, Kind: loadKind, Position: Point{200, 0}, Value: value})
	}
	for _, p := range parts {
		c, err := MakeComponent(p.ID, p.Kind, p.Position, p.Value)
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
	}
	pairs := [][2]string{{"V1.positive", "P1.1"}, {"P1.2", "GND.reference"}, {"V1.negative", "GND.reference"}}
	if loadKind != "" {
		id := "R1"
		if loadKind == "capacitor" {
			id = "C1"
		}
		pairs = append(pairs, [2]string{"P1.wiper", id + ".1"}, [2]string{id + ".2", "GND.reference"})
	}
	for i, p := range pairs {
		g.Wires = append(g.Wires, Wire{ID: fmtWire(i), From: p[0], To: p[1], Connected: true})
	}
	return g
}
func TestPotentiometerDividerEndpointsLoadedConservation(t *testing.T) {
	for _, loaded := range []bool{false, true} {
		kind := ""
		if loaded {
			kind = "resistor"
		}
		s := newGraphService(SolveLED, potGraph(t, kind))
		settled(t, s)
		for i, w := range []float64{0, .25, .5, .75, 1} {
			call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: w}, fmtWire(i))
			state := s.Snapshot()
			if state.PotentiometerModel == nil || state.Mixed == nil || len(state.Mixed.Potentiometers) != 1 {
				t.Fatalf("pot unavailable: %+v", state.Diagnostics)
			}
			result := state.Mixed.Potentiometers[0]
			// Independent divider equation, including the model's finite track endpoints.
			upper, lower := 1+9998*(1-w), 1+9998*w
			effective := lower
			if loaded {
				effective = 1 / (1/lower + 1./10000)
			}
			want := 5 * effective / (upper + effective)
			if math.Abs(result.WiperVoltage-want) > 1e-9 || result.UpperResistance+result.LowerResistance != 10000 || math.Abs(result.Terminal1Current+result.Terminal2Current+result.WiperCurrent) > 1e-12 || state.Mixed.KCLResidual > 1e-10 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
				t.Fatalf("loaded=%v w=%g result %+v reference %g", loaded, w, result, want)
			}
			if !loaded && math.Abs(result.WiperCurrent) > 1e-12 {
				t.Fatal("unloaded wiper carried current")
			}
		}
		s.Close()
	}
}
func TestPotentiometerRCWiperPreservesChargeAndLiveClock(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .005}, "charge")
	before := s.Snapshot()
	want := 2.5 * (1 - math.Exp(-.005/.0025))
	if math.Abs(before.Mixed.Capacitors[0].Voltage-want) > .002 {
		t.Fatalf("Thevenin RC reference got %g want %g", before.Mixed.Capacitors[0].Voltage, want)
	}
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .75}, "wiper")
	after := s.Snapshot()
	if after.RunID != before.RunID || after.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(after.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-10 || after.Transient.Events[len(after.Transient.Events)-1].Mode != "wiper:P1:0.75" {
		t.Fatal("wiper movement lost charge/time/run/event")
	}
	call(t, s, "simulation.step", StepParams{Duration: .005}, "second-charge")
	next := s.Snapshot()
	target := 5 * 7499.5 / 10000
	tau := (2500.5 * 7499.5 / 10000) * 1e-6
	want = target + (before.Mixed.Capacitors[0].Voltage-target)*math.Exp(-.005/tau)
	if math.Abs(next.Mixed.Capacitors[0].Voltage-want) > .002 {
		t.Fatalf("changed Thevenin RC reference got %g want %g", next.Mixed.Capacitors[0].Voltage, want)
	}
	duration := .005
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:P1.wiper", "i:P1.wiper", "v:P1"}}, "verify")
	if record.Metadata.PotentiometerModel == nil || record.Metadata.Mixed.Potentiometers[0].WiperPosition != .75 {
		t.Fatal("Verify lost pot model/wiper")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume")
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .5}, "running-wiper")
	start := s.Snapshot()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.Transient.Sample.Time > start.Transient.Sample.Time {
			if state.RunID != start.RunID || state.Paused || state.Transient.Stale {
				t.Fatal("wiper stopped the live clock")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("wiper live clock did not continue")
}
func TestPotentiometerRheostatUnusedAndTiedEnd(t *testing.T) {
	for _, tied := range []bool{false, true} {
		g := potGraph(t, "")
		g.Wires[1].From = "P1.wiper"
		if tied {
			g.Wires = append(g.Wires, Wire{ID: "Tie", From: "P1.2", To: "P1.wiper", Connected: true})
		}
		s := newGraphService(SolveLED, g)
		state := settled(t, s)
		if state.Mixed == nil || math.Abs(state.Mixed.Potentiometers[0].Terminal1Current-.001) > 1e-12 {
			t.Fatalf("rheostat tied=%v: %+v", tied, state.Diagnostics)
		}
		s.Close()
	}
}
func TestPotentiometerValidationCopiesAndPortableRecords(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "resistor"))
	defer s.Close()
	settled(t, s)
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .75}, "wiper")
	before := s.Snapshot()
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .75}, "noop")
	if s.Snapshot().RunID != before.RunID || s.Snapshot().ElectricalRevision != before.ElectricalRevision {
		t.Fatal("no-op wiper restarted")
	}
	for _, w := range []float64{-.1, 1.1} {
		refuse(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: w}, "invalid-params")
	}
	refuse(t, s, "potentiometer.wiper.set", WiperParams{ID: "R1", Position: .5}, "invalid-params")
	refuse(t, s, "component.set", ValueParams{ID: "P1", Value: "1Ω"}, "invalid-params")
	snapshot := s.Snapshot()
	for i := range snapshot.Components {
		if snapshot.Components[i].WiperPosition != nil {
			*snapshot.Components[i].WiperPosition = 0
		}
	}
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("snapshot aliases wiper")
	}
	verify(t, s, VerifyParams{ProbeIDs: []string{"v:P1.wiper", "i:P1.1", "i:P1.2", "i:P1.wiper"}}, "verify")
	path := filepath.Join(t.TempDir(), "pot.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	restored := settled(t, fresh)
	if restored.Document.RecordingCount != 1 || restored.PotentiometerModel == nil || !reflect.DeepEqual(restored.Components, before.Components) || restored.Mixed.Potentiometers[0].WiperVoltage != before.Mixed.Potentiometers[0].WiperVoltage {
		t.Fatal("pot project/record failed round trip")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	originalGeneration := fresh.Snapshot().Document.Generation
	for _, mutation := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"model", func(d map[string]any) { d["potentiometer_model"].(map[string]any)["end_resistance_ohm"] = 0 }},
		{"missing-model", func(d map[string]any) { delete(d, "potentiometer_model") }},
		{"range", func(d map[string]any) {
			for _, raw := range d["graph"].(map[string]any)["components"].([]any) {
				p := raw.(map[string]any)
				if p["kind"] == "potentiometer" {
					p["wiper_position"] = -.1
				}
			}
		}},
		{"missing-wiper", func(d map[string]any) {
			for _, raw := range d["graph"].(map[string]any)["components"].([]any) {
				p := raw.(map[string]any)
				if p["kind"] == "potentiometer" {
					delete(p, "wiper_position")
				}
			}
		}},
		{"foreign-wiper", func(d map[string]any) {
			for _, raw := range d["graph"].(map[string]any)["components"].([]any) {
				p := raw.(map[string]any)
				if p["kind"] == "resistor" {
					p["wiper_position"] = .5
				}
			}
		}},
	} {
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		mutation.change(document)
		bad, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		badPath := filepath.Join(t.TempDir(), mutation.name+".elab")
		if err = os.WriteFile(badPath, bad, 0600); err != nil {
			t.Fatal(err)
		}
		refuse(t, fresh, "project.load", LoadParams{Path: badPath}, "invalid-document")
		if fresh.Snapshot().Document.Generation != originalGeneration {
			t.Fatal("invalid pot project replaced the current document")
		}
	}

}

func TestPotentiometerBatchUndoAndDuplicateState(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	wiper, _ := json.Marshal(WiperParams{ID: "P1", Position: .8})
	value, _ := json.Marshal(ValueParams{ID: "P1", Value: "20k"})
	call(t, s, "edit.batch", BatchParams{Name: "Adjust track", Edits: []BatchEdit{{Command: "component.set", Params: value}, {Command: "potentiometer.wiper.set", Params: wiper}}}, "batch")
	state := s.Snapshot()
	if !state.Transient.Stale || !state.Mixed.Stale || state.UndoName != "Adjust track" {
		t.Fatal("paused batch failed to invalidate the previous result")
	}
	call(t, s, "simulation.restart", struct{}{}, "restart-batch")
	state = s.Snapshot()
	if state.Transient.Sample.Time != 0 || state.UndoName != "Adjust track" || state.Mixed.Potentiometers[0].Resistance != 20000 || state.Mixed.Potentiometers[0].WiperPosition != .8 {
		t.Fatal("atomic track edit did not restart as a named history entry")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	call(t, s, "simulation.restart", struct{}{}, "restart-undo")
	if state = s.Snapshot(); state.Mixed.Potentiometers[0].Resistance != 10000 || state.Mixed.Potentiometers[0].WiperPosition != .5 {
		t.Fatal("pot batch undo failed")
	}
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8}, "set")
	call(t, s, "component.duplicate", DuplicateParams{ID: "P1"}, "duplicate")
	copy, ok := s.graph.Component(s.Snapshot().Selected)
	if !ok || copy.WiperPosition == nil || *copy.WiperPosition != .8 {
		t.Fatal("pot duplication lost wiper")
	}
	*copy.WiperPosition = .2
	original, _ := s.graph.Component("P1")
	if *original.WiperPosition != .8 {
		t.Fatal("pot duplicate aliases original")
	}
}

func TestPotentiometerOpenAndAllPinsShortDiagnostics(t *testing.T) {
	for _, allShort := range []bool{false, true} {
		g := potGraph(t, "")
		if allShort {
			g.Wires[1].To = "P1.1"
			g.Wires = append(g.Wires, Wire{ID: "Short", From: "P1.1", To: "P1.wiper", Connected: true})
		} else {
			g.Wires = append(g.Wires[:1], g.Wires[2:]...)
		}
		_, _, diagnostic := g.Compile()
		want := "open-terminal"
		if allShort {
			want = "shorted-component"
		}
		if diagnostic == nil || diagnostic.Code != want || diagnostic.ObjectID != "P1" {
			t.Fatalf("pot diagnostic: %+v", diagnostic)
		}
	}
}
