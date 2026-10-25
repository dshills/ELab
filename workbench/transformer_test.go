package workbench

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func transformerOpenFixture(reversed bool) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "6", -200, 0, 0)
	b.ground("GP", "V1.negative", -200, 180)
	b.part("T1", "transformer", "", 100, 0, 0)
	b.graph.Components[2].Power.Transformer.Reversed = reversed
	b.wire("V1.positive", "T1.primary1")
	b.wire("GP.reference", "T1.primary2")
	b.ground("GS", "T1.secondary2", 200, 180)
	return b.graph
}
func TestTransformerOpenWindingRLFluxPolarityAndEnergy(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g := transformerOpenFixture(reverse)
		k := batteryKernel(t, g)
		trace := k.trace
		sign := 1.
		if reverse {
			sign = -1
		}
		if math.Abs(k.mixed.result.Transformers[0].SecondaryVoltage-sign*.98*.5*6) > 1e-8 {
			t.Fatal("open winding startup derivative", reverse, k.mixed.result.Transformers)
		}
		for j := 1; j <= 100; j++ {
			at := float64(j) * .0001
			next, updated, err := advanceMixed(k.mixed, trace, at)
			if err != nil {
				t.Fatal(j, err)
			}
			k.mixed, trace = next, updated
			r := next.result
			u := r.Transformers[0]
			wantI := 6 * (1 - math.Exp(-at/.1))
			wantV := sign * .98 * .5 * (6 - wantI)
			if math.Abs(u.PrimaryCurrent-wantI) > 1e-7 || math.Abs(u.SecondaryVoltage-wantV) > 1e-7 || math.Abs(u.SecondaryCurrent) > 1e-12 || math.Abs(u.Energy-.05*wantI*wantI) > 1e-8 {
				t.Fatal("open RL/mutual flux", reverse, at, u, wantI, wantV)
			}
			if math.Abs(r.PowerResidual) > 1e-8 || r.KCLResidual > 1e-9 || len(r.Inductors) != 0 {
				t.Fatal("transformer conservation/private winding leak", r)
			}
			if err := validateRecordedTransformers(k.compiled, g, r); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestTransformerShortCircuitInitialSlopeAndFiniteWindingLoss(t *testing.T) {
	g := transformerOpenFixture(false)
	g.Wires = append(g.Wires, Wire{ID: "SHORT", From: "T1.secondary1", To: "T1.secondary2", Connected: true})
	k := batteryKernel(t, g)
	next, _, err := advanceMixed(k.mixed, k.trace, 1e-8)
	if err != nil {
		t.Fatal(err)
	}
	u := next.result.Transformers[0]
	slope := 6 / (.1 * (1 - .98*.98))
	if math.Abs(u.PrimaryCurrent/1e-8/slope-1) > 1e-4 || math.Abs(u.SecondaryCurrent/u.PrimaryCurrent+(.98/.5)) > 1e-4 {
		t.Fatal("mutual short-circuit derivative", u, slope)
	}
	if u.Energy < 0 || u.WindingLoss <= 0 || math.Abs(next.result.PowerResidual) > 1e-8 {
		t.Fatal("short circuit energy/loss", u)
	}
}
func TestTransformerACLoadedRatioAndMutualImpedance(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g := TransformerExampleGraph(false)
		g.Components[2].Power.Transformer.Reversed = reverse
		cfg := acSettings("VIN", "v:T1.secondary")
		cfg.Start = "10 Hz"
		cfg.Stop = "10 kHz"
		cfg.Points = 31
		r := sweepForTest(t, g, cfg)
		M := .98 * .1 * .5
		if reverse {
			M = -M
		}
		for _, p := range r.Points {
			w := 2 * math.Pi * p.Input
			zp := complex(1, w*.1)
			zs := complex(101, w*.025)
			ip := 1 / (zp + complex(w*w*M*M, 0)/zs)
			want := complex(0, w*M*100) / zs * ip
			actual := complex(p.AC[0].Real, p.AC[0].Imaginary)
			if cmplx.Abs(actual-want) > 1e-9 {
				t.Fatal("coupled AC impedance", reverse, p, want)
			}
		}
	}
}
func TestTransformerPositiveDefiniteBoundsAndResourceReservation(t *testing.T) {
	for _, k := range []float64{0, .5, .98, .9999} {
		c := transformerDefault()
		c.Coupling = k
		if err := validateTransformer(c); err != nil {
			t.Fatal(err)
		}
		g := transformerOpenFixture(false)
		g.Components[2].Power.Transformer = &c
		kernel := batteryKernel(t, g)
		n := kernel.mixed.net
		if n.minimumWindingInductance() <= 0 {
			t.Fatal("nonpositive leakage eigenvalue")
		}
		for _, currents := range [][2]float64{{1, -2}, {-1, 2}, {1, 2}, {-10, -3}} {
			x := append([]float64{}, kernel.mixed.x...)
			x[n.transformerCurrentIndex("T1", false)] = currents[0]
			x[n.transformerCurrentIndex("T1", true)] = currents[1]
			u := n.transformerResult(g.Components[2], x)
			if !finite(u.Energy) || u.Energy < 0 {
				t.Fatal("negative coupled energy", k, u)
			}
		}
	}
	for _, c := range []TransformerConfig{{Inductance: .1, Ratio: .5, Coupling: 1, PrimaryResistance: 1, SecondaryResistance: 1}, {Inductance: 100, Ratio: 10, Coupling: .98, PrimaryResistance: 1, SecondaryResistance: 1}} {
		if validateTransformer(c) == nil {
			t.Fatal("invalid/singular winding parameters accepted")
		}
	}
	usage := powerResources(transformerOpenFixture(false))
	if usage.inductors != 2 || usage.parts != 1 {
		t.Fatal("coupled windings bypass existing resource slots", usage)
	}
}
func TestTransformerSupplyRectificationSmoothingAndRegulation(t *testing.T) {
	g := TransformerExampleGraph(true)
	k := batteryKernel(t, g)
	trace := k.trace
	low, high := false, false
	for j := 1; j <= 1000; j++ {
		next, updated, err := advanceMixed(k.mixed, trace, float64(j)*.0001)
		if err != nil {
			t.Fatal("supply", j, err)
		}
		k.mixed, trace = next, updated
		r := next.result
		if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-7 {
			t.Fatal("supply conservation", j, r.KCLResidual, r.PowerResidual)
		}
		if j > 800 {
			low = low || r.Transformers[0].SecondaryVoltage < 0
			high = high || r.Transformers[0].SecondaryVoltage > 0
			if r.Capacitors[0].ID != "C1" || r.Capacitors[0].Voltage < 4 || r.Regulators[0].OutputVoltage < 3.29 || r.Regulators[0].OutputVoltage > 3.31 {
				t.Fatal("supply failed to smooth/regulate", j, r.Capacitors, r.Regulators)
			}
		}
	}
	if !low || !high {
		t.Fatal("supply did not rectify both AC polarities")
	}
}

func TestTransformerCapturedStatePersistenceUndoAndFailureAtomicity(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "coupled-windings"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	cfg := transformerDefault()
	cfg.Ratio = .75
	cfg.Reversed = true
	call(t, s, "transformer.configure", TransformerParams{"T1", cfg}, "configure")
	settled(t, s)
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	settled(t, s)
	p, _ := s.graph.Component("T1")
	if p.Power.Transformer.Reversed {
		t.Fatal("dot polarity undo")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	settled(t, s)
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	duration, step := .005, .0001
	_, v := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:T1.secondary", "i:T1.primary1", "i:T1.secondary1"}}, "capture")
	rec := s.operations[v.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: rec.metadata, Rows: rec.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("coupled capture", err)
	}
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	obj["metadata"].(map[string]any)["mixed"].(map[string]any)["transformers"].([]any)[0].(map[string]any)["magnetic_energy_j"] = 42
	raw, _ = json.Marshal(obj)
	if _, err := decodeRecording(raw); err == nil {
		t.Fatal("forged coupled magnetic energy")
	}
	path := filepath.Join(t.TempDir(), "coupled.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	before := settled(t, s)
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	after := settled(t, fresh)
	if !reflect.DeepEqual(before.Components, after.Components) || after.Document.Dirty || after.Document.RecordingCount != 1 {
		t.Fatal("coupled project roundtrip")
	}
	raw, _ = json.Marshal(cfg)
	var parameters map[string]any
	json.Unmarshal(raw, &parameters)
	delete(parameters, "reversed_secondary")
	refuse(t, s, "transformer.configure", map[string]any{"id": "T1", "parameters": parameters}, "invalid-params")
	g := transformerOpenFixture(false)
	k := batteryKernel(t, g)
	saved, _ := json.Marshal(k.mixed.result)
	budget := newIntegrationBudget()
	budget.elapsed = func() time.Duration { return MaxIntegrationWallTime + 1 }
	if _, _, err := advanceMixedBudget(k.mixed, k.trace, .001, budget); err == nil {
		t.Fatal("forced work limit ignored")
	}
	current, _ := json.Marshal(k.mixed.result)
	if string(saved) != string(current) {
		t.Fatal("failed work changed accepted flux/current")
	}
}

func TestTransformerExampleLabelsLeavePhysicalWiresContinuous(t *testing.T) {
	assertLabelMasksLeaveWiresContinuous(t, TransformerExampleGraph(false), []string{"T1", "RL"})
	assertLabelMasksLeaveWiresContinuous(t, TransformerExampleGraph(true), []string{"T1", "D1", "D2", "D3", "D4", "C1", "U1", "RSN", "CSN", "RREF", "RLOAD"})
}
