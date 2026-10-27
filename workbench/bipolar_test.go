package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const bipolarReferenceVT = 1.380649e-23 * 300.15 / 1.602176634e-19

// Scalar references use the published transport-form equations and fixed
// parameters, independently of the network residual/Jacobian and solver.
func bipolarReference(vbe, vce float64) (ib, ic float64) {
	forward := 1e-14 * math.Expm1(vbe/bipolarReferenceVT)
	reverse := 1e-14 * math.Expm1((vbe-vce)/bipolarReferenceVT)
	return forward/100 + reverse, forward - 2*reverse
}

func bipolarDCInput(g Graph, input float64) Graph {
	g = g.Clone()
	for i, part := range g.Components {
		if part.ID == "VIN" {
			g.Components[i], _ = MakeComponent(part.ID, "dc-source", part.Position, FormatValue(input, "V"))
			g.Components[i].Rotation = part.Rotation
		}
	}
	return g
}

func bipolarSwitchReference(drive float64) (base, collector, ib, ic float64) {
	load := func(vc float64) float64 {
		lo, hi := 0., .005
		for range 90 {
			i := (lo + hi) / 2
			drop := 2*bipolarReferenceVT*math.Log1p(i/1e-20) + 1001*i
			if drop > 5-vc {
				hi = i
			} else {
				lo = i
			}
		}
		return (lo + hi) / 2
	}
	baseAt := func(vc float64) float64 {
		lo, hi := 0., drive
		for range 90 {
			vb := (lo + hi) / 2
			b, _ := bipolarReference(vb, vc)
			if b+vb/100000 > (drive-vb)/10000 {
				hi = vb
			} else {
				lo = vb
			}
		}
		return (lo + hi) / 2
	}
	lo, hi := 0., 5.
	for range 90 {
		vc := (lo + hi) / 2
		_, current := bipolarReference(baseAt(vc), vc)
		if current > load(vc) {
			hi = vc
		} else {
			lo = vc
		}
	}
	collector = (lo + hi) / 2
	base = baseAt(collector)
	ib, ic = bipolarReference(base, collector)
	return
}

func followerReference(input float64) (emitter, baseCurrent float64) {
	lo, hi := 0., input/1000
	for range 90 {
		ie := (lo + hi) / 2
		vbe := bipolarReferenceVT * math.Log1p(ie/(1.01e-14))
		if ie*1000+vbe+1000*ie/101 > input {
			hi = ie
		} else {
			lo = ie
		}
	}
	ie := (lo + hi) / 2
	return 1000 * ie, ie / 101
}

func TestBipolarThreeTerminalStampPolarityAndJacobian(t *testing.T) {
	for _, kind := range []string{"npn", "pnp"} {
		sign := bipolarPolarity(kind)
		for _, bias := range [][3]float64{{.7, 4, 0}, {.7, .1, 0}, {0, 5, 0}, {.1, .7, 0}, {2.7, 6, 2}} {
			ib, ic, _, _, _, _ := bipolarLinear(kind, sign*(bias[0]-bias[2]), sign*(bias[1]-bias[2]))
			wantB, wantC := bipolarReference(bias[0]-bias[2], bias[1]-bias[2])
			if math.Abs(ib-sign*wantB) > 1e-12 || math.Abs(ic-sign*wantC) > 1e-12 {
				t.Fatal("polarity/equation mismatch", kind, bias, ib, ic, wantB, wantC)
			}
		}
		for _, ground := range []int{-1, 0, 1, 2} {
			p, _ := MakeComponent("Q", kind, Point{}, "")
			n := &mixedNetwork{c: Compiled{TerminalNet: map[string]string{}}, index: map[string]int{}}
			x := []float64{}
			physical := []float64{sign * 2.7, sign * 6, sign * 2}
			for i, pin := range p.Terminals {
				n.c.TerminalNet["Q."+pin] = pin
				if i == ground {
					n.index[pin] = -1
				} else {
					n.index[pin] = len(x)
					v := physical[i]
					if ground >= 0 {
						v -= physical[ground]
					}
					x = append(x, v)
				}
			}
			evaluate := func(v []float64) ([][]float64, []float64) {
				a, f := denseMatrix(len(v)), make([]float64, len(v))
				n.stampBipolar(p, v, func(row, col int, value float64) {
					if row >= 0 && col >= 0 {
						a[row][col] += value
					}
				}, f)
				return a, f
			}
			a, f := evaluate(x)
			if ground < 0 && math.Abs(f[0]+f[1]+f[2]) > 1e-14 {
				t.Fatal("device violates KCL", f)
			}
			for col := range x {
				up, down := append([]float64{}, x...), append([]float64{}, x...)
				up[col] += 1e-6
				down[col] -= 1e-6
				_, fp := evaluate(up)
				_, fm := evaluate(down)
				for row := range f {
					if math.Abs(a[row][col]-(fp[row]-fm[row])/2e-6) > 1e-7 {
						t.Fatal("incorrect full three-terminal Jacobian", kind, ground, row, col)
					}
				}
			}
		}
	}
}

