package workbench

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func classABDC(input float64) Graph {
	g := DiscreteClassABExampleGraph()
	parts := g.Components[:0]
	for _, p := range g.Components {
		if p.ID == "C1" {
			continue
		}
		if p.ID == "VIN" {
			p.Sine.Amplitude, p.Sine.Offset, p.Value = 0, input, 0
		}
		parts = append(parts, p)
	}
	g.Components = parts
	wires := g.Wires[:0]
	for _, w := range g.Wires {
		if w.From != "C1.1" && w.From != "C1.2" && w.To != "C1.1" && w.To != "C1.2" {
			wires = append(wires, w)
		}
	}
	g.Wires = wires
	return g
}
func classABTransistor(t *testing.T, r *MixedResult, id string) MixedTransistor {
	t.Helper()
	for _, q := range r.Transistors {
		if q.TransistorID == id {
			return q
		}
	}
	t.Fatal("missing transistor", id)
	return MixedTransistor{}
}
func classABConservation(t *testing.T, r *MixedResult) {
	t.Helper()
	if r == nil || r.Stale || math.IsNaN(r.KCLResidual) || r.KCLResidual > 1e-9 || math.IsNaN(r.PowerResidual) || math.Abs(r.PowerResidual) > 1e-8 {
		t.Fatal("invalid amplifier balance", r)
	}
	for _, q := range r.Transistors {
		if math.IsNaN(q.CollectorCurrent) || math.IsInf(q.CollectorCurrent, 0) || math.Abs(q.EmitterCurrent-q.BaseCurrent-q.CollectorCurrent) > 1e-12 || q.TransistorPower < -1e-10 {
			t.Fatal("invalid transistor measurement", q)
		}
	}
}

// Independent scalar bias loop: two Shockley reference diodes (n=1.7,
// IS=1 nA, RS=1 ohm), Q8's Ebers-Moll base/emitter current and RT/RREF.
// This does not use the network stamps, Jacobian, Newton solver or its result.
func classABTailReference() float64 {
	lo, hi := 0., .002
	for range 100 {
		f := (lo + hi) / 2
		ib, ie := f/100-1e-14, 1.01*f+1e-14
		vb := bipolarReferenceVT*math.Log1p(f/1e-14) + 680*ie
		diodeCurrent := (9-vb)/2200 - ib
		diodeVoltage := 1.7*bipolarReferenceVT*math.Log1p(diodeCurrent/1e-9) + diodeCurrent
		if vb > 2*diodeVoltage {
			hi = f
		} else {
			lo = f
		}
	}
	return (lo+hi)/2 + 2e-14
}
func TestDiscreteClassABOperatingPointAndHandoff(t *testing.T) {
	var zero, positive, negative float64
	for _, input := range []float64{0, .01, -.01, .1, -.1} {
		t.Run(fmt.Sprint(input), func(t *testing.T) {
			state, c := solveFloatingDC(t, classABDC(input))
			r := state.result
			classABConservation(t, r)
			if len(r.Transistors) != 8 || len(r.OpAmps) != 0 {
				t.Fatal("amplifier must be entirely discrete")
			}
			output := r.Nodes[c.TerminalNet["RL.1"]]
			if math.Abs(output-11*input) > .05 {
				t.Fatal("closed-loop gain/offset", input, output)
			}
			if math.Abs(classABTransistor(t, r, "Q8").CollectorCurrent-classABTailReference()) > 1e-9 {
				t.Fatal("tail differs from scalar reference")
			}
			q1, q3, q4 := classABTransistor(t, r, "Q1"), classABTransistor(t, r, "Q3"), classABTransistor(t, r, "Q4")
			if math.Abs(q3.CollectorCurrent-q4.CollectorCurrent) > 1e-10 || math.Abs(-q3.CollectorCurrent-q1.CollectorCurrent*100/102) > 1e-10 {
				t.Fatal("current mirror/base-current loading", q1, q3, q4)
			}
			upper, lower := classABTransistor(t, r, "Q6"), classABTransistor(t, r, "Q7")
			switch input {
			case 0:
				zero = output
				if upper.CollectorCurrent < 1e-5 || lower.CollectorCurrent > -1e-5 {
					t.Fatal("class AB idle conduction missing")
				}
			case .1:
				positive = output
				if upper.CollectorCurrent < .01 || math.Abs(lower.CollectorCurrent) > 1e-7 {
					t.Fatal("positive handoff", upper, lower)
				}
			case -.1:
				negative = output
				if lower.CollectorCurrent > -.01 || math.Abs(upper.CollectorCurrent) > 1e-7 {
					t.Fatal("negative handoff", upper, lower)
				}
			}
			t.Logf("VIN=%g V output=%.9g V tail=%.9g A KCL=%.3g A power=%.3g W", input, output, classABTransistor(t, r, "Q8").CollectorCurrent, r.KCLResidual, r.PowerResidual)
		})
	}
	gain := (positive - negative) / .2
	if math.Abs(zero) > .03 || math.Abs(gain-11) > .12 || math.Abs((positive+negative)/2-zero) > .002 {
		t.Fatal("gain, DC offset or symmetry", gain, zero, positive, negative)
	}
}

