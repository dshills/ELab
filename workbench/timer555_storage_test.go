package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func TestTimer555ExamplesCaptureSaveRecoveryAndStrictImport(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	for _, id := range []string{"555-flasher", "555-oscillator", "555-one-shot"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.Timer555Model == nil || *initial.Timer555Model != Timer555DeviceModel() || initial.Limits.Timers555 != 2 || initial.Document.Dirty || initial.Analysis != Timer555Analysis {
				t.Fatal("timer discovery", initial.Diagnostics)
			}
			duration, step := 1.2, .005
			before := s.Snapshot()
			_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "v:C1", "i:U1", "i:U1.discharge"}}, "capture")
			if view.Metadata.Engine != Timer555EngineID || view.Metadata.Timer555Model == nil || !reflect.DeepEqual(before.Transient, s.Snapshot().Transient) {
				t.Fatal("555 capture changed live state or lost provenance")
			}
			captured := s.operations[view.Operation.ID].record
			raw, _ := json.Marshal(VerificationDocument{Metadata: captured.metadata, Rows: captured.rows})
			decoded, err := decodeRecording(raw)
			if err != nil {
				t.Fatal("valid 555 recording refused", err)
			}
			detached := cloneVerificationMetadata(decoded.metadata)
			detached.Timer555Model.OutputResistance = 1
			if decoded.metadata.Timer555Model.OutputResistance != 50 {
				t.Fatal("capture shares model pointer")
			}
			path := filepath.Join(t.TempDir(), id+".elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "load")
			restored := settled(t, fresh)
			if !reflect.DeepEqual(restored.Components, initial.Components) || !reflect.DeepEqual(restored.Wires, initial.Wires) || restored.Document.Dirty || restored.Document.RecordingCount != 1 || !restored.Mixed.Timers555[0].High {
				t.Fatal("555 project/capture restoration", restored.Diagnostics)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			d, err := decodeDocument(data)
			if err != nil {
				t.Fatal(err)
			}
			inspection := inspectDocument(d)
			if inspection.Timer555Model == nil {
				t.Fatal("recovery inspection lost timer model")
			}
			var document map[string]any
			json.Unmarshal(data, &document)
			delete(document, "timer555_model")
			bad, _ := json.Marshal(document)
			if _, err = decodeDocument(bad); err == nil {
				t.Fatal("missing 555 model accepted")
			}
			var record map[string]any
			json.Unmarshal(raw, &record)
			metadata := record["metadata"].(map[string]any)
			mixed := metadata["mixed"].(map[string]any)
			timer := mixed["timers555"].([]any)[0].(map[string]any)
			for _, field := range []string{"high", "powered", "model", "pin_currents_a"} {
				saved := timer[field]
				delete(timer, field)
				bad, _ := json.Marshal(record)
				if _, err = decodeRecording(bad); err == nil {
					t.Fatal("missing timer field accepted", field)
				}
				timer[field] = saved
			}
			saved := timer["pin_currents_a"]
			timer["pin_currents_a"] = saved.([]any)[:7]
			bad, _ = json.Marshal(record)
			if _, err = decodeRecording(bad); err == nil {
				t.Fatal("incomplete pin-current array accepted")
			}
			timer["pin_currents_a"] = saved
			timer["power_w"] = 1.
			bad, _ = json.Marshal(record)
			if _, err = decodeRecording(bad); err == nil {
				t.Fatal("tampered timer result accepted")
			}
			state := s.Snapshot()
			state.Timer555Model.OutputResistance = 2
			if s.Snapshot().Timer555Model.OutputResistance != 50 {
				t.Fatal("snapshot aliases model")
			}
		})
	}
}
func TestTimer555CompileLimitsOpenInputsAndTransientOnlySweep(t *testing.T) {
	g := Timer555FlasherExampleGraph()
	for _, pin := range []string{"trigger", "threshold", "reset", "vcc", "gnd"} {
		broken := g.Clone()
		wires := []Wire{}
		for _, wire := range broken.Wires {
			if wire.From != "U1."+pin && wire.To != "U1."+pin {
				wires = append(wires, wire)
			}
		}
		broken.Wires = wires
		if _, _, d := broken.Compile(); d == nil {
			t.Fatal("unwired required timer pin accepted", pin)
		}
	}
	if _, err := newSweepPlan(g, SweepSettings{ComponentID: "RA", Parameter: "resistance", Start: "10k", Stop: "20k", Points: 3}); err == nil {
		t.Fatal("555 DC sweep silently chose latch history")
	}
	for i := 2; i <= 3; i++ {
		id := fmt.Sprintf("U%d", i)
		p, _ := MakeComponent(id, "timer555", Point{X: float64(i) * 300, Y: 600}, "")
		g.Components = append(g.Components, p)
		for _, pin := range timer555Pins {
			g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%s_%s", id, pin), From: id + "." + pin, To: "U1." + pin, Connected: true})
		}
		_, _, d := g.Compile()
		if i == 2 && d != nil {
			t.Fatal("second timer refused", d)
		}
		if i == 3 && (d == nil || d.Code != "mixed-limit") {
			t.Fatal("third timer limit absent", d)
		}
	}
	registry := SchemaRegistry()
	var schema map[string]any
	json.Unmarshal(registry["urn:elab:timer555-result:1"], &schema)
	pins := schema["properties"].(map[string]any)["pin_currents_a"].(map[string]any)
	if pins["type"] != "array" || pins["minItems"] != float64(8) || pins["maxItems"] != float64(8) {
		t.Fatal("pin-current discovery shape", pins)
	}
	for _, rotation := range []int{0, 90, 180, 270} {
		p, _ := MakeComponent("U1", "timer555", Point{}, "")
		p.Rotation = rotation
		seen := map[Point]bool{}
		for _, pin := range p.Terminals {
			at, ok := TerminalPosition(p, pin)
			if !ok || seen[at] || math.Hypot(at.X, at.Y) < 90 {
				t.Fatal("555 pin geometry", rotation, pin, at)
			}
			seen[at] = true
		}
	}
}

func TestTimer555ExampleCaptionsClearPowerAndBypassPins(t *testing.T) {
	for _, g := range []Graph{Timer555FlasherExampleGraph(), Timer555OscillatorExampleGraph(), Timer555OneShotExampleGraph()} {
		timer, _ := g.Component("U1")
		x, y, text := schematicPartLabel(timer)
		supply, _ := TerminalPosition(timer, "vcc")
		// The upper supply feed shares the VCC pin's X coordinate. Conservative
		// export masks must not erase it above the symbol.
		if supply.X >= x-3 && supply.X <= x+schematicLabelWidth(text) {
			t.Fatal("555 caption masks VCC feed")
		}
		cap, _ := g.Component("CC")
		ground, _ := g.Component("GNDU")
		cx, cy, ct := schematicPartLabel(cap)
		gx, gy, gt := schematicPartLabel(ground)
		if cx-3 < gx+schematicLabelWidth(gt) && cx+schematicLabelWidth(ct) > gx-3 && math.Abs(cy-gy) < 20 {
			t.Fatal("bypass and ground captions crowd")
		}
		if y > timer.Position.Y-140 {
			t.Fatal("555 caption crowds upper pin route")
		}
	}
}