func TestPNPHighSideSwitchIndependentLoadLineAndSigns(t *testing.T) {
	for _, input := range []float64{0, 1, 4, 4.2, 4.5, 5} {
		state, c := solveFloatingDC(t, bipolarDCInput(PNPExampleGraph(), input))
		r := state.result
		q := r.Transistors[0]
		vb, vc, ib, ic := bipolarSwitchReference(5 - input)
		if math.Abs(q.VBE+vb) > 2e-6 || math.Abs(q.VCE+vc) > 2e-6 || math.Abs(q.BaseCurrent+ib) > 1e-10 || math.Abs(q.CollectorCurrent+ic) > 1e-9 || math.Abs(r.LEDs[0].Current-ic) > 1e-9 {
			t.Fatal("PNP independent reference", input, q, vb, vc, ib, ic)
		}
		if q.Model != PNPDeviceModel() || q.EmitterCurrent != q.BaseCurrent+q.CollectorCurrent || q.TransistorPower < -1e-12 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
			t.Fatal("PNP metadata/current/power balance", r)
		}
		if math.Abs(r.Nodes[c.TerminalNet["Q1.emitter"]]-5) > 1e-12 {
			t.Fatal("emitter supply was treated as ground")
		}
	}
}

func TestBipolarEmitterFollowersAndComplementaryBuffer(t *testing.T) {
	for _, input := range []float64{0, 1, 2.5, 4} {
		state, c := solveFloatingDC(t, bipolarDCInput(EmitterFollowerExampleGraph(), input))
		ve, ib := followerReference(input)
		q := state.result.Transistors[0]
		if math.Abs(state.result.Nodes[c.TerminalNet["Q1.emitter"]]-ve) > 2e-6 || math.Abs(q.BaseCurrent-ib) > 1e-10 || math.Abs(q.EmitterCurrent-ve/1000) > 1e-9 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("emitter follower load-line", input, q, ve, ib)
		}
	}
	for _, input := range []float64{0, .1, .5, 1, 2.5} {
		for _, negative := range []bool{false, true} {
			g := bipolarDCInput(ComplementaryBipolarExampleGraph(), input)
			if negative {
				for i := range g.Wires {
					for _, ref := range []*string{&g.Wires[i].From, &g.Wires[i].To} {
						if *ref == "VIN.positive" {
							*ref = "VIN.negative"
						} else if *ref == "VIN.negative" {
							*ref = "VIN.positive"
						}
					}
				}
			}
			state, c := solveFloatingDC(t, g)
			ve, _ := followerReference(input)
			if negative {
				ve = -ve
			}
			r := state.result
			if math.Abs(r.Nodes[c.TerminalNet["Q1.emitter"]]-ve) > 2e-6 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
				t.Fatal("complementary buffer independent load-line", input, negative, r, ve)
			}
			if input == 2.5 && ((!negative && r.Transistors[0].CollectorCurrent < .001) || (negative && r.Transistors[1].CollectorCurrent > -.001)) {
				t.Fatal("wrong device conducts", r.Transistors)
			}
		}
	}
}

