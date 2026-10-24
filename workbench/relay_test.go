package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"
)

func TestRelayCoilExponentialThresholdAndIsolatedContacts(t *testing.T) {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "5", 0, 0, 0)
	b.ground("GND", "V1.negative", 0, 200)
	b.part("K1", "relay", "100mH", 200, 0, 0)
	b.wire("V1.positive", "K1.coil1")
	b.wire("GND.reference", "K1.coil2")
	b.wire("V1.positive", "K1.common")
	b.part("RL", "resistor", "1k", 400, 0, 0)
	b.wire("K1.no", "RL.1")
	b.wire("RL.2", "GND.reference")
	c, _, d := b.graph.Compile()
	if d != nil {
		t.Fatal(d)
	}
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	last := false
	pick := 0.
	for k := 1; k <= 100; k++ {
		at := float64(k) * .0001
		if err := kernel.advance(t.Context(), at); err != nil {
			t.Fatal(at, err)
		}
		r := kernel.mixed.result.Controls[0]
		want := .05 * (1 - math.Exp(-at/.001))
		if math.Abs(r.Current-want) > 1e-8 {
			t.Fatal("coil exponential", at, r.Current, want)
		}
		if r.Memory.High && !last {
			for j := 1; j < len(kernel.trace.Samples); j++ {
				a, b := kernel.trace.Samples[j-1], kernel.trace.Samples[j]
				if a.Time == b.Time && a.frame.values["i:K1.no"] != b.frame.values["i:K1.no"] {
					pick = b.Time
					break
				}
			}
		}
		last = r.Memory.High
		rno, rnc := 1e9, 1.
		if r.Memory.High {
			rno, rnc = rnc, rno
		}
		wantPower := 100*r.Current*r.Current + rno*r.PinCurrents[3]*r.PinCurrents[3] + rnc*r.PinCurrents[4]*r.PinCurrents[4]
		if math.Abs(r.Power-wantPower) > 1e-10 {
			t.Fatal("relay dissipation", r.Power, wantPower)
		}
		if kernel.mixed.result.KCLResidual > 1e-9 || math.Abs(kernel.mixed.result.PowerResidual) > 1e-7 {
			t.Fatal("relay power balance", kernel.mixed.result.PowerResidual)
		}
	}
	want := -.001 * math.Log(1-.03/.05)
	if math.Abs(pick-want) > 2e-8 {
		t.Fatal("pickup timing", pick, want)
	}
	p, _ := b.graph.Component("K1")
	v := [5]float64{3, -2, 9, 1, 4}
	e := controlCurrents(p, v, ControlMemory{High: true})
	v[0] = 17
	v[1] = 5
	other := controlCurrents(p, v, ControlMemory{High: true})
	if e != other || e.Current[0] != 0 || e.Current[1] != 0 {
		t.Fatal("galvanic contact/coil coupling")
	}
	m := ControlMemory{High: true}
	next, _ := controlDecision(p, v, m, 0, .02)
	if !next.High {
		t.Fatal("relay hold band")
	}
	next, _ = controlDecision(p, v, m, 0, .009)
	if next.High {
		t.Fatal("dropout")
	}
}
func TestRelayProtectedDriverAndPulseRelease(t *testing.T) {
	for _, manual := range []bool{false, true} {
		g := RelayExampleGraph(manual)
		c, _, d := g.Compile()
		if d != nil {
			t.Fatal(d)
		}
		kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal("startup", manual, err)
		}
		if manual {
			net := *kernel.mixed.net
			net.controlMemory = append([]ControlMemory{}, net.controlMemory...)
			if err := net.stageButton("button:S1:click", 0); err != nil {
				t.Fatal(err)
			}
			kernel.mixed, err = net.solveContext(t.Context(), kernel.mixed.x, kernel.mixed.capV, 0, kernel.trace, 0)
			if err != nil {
				t.Fatal(err)
			}
		}
		on, off, flyback := false, false, false
		for k := 1; k <= 400; k++ {
			if err := kernel.advance(t.Context(), float64(k)*.0002); err != nil {
				t.Fatal(manual, k, err)
			}
			for _, r := range kernel.mixed.result.Controls {
				if r.ID == "K1" {
					on = on || r.Memory.High
					if on && !r.Memory.High {
						off = true
					}
					if r.Voltage < -.1 && r.Current > 1e-5 {
						flyback = true
					}
					if r.Energy < 0 || r.Power < 0 {
						t.Fatal("negative coil power/energy")
					}
				}
			}
			if kernel.mixed.result.KCLResidual > 1e-9 || math.Abs(kernel.mixed.result.PowerResidual) > 1e-7 {
				t.Fatal("protected relay balance")
			}
		}
		if !on || !off || !flyback {
			t.Fatal("relay driver did not exercise pickup/release/flyback", manual, on, off, flyback)
		}
		if err := validateRecordedControls(c, g, kernel.mixed.result); err != nil {
			t.Fatal("relay captured state", err)
		}
	}
}
func TestRelaySharedCaptureAndProjectReopen(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "relay-flasher"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := .04, .0002
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"i:K1", "v:K1", "i:D2", "i:D1"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("relay capture", err)
	}
	path := filepath.Join(t.TempDir(), "relay.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	a := settled(t, fresh)
	if a.Document.RecordingCount != 1 || a.ControlModel == nil {
		t.Fatal("relay project")
	}
}
