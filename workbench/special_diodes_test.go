package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Independently solve an external-resistance load line in current space. These
// fixed constants deliberately do not call the device evaluator or MNA solver.
func referenceSchottkyCurrent(v, r float64) float64 {
	if v <= 0 {
		return -2e-7
	}
	nv := 1.05 * 1.380649e-23 * 300.15 / 1.602176634e-19
	lo, hi := 0., v/(r+2)
	for range 90 {
		i := (lo + hi) / 2
		if i*(r+2)+nv*math.Log1p(i/2e-7) > v {
			hi = i
		} else {
			lo = i
		}
	}
	return (lo + hi) / 2
}
func specialSeriesGraph(t *testing.T, kind string, reverse bool, supply float64) Graph {
	t.Helper()
	g := siliconSeriesGraph(t, reverse)
	g.Components[0].Value = supply
	p, err := MakeComponent("D1", kind, Point{200, 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components[2] = p
	return g
}
func TestSpecialDiodeDCLoadLinesAndConservation(t *testing.T) {
	for _, kind := range []string{"zener", "schottky"} {
		for _, reverse := range []bool{false, true} {
			for _, supply := range []float64{0, .5, 3, 5.1, 6, 12} {
				g := specialSeriesGraph(t, kind, reverse, supply)
				state, _ := solveFloatingDC(t, g)
				r := state.result
				d := r.Diodes[0]
				want := referenceSiliconCurrent(supply, 1000)
				if kind == "schottky" {
					want = referenceSchottkyCurrent(supply, 1000)
				}
				if reverse {
					want = -1e-9
					if kind == "schottky" {
						want = -2e-7
					}
					if kind == "zener" && supply > 5.1 {
						want = -(supply - 5.1 + 10e-9) / 1010
					}
				}
				if supply == 0 {
					want = 0
				}
				if math.Abs(d.Current-want) > 2e-10 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 || d.Power < -1e-12 {
					t.Fatalf("%s reverse=%v supply=%g: %+v want %g; residuals %g/%g", kind, reverse, supply, d, want, r.KCLResidual, r.PowerResidual)
				}
			}
		}
	}
	sch, _ := solveFloatingDC(t, specialSeriesGraph(t, "schottky", false, 1))
	silicon, _ := solveFloatingDC(t, siliconSeriesGraph(t, false))
	if sch.result.Diodes[0].Voltage >= .3 || silicon.result.Diodes[0].Voltage <= .6 {
		t.Fatal("forward barrier distinction lost")
	}
}
func TestSpecialDiodeJacobianAndKnee(t *testing.T) {
	for _, kind := range []string{"zener", "schottky"} {
		p, _ := MakeComponent("D1", kind, Point{}, "")
		previous := -math.MaxFloat64
		for _, v := range []float64{-30, -12, -5.2, -5.10001, -5.09999, -3, -.1, 0, .1, .3, .7, 12, 30} {
			i, g := deviceDiodeCurrent(p, v)
			left, _ := deviceDiodeCurrent(p, v-1e-7)
			right, _ := deviceDiodeCurrent(p, v+1e-7)
			if !finite(i) || !finite(g) || g < 0 || i < previous || math.Abs(g-(right-left)/2e-7) > 1e-7+1e-4*g {
				t.Fatalf("%s at %g: %g/%g", kind, v, i, g)
			}
			previous = i
		}
	}
	p, _ := MakeComponent("D1", "zener", Point{}, "")
	i, _ := deviceDiodeCurrent(p, -5.1)
	left, _ := deviceDiodeCurrent(p, -5.1-1e-10)
	right, _ := deviceDiodeCurrent(p, -5.1+1e-10)
	if math.Abs(i+1e-9) > 1e-15 || math.Abs(left-right) > 2e-11 {
		t.Fatal("Zener knee is not continuous", left, i, right)
	}
}
func TestZenerLineAndLoadRegulation(t *testing.T) {
	for _, load := range []float64{100, 1000, 4700, 100000} {
		for _, input := range []float64{2, 4, 5.1, 6, 8, 10, 12} {
			g := ZenerExampleGraph()
			g.Components[0], _ = MakeComponent("V1", "dc-source", g.Components[0].Position, FormatValue(input, "V"))
			g.Components[3].Value = load
			state, c := solveFloatingDC(t, g)
			// Closed-form divider below the knee; parallel shunt law above it.
			want := (input/470 - 1e-9) / (1./470 + 1/load)
			if want > 5.1 {
				want = (input/470 + 5.1/10 - 1e-9) / (1./470 + 1/load + 1./10)
			}
			got := state.result.Nodes[c.TerminalNet["RL.1"]]
			if math.Abs(got-want) > 2e-7 {
				t.Fatalf("line/load input %g load %g: %g != %g", input, load, got, want)
			}
			if state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
				t.Fatal("regulator violates conservation")
			}
		}
	}
}

func TestSpecialDiodeExamplesIndependentTransientAndRoundTrip(t *testing.T) {
	for _, id := range []string{"zener", "schottky"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.Example.Info.ID != id || initial.Document.Dirty || initial.Mixed == nil {
				t.Fatal("example defaults", initial.Diagnostics)
			}
			refs := []float64{0, 0}
			maximum := 0.
			slope := func(at, v float64, branch int) float64 {
				input := math.Sin(20 * math.Pi * at)
				i := referenceSchottkyCurrent(input-v, 100)
				if branch == 1 {
					i = referenceSiliconCurrent(input-v, 100)
				}
				return (i - v/10000) / 10e-6
			}
			for ms := 1; ms <= 100; ms++ {
				if id == "schottky" {
					for j := range 200 {
						at := float64((ms-1)*200+j) * 5e-6
						dt := 5e-6
						for branch, v := range refs {
							a := slope(at, v, branch)
							b := slope(at+dt/2, v+dt*a/2, branch)
							c := slope(at+dt/2, v+dt*b/2, branch)
							d := slope(at+dt, v+dt*c, branch)
							refs[branch] += dt * (a + 2*b + 2*c + d) / 6
						}
					}
				}
				call(t, s, "simulation.step", StepParams{Duration: .001}, fmt.Sprintf("step-%d", ms))
				state := s.Snapshot()
				if state.StepWork.Status != "completed" || state.Mixed == nil || state.Mixed.Stale {
					t.Fatal("transient failed", state.StepWork, state.Diagnostics)
				}
				if id == "zener" {
					input := 10 + 2*math.Sin(20*math.Pi*float64(ms)*.001)
					want := (input/470 + 5.1/10 - 1e-9) / (1./470 + 1./4700 + 1./10)
					got := state.Mixed.Resistors[1].Voltage
					if math.Abs(got-want) > 2e-7 || state.Mixed.Diodes[0].Current >= 0 {
						t.Fatal("live regulator differs from closed form", got, want)
					}
				} else {
					for branch, c := range state.Mixed.Capacitors {
						err := math.Abs(c.Voltage - refs[branch])
						maximum = math.Max(maximum, err)
						if err > .001 {
							t.Fatalf("%s at %d: %g vs RK4 %g", c.ID, ms, c.Voltage, refs[branch])
						}
					}
					if ms == 25 && state.Mixed.Capacitors[0].Voltage < state.Mixed.Capacitors[1].Voltage+.2 {
						t.Fatal("Schottky comparison lacks expected benefit")
					}
					if ms == 75 && math.Abs(state.Mixed.Diodes[0].Current+2e-7) > 1e-12 {
						t.Fatal("Schottky reverse leakage absent")
					}
				}
				if state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
					t.Fatal("transient violates conservation")
				}
			}
			t.Logf("maximum RK4 error %.9g V", maximum)
			before := s.Snapshot()
			duration := .1
			_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:D1", "i:D1"}}, "verify")
			if !reflect.DeepEqual(before.Transient, s.Snapshot().Transient) {
				t.Fatal("Verify changed live state")
			}
			if id == "zener" && record.Metadata.ZenerModel == nil || id == "schottky" && (record.Metadata.SchottkyModel == nil || record.Metadata.DiodeModel == nil) {
				t.Fatal("missing model provenance")
			}
			path := filepath.Join(t.TempDir(), id+".elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
			got := settled(t, fresh)
			if !reflect.DeepEqual(got.Components, initial.Components) || got.Document.RecordingCount != 1 || got.Document.Dirty || got.Example.Info.ID != id {
				t.Fatal("project/capture round trip lost data")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err = json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			field := id + "_model"
			delete(doc, field)
			bad, _ := json.Marshal(doc)
			if _, err = decodeDocument(bad); err == nil {
				t.Fatal("missing model accepted")
			}
			if id == "zener" {
				doc[field] = map[string]any{"id": ZenerModelID, "dynamic_resistance_ohm": 1}
			} else {
				doc[field] = SiliconDiodeModel()
			}
			bad, _ = json.Marshal(doc)
			if _, err = decodeDocument(bad); err == nil {
				t.Fatal("tampered model accepted")
			}
			c, _, diagnostic := record.Metadata.Graph.Compile()
			if diagnostic != nil {
				t.Fatal(diagnostic)
			}
			corrupt := cloneMixed(record.Metadata.Mixed)
			corrupt.Diodes[0].Current += .001
			if validateRecordedDiodes(c, record.Metadata.Graph, corrupt) == nil {
				t.Fatal("unprobed diode tampering accepted")
			}
			duplicate := cloneMixed(record.Metadata.Mixed)
			duplicate.Diodes = append(duplicate.Diodes, duplicate.Diodes[0])
			if validateRecordedDiodes(c, record.Metadata.Graph, duplicate) == nil {
				t.Fatal("duplicate diode accepted")
			}
		})
	}
}