func TestBipolarExampleLiveCapturePersistenceAndModelValidation(t *testing.T) {
	for _, id := range []string{"pnp", "emitter-follower", "bipolar-push-pull"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.Mixed == nil || initial.Document.Dirty || initial.Example.Modified || initial.Selected != "Q1" {
				t.Fatal("invalid guided startup", initial.Diagnostics)
			}
			call(t, s, "simulation.step", StepParams{Duration: .075}, "step")
			if s.Snapshot().Transient.Sample.Time < .075 || s.Snapshot().Mixed.Stale {
				t.Fatal("live bipolar stepping failed")
			}
			duration, step := .1, .001
			_, captured := verify(t, s, VerifyParams{Duration: &duration, Step: &step}, "verify")
			m := captured.Metadata
			if (id != "emitter-follower") != (m.PNPModel != nil) || (id != "pnp") != (m.NPNModel != nil) {
				t.Fatal("wrong captured model inventory")
			}
			stored := s.operations[captured.Operation.ID].record
			data, err := encodeVerificationExport(stored, "json")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeRecording(data); err != nil {
				t.Fatal("fresh capture refused", err)
			}
			path := filepath.Join(t.TempDir(), id+".elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "load")
			loaded := settled(t, fresh)
			if loaded.Document.Dirty || loaded.Document.RecordingCount != 1 || !reflect.DeepEqual(initial.Components, loaded.Components) || loaded.Mixed == nil || loaded.Transient.Sample.Time != 0 {
				t.Fatal("saved bipolar project differs", loaded.Diagnostics)
			}
			for _, corrupt := range []string{"polarity", "current", "emitter-current", "voltage", "power", "identity", "missing-model"} {
				var doc VerificationDocument
				_ = json.Unmarshal(data, &doc)
				q := &doc.Metadata.Mixed.Transistors[0]
				switch corrupt {
				case "polarity":
					q.Model = PNPDeviceModel()
					if id == "pnp" {
						q.Model = TransistorModel()
					}
				case "current":
					q.CollectorCurrent += .001
				case "emitter-current":
					q.EmitterCurrent += .001
				case "voltage":
					q.VBE += .1
				case "power":
					q.TransistorPower += .001
				case "identity":
					q.TransistorID = "foreign"
				case "missing-model":
					doc.Metadata.PNPModel = nil
					doc.Metadata.NPNModel = nil
				}
				bad, _ := json.Marshal(doc)
				if _, err := decodeRecording(bad); err == nil {
					t.Fatal("accepted inconsistent bipolar capture", corrupt)
				}
			}
			if id != "emitter-follower" {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var doc Document
				_ = json.Unmarshal(raw, &doc)
				doc.PNPModel = nil
				bad, _ := json.Marshal(doc)
				if _, err := decodeDocument(bad); err == nil {
					t.Fatal("PNP graph without pinned model accepted")
				}
				copy := fresh.Snapshot()
				copy.PNPModel.ForwardBeta = 99
				if fresh.Snapshot().PNPModel.ForwardBeta != 100 {
					t.Fatal("snapshot aliases PNP parameters")
				}
			}
		})
	}
}

func TestBipolarCancellationAndCombinedLimit(t *testing.T) {
	g := bipolarDCInput(ComplementaryBipolarExampleGraph(), 2.5)
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newMixedNetwork(c).solveContext(ctx, nil, nil, 0, &Transient{}, 0); err != context.Canceled {
		t.Fatal("solver cancellation", err)
	}
	for i := 3; i <= MaxNetworkTransistors+1; i++ {
		q, err := MakeComponent(fmt.Sprintf("Q%d", i), "pnp", Point{700, float64(100 * i)}, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, q)
	}
	if _, _, d := g.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("combined transistor limit not enforced", d)
	}
}

