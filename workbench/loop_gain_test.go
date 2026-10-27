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

func loopSettings() SweepSettings {
	return SweepSettings{ComponentID: "U1", Parameter: "loop-frequency", Spacing: "logarithmic", Start: ".1 Hz", Stop: "100 kHz", Points: 201}
}
func TestLoopReturnRatioAgainstIndependentLoadedTwoPole(t *testing.T) {
	g := FeedbackLoopExampleGraph()
	r := sweepForTest(t, g, loopSettings())
	if r.Loop == nil || len(r.Loop.Unity) != 1 || len(r.Loop.Phase) != 0 || r.Loop.PhaseReason == "" {
		t.Fatal("loop margins", r.Loop)
	}
	gain := 1e5
	wp := 2 * math.Pi * 1000 / gain
	gout := (100. / 100.01) / 50
	conductance := gout + 1e-4 + 1e-7 + 1e-9
	for _, p := range r.Points {
		w := 2 * math.Pi * p.Input
		want := complex(gout*gain*wp, 0) / (complex(wp, w) * complex(conductance, w*1e-6))
		got := complex(p.AC[0].Real, p.AC[0].Imaginary)
		if cmplx.Abs(got-want) > 1e-8*math.Max(1, cmplx.Abs(want)) {
			t.Fatal("independent loaded return ratio", p.Input, got, want)
		}
	}
	lo, hi := 0., 2*math.Pi*10000
	for k := 0; k < 80; k++ {
		w := (lo + hi) / 2
		mag := gout * gain * wp / (math.Hypot(wp, w) * math.Hypot(conductance, w*1e-6))
		if mag > 1 {
			lo = w
		} else {
			hi = w
		}
	}
	w := (lo + hi) / 2
	pm := 180 - (math.Atan(w/wp)+math.Atan(w*1e-6/conductance))*180/math.Pi
	if math.Abs(r.Loop.Unity[0].Frequency-w/(2*math.Pi)) > w/(2*math.Pi)*.001 || math.Abs(r.Loop.Unity[0].Margin-pm) > .02 {
		t.Fatal("independent unity/phase margin", r.Loop.Unity, w/(2*math.Pi), pm)
	}
	signal := sweepForTest(t, g, acSettings("VIN", "v:U1"))
	if signal.Loop != nil || r.Points[0].Values[0] < 40 || math.Abs(signal.Points[0].Values[0]) > 1 {
		t.Fatal("loop confused with signal gain")
	}
	cfg := loopSettings()
	cfg.Start, cfg.Stop = cfg.Stop, cfg.Start
	if _, err := newSweepPlan(g, cfg); err == nil {
		t.Fatal("descending loop band accepted")
	}
	if _, err := newSweepPlan(OpAmpFollowerExampleGraph(), loopSettings()); err == nil {
		t.Fatal("memoryless opamp injection advertised")
	}
}
func TestLoopMarginCrossingsAgainstIndependentThreePole(t *testing.T) {
	r := &SweepResult{Target: SweepTarget{ComponentID: "U1"}}
	previous := 0.
	for k := 0; k < 201; k++ {
		f := math.Pow(10, float64(k)*3/200)
		z := 4 / cmplx.Pow(complex(1, f/100), 3)
		db, phase, defined := acValues(z, previous, k > 0)
		previous = phase
		r.Points = append(r.Points, SweepPoint{Input: f, Values: []float64{db, phase}, AC: []ACResponse{{Real: real(z), Imaginary: imag(z), PhaseDefined: defined}}})
	}
	m, err := loopMargins(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Unity) != 1 || len(m.Phase) != 1 {
		t.Fatal("missing three-pole crossings", m)
	}
	fc := 100 * math.Sqrt(math.Pow(4, 2./3)-1)
	pm := 180 - 3*math.Atan(fc/100)*180/math.Pi
	if math.Abs(m.Unity[0].Frequency-fc) > .05 || math.Abs(m.Unity[0].Margin-pm) > .03 || math.Abs(m.Phase[0].Frequency-100*math.Sqrt(3)) > .05 || math.Abs(m.Phase[0].Margin-20*math.Log10(2)) > .005 {
		t.Fatal("three-pole margins", m, fc, pm)
	}
	// Repeated exact endpoint arrivals are not counted twice.
	r.Points = []SweepPoint{{Input: 1, Values: []float64{1, -170}, AC: []ACResponse{{PhaseDefined: true}}}, {Input: 2, Values: []float64{0, -180}, AC: []ACResponse{{PhaseDefined: true}}}, {Input: 3, Values: []float64{-1, -190}, AC: []ACResponse{{PhaseDefined: true}}}}
	m, err = loopMargins(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Unity) != 1 || len(m.Phase) != 1 || m.Unity[0].Frequency != 2 || m.Phase[0].Frequency != 2 {
		t.Fatal("endpoint deduplication", m)
	}
}
func TestLoopSharedCaptureSaveTamperingAndCancellation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "feedback-loop"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	cfg := loopSettings()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	after := awaitSweep(t, s, "completed")
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Mixed, after.Mixed) || after.RunID != before.RunID || after.DCSweep.Result.Loop == nil || !after.DCSweep.Operations[0].SourceUnchanged {
		t.Fatal("loop touched live source", after.DCSweep)
	}
	s.mu.Lock()
	record := cloneSweepRecord(s.sweeps.operations[after.DCSweep.Selected].record)
	s.mu.Unlock()
	raw, _ := json.Marshal(record)
	if _, err := validateSweepRecord(raw); err != nil {
		t.Fatal("loop capture", err)
	}
	path := filepath.Join(t.TempDir(), "loop.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	opened := settled(t, fresh)
	if !reflect.DeepEqual(after.DCSweep.Result, opened.DCSweep.Result) {
		t.Fatal("loop persistence")
	}
	after.DCSweep.Result.Loop.Unity[0].Margin = 999
	if s.Snapshot().DCSweep.Result.Loop.Unity[0].Margin == 999 {
		t.Fatal("loop snapshot aliases")
	}
	record.Result.Loop.Unity[0].Margin += 1
	raw, _ = json.Marshal(record)
	if _, err := validateSweepRecord(raw); err == nil {
		t.Fatal("forged margin")
	}
	// Do not derive loop crossings from unvalidated huge unwrapped phase values.
	record.Result.Points[0].Values[1] = 1e300
	raw, _ = json.Marshal(record)
	if _, err := validateSweepRecord(raw); err == nil {
		t.Fatal("forged unwrapped phase")
	}
	plan, err := newSweepPlan(FeedbackLoopExampleGraph(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := computeDCSweep(ctx, plan, func(int) {}); err == nil || result != nil {
		t.Fatal("loop cancellation published result")
	}
}

func TestLoopMarginsRefuseUntrustedShapesAndHugePhase(t *testing.T) {
	for _, r := range []*SweepResult{nil, {}, {Points: []SweepPoint{{Input: 1, Values: []float64{0, 0}, AC: []ACResponse{{PhaseDefined: true}}}, {Input: 2, Values: []float64{0, 1e300}, AC: []ACResponse{{PhaseDefined: true}}}}}, {Points: []SweepPoint{{Input: 1}, {Input: 2}}}} {
		if _, err := loopMargins(r); err == nil {
			t.Fatal("unbounded/malformed margin data accepted")
		}
	}
}
