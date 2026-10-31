package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func TestButtonDeterministicBounceAndAutomaticRelease(t *testing.T) {
	m := ControlMemory{Pressed: true, Bouncing: true, StartedAt: .123, ReleaseAt: .173}
	for k, want := range []bool{true, false, true, false, true} {
		got := buttonAt("bouncing-pushbutton", m, .123+float64(k)*.001)
		if got.ContactClosed != want {
			t.Fatal("press bounce", k, got)
		}
	}
	for k, want := range []bool{false, true, false, true, false} {
		got := buttonAt("bouncing-pushbutton", m, .173+float64(k)*.001)
		if got.ContactClosed != want || got.Pressed {
			t.Fatal("release bounce", k, got)
		}
	}
	p, _ := MakeComponent("S1", "pushbutton", Point{}, "")
	e := controlCurrents(p, [5]float64{5, 2}, ControlMemory{ContactClosed: true})
	if e.Current[0] != 3 || e.Current[1] != -3 {
		t.Fatal("closed resistance/KCL")
	}
	e = controlCurrents(p, [5]float64{5, 2}, ControlMemory{})
	if math.Abs(e.Current[0]-3e-9) > 1e-20 {
		t.Fatal("off leakage")
	}
}
func TestButtonCounterAndRCDebounceThroughSharedCommands(t *testing.T) {
	for _, id := range []string{"button-counter", "button-debounce"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			initial := settled(t, s)
			if initial.Analysis != ControlAnalysis || initial.ControlModel == nil || initial.Document.Dirty {
				t.Fatal("button startup", initial.Diagnostics)
			}
			call(t, s, "button.click", IDParams{ID: "SCOUNT"}, "click")
			pressed := settled(t, s)
			if !pressed.Mixed.Controls[0].Memory.Pressed || pressed.Document.Dirty || pressed.RunID != initial.RunID {
				t.Fatal("button click changed project/run")
			}
			call(t, s, "simulation.step", StepParams{Duration: .25}, "finish-click")
			after := s.Snapshot()
			counts := map[string]int{}
			for _, u := range after.Mixed.Sequential {
				counts[u.ID] = u.Memory.Value
			}
			want := 1
			if id == "button-debounce" {
				want = 5
				if counts["U4"] != 1 {
					t.Fatal("RC did not debounce to one count", counts)
				}
			}
			if counts["U2"] != want {
				t.Fatal("raw counter", counts, want)
			}
			for _, u := range after.Mixed.Controls {
				if u.ID == "SCOUNT" && (u.Memory.Pressed || u.Memory.ContactClosed || u.Memory.Bouncing || u.Memory.ReleaseAt != 0) {
					t.Fatal("button failed to release", u.Memory)
				}
			}
			if after.Document.Dirty || after.RunID != initial.RunID || after.Mixed.KCLResidual > 1e-8 || math.Abs(after.Mixed.PowerResidual) > 1e-7 {
				t.Fatal("button live invariants")
			}
			call(t, s, "button.set", ButtonParams{ID: "SRESET", Pressed: true}, "reset")
			reset := settled(t, s)
			for _, u := range reset.Mixed.Sequential {
				if u.Memory.Value != 0 {
					t.Fatal("momentary async reset")
				}
			}
			call(t, s, "button.set", ButtonParams{ID: "SRESET", Pressed: false}, "release")
			settled(t, s)
			call(t, s, "button.set", ButtonParams{ID: "SCOUNT", Pressed: true}, "hold")
			settled(t, s)
			call(t, s, "simulation.step", StepParams{Duration: .1}, "held-step")
			if !s.Snapshot().Mixed.Controls[0].Memory.Pressed {
				t.Fatal("held button auto released")
			}
			call(t, s, "simulation.restart", struct{}{}, "restart")
			fresh := settled(t, s)
			for _, u := range fresh.Mixed.Controls {
				if u.Memory != (ControlMemory{}) {
					t.Fatal("restart retained runtime button", u)
				}
			}
		})
	}
}
func TestButtonCoarseOutputDoesNotMissBounceAndCapturesValidate(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "button-debounce"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	cfg := s.Snapshot().TransientSettings
	cfg.StepSeconds = .01
	call(t, s, "simulation.transient.set", cfg, "coarse")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	call(t, s, "button.click", IDParams{ID: "SCOUNT"}, "click")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .25}, "coarse-step")
	a := s.Snapshot()
	for _, u := range a.Mixed.Sequential {
		want := 5
		if u.ID == "U4" {
			want = 1
		}
		if u.Memory.Value != want {
			t.Fatal("coarse output missed contact edges", u)
		}
	}
	if err := validateRecordedControls(s.mixedState.net.c, s.graph, a.Mixed); err != nil {
		t.Fatal("valid runtime state", err)
	}
	a.Mixed.Controls[0].Memory.ContactClosed = true
	if err := validateRecordedControls(s.mixedState.net.c, s.graph, a.Mixed); err == nil {
		t.Fatal("inconsistent contact memory accepted")
	}
	duration, step := .02, .0005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U2.clk", "v:U4.clk"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("button capture", err)
	}
	path := filepath.Join(t.TempDir(), "buttons.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	r := settled(t, fresh)
	if r.ControlModel == nil || r.Document.RecordingCount != 1 || r.Document.Dirty {
		t.Fatal("button project")
	}
	for _, u := range r.Mixed.Controls {
		if u.Memory != (ControlMemory{}) {
			t.Fatal("reopen pressed a button")
		}
	}
}