func TestBipolarEmitterCapacitorTransientAgainstIndependentODE(t *testing.T) {
	g := bipolarDCInput(EmitterFollowerExampleGraph(), 2.5)
	cap, _ := MakeComponent("CE", "capacitor", Point{650, 0}, "1uF")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{ID: "WC1", From: "CE.1", To: "Q1.emitter", Connected: true}, Wire{ID: "WC2", From: "CE.2", To: "GNDL.reference", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	mirror := g.Clone()
	for i, part := range mirror.Components {
		if part.Kind == "npn" {
			mirror.Components[i], _ = MakeComponent(part.ID, "pnp", part.Position, "")
		}
	}
	for i := range mirror.Wires {
		for _, ref := range []*string{&mirror.Wires[i].From, &mirror.Wires[i].To} {
			for _, id := range []string{"VIN", "VCC"} {
				if *ref == id+".positive" {
					*ref = id + ".negative"
				} else if *ref == id+".negative" {
					*ref = id + ".positive"
				}
			}
		}
	}
	pnp := newGraphService(SolveLED, mirror)
	defer pnp.Close()
	call(t, pnp, "simulation.pause", struct{}{}, "pause")
	call(t, pnp, "simulation.restart", struct{}{}, "restart")
	settled(t, pnp)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	initial := settled(t, s)
	if initial.Mixed == nil || math.Abs(initial.Mixed.Capacitors[0].Voltage) > 1e-12 {
		t.Fatal("emitter capacitor startup", initial.Diagnostics)
	}
	// One-state ODE: solve the base resistor load line independently at each
	// RK4 evaluation, then C*dVE/dt = IE - VE/RL. No MNA or ELab integrator.
	rate := func(ve float64) float64 {
		lo, hi := ve, 2.5
		for range 70 {
			vb := (lo + hi) / 2
			ib, _ := bipolarReference(vb-ve, 5-ve)
			if ib > (2.5-vb)/1000 {
				hi = vb
			} else {
				lo = vb
			}
		}
		ib, ic := bipolarReference((lo+hi)/2-ve, 5-ve)
		return (ib + ic - ve/1000) / 1e-6
	}
	ve, at := 0., 0.
	for _, end := range []float64{.0001, .001, .005} {
		for at < end-1e-15 {
			h := math.Min(1e-7, end-at)
			a := rate(ve)
			b := rate(ve + h*a/2)
			c := rate(ve + h*b/2)
			d := rate(ve + h*c)
			ve += h * (a + 2*b + 2*c + d) / 6
			at += h
		}
		before := s.Snapshot().Transient.Sample.Time
		call(t, s, "simulation.step", StepParams{Duration: end - before}, FormatValue(end, "s"))
		call(t, pnp, "simulation.step", StepParams{Duration: end - before}, FormatValue(end, "s"))
		r := s.Snapshot().Mixed
		pr := pnp.Snapshot().Mixed
		if math.Abs(r.Capacitors[0].Voltage-ve) > 1e-5 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
			t.Fatal("emitter RC ODE reference", end, r.Capacitors[0], ve, r.KCLResidual, r.PowerResidual)
		}
		if math.Abs(pr.Capacitors[0].Voltage+ve) > 1e-5 || math.Abs(pr.Transistors[0].CollectorCurrent+r.Transistors[0].CollectorCurrent) > 1e-9 || pr.KCLResidual > 1e-9 || math.Abs(pr.PowerResidual) > 1e-8 {
			t.Fatal("mirrored PNP RC reference", end, pr, ve)
		}
	}
}