func TestSpecialDiodeValueEditsClonesAndDCSweeps(t *testing.T) {
	g := ZenerExampleGraph()
	g.Components[0], _ = MakeComponent("V1", "dc-source", g.Components[0].Position, "10")
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	for _, value := range []string{"0 V", "25 V", "NaN", "1k V"} {
		before := s.Snapshot()
		refuse(t, s, "component.set", ValueParams{ID: "D1", Value: value}, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("refused Zener value changed state")
		}
	}
	call(t, s, "component.set", ValueParams{ID: "D1", Value: "3300 mV"}, "knee")
	state := settled(t, s)
	if math.Abs(state.Components[2].Value-3.3) > 1e-12 || state.Mixed.Resistors[1].Voltage < 3.3 || state.Mixed.Resistors[1].Voltage > 3.5 {
		t.Fatal("shared Zener value edit did not affect regulation")
	}
	state.ZenerModel.DynamicResistance = 1
	if s.Snapshot().ZenerModel.DynamicResistance != 10 {
		t.Fatal("snapshot aliases Zener model")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	if settled(t, s).Components[2].Value != 5.1 {
		t.Fatal("Zener undo failed")
	}
	call(t, s, "edit.redo", struct{}{}, "redo")
	if math.Abs(settled(t, s).Components[2].Value-3.3) > 1e-12 {
		t.Fatal("Zener redo failed")
	}
	cfg := DefaultTransientSettings()
	for _, graph := range []Graph{g, SchottkyExampleGraph()} {
		source := dcCircuitSnapshot(graph, cfg)
		raw, err := json.Marshal(source.document())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = decodeDocument(raw); err != nil {
			t.Fatal("DC source provenance", err)
		}
		owned := cloneSweepMetadata(SweepMetadata{Source: source})
		if source.ZenerModel != nil {
			owned.Source.ZenerModel.DynamicResistance = 3
			if source.ZenerModel.DynamicResistance != 10 {
				t.Fatal("sweep aliases Zener model")
			}
		}
		if source.SchottkyModel != nil {
			owned.Source.SchottkyModel.SaturationCurrent = 1
			if source.SchottkyModel.SaturationCurrent != 2e-7 {
				t.Fatal("sweep aliases Schottky model")
			}
		}
	}
	result := sweepForTest(t, g, SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "12 V", Points: 25, ProbeIDs: []string{"v:RL", "i:D1"}})
	for _, p := range result.Points {
		want := (p.Input/470 - 1e-9) / (1./470 + 1./4700)
		if want > 5.1 {
			want = (p.Input/470 + 5.1/10 - 1e-9) / (1./470 + 1./4700 + .1)
		}
		if p.Input == 0 {
			want = 0
		}
		if math.Abs(p.Values[0]-want) > 2e-7 {
			t.Fatal("Zener sweep differs from line/load reference", p.Input, p.Values[0], want)
		}
	}
}

