package workbench

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
)

func TestMixedFlasherKernel(t *testing.T) {
	c, status, d := FlasherExampleGraph().Compile()
	if d != nil {
		t.Fatalf("%s %+v", status, d)
	}
	n := newMixedNetwork(c)
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = .005
	cfg.PlaybackSpeed = .5
	tr := &Transient{Settings: cfg, Mode: "auto", RunID: 1, ElectricalRevision: 1}
	old := make([]float64, len(n.caps))
	for i, p := range n.caps {
		if p.InitialVoltage != nil {
			old[i] = *p.InitialVoltage
		}
	}
	s, err := n.solve(nil, old, 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("initialized %d transistors and %d capacitors", len(s.result.Transistors), len(s.result.Capacitors))
	transitions := 0
	last := s.result.Transistors[0].State
	min, max := 1., 0.
	for i := 0; i < 600; i++ {
		// This checks sustained numerical oscillation, not host scheduling.
		// Keep the attempt cap; elapsed-time rejection has dedicated tests.
		budget := newIntegrationBudget()
		budget.elapsed = func() time.Duration { return 0 }
		s, err = s.advanceBudget(tr, s.result.Time+.005, budget)
		if err != nil {
			t.Fatalf("at %d: %v", i, err)
		}
		r := s.result
		if r.LEDs[0].Current < min {
			min = r.LEDs[0].Current
		}
		if r.LEDs[0].Current > max {
			max = r.LEDs[0].Current
		}
		if r.Transistors[0].State != last {
			transitions++
			last = r.Transistors[0].State

		}
	}
	if transitions < 5 || min > 1e-5 || max < .002 {
		t.Fatalf("no sustained oscillation: transitions %d I=%g..%g", transitions, min, max)
	}
}

func TestMixedCapacitanceSetsFlasherPeriod(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	simulate := func(scale float64) []float64 {
		g := FlasherExampleGraph()
		for i, p := range g.Components {
			if p.Kind == "capacitor" {
				g.Components[i].Value *= scale
			}
		}
		c, _, d := g.Compile()
		if d != nil {
			t.Fatal(d)
		}
		n := newMixedNetwork(c)
		cfg := DefaultTransientSettings()
		// A finer output grid resolves the switching interval under race instrumentation
		// while keeping the same duration, solver tolerances and per-output work limits.
		cfg.StepSeconds = .0025 * scale
		tr := &Transient{Settings: cfg, Mode: "auto"}
		old := []float64{.1, 0}
		s, err := n.solve(nil, old, 0, tr, 0)
		if err != nil {
			t.Fatal(err)
		}
		edges := []float64{}
		high := s.result.LEDs[0].Current > .001
		for i := 0; i < 1200; i++ {
			s, err = s.advance(tr, float64(i+1)*cfg.StepSeconds)
			if err != nil {
				t.Fatal(err)
			}
			r := s.result
			next := r.LEDs[0].Current > .001
			if next && !high && r.Time > .4*scale {
				edges = append(edges, r.Time)
			}
			high = next
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
				t.Fatalf("conservation KCL %g power %g", r.KCLResidual, r.PowerResidual)
			}
		}
		if len(edges) < 4 {
			t.Fatalf("too few periodic rising edges: %v", edges)
		}
		return edges
	}
	a, b := simulate(1), simulate(2)
	pa, pb := a[2]-a[1], b[2]-b[1]
	t.Logf("periods: 10uF %.6g s; 20uF %.6g s", pa, pb)
	if math.Abs(pb/pa-2) > .03 {
		t.Fatal("doubling capacitance did not double the physical period")
	}
}
func mixedBaseCapGraph() Graph {
	g := NPNPulseExampleGraph()
	cap, _ := MakeComponent("Timing", "capacitor", Point{400, 60}, "1uF")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{"WC1", "Timing.1", "Q1.base", true}, Wire{"WC2", "Timing.2", "GND1.reference", true})
	return g
}
func TestMixedPulseCapacitorContinuityAndLowVoltageRCReference(t *testing.T) {
	g := mixedBaseCapGraph()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = .001
	cfg.HighSeconds = .02
	cfg.LowSeconds = .02
	tr := &Transient{Engine: MixedEngineID, Settings: cfg, Mode: "auto", Samples: []TraceSample{}, Events: []InputEvent{{0, "restart"}}}
	s, err := n.solve(nil, []float64{0}, 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	tr.Sample = mixedSample(c, s.result, 5)
	for i := 0; i < 20; i++ {
		s, tr, err = advanceMixed(s, tr, float64(i+1)*.001)
		if err != nil {
			t.Fatal(err)
		}
	}
	if tr.Sample.Input != 0 || tr.Sample.State != "saturation" || tr.Sample.CapacitorVoltage < .65 {
		t.Fatal("capacitor did not retain base voltage at falling input edge")
	}
	edgeV := []float64{}
	for _, p := range tr.Samples {
		if math.Abs(p.Time-.02) < 1e-12 {
			edgeV = append(edgeV, p.CapacitorVoltage)
		}
	}
	if len(edgeV) < 2 || math.Abs(edgeV[0]-edgeV[1]) > 1e-8 {
		t.Fatal("voltage jumped at pulse edge")
	}
	for i := 20; i < 39; i++ {
		s, tr, err = advanceMixed(s, tr, float64(i+1)*.001)
		if err != nil {
			t.Fatal(err)
		}
	}
	if tr.Sample.Current > .00001 || tr.Sample.CapacitorVoltage > .2 {
		t.Fatal("base capacitor did not discharge and turn LED off")
	}
	// Below turn-on, transistor loading is negligible: compare to ideal RC.
	for i, p := range g.Components {
		if p.ID == "VIN" {
			p.Kind = "dc-source"
			p.Model = "ideal-dc.v1"
			p.Name = "DC source"
			p.Value = .3
			g.Components[i] = p
		}
	}
	c, _, _ = g.Compile()
	n = newMixedNetwork(c)
	tr = &Transient{Settings: cfg, Mode: "auto"}
	s, err = n.solve(nil, []float64{0}, 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		s, err = s.advance(tr, float64(i+1)*.0001)
		if err != nil {
			t.Fatal(err)
		}
	}
	expected := .3 * (1 - math.Exp(-1))
	if math.Abs(s.capV[0]-expected) > .0004 {
		t.Fatalf("low-voltage RC expected %.9g got %.9g", expected, s.capV[0])
	}
}
func pausedFlasher(t *testing.T) *Service {
	t.Helper()
	s := New()
	call(t, s, "example.flasher", ReplaceParams{}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	return s
}
func TestMixedServiceHistoryPersistenceAndRuntime(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := pausedFlasher(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.3}, "step")
	before := s.Snapshot()
	if before.Mixed == nil || before.Mixed.Time != before.Transient.Sample.Time || before.NPN != nil {
		t.Fatal("missing coupled result")
	}
	call(t, s, "component.move", MoveParams{ID: "C1", Position: Point{-20, 30}}, "move")
	if s.Snapshot().RunID != before.RunID || !reflect.DeepEqual(s.Snapshot().Mixed, before.Mixed) {
		t.Fatal("geometry reset capacitor memory")
	}
	call(t, s, "simulation.input.set", InputParams{"low"}, "queue-off")
	if s.Snapshot().Transient.Sample != before.Transient.Sample {
		t.Fatal("queued override changed paused state")
	}
	call(t, s, "simulation.step", StepParams{.005}, "power-off")
	if s.Snapshot().Transient.Mode != "low" || s.Snapshot().RunID != before.RunID {
		t.Fatal("override restarted run")
	}
	call(t, s, "capacitor.initial.set", ValueParams{ID: "C1", Value: "-0.2 V"}, "initial")
	if !s.Snapshot().Mixed.Stale || !s.Snapshot().Transient.Stale {
		t.Fatal("initial-condition edit not stale")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	call(t, s, "simulation.restart", struct{}{}, "reset-after-undo")
	path := filepath.Join(t.TempDir(), "Flasher.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := fresh.Snapshot()
	if loaded.Mixed == nil || loaded.Mixed.Time != 0 || loaded.Transient.Mode != "auto" || loaded.Document.Dirty {
		t.Fatal("load did not initialize Auto run")
	}
	if *loaded.Components[9].InitialVoltage != .1 {
		t.Fatal("lost signed startup override")
	}
	copy := s.Snapshot()
	copy.Mixed.Nodes["bad"] = 99
	*copy.Components[9].InitialVoltage = 99
	if s.Snapshot().Mixed.Nodes["bad"] == 99 || *s.Snapshot().Components[9].InitialVoltage == 99 {
		t.Fatal("snapshot aliases mixed state or initial conditions")
	}
	// A live sample advances time without a command, then pause freezes it.
	call(t, fresh, "simulation.pause", struct{}{}, "freeze")
	frozen := fresh.Snapshot()
	call(t, fresh, "simulation.resume", struct{}{}, "continue")
	time.Sleep(120 * time.Millisecond)
	if fresh.Snapshot().Mixed.Time <= frozen.Mixed.Time || fresh.Snapshot().RunID != frozen.RunID {
		t.Fatal("live coupled clock did not continue")
	}
	call(t, fresh, "simulation.pause", struct{}{}, "freeze-again")
	frozen = fresh.Snapshot()
	time.Sleep(80 * time.Millisecond)
	if fresh.Snapshot().Mixed.Time != frozen.Mixed.Time {
		t.Fatal("paused coupled clock advanced")
	}
}

func TestMixedFailedStepIsAtomicAndInvalidInitialConditionsAreRejected(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	call(t, s, "simulation.input.set", InputParams{"low"}, "queue")
	before := s.Snapshot()
	rev := before.Revision
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "tiny", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":1e-12}`), ExpectedRevision: &rev, Key: "tiny"}, "test")
	if response.Error == nil || response.Error.Code != "solve-failed" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("failed coupled step committed partial time, state or queued input")
	}
	for _, value := range []string{"13 V", "-13 V", "NaN"} {
		raw, _ := json.Marshal(ValueParams{ID: "C1", Value: value})
		response = s.Dispatch(context.Background(), Request{Version: 1, ID: value, Command: "capacitor.initial.set", Params: raw, ExpectedRevision: &rev, Key: value}, "test")
		if response.Error == nil || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid initial voltage altered state")
		}
	}
	path := filepath.Join(t.TempDir(), "Conditions.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-conditions")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	_ = json.Unmarshal(data, &doc)
	for _, test := range []struct {
		id    string
		value float64
	}{{"C1", 13}, {"RB1", .1}} {
		bad := doc
		bad.Graph = doc.Graph.Clone()
		for i, p := range bad.Graph.Components {
			if p.ID == test.id {
				v := test.value
				bad.Graph.Components[i].InitialVoltage = &v
			}
		}
		raw, _ := json.Marshal(bad)
		if _, err := decodeDocument(raw); err == nil {
			t.Fatal("loaded invalid capacitor metadata")
		}
	}
}
func TestMixedConstraintDiagnosticsAndRCInitialPolarity(t *testing.T) {
	g := mixedBaseCapGraph()
	extra, _ := MakeComponent("Parallel", "capacitor", Point{500, 60}, "1uF")
	g.Components = append(g.Components, extra)
	g.Wires = append(g.Wires, Wire{"P1", "Parallel.1", "Timing.1", true}, Wire{"P2", "Parallel.2", "Timing.2", true})
	_, status, d := g.Compile()
	if d == nil || status != "unsupported" || d.Code != "ideal-constraint-loop" {
		t.Fatal("redundant capacitor constraints not diagnosed")
	}
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "capacitor.initial.set", ValueParams{ID: "C1", Value: "-1 V"}, "negative-initial")
	call(t, s, "simulation.restart", struct{}{}, "negative-start")
	if s.Snapshot().Transient.Sample.Voltage != -1 {
		t.Fatal("RC signed capacitor initialization lost")
	}
	// Reverse capacitor terminals; pin 1 minus pin 2 stays -1 V, while the
	// measured output-to-ground voltage becomes +1 V.
	s.mu.Lock()
	for i := range s.graph.Wires {
		if s.graph.Wires[i].To == "C1.1" {
			s.graph.Wires[i].To = "C1.2"
		} else if s.graph.Wires[i].From == "C1.2" {
			s.graph.Wires[i].From = "C1.1"
		}
	}
	s.mu.Unlock()
	call(t, s, "simulation.restart", struct{}{}, "reversed-start")
	if s.Snapshot().Transient.Sample.Voltage != 1 {
		t.Fatal("RC initial condition ignored named pin polarity")
	}
}
