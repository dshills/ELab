package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestDigitalDisplayDecoderTruthAndJacobian(t *testing.T) {
	expected := []string{"abcdef", "bc", "abdeg", "abcdg", "bcfg", "acdfg", "acdefg", "abc", "abcdefg", "abcdfg", "", "", "", "", "", ""}
	for _, span := range []float64{3.3, 5, 12} {
		for _, ground := range []float64{-2, 0, 3} {
			for code, segments := range expected {
				var v [13]float64
				for k := range v {
					v[k] = ground
				}
				v[11] += span
				for k := range 4 {
					if code&(1<<k) != 0 {
						v[k] += span
					}
				}
				e := decoderCurrents(v)
				for k, pin := range segmentNames {
					g := 1 / LogicDeviceModel().OutputResistance
					enable := span * span / (span*span + .01)
					want := 0.
					if strings.Contains(segments, pin) {
						want = -span * g * enable
					}
					if math.Abs(e.Current[4+k]-want) > 1e-12 {
						t.Fatal("decoder truth", span, ground, code, pin, e.Current[4+k], want)
					}
				}
			}
		}
	}
	for _, span := range []float64{-1, 0, 1e-9, 5, 12} {
		var v [13]float64
		v[11] = span
		for k := range 4 {
			v[k] = span * (.35 + .08*float64(k))
		}
		for k := 4; k < 11; k++ {
			v[k] = float64(k) * .2
		}
		e := decoderCurrents(v)
		sum, power := 0., 0.
		for k, i := range e.Current {
			sum += i
			power += (v[k] - v[12]) * i
		}
		if math.Abs(sum) > 1e-12 || power < -1e-12 {
			t.Fatal("passivity/KCL", span, sum, power)
		}
		if math.Abs(span) < 1e-6 {
			continue
		}
		for col := range v {
			a, b := v, v
			a[col] += 1e-6
			b[col] -= 1e-6
			ea, eb := decoderCurrents(a), decoderCurrents(b)
			for row := range v {
				want := (ea.Current[row] - eb.Current[row]) / 2e-6
				if math.Abs(want-e.Jacobian[row][col]) > 1e-7 {
					t.Fatal("decoder derivative", span, row, col, want, e.Jacobian[row][col])
				}
			}
		}
	}
}
func TestDigitalDisplaySegmentElectricalLawAndIsolation(t *testing.T) {
	f := newOpAmpNetworkFixture()
	f.add(t, "DS1", "seven-segment", "", "a", "b", "c", "d", "e", "f", "g", "0")
	n := f.network()
	p := n.c.MixedGraph.Components[0]
	var v [13]float64
	v[4] = 2.1
	e := n.displayCurrents(p, v)
	i, g := n.ledCurrent(2.1)
	if math.Abs(e.Current[4]-i) > 1e-12 || e.Jacobian[4][4] != g || i <= 0 {
		t.Fatal("segment LED law")
	}
	for k := 5; k <= 10; k++ {
		if e.Current[k] != 0 {
			t.Fatal("unpowered segment lights", k)
		}
	}
	if e.Current[12] != -i {
		t.Fatal("common return current")
	}
	v[4] = -5
	e = n.displayCurrents(p, v)
	if e.Current[4] != 0 {
		t.Fatal("undeclared reverse breakdown")
	}
}
func TestDigitalDisplay555CounterAllDigits(t *testing.T) {
	g := DisplayCounterExampleGraph()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if kernel.mixed.result.Sequential[0].Memory.Value != 0 {
		t.Fatal("phantom initial count")
	}
	seen := map[int]bool{}
	last, changes := 0, 0
	for k := 0; k < 1800; k++ {
		if k > 0 {
			if err := kernel.advance(t.Context(), float64(k)*.005); err != nil {
				t.Fatal(k, err)
			}
		}
		r := kernel.mixed.result
		digit := -1
		for _, u := range r.Displays {
			if u.ID == "DS1" {
				digit = u.Digit
				sum := 0.
				for _, i := range u.SegmentCurrents {
					sum += i
					if i < -1e-12 {
						t.Fatal("negative segment current")
					}
				}
				if math.Abs(sum+u.PinCurrents[12]) > 1e-10 {
					t.Fatal("common current balance")
				}
			}
		}
		count := r.Sequential[0].Memory.Value
		if digit != count {
			t.Fatal("lit segments do not encode counter", k, digit, count)
		}
		seen[digit] = true
		if count != last {
			changes++
			if count != (last+1)%10 {
				t.Fatal("missed/wrong count", last, count)
			}
			last = count
		}
		if r.KCLResidual > 1e-8 || math.Abs(r.PowerResidual) > 1e-7 {
			t.Fatal("electrical balance", k, r.KCLResidual, r.PowerResidual)
		}
	}
	if len(seen) != 10 || changes < 10 {
		t.Fatal("counter did not cycle through all digits", seen, changes)
	}
}
func TestDigitalDisplaySharedPersistenceAndStrictCapture(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "display-counter"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := .05, .005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"i:DS1.a", "v:U3.a"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("valid capture", err)
	}
	var object map[string]any
	json.Unmarshal(raw, &object)
	meta := object["metadata"].(map[string]any)
	display := meta["mixed"].(map[string]any)["displays"].([]any)[0].(map[string]any)
	for _, key := range []string{"lit", "pin_currents_a", "segment_currents_a", "model"} {
		saved := display[key]
		delete(display, key)
		bad, _ := json.Marshal(object)
		if _, err := decodeRecording(bad); err == nil {
			t.Fatal("missing display data", key)
		}
		display[key] = saved
	}
	display["digit"] = 99
	bad, _ := json.Marshal(object)
	if _, err := decodeRecording(bad); err == nil {
		t.Fatal("forged digit")
	}
	path := filepath.Join(t.TempDir(), "display.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	a := settled(t, fresh)
	if a.DisplayModel == nil || a.Document.RecordingCount != 1 || a.Document.Dirty {
		t.Fatal("display reopen")
	}
	a.Mixed.Displays[0].Lit[0] = !a.Mixed.Displays[0].Lit[0]
	a.DisplayModel.VisibleCurrent = 1
	if fresh.Snapshot().DisplayModel.VisibleCurrent == 1 {
		t.Fatal("snapshot model alias")
	}
	doc := s.documentLocked()
	if inspectDocument(doc).DisplayModel == nil {
		t.Fatal("recovery model")
	}
	doc.DisplayModel = nil
	bad, _ = json.Marshal(doc)
	if _, err := decodeDocument(bad); err == nil {
		t.Fatal("model-less display project")
	}
	for _, kind := range []string{"bcd-decoder", "seven-segment"} {
		p, _ := MakeComponent("U", kind, Point{}, "")
		for _, angle := range []int{0, 90, 180, 270} {
			p.Rotation = angle
			for _, pin := range p.Terminals {
				if _, ok := TerminalPosition(p, pin); !ok {
					t.Fatal("missing pin", kind, angle, pin)
				}
			}
			svg := schematicSVG(Graph{Components: []Component{p}}, 1, 1)
			if !strings.Contains(string(svg), "display_model") {
				t.Fatal("export model", fmt.Sprint(angle))
			}
		}
	}
}