func TestZenerCapacitorClampChargeAndRelease(t *testing.T) {
	g := ZenerExampleGraph()
	g.Components[0], _ = MakeComponent("V1", "pulse-source", g.Components[0].Position, "10")
	cap, _ := MakeComponent("C1", "capacitor", Point{500, 20}, "1u")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{ID: "WC1", From: "C1.1", To: "D1.cathode", Connected: true}, Wire{ID: "WC2", From: "C1.2", To: "GND1.reference", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.input.set", InputParams{Mode: "high"}, "high")
	parallel := 1 / (1./470 + 1./4700)
	tau := parallel * 1e-6
	divider := (10/470. - 1e-9) * parallel
	kneeAt := -tau * math.Log1p(-5.1/divider)
	shuntR := 1 / (1/parallel + .1)
	reg := (10/470. + .51 - 1e-9) * shuntR
	maximum := 0.
	for ms := 1; ms <= 10; ms++ {
		call(t, s, "simulation.step", StepParams{Duration: .0001}, fmt.Sprint("charge-", ms))
		at := float64(ms) * .0001
		want := divider * -math.Expm1(-at/tau)
		if at > kneeAt {
			want = reg + (5.1-reg)*math.Exp(-(at-kneeAt)/(shuntR*1e-6))
		}
		got := s.Snapshot().Mixed.Capacitors[0].Voltage
		maximum = math.Max(maximum, math.Abs(got-want))
		if math.Abs(got-want) > .001 {
			t.Fatal("capacitor clamp charge reference", at, got, want)
		}
	}
	before := s.Snapshot().Mixed.Capacitors[0].Voltage
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "release")
	if s.Snapshot().Mixed.Capacitors[0].Voltage != before {
		t.Fatal("input command jumped capacitor voltage")
	}
	call(t, s, "simulation.step", StepParams{Duration: .001}, "discharge")
	// Initially discharge through both ballast and breakdown. Below the knee,
	// only the linear parallel load and the 1 nA reverse leakage remain.
	lowerEq := (.51 - 1e-9) * shuntR
	fallKnee := shuntR * 1e-6 * math.Log((before-lowerEq)/(5.1-lowerEq))
	want := -1e-9*parallel + (5.1+1e-9*parallel)*math.Exp(-(.001-fallKnee)/tau)
	final := s.Snapshot()
	got := final.Mixed.Capacitors[0].Voltage
	if math.Abs(got-want) > .001 || final.Mixed.Diodes[0].Current > 0 || final.Mixed.KCLResidual > 1e-9 || math.Abs(final.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("capacitor release reference", got, want, final.Diagnostics)
	}
	t.Logf("maximum charge reference error %.9g V; release %.9g V", maximum, math.Abs(got-want))
}