func bipolarBoundaryGraph(count int, polarity string) Graph {
	b := newExampleBuilder()
	b.part("VP", "dc-source", "5", 0, 0, 0)
	b.part("VN", "dc-source", "5", 0, 200, 0)
	b.ground("G", "VP.negative", 0, 100)
	b.wire("VN.positive", "G.reference")
	for i := 0; i < count; i++ {
		kind := polarity
		if kind == "mixed" {
			kind = "npn"
			if i%2 != 0 {
				kind = "pnp"
			}
		}
		id, load := fmt.Sprintf("Q%d", i+1), fmt.Sprintf("R%d", i+1)
		b.part(id, kind, "", float64(i*100), 400, 0)
		b.part(load, "resistor", "1k", float64(i*100), 600, 0)
		b.wire(id+".base", "G.reference")
		b.wire(id+".emitter", load+".1")
		if kind == "npn" {
			b.wire(id+".collector", "VP.positive")
			b.wire(load+".2", "VN.negative")
		} else {
			b.wire(id+".collector", "VN.negative")
			b.wire(load+".2", "VP.positive")
		}
	}
	return b.graph
}
func TestSixteenBipolarDevicesAgainstScalarLoadLine(t *testing.T) {
	lo, hi := 0., .005
	for range 100 {
		f := (lo + hi) / 2
		vbe := bipolarReferenceVT * math.Log1p(f/1e-14)
		if 1000*(1.01*f+1e-14)+vbe > 5 {
			hi = f
		} else {
			lo = f
		}
	}
	expected := (lo+hi)/2 + 2e-14
	for _, kind := range []string{"npn", "pnp", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			state, _ := solveFloatingDC(t, bipolarBoundaryGraph(16, kind))
			classABConservation(t, state.result)
			if len(state.result.Transistors) != 16 {
				t.Fatal("capacity dropped a transistor")
			}
			for _, q := range state.result.Transistors {
				sign := 1.
				if q.Model.ID == PNPModelID {
					sign = -1
				}
				if math.Abs(q.CollectorCurrent-sign*expected) > 1e-9 {
					t.Fatal("boundary load line", q, expected)
				}
			}
		})
	}
	limits := Limits()
	if limits.Transistors != 16 || limits.PNP != 16 || limits.CombinedBipolar != 16 {
		t.Fatal("shared capacity discovery differs", limits)
	}
	s := newGraphService(SolveLED, bipolarBoundaryGraph(17, "mixed"))
	defer s.Close()
	initial := s.Snapshot()
	if initial.Status != "unsupported" || len(initial.Diagnostics) != 1 || initial.Diagnostics[0].Code != "mixed-limit" {
		t.Fatal("excess capacity not refused", initial.Diagnostics)
	}
	call(t, s, "component.move", MoveParams{ID: "Q17", Position: Point{0, 2000}}, "move")
	path := filepath.Join(t.TempDir(), "excess.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	if !reflect.DeepEqual(s.Snapshot().Components, fresh.Snapshot().Components) || fresh.Snapshot().Status != "unsupported" {
		t.Fatal("excess drawing not retained")
	}
}
func TestBipolarSourceSteppingIsPrivateAndBounded(t *testing.T) {
	g := classABDC(-.1)
	original := g.Clone()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	trace := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	state, err := n.bipolarSourceStep(context.Background(), nil, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	classABConservation(t, state.result)
	if n.startupSourceScale != nil || !reflect.DeepEqual(g, original) || n.sourceVoltage(c.Source, trace, 0) != sourceVoltageAt(c.Source, trace, 0, true) {
		t.Fatal("source stepping leaked trial state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := n.bipolarSourceStep(ctx, nil, 0, trace, 0); err != context.Canceled {
		t.Fatal("source stepping ignored cancellation", err)
	}
	reactive, _, d := DiscreteClassABExampleGraph().Compile()
	if d != nil {
		t.Fatal(d)
	}
	if newMixedNetwork(reactive).canStepBipolarSources() {
		t.Fatal("fallback must not replace reactive initial conditions")
	}
}
func TestDiscreteClassABLiveCaptureAndPersistence(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "discrete-class-ab"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	initial := settled(t, s)
	classABConservation(t, initial.Mixed)
	if initial.Document.Dirty || initial.Example.Modified || initial.Selected != "Q6" || len(initial.Scope.Selected) != 6 {
		t.Fatal("guided example startup")
	}
	duration := .2
	probes := []string{"v:VIN.positive", "v:RL", "i:Q6.collector", "i:Q7.collector", "i:Q8.collector", "v:C1"}
	var coarse []VerificationRow
	for _, step := range []float64{.0005, .00025} {
		_, record := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, fmt.Sprint(step))
		if record.Metadata.NPNModel == nil || record.Metadata.PNPModel == nil || len(record.Metadata.Mixed.Transistors) != 8 {
			t.Fatal("capture lost device provenance")
		}
		var rows []VerificationRow
		for offset := 0; ; {
			page := call(t, s, "operation.samples", VerificationPageParams{ID: record.Operation.ID, Offset: offset, Limit: 200}, fmt.Sprintf("rows-%g-%d", step, offset)).Result.(VerificationPage)
			rows = append(rows, page.Rows...)
			if page.NextOffset == page.Total {
				break
			}
			offset = page.NextOffset
		}
		if len(rows) < 400 {
			t.Fatal("capture incomplete", len(rows))
		}
		pos, neg := false, false
		for _, row := range rows {
			if row.Time < .05 {
				continue
			}
			if math.Abs(row.Values[1]-11*row.Values[0]) > .055 || math.Abs(row.Values[4]-classABTailReference()) > 1e-9 {
				t.Fatal("transient gain/tail", row)
			}
			if row.Values[1] > .9 && row.Values[2] > .009 && math.Abs(row.Values[3]) < 1e-6 {
				pos = true
			}
			if row.Values[1] < -.9 && row.Values[3] < -.009 && math.Abs(row.Values[2]) < 1e-6 {
				neg = true
			}
		}
		if !pos || !neg {
			t.Fatal("capture omitted output handoff")
		}
		if coarse == nil {
			coarse = rows
		} else {
			for _, a := range coarse {
				if a.Time < .05 {
					continue
				}
				best := math.Inf(1)
				delta := 0.
				for _, b := range rows {
					if d := math.Abs(a.Time - b.Time); d < best {
						best = d
						delta = math.Abs(a.Values[1] - b.Values[1])
					}
				}
				if best < 1e-10 && delta > .002 {
					t.Fatal("step refinement differs", a.Time, delta)
				}
			}
		}
		data, err := encodeVerificationExport(s.operations[record.Operation.ID].record, "json")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRecording(data); err != nil {
			t.Fatal("capture validation", err)
		}
	}
	if !reflect.DeepEqual(initial.Mixed, s.Snapshot().Mixed) {
		t.Fatal("private verification changed live circuit")
	}
	call(t, s, "simulation.step", StepParams{Duration: .2}, "step")
	live := settled(t, s)
	classABConservation(t, live.Mixed)
	if math.Abs(live.Mixed.Time-.2) > 1e-10 || live.Transient.Stale {
		t.Fatal("live simulation did not advance")
	}
	path := filepath.Join(t.TempDir(), "class-ab.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	classABConservation(t, loaded.Mixed)
	if !reflect.DeepEqual(initial.Components, loaded.Components) || loaded.Document.RecordingCount != 2 || loaded.Document.Dirty || loaded.Transient.Sample.Time != 0 {
		t.Fatal("roundtrip lost graph/recording/initial state")
	}
}

func TestBipolarCapacityIncludesOptocouplerPhototransistor(t *testing.T) {
	for _, count := range []int{15, 16} {
		g := bipolarBoundaryGraph(count, "mixed")
		part, err := MakeComponent("OC1", "optocoupler", Point{0, 1000}, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, part)
		for _, pin := range part.Terminals {
			g.Wires = append(g.Wires, Wire{ID: "WO" + pin, From: "OC1." + pin, To: "G.reference", Connected: true})
		}
		_, _, diagnostic := g.Compile()
		if count == 15 {
			if diagnostic != nil {
				t.Fatal("16 including phototransistor refused", diagnostic)
			}
		} else if diagnostic == nil || diagnostic.Code != "mixed-limit" {
			t.Fatal("17 including phototransistor accepted", diagnostic)
		}
	}
}