func TestButtonSwitchedRCMatchesIndependentExponentials(t *testing.T) {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "5", 0, 0, 0)
	b.ground("GND", "V1.negative", 0, 200)
	b.part("S1", "bouncing-pushbutton", "", 200, 0, 0)
	b.wire("V1.positive", "S1.1")
	b.part("R1", "resistor", "1k", 400, 0, 0)
	b.wire("S1.2", "R1.1")
	b.part("C1", "capacitor", "10u", 600, 0, 0)
	b.wire("R1.2", "C1.1")
	b.wire("C1.2", "GND.reference")
	b.part("RP", "resistor", "10k", 600, 200, 0)
	b.wire("RP.1", "C1.1")
	b.wire("RP.2", "GND.reference")
	c, _, d := b.graph.Compile()
	if d != nil {
		t.Fatal(d)
	}
	expected := 0.
	previous := 0.
	for k, end := range []float64{.001, .002, .003, .004, .02} {
		top := 1001.
		if k%2 != 0 {
			top = 1e9 + 1000
		}
		tau := top * 10000 / (top + 10000) * 1e-5
		target := 5 * 10000 / (top + 10000)
		expected = target + (expected-target)*math.Exp(-(end-previous)/tau)
		previous = end
	}
	for _, step := range []float64{.0005, .005, .01} {
		kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal(err)
		}
		net := *kernel.mixed.net
		net.controlMemory = append([]ControlMemory{}, net.controlMemory...)
		if err := net.stageButton("button:S1:press", 0); err != nil {
			t.Fatal(err)
		}
		state, err := net.solveContext(t.Context(), kernel.mixed.x, kernel.mixed.capV, 0, kernel.trace, 0)
		if err != nil {
			t.Fatal(err)
		}
		kernel.mixed = state
		for at := step; at < .02+step/2; at += step {
			if err := kernel.advance(t.Context(), math.Min(at, .02)); err != nil {
				t.Fatal(step, err)
			}
		}
		// This is accumulated trajectory error, not the local full/half-step
		// tolerance. Bound it to 100 µV across all three output grids.
		if math.Abs(kernel.mixed.capV[0]-expected) > 1e-4 {
			t.Fatal("switched RC exponential", step, kernel.mixed.capV[0], expected)
		}
	}
}