func TestSpecialDiodeCombinedBudgetAndMixedProvenance(t *testing.T) {
	g := specialSeriesGraph(t, "diode", false, 5)
	for k := 1; k <= 8; k++ {
		kind := []string{"zener", "schottky", "diode"}[k%3]
		id := fmt.Sprintf("DX%d", k)
		p, _ := MakeComponent(id, kind, Point{X: float64(k) * 100, Y: 300}, "")
		g.Components = append(g.Components, p)
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("WX%dA", k), From: "R1.2", To: id + ".anode", Connected: true}, Wire{ID: fmt.Sprintf("WX%dB", k), From: id + ".cathode", To: "GND.reference", Connected: true})
		_, _, d := g.Compile()
		if k < 8 && d != nil {
			t.Fatal("combined diode budget refused early", k, d)
		}
		if k == 8 && (d == nil || d.Code != "mixed-limit") {
			t.Fatal("ninth mixed diode did not refuse", d)
		}
	}
	// Put silicon first to ensure provenance traversal does not stop before the
	// other models. All three kinds share one diode result array, not identities.
	g.Components = g.Components[:len(g.Components)-1]
	g.Wires = g.Wires[:len(g.Wires)-2]
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := settled(t, s)
	if state.DiodeModel == nil || state.ZenerModel == nil || state.SchottkyModel == nil || len(state.Mixed.Diodes) != 8 {
		t.Fatal("mixed model traversal incomplete", state.Diagnostics)
	}
	_, rec := verify(t, s, VerifyParams{}, "capture")
	if len(recordingModelHashes(*rec.Metadata)) < 3 {
		t.Fatal("mixed models not hashed")
	}
	s.mu.Lock()
	doc := s.documentLocked()
	s.mu.Unlock()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodeDocument(raw); err != nil {
		t.Fatal("mixed model project rejected", err)
	}
}