func TestBipolarEmitterResistorAmplifierReference(t *testing.T) {
	for _, input := range []float64{.8, 1, 1.5, 2} {
		g := bipolarDCInput(NPNExampleGraph(), input)
		// Replace the collector LED with a wire; add explicit emitter feedback.
		g = resistiveNPN(g)
		re, _ := MakeComponent("RE", "resistor", Point{400, 200}, "470")
		g.Components = append(g.Components, re)
		for i := range g.Wires {
			if g.Wires[i].From == "Q1.emitter" && g.Wires[i].To == "GND1.reference" {
				g.Wires[i].To = "RE.1"
			}
		}
		g.Wires = append(g.Wires, Wire{ID: "WRE", From: "RE.2", To: "GND1.reference", Connected: true})
		state, c := solveFloatingDC(t, g)
		lo, hi := 0., input/(10000+101*470)
		for range 90 {
			ib := (lo + hi) / 2
			drop := bipolarReferenceVT*math.Log1p(100*ib/1e-14) + ib*(10000+101*470)
			if drop > input {
				hi = ib
			} else {
				lo = ib
			}
		}
		ib := (lo + hi) / 2
		ve := 101 * ib * 470
		vc := 5 - 100*ib*1000
		q := state.result.Transistors[0]
		if q.State != "active" || math.Abs(q.BaseCurrent-ib) > 1e-10 || math.Abs(q.CollectorCurrent-100*ib) > 1e-9 || math.Abs(q.VCE-(vc-ve)) > 2e-6 || math.Abs(state.result.Nodes[c.TerminalNet["Q1.emitter"]]-ve) > 2e-6 {
			t.Fatal("emitter feedback reference", input, q, ib, ve, vc)
		}
	}
}

func TestBipolarDiodeConnectedAndGroundedBase(t *testing.T) {
	for _, kind := range []string{"npn", "pnp"} {
		g := EmptyGraph()
		for i, part := range []struct{ id, kind, value string }{{"V1", "dc-source", "5 V"}, {"R1", "resistor", "1k"}, {"Q1", kind, ""}, {"GND", "ground", ""}} {
			p, _ := MakeComponent(part.id, part.kind, Point{X: float64(i) * 160}, part.value)
			g.Components = append(g.Components, p)
		}
		for i, pair := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "Q1.base"}, {"Q1.base", "Q1.collector"}, {"Q1.emitter", "GND.reference"}, {"V1.negative", "GND.reference"}} {
			if kind == "pnp" {
				if pair[0] == "V1.positive" {
					pair[0] = "V1.negative"
				} else if pair[0] == "V1.negative" {
					pair[0] = "V1.positive"
				}
			}
			g.Wires = append(g.Wires, Wire{ID: "W" + string(rune('A'+i)), From: pair[0], To: pair[1], Connected: true})
		}
		state, _ := solveFloatingDC(t, g)
		lo, hi := 0., 1.
		for range 90 {
			v := (lo + hi) / 2
			if 1.01e-14*math.Expm1(v/bipolarReferenceVT) > (5-v)/1000 {
				hi = v
			} else {
				lo = v
			}
		}
		want := (lo + hi) / 2
		q := state.result.Transistors[0]
		if math.Abs(q.VBE-bipolarPolarity(kind)*want) > 1e-7 || math.Abs(q.VCE-q.VBE) > 1e-12 || math.Abs(q.EmitterCurrent-bipolarPolarity(kind)*(5-want)/1000) > 1e-10 {
			t.Fatal("diode-connected BJT reference", kind, q, want)
		}
	}
	// A true common-base stage: base at zero, emitter resistor to -1 V,
	// collector resistor to +5 V. The reference pin must never index the matrix.
	g := bipolarDCInput(EmitterFollowerExampleGraph(), 1)
	for i := range g.Wires {
		w := &g.Wires[i]
		switch {
		case w.From == "VIN.positive":
			w.From = "Q1.base"
			w.To = "GNDIN.reference"
		case w.From == "RB.2":
			w.From = "VIN.negative"
			w.To = "RL.2"
		case w.From == "RL.2":
			w.From = "VIN.positive"
			w.To = "GNDL.reference"
		case w.From == "VIN.negative":
			w.From = "VCC.positive"
			w.To = "RB.1"
		case w.From == "Q1.collector":
			w.From = "RB.2"
			w.To = "Q1.collector"
		}
	}
	state, c := solveFloatingDC(t, g)
	lo, hi := -1., 0.
	for range 90 {
		ve := (lo + hi) / 2
		ie := 1.01e-14 * math.Expm1(-ve/bipolarReferenceVT)
		if ie > (ve+1)/1000 {
			lo = ve
		} else {
			hi = ve
		}
	}
	want := (lo + hi) / 2
	if math.Abs(state.result.Nodes[c.TerminalNet["Q1.emitter"]]-want) > 1e-7 || math.Abs(state.result.Transistors[0].VBE+want) > 1e-7 {
		t.Fatal("common-base reference", state.result, want)
	}
}

