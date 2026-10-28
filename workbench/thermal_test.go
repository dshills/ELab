package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func thermalResistorGraph(alpha float64, self bool) Graph {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -200, 0, 0)
	b.part("R1", "resistor", "100", 200, 0, 0)
	b.wire("V1.positive", "R1.1")
	b.ground("GND", "V1.negative", -200, 200)
	b.wire("R1.2", "GND.reference")
	cfg := DefaultThermalConfig("resistor")
	cfg.Alpha = alpha
	cfg.SelfHeating = self
	cfg.Resistance = 100
	cfg.Capacitance = .001
	for k, p := range b.graph.Components {
		if p.ID == "R1" {
			b.graph.Components[k].Thermal = &cfg
		}
	}
	return b.graph
}
func TestThermalResistorAgainstIndependentHeatRC(t *testing.T) {
	g := thermalResistorGraph(0, true)
	integrate := func(step float64) *mixedState {
		s, tr := initializeLossRC(t, g)
		if len(s.result.Thermal) != 1 || s.result.Thermal[0].Temperature != 27 {
			t.Fatal("thermal startup", s.result.Thermal)
		}
		for k := 1; k <= int(math.Round(.1/step)); k++ {
			next, err := s.advance(tr, float64(k)*step)
			if err != nil {
				t.Fatal(err)
			}
			s = next
		}
		return s
	}
	coarse, fine := integrate(.001), integrate(.0005)
	want := 27 + 25*(1-math.Exp(-1))
	ce := math.Abs(coarse.result.Thermal[0].Temperature - want)
	fe := math.Abs(fine.result.Thermal[0].Temperature - want)
	if ce > 1e-3 || fe > ce*.6 {
		t.Fatal("thermal RC refinement", coarse.result.Thermal, fine.result.Thermal, want, ce, fe)
	}
	r := fine.result.Thermal[0]
	if math.Abs(r.StoredHeat-.001*(r.Temperature-27)) > 1e-12 || math.Abs(r.ConductiveHeat-.25) > 1e-12 || math.Abs(r.Balance) > 1e-7 || math.Abs(fine.result.PowerResidual) > 1e-8 || fine.result.KCLResidual > 1e-9 {
		t.Fatal("separate thermal/electrical balance", r, fine.result.PowerResidual, fine.result.KCLResidual)
	}
	if len(fine.result.Capacitors) != 0 {
		t.Fatal("thermal state exposed as electrical capacitor")
	}
}
func TestThermalResistorAmbientAndElectricalFeedback(t *testing.T) {
	g := thermalResistorGraph(.004, false)
	for k, p := range g.Components {
		if p.Thermal != nil {
			g.Components[k].Thermal.Ambient = 50
		}
	}
	s, _ := initializeLossRC(t, g)
	wantR := 100 * (1 + .004*23)
	if math.Abs(s.result.Resistors[0].Current-5/wantR) > 1e-12 || s.result.Thermal[0].Resistance != wantR {
		t.Fatal("independent resistor TC", s.result)
	}
	// DC self-heating: theta=Rth*V²/(R0*(1+alpha*theta)).
	g = thermalResistorGraph(.004, true)
	r, _, err := solveDCOperatingPoint(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	theta := (-1 + math.Sqrt(1+4*.004*25)) / (2 * .004)
	got := r.Thermal[0]
	if math.Abs(got.Temperature-(27+theta)) > 1e-5 || math.Abs(got.ConductiveHeat-got.Transfer) > 1e-7 || math.Abs(got.StorageRate) > 1e-12 {
		t.Fatal("independent coupled thermal DC quadratic", got, theta)
	}
}
func TestThermalBipolarTemperatureAndJacobian(t *testing.T) {
	for _, kind := range []string{"npn", "pnp"} {
		p := thermalTestPart(t, "Q1", kind, Point{}, "")
		cfg := DefaultThermalConfig(kind)
		p.Thermal = &cfg
		sign := bipolarPolarity(kind)
		for _, temperature := range []float64{-20, 27, 80} {
			vb, vc := sign*.6, sign*2
			ib, ic, bb, bc, cb, cc, bt, ct := thermalBipolarLinear(p, temperature, vb, vc)
			// Independent reciprocal transport reference using the declared generic law.
			tk := 273.15 + temperature
			ref := 300.15
			is := 1e-14 * math.Pow(tk/ref, 3) * math.Exp(1.11/8.617333262145e-5*(1/ref-1/tk))
			vt := 1.380649e-23 * tk / 1.602176634e-19
			forward, reverse := is*math.Expm1(.6/vt), is*math.Expm1((.6-2)/vt)
			wantB := sign * (forward/100 + reverse)
			wantC := sign * (forward - 2*reverse)
			if math.Abs(ib-wantB) > 1e-12*math.Max(1, math.Abs(ib)) || math.Abs(ic-wantC) > 1e-12*math.Max(1, math.Abs(ic)) {
				t.Fatal("temperature reference", kind, temperature, ib, ic, wantB, wantC)
			}
			dv, dt := 1e-7, 1e-3
			bp, cp, _, _, _, _, _, _ := thermalBipolarLinear(p, temperature, vb+dv, vc)
			bm, cm, _, _, _, _, _, _ := thermalBipolarLinear(p, temperature, vb-dv, vc)
			for k, q := range [][2]float64{{bb, (bp - bm) / (2 * dv)}, {cb, (cp - cm) / (2 * dv)}} {
				if math.Abs(q[0]-q[1]) > 1e-6*math.Max(1, math.Abs(q[0])) {
					t.Fatal("base derivative", k, q)
				}
			}
			bp, cp, _, _, _, _, _, _ = thermalBipolarLinear(p, temperature, vb, vc+dv)
			bm, cm, _, _, _, _, _, _ = thermalBipolarLinear(p, temperature, vb, vc-dv)
			for _, q := range [][2]float64{{bc, (bp - bm) / (2 * dv)}, {cc, (cp - cm) / (2 * dv)}} {
				if math.Abs(q[0]-q[1]) > 1e-6*math.Max(1, math.Abs(q[0])) {
					t.Fatal("collector derivative", q)
				}
			}
			bp, cp, _, _, _, _, _, _ = thermalBipolarLinear(p, temperature+dt, vb, vc)
			bm, cm, _, _, _, _, _, _ = thermalBipolarLinear(p, temperature-dt, vb, vc)
			for _, q := range [][2]float64{{bt, (bp - bm) / (2 * dt)}, {ct, (cp - cm) / (2 * dt)}} {
				if math.Abs(q[0]-q[1]) > 1e-6*math.Max(1, math.Abs(q[0])) {
					t.Fatal("temperature derivative", q)
				}
			}
		}
	}
}

func TestThermalCaptureAndProjectIntegrity(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "self-heating"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration, step := .02, .001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"i:R1"}}, "thermal-capture")
	s.mu.Lock()
	rec := s.operations[op.Operation.ID].record
	doc := VerificationDocument{Metadata: cloneVerificationMetadata(rec.metadata), Rows: append([]VerificationRow{}, rec.rows...)}
	s.mu.Unlock()
	raw, _ := json.Marshal(doc)
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("valid thermal capture", err)
	}
	original := cloneVerificationMetadata(doc.Metadata)
	for _, change := range []func(*ThermalReading){func(r *ThermalReading) { r.Temperature += 1 }, func(r *ThermalReading) { r.StoredHeat += .001 }, func(r *ThermalReading) { r.StorageRate += .01 }, func(r *ThermalReading) { r.ConductiveHeat += .01 }, func(r *ThermalReading) { r.Model = "other" }} {
		doc.Metadata = cloneVerificationMetadata(original)
		change(&doc.Metadata.Mixed.Thermal[0])
		raw, _ = json.Marshal(doc)
		if _, err := decodeRecording(raw); err == nil {
			t.Fatal("forged thermal reading accepted")
		}
	}
	doc.Metadata = cloneVerificationMetadata(original)
	doc.Metadata.Mixed.Resistors[0].Current += .01
	raw, _ = json.Marshal(doc)
	if _, err := decodeRecording(raw); err == nil {
		t.Fatal("forged thermal resistor current accepted")
	}
	path := filepath.Join(t.TempDir(), "thermal.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	settled(t, fresh)
	if !reflect.DeepEqual(s.Snapshot().Components, fresh.Snapshot().Components) || fresh.Snapshot().Document.RecordingCount != 1 {
		t.Fatal("thermal project round trip")
	}
	cfg := DefaultThermalConfig("resistor")
	call(t, s, "thermal.configure", ThermalParams{ID: "R1", Enabled: true, Parameters: cfg}, "change")
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, fresh.Snapshot().Components) {
		t.Fatal("thermal undo")
	}
	snap := s.Snapshot()
	snap.Components[1].Thermal.Ambient = 99
	if reflect.DeepEqual(snap.Components, s.Snapshot().Components) {
		t.Fatal("thermal config alias")
	}
}
func TestThermalTemperatureSweepIndependentReference(t *testing.T) {
	g := thermalResistorGraph(.004, false)
	plan, err := newSweepPlan(g, SweepSettings{ComponentID: "R1", Parameter: "temperature", Start: "-20", Stop: "80", Points: 3, ProbeIDs: []string{"i:R1"}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := computeDCSweep(context.Background(), plan, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range r.Points {
		want := 5 / (100 * (1 + .004*(p.Input-27)))
		if math.Abs(p.Values[0]-want) > 1e-10 {
			t.Fatal("ambient sweep", p, want)
		}
	}
	if g.Components[1].Thermal.Ambient != 27 {
		t.Fatal("sweep mutated graph")
	}
}
func TestThermalFullStampJacobianAndBounds(t *testing.T) {
	for _, kind := range []string{"resistor", "diode", "npn", "pnp"} {
		p := thermalTestPart(t, "DUT", kind, Point{}, "")
		if kind == "resistor" {
			p.Value = 100
		}
		cfg := DefaultThermalConfig(kind)
		cfg.SelfHeating = true
		p.Thermal = &cfg
		c := Compiled{Ground: "g", TerminalNet: map[string]string{}, MixedGraph: Graph{Components: []Component{p}}, Analysis: NetworkTransientAnalysis}
		for k, term := range p.Terminals {
			id := fmt.Sprintf("n%d", k)
			c.Nets = append(c.Nets, Net{ID: id})
			c.TerminalNet[p.ID+"."+term] = id
		}
		n := newMixedNetwork(c)
		x := make([]float64, n.matrixSize(0))
		if IsBipolarKind(kind) {
			x[n.pin(p, "base")] = bipolarPolarity(kind) * .6
			x[n.pin(p, "collector")] = bipolarPolarity(kind) * 2
		} else {
			x[n.pin(p, p.Terminals[0])] = .5
		}
		x[n.index[thermalNode(p.ID)]] = .1
		stamp := func(v []float64) ([][]float64, []float64) {
			j := make([][]float64, len(v))
			for k := range j {
				j[k] = make([]float64, len(v))
			}
			f := make([]float64, len(v))
			n.stampThermal(p, v, func(a, b int, d float64) {
				if a >= 0 && b >= 0 {
					j[a][b] += d
				}
			}, f)
			return j, f
		}
		j, _ := stamp(x)
		for col := range n.nodes {
			h := 1e-7
			xp, xm := append([]float64{}, x...), append([]float64{}, x...)
			xp[col] += h
			xm[col] -= h
			_, fp := stamp(xp)
			_, fm := stamp(xm)
			for row := range n.nodes {
				want := (fp[row] - fm[row]) / (2 * h)
				if math.Abs(j[row][col]-want) > 2e-6*math.Max(1, math.Abs(want)) {
					t.Fatal("thermal full Jacobian", kind, row, col, j[row][col], want)
				}
			}
		}
		x[n.index[thermalNode(p.ID)]] = 2
		if n.validateThermalState(x) == nil {
			t.Fatal("out of range temperature accepted")
		}
	}
}
func TestThermalExamplesAndDisabledEquivalence(t *testing.T) {
	for _, id := range []string{"self-heating", "class-ab-temperature"} {
		s := New()
		t.Cleanup(s.Close)
		call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
		call(t, s, "simulation.pause", struct{}{}, "pause")
		snap := settled(t, s)
		if snap.Mixed == nil || len(snap.Mixed.Thermal) == 0 {
			t.Fatal("thermal example", id, snap.Diagnostics)
		}
		call(t, s, "simulation.step", StepParams{Duration: .0001}, "step")
	}
	g := thermalResistorGraph(.004, false)
	g.Components[1].Thermal = nil
	a, _, err := solveDCOperatingPoint(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultThermalConfig("resistor")
	s := New()
	defer s.Close()
	s.graph = g.Clone()
	s.usedIDs = map[string]bool{}
	raw, _ := json.Marshal(ThermalParams{ID: "R1", Enabled: false, Parameters: cfg})
	if err := s.configureThermalLocked(raw); err != nil {
		t.Fatal(err)
	}
	b, _, err := solveDCOperatingPoint(context.Background(), s.graph, nil)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("disabled model changed original law", err)
	}
}

func TestThermalDiodeTemperatureAndResourceAccounting(t *testing.T) {
	p := thermalTestPart(t, "D1", "diode", Point{}, "")
	cfg := DefaultThermalConfig("diode")
	p.Thermal = &cfg
	c := Compiled{Ground: "g", TerminalNet: map[string]string{"D1.anode": "a", "D1.cathode": "g"}, Nets: []Net{{ID: "g"}, {ID: "a"}}, MixedGraph: Graph{Components: []Component{p}}}
	n := newMixedNetwork(c)
	solveVoltage := func(temp float64) float64 {
		p.Thermal.Ambient = temp
		lo, hi := 0., 1.
		for k := 0; k < 80; k++ {
			mid := (lo + hi) / 2
			x := make([]float64, n.matrixSize(0))
			x[n.pin(p, "anode")] = mid
			if n.evaluateThermal(p, x).currents[0] > .001 {
				hi = mid
			} else {
				lo = mid
			}
		}
		return (lo + hi) / 2
	}
	cold, warm := solveVoltage(0), solveVoltage(80)
	if !(warm < cold-.05) {
		t.Fatal("constant current forward drop did not decrease", cold, warm)
	}
	// Independent implicit Shockley/series-R equation, not another device solve.
	tk := 353.15
	is := SiliconDiodeModel().SaturationCurrent * math.Pow(tk/300.15, 3/1.7) * math.Exp(1.11/(1.7*8.617333262145e-5)*(1/300.15-1/tk))
	want := 1.7*1.380649e-23*tk/1.602176634e-19*math.Log1p(.001/is) + .001*SiliconDiodeModel().SeriesResistance
	if math.Abs(warm-want) > 1e-10 {
		t.Fatal("independent diode equation", warm, want)
	}
	g := thermalResistorGraph(0, true)
	u := powerResources(g)
	if u.parts != 2 || u.caps != 1 || u.resistors != 1 || u.nodes != 1 {
		t.Fatal("thermal state escaped existing resource accounting", u)
	}
	for k := 0; k < MaxNetworkCapacitors; k++ {
		cap := thermalTestPart(t, fmt.Sprintf("C%d", k), "capacitor", Point{X: float64(k) * 100, Y: 400}, "1u")
		g.Components = append(g.Components, cap)
	}
	_, _, d := g.Compile()
	if d == nil || d.Code != "mixed-limit" {
		t.Fatal("thermal storage should exceed original capacitor budget", d)
	}
	g = thermalResistorGraph(0, true)
	g.Components[1].Thermal.Resistance = 2000
	if _, _, err := solveDCOperatingPoint(context.Background(), g, nil); err == nil {
		t.Fatal("out of range hot operating point accepted")
	}
}

func thermalTestPart(t *testing.T, id, kind string, point Point, value string) Component {
	t.Helper()
	p, err := MakeComponent(id, kind, point, value)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestThermalUnsupportedKindsRefuse(t *testing.T) {
	for _, kind := range []string{"led", "zener", "schottky", "nmos", "pmos", "njfet", "pjfet", "capacitor", "opamp"} {
		p := thermalTestPart(t, "DUT", kind, Point{}, "")
		cfg := DefaultThermalConfig(kind)
		p.Thermal = &cfg
		if validateThermal(p) == nil {
			t.Fatal("unsupported thermal kind accepted", kind)
		}
	}
}