func TestZenerInductorReverseClampAndRelease(t *testing.T) {
	g := RLExampleGraph()
	g.Components[0], _ = MakeComponent("V1", "dc-source", g.Components[0].Position, "0")
	initial := .1
	g.Components[2].InitialCurrent = &initial
	d, _ := MakeComponent("D1", "zener", Point{400, 0}, "5.1")
	g.Components = append(g.Components, d)
	g.Wires = append(g.Wires, Wire{ID: "WD1", From: "D1.anode", To: "L1.1", Connected: true}, Wire{ID: "WD2", From: "D1.cathode", To: "GND1.reference", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	startup := s.Snapshot()
	if startup.Mixed.Inductors[0].Current != initial || startup.Mixed.Diodes[0].Voltage >= -5.1 {
		t.Fatal("coil startup did not establish reverse clamp", startup.Diagnostics)
	}
	// In breakdown: L*dI/dt=-(I+Vz/Rz-Is)/(1/R+1/Rz).
	// At I=Vz/R+Is the Zener blocks; then L*dI/dt=-R*(I-Is).
	offset := .51 - 1e-9
	threshold := .051 + 1e-9
	kneeAt := .11 * math.Log((initial+offset)/(threshold+offset))
	maximum := 0.
	for ms := 1; ms <= 30; ms++ {
		call(t, s, "simulation.step", StepParams{Duration: .001}, fmt.Sprint("decay-", ms))
		at := float64(ms) * .001
		want := (initial+offset)*math.Exp(-at/.11) - offset
		if at > kneeAt {
			want = 1e-9 + (threshold-1e-9)*math.Exp(-(at-kneeAt)/.01)
		}
		state := s.Snapshot()
		got := state.Mixed.Inductors[0].Current
		maximum = math.Max(maximum, math.Abs(got-want))
		if state.StepWork.Status != "completed" || math.Abs(got-want) > 2e-6 || state.Mixed.Diodes[0].Power < -1e-12 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatal("coil reverse clamp reference", at, got, want, state.Diagnostics)
		}
	}
	t.Logf("maximum coil decay reference error %.9g A", maximum)
}