func TestBipolarInductorCapacitorPolaritySymmetry(t *testing.T) {
	positive := inductorNPNCapGraph(t)
	for i, p := range positive.Components {
		if p.Kind == "pulse-source" {
			positive.Components[i], _ = MakeComponent(p.ID, "dc-source", p.Position, "5 V")
		}
	}
	negative := positive.Clone()
	for i, p := range negative.Components {
		if p.Kind == "npn" {
			negative.Components[i], _ = MakeComponent(p.ID, "pnp", p.Position, "")
		}
		if IsVoltageSource(p.Kind) || p.Kind == "diode" {
			one, two := "positive", "negative"
			if p.Kind == "diode" {
				one, two = "anode", "cathode"
			}
			for j := range negative.Wires {
				for _, ref := range []*string{&negative.Wires[j].From, &negative.Wires[j].To} {
					if *ref == p.ID+"."+one {
						*ref = p.ID + "." + two
					} else if *ref == p.ID+"."+two {
						*ref = p.ID + "." + one
					}
				}
			}
		}
	}
	kernels := []*simulationKernel{}
	for _, g := range []Graph{positive, negative} {
		c, _, d := g.Compile()
		if d != nil {
			t.Fatal(d)
		}
		k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal(err)
		}
		kernels = append(kernels, k)
	}
	ends := []float64{.0001}
	for i := 1; i <= 25; i++ {
		ends = append(ends, float64(i)*.001)
	}
	for _, end := range ends {
		for _, k := range kernels {
			// Race instrumentation is not a workstation latency benchmark.
			// Preserve the production attempt limit and all numerical tests.
			budget := newIntegrationBudget()
			budget.elapsed = func() time.Duration { return 0 }
			state, trace, err := advanceMixedBudget(k.mixed, k.trace, end, budget)
			if err != nil {
				t.Fatal(err)
			}
			k.mixed, k.trace = state, trace
		}
		a, b := kernels[0].mixed.result, kernels[1].mixed.result
		if math.Abs(a.Inductors[0].Current+b.Inductors[0].Current) > 1e-9 || math.Abs(a.Capacitors[0].Voltage+b.Capacitors[0].Voltage) > 1e-7 || math.Abs(a.Transistors[0].CollectorCurrent+b.Transistors[0].CollectorCurrent) > 1e-9 || b.KCLResidual > 1e-9 || math.Abs(b.PowerResidual) > 1e-8 {
			t.Fatal("PNP coupled L/C polarity or conservation", end, a, b)
		}
	}
}

func TestBipolarExamplePowerCaptionsClearLoadAndGround(t *testing.T) {
	for _, g := range []Graph{PNPExampleGraph(), EmitterFollowerExampleGraph(), ComplementaryBipolarExampleGraph()} {
		for _, supply := range []string{"VCC", "VEE"} {
			p, ok := g.Component(supply)
			if !ok {
				continue
			}
			x, y, text := schematicPartLabel(p)
			for _, id := range []string{"RL", "GNDL", "GND1"} {
				other, ok := g.Component(id)
				if !ok {
					continue
				}
				ox, oy, otext := schematicPartLabel(other)
				if x-3 <= ox+schematicLabelWidth(otext)+4 && x+schematicLabelWidth(text)+4 >= ox-3 && y-18 <= oy+9 && y+9 >= oy-18 {
					t.Fatal("supply caption crowds load or ground", supply, id, p.Position, other.Position)
				}
			}
		}
	}
}
