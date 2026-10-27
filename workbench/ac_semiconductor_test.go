package workbench

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"testing"
)

func acTransistorFixture(kind string) Graph {
	b := newExampleBuilder()
	negative := kind == "pnp" || kind == "pmos"
	bias := 2.5
	if IsMOSFETKind(kind) {
		bias = 2
	}
	if negative {
		bias = -bias
	}
	b.part("VIN", "sine-source", ".1", -400, 0, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: .1, Offset: bias, Frequency: 10}
	b.ground("GI", "VIN.negative", -400, 100)
	b.part("VCC", "dc-source", "5", 400, 0, 0)
	rail := "VCC.positive"
	if negative {
		rail = "VCC.negative"
		b.ground("GS", "VCC.positive", 400, 100)
	} else {
		b.ground("GS", "VCC.negative", 400, 100)
	}
	b.part("Q", kind, "", 0, 0, 0)
	b.part("RL", "resistor", "1k", 0, 200, 0)
	if IsBipolarKind(kind) {
		b.ground("GL", "RL.2", 0, 300)
		b.part("RB", "resistor", "1k", -200, 0, 0)
		b.wire("VIN.positive", "RB.1")
		b.wire("RB.2", "Q.base")
		b.wire("Q.collector", rail)
		b.wire("Q.emitter", "RL.1")
	} else {
		b.ground("GL", "Q.source", 0, 100)
		b.wire("Q.gate", "VIN.positive")
		b.wire("RL.1", rail)
		b.wire("RL.2", "Q.drain")
		for i, p := range b.graph.Components {
			if p.ID == "RL" {
				b.graph.Components[i].Value = 100
			}
		}
	}
	b.part("CL", "capacitor", "1u", 200, 200, 0)
	output := "Q.emitter"
	if IsMOSFETKind(kind) {
		output = "Q.drain"
	}
	b.wire("CL.1", output)
	b.ground("GC", "CL.2", 200, 300)
	return b.graph
}
func TestACBipolarFollowersAndMOSAgainstScalarSmallSignal(t *testing.T) {
	for _, kind := range []string{"npn", "pnp", "nmos", "pmos"} {
		t.Run(kind, func(t *testing.T) {
			g := acTransistorFixture(kind)
			probe := "v:Q.emitter"
			if IsMOSFETKind(kind) {
				probe = "v:Q.drain"
			}
			r := sweepForTest(t, g, acSettings("VIN", probe))
			for _, p := range r.Points {
				var want complex128
				if IsBipolarKind(kind) {
					emitter, _ := followerReference(2.5)
					forward := emitter / 1000 / 1.01
					gm := forward / bipolarReferenceVT
					gpi := gm / 100
					want = complex(gm+gpi, 0) / (complex(gm+gpi, 0) + complex(1./1000, 2*math.Pi*p.Input*1e-6)*complex(1+1000*gpi, 0))
				} else {
					drain := (5. - 100*.0125) / (1 + 100*.0125*.02)
					gm := .1 * .5 * (1 + .02*drain)
					gd := .0125 * .02
					want = complex(-gm, 0) / complex(gd+.01, 2*math.Pi*p.Input*1e-6)
				}
				z := p.AC[0]
				if cmplx.Abs(complex(z.Real, z.Imaginary)-want) > 1e-7 {
					t.Fatal(kind, "small-signal reference", p.Input, z, want)
				}
			}
		})
	}
}
func TestACDiodeDynamicResistanceAndZenerBreakdown(t *testing.T) {
	for _, kind := range []string{"diode", "schottky", "zener", "led"} {
		t.Run(kind, func(t *testing.T) {
			g := ExampleGraph()
			if kind != "led" {
				p, err := MakeComponent("D1", kind, Point{200, 0}, "")
				if err != nil {
					t.Fatal(err)
				}
				replaced := false
				for i, part := range g.Components {
					if part.ID == "D1" {
						g.Components[i] = p
						replaced = true
						break
					}
				}
				if !replaced {
					t.Fatal("diode fixture lacks D1")
				}
			}
			r := sweepForTest(t, g, acSettings("V1", "v:D1"))
			nv, is, rs, current := 2*bipolarReferenceVT, 1e-20, 1., 0.
			switch kind {
			case "led":
				lo, hi := 0., .005
				for range 100 {
					mid := (lo + hi) / 2
					if nv*math.Log1p(mid/is)+(1000+rs)*mid > 5 {
						hi = mid
					} else {
						lo = mid
					}
				}
				current = (lo + hi) / 2
			case "schottky":
				nv, is, rs = 1.05*bipolarReferenceVT, 2e-7, 2
				current = referenceSchottkyCurrent(5, 1000)
			default:
				nv, is, rs = 1.7*bipolarReferenceVT, 1e-9, 1
				current = referenceSiliconCurrent(5, 1000)
			}
			rd := rs + nv/(current+is)
			want := rd / (1000 + rd)
			for _, p := range r.Points {
				if math.Abs(p.AC[0].Real-want) > 1e-9 || math.Abs(p.AC[0].Imaginary) > 1e-12 {
					t.Fatal("diode tangent", kind, p, want)
				}
			}
		})
	}
	g := specialSeriesGraph(t, "zener", true, 9)
	r := sweepForTest(t, g, acSettings("V1", "v:D1"))
	for _, p := range r.Points {
		if math.Abs(p.AC[0].Real+10./1010) > 1e-8 {
			t.Fatal("reverse Zener tangent", p)
		}
	}
}
func TestACDiscreteAmplifierCaptureAndBiasDoesNotMutateLive(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "example.load", ExampleLoadParams{ID: "discrete-class-ab"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause-again")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	cfg := acSettings("VIN", "v:RL")
	cfg.Start = ".01 Hz"
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	awaitSweep(t, s, "completed")
	after := s.Snapshot()
	r := after.DCSweep.Result
	// Central DC perturbations independently check the solved AC Jacobian's low-frequency gain.
	positive, c := solveFloatingDC(t, classABDC(1e-5))
	negative, _ := solveFloatingDC(t, classABDC(-1e-5))
	want := (positive.result.Nodes[c.TerminalNet["RL.1"]] - negative.result.Nodes[c.TerminalNet["RL.1"]]) / 2e-5
	if r == nil || len(r.Points) != 201 || len(r.Points[0].AC) != 1 {
		t.Fatal("amplifier AC capture shape")
	}
	if math.Abs(r.Points[0].AC[0].Real-want) > 1e-5 {
		t.Fatal("amplifier bias/gain", r.Points[0], want)
	}
	if !reflect.DeepEqual(before.Mixed, after.Mixed) || before.RunID != after.RunID {
		t.Fatal("AC changed accepted transient state")
	}
	s.mu.Lock()
	op := s.sweeps.operations[after.DCSweep.Selected]
	var record *SweepRecord
	if op != nil {
		record = cloneSweepRecord(op.record)
	}
	s.mu.Unlock()
	if record == nil {
		t.Fatal("selected AC operation missing its completed record")
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSweepRecord(data); err != nil {
		t.Fatal("semiconductor capture", err)
	}
	path := filepath.Join(t.TempDir(), "amplifier-ac.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	if !reflect.DeepEqual(settled(t, fresh).DCSweep.Result, r) {
		t.Fatal("AC roundtrip changed response")
	}
	// A DC bias opens the authored capacitor, allowing the larger-circuit cold fallback.
	g := DiscreteClassABExampleGraph()
	for i, p := range g.Components {
		if p.ID == "VIN" {
			g.Components[i].Sine.Offset = -.1
		}
	}
	sweepForTest(t, g, cfg)
}
