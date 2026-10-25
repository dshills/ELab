package workbench

import (
	"context"
	"encoding/json"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"testing"
)

func acSettings(source, output string) SweepSettings {
	return SweepSettings{ComponentID: source, Parameter: "frequency", Start: "1 Hz", Stop: "100 kHz", Points: 201, Spacing: "logarithmic", ProbeIDs: []string{output}}
}
func TestACRCGainPhaseAndCutoffAgainstComplexReference(t *testing.T) {
	r := sweepForTest(t, ACRCExampleGraph(), acSettings("V1", "v:C1"))
	for _, p := range r.Points {
		want := 1 / complex(1, 2*math.Pi*p.Input*.001)
		z := p.AC[0]
		if cmplx.Abs(complex(z.Real, z.Imaginary)-want) > 1e-10 {
			t.Fatal("RC transfer", p, want)
		}
	}
	if len(r.AC) != 1 || len(r.AC[0].HalfPowerHz) != 1 || math.Abs(r.AC[0].HalfPowerHz[0]-1/(2*math.Pi*.001)) > .1 {
		t.Fatal("RC cutoff estimate", r.AC)
	}
}
func TestACRLCResonanceAndOpAmpFilter(t *testing.T) {
	r := sweepForTest(t, RLCExampleGraph(), acSettings("V1", "v:C1"))
	for _, p := range r.Points {
		w := 2 * math.Pi * p.Input
		want := 1 / complex(1-w*w*.1*10e-6, w*50*10e-6)
		z := p.AC[0]
		if cmplx.Abs(complex(z.Real, z.Imaginary)-want) > 1e-10 {
			t.Fatal("RLC transfer", p, want)
		}
	}
	if r.AC[0].PeakFrequency < 130 || r.AC[0].PeakFrequency > 160 || len(r.AC[0].HalfPowerHz) != 2 {
		t.Fatal("RLC resonance", r.AC)
	}
	g := OpAmpInvertingExampleGraph()
	// Capacitor shunts RF to form an inverting low-pass filter around zero DC.
	b := exampleBuilder{graph: g}
	b.part("CF", "capacitor", "10n", 0, 400, 0)
	b.wire("CF.1", "RF.1")
	b.wire("CF.2", "RF.2")
	r = sweepForTest(t, b.graph, acSettings("VIN", "v:U1"))
	for _, p := range r.Points {
		w := 2 * math.Pi * p.Input
		want := -2 / complex(1, w*20000*10e-9)
		z := p.AC[0]
		if cmplx.Abs(complex(z.Real, z.Imaginary)-want) > .002 {
			t.Fatal("opamp lowpass", p, want)
		}
	}
}
func TestACCaptureSaveReopenAndNoLiveMutation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	s.mu.Lock()
	s.graph = ACRCExampleGraph()
	s.mu.Unlock()
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	cfg := acSettings("V1", "v:C1")
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "ac")
	awaitSweep(t, s, "completed")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID {
		t.Fatal("AC changed live physics")
	}
	op := s.sweeps.operations[after.DCSweep.Selected]
	data, _ := json.Marshal(op.record)
	if _, err := validateSweepRecord(data); err != nil {
		t.Fatal("AC capture", err)
	}
	original := op.record.Result.Points[0].Values[0]
	op.record.Result.Points[0].Values[0]++
	data, _ = json.Marshal(op.record)
	if _, err := validateSweepRecord(data); err == nil {
		t.Fatal("forged AC gain accepted")
	}
	op.record.Result.Points[0].Values[0] = original
	path := filepath.Join(t.TempDir(), "filter.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	settled(t, s)
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	a := settled(t, fresh)
	if a.DCSweep == nil || a.DCSweep.Result == nil || !reflect.DeepEqual(a.DCSweep.Result, after.DCSweep.Result) {
		t.Fatal("AC project roundtrip")
	}
}
func TestACRefusalAndCancellation(t *testing.T) {
	if _, err := newSweepPlan(Timer555FlasherExampleGraph(), acSettings("VCC", "v:C1")); err == nil {
		t.Fatal("history-dependent AC accepted")
	}
	cfg := acSettings("V1", "i:R1")
	if _, err := newSweepPlan(ACRCExampleGraph(), cfg); err == nil {
		t.Fatal("current output accepted as voltage")
	}
	cfg = acSettings("V1", "v:C1")
	plan, err := newSweepPlan(ACRCExampleGraph(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := computeACSweep(ctx, plan, func(int) {}); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestACDescendingUnwrapAndZeroResponse(t *testing.T) {
	cfg := acSettings("V1", "v:C1")
	cfg.Start, cfg.Stop = cfg.Stop, cfg.Start
	r := sweepForTest(t, ACRCExampleGraph(), cfg)
	if math.Abs(r.Points[len(r.Points)-1].Values[1]) > 1 || len(r.AC[0].HalfPowerHz) != 1 {
		t.Fatal("descending phase/cutoff", r.AC)
	}
	cfg = acSettings("V1", "v:V1.negative")
	r = sweepForTest(t, ACRCExampleGraph(), cfg)
	for _, p := range r.Points {
		if p.Values[0] != acDBFloor || p.Values[1] != 0 || p.AC[0].PhaseDefined {
			t.Fatal("zero transfer must disclose undefined phase", p)
		}
	}
}
func TestBatteryIsNotAnIdealBiasSweepTarget(t *testing.T) {
	for _, target := range sweepTargets(BatteryExampleGraph(false)) {
		if target.ComponentID == "B1" {
			t.Fatal("battery reservoir cannot be silently replaced with ideal bias")
		}
	}
}
