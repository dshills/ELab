package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTransistorProfileLawAndJacobian(t *testing.T) {
	for _, kind := range []string{"npn", "pnp"} {
		t.Run(kind, func(t *testing.T) {
			v := DefaultTransistorProfile()
			p := Component{Kind: kind, Device: &DeviceParameters{Bipolar: &v}}
			sign := bipolarPolarity(kind)
			b, c := sign*.63, sign*4.
			ib, ic, bb, bc, cb, cc := componentBipolarLinear(p, b, c)
			vt := 1.380649e-23 * 300.15 / 1.602176634e-19
			f, r := v.SaturationCurrent*math.Expm1(.63/vt), v.SaturationCurrent*math.Expm1((.63-4)/vt)
			if math.Abs(ib-sign*(f/v.ForwardBeta+r/v.ReverseBeta)) > 1e-15 || math.Abs(ic-sign*(f-r*(1+1/v.ReverseBeta)+f*4/v.EarlyVoltage)) > 1e-15 {
				t.Fatal("independent transport/Early law", ib, ic)
			}
			h := 1e-6
			bplus, cplus, _, _, _, _ := componentBipolarLinear(p, b+h, c)
			bminus, cminus, _, _, _, _ := componentBipolarLinear(p, b-h, c)
			qplus, rplus, _, _, _, _ := componentBipolarLinear(p, b, c+h)
			qminus, rminus, _, _, _, _ := componentBipolarLinear(p, b, c-h)
			for _, pair := range [][2]float64{{bb, (bplus - bminus) / (2 * h)}, {cb, (cplus - cminus) / (2 * h)}, {bc, (qplus - qminus) / (2 * h)}, {cc, (rplus - rminus) / (2 * h)}} {
				if math.Abs(pair[0]-pair[1]) > 1e-8*math.Max(1, math.Abs(pair[0])) {
					t.Fatal("analytic derivative", pair)
				}
			}
		})
	}
}
func TestTransistorProfileCaptureHistoryAndPersistence(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "transistor-profile"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	if before.Mixed == nil || len(before.Mixed.Transistors) != 1 {
		t.Fatal(before.Diagnostics)
	}
	q := before.Mixed.Transistors[0]
	c, _, d := (Graph{Components: before.Components, Wires: before.Wires}).Compile()
	if d != nil {
		t.Fatal(d)
	}
	sample := mixedSample(c, before.Mixed, 0)
	if math.Abs(sample.Energy-q.JunctionEnergy) > 1e-20 {
		t.Fatal("scope energy excluded private junctions", sample.Energy, q.JunctionEnergy)
	}
	if q.Model.ID != TransistorProfileModelID+".npn" || q.JunctionEnergy <= 0 || q.JunctionBECurrent != 0 || q.JunctionBCCurrent != 0 || math.Abs(q.EmitterCurrent-q.BaseCurrent-q.CollectorCurrent) > 1e-14 {
		t.Fatal("biased junction startup", q)
	}
	duration, step := .002, .0005
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:Q1.emitter", "i:Q1.base", "i:Q1.collector"}}, "verify")
	s.mu.Lock()
	source := s.operations[op.Operation.ID].record
	record := &verificationRecord{metadata: cloneVerificationMetadata(source.metadata)}
	for _, row := range source.rows {
		row.Values = append([]float64{}, row.Values...)
		record.rows = append(record.rows, row)
	}
	s.mu.Unlock()
	data, err := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(data); err != nil {
		t.Fatal("profile capture", err)
	}
	if len(record.metadata.Mixed.Transistors) != 1 || math.Abs(record.metadata.Mixed.PowerResidual) > 1e-7 {
		t.Fatal("profile capture conservation", record.metadata.Mixed)
	}
	path := filepath.Join(t.TempDir(), "profile.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	other := New()
	defer other.Close()
	call(t, other, "project.load", LoadParams{Path: path}, "open")
	if !reflect.DeepEqual(before.Components, other.Snapshot().Components) {
		t.Fatal("profile project changed parameters")
	}
	v := DefaultTransistorProfile()
	v.ForwardBeta = 300
	call(t, s, "transistor.profile.set", TransistorProfileParams{ID: "Q1", Enabled: true, Parameters: v}, "set")
	if !s.Snapshot().Document.Dirty {
		t.Fatal("model edit not saved/history edit")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	p, _ := (Graph{Components: s.Snapshot().Components}).Component("Q1")
	if p.Device == nil || p.Device.Bipolar == nil || p.Device.Bipolar.ForwardBeta != 100 {
		t.Fatal("profile undo", p)
	}
	call(t, s, "transistor.profile.set", TransistorProfileParams{ID: "Q1", Parameters: v}, "disable")
	p, _ = (Graph{Components: s.Snapshot().Components}).Component("Q1")
	if p.Device != nil {
		t.Fatal("generic restoration")
	}
	call(t, s, "simulation.restart", struct{}{}, "generic-restart")
	settled(t, s)
	bad := v
	bad.Version = "future"
	refuse(t, s, "transistor.profile.set", TransistorProfileParams{ID: "Q1", Enabled: true, Parameters: bad}, "invalid-params")
}
func TestTransistorProfileACChargeAndResourceBounds(t *testing.T) {
	g := acTransistorFixture("npn")
	v := DefaultTransistorProfile()
	v.CBE, v.CBC = 100e-9, 10e-9
	for i, p := range g.Components {
		if p.ID == "Q" {
			g.Components[i].Device = &DeviceParameters{Bipolar: &v}
		}
	}
	with := sweepForTest(t, g, acSettings("VIN", "v:Q.emitter"))
	for i, p := range g.Components {
		if p.ID == "Q" {
			g.Components[i].Device.Bipolar.CBE = 0
			g.Components[i].Device.Bipolar.CBC = 0
		}
	}
	without := sweepForTest(t, g, acSettings("VIN", "v:Q.emitter"))
	if reflect.DeepEqual(with.Points, without.Points) {
		t.Fatal("junction admittance absent")
	}
	for i, p := range g.Components {
		if p.ID == "Q" {
			g.Components[i].Device.Bipolar.CBE = 1e-9
			g.Components[i].Device.Bipolar.CBC = 1e-9
		}
	}
	usage := powerResources(g)
	if usage.caps != 2 || usage.parts != 2 {
		t.Fatal("profile hidden storage accounting", usage)
	}
}

func TestTransistorProfileACAgainstIndependentTwoNodeCircuit(t *testing.T) {
	b := newExampleBuilder()
	b.part("VIN", "sine-source", ".1", -300, 0, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: .1, Offset: .65, Frequency: 10}
	b.ground("GI", "VIN.negative", -300, 100)
	b.part("VCC", "dc-source", "5", 300, 0, 0)
	b.ground("GS", "VCC.negative", 300, 100)
	b.part("RB", "resistor", "1k", -100, 0, 0)
	b.part("RC", "resistor", "1k", 100, 0, 0)
	b.part("Q", "npn", "", 0, 100, 0)
	b.ground("GE", "Q.emitter", 0, 200)
	b.wire("VIN.positive", "RB.1")
	b.wire("RB.2", "Q.base")
	b.wire("VCC.positive", "RC.1")
	b.wire("RC.2", "Q.collector")
	v := DefaultTransistorProfile()
	v.CBE, v.CBC = 100e-9, 10e-9
	for i, p := range b.graph.Components {
		if p.ID == "Q" {
			b.graph.Components[i].Device = &DeviceParameters{Bipolar: &v}
		}
	}
	result, _, err := solveDCOperatingPoint(t.Context(), b.graph, dcSourceLevels(b.graph))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Transistors) != 1 {
		t.Fatal("missing transistor bias")
	}
	q := result.Transistors[0]
	vt := 1.380649e-23 * 300.15 / 1.602176634e-19
	gf := v.SaturationCurrent * math.Exp(q.VBE/vt) / vt
	gr := v.SaturationCurrent * math.Exp((q.VBE-q.VCE)/vt) / vt
	f := v.SaturationCurrent * math.Expm1(q.VBE/vt)
	bb, bc := gf/v.ForwardBeta+gr/v.ReverseBeta, -gr/v.ReverseBeta
	cb, cc := gf-(1+1/v.ReverseBeta)*gr+gf*q.VCE/v.EarlyVoltage, (1+1/v.ReverseBeta)*gr+f/v.EarlyVoltage
	sweep := sweepForTest(t, b.graph, acSettings("VIN", "v:Q.collector"))
	for _, p := range sweep.Points {
		w := 2 * math.Pi * p.Input
		a := complex(.001+bb, w*(v.CBE+v.CBC))
		z := complex(bc, -w*v.CBC)
		c := complex(cb, -w*v.CBC)
		d := complex(.001+cc, w*v.CBC)
		want := -c * .001 / (a*d - z*c)
		if len(p.AC) != 1 {
			t.Fatal("missing AC response")
		}
		got := complex(p.AC[0].Real, p.AC[0].Imaginary)
		if math.Hypot(real(got-want), imag(got-want)) > 1e-8 {
			t.Fatal("independent profile AC", p.Input, got, want)
		}
	}
}

func TestTransistorProfileBiasDoesNotImposeRedundantJunctionVoltages(t *testing.T) {
	b := newExampleBuilder()
	b.part("VB", "sine-source", ".01", 0, 0, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: .01, Offset: .63, Frequency: 10}
	b.ground("GB", "VB.negative", 0, 100)
	b.part("VC", "dc-source", "4", 200, 0, 0)
	b.ground("GC", "VC.negative", 200, 100)
	b.part("Q", "npn", "", 100, 0, 0)
	b.ground("GE", "Q.emitter", 100, 100)
	b.wire("VB.positive", "Q.base")
	b.wire("VC.positive", "Q.collector")
	b.part("RL", "resistor", "1k", 300, 0, 0)
	b.wire("RL.1", "VC.positive")
	b.ground("GL", "RL.2", 300, 100)
	v := DefaultTransistorProfile()
	for i, p := range b.graph.Components {
		if p.ID == "Q" {
			b.graph.Components[i].Device = &DeviceParameters{Bipolar: &v}
		}
	}
	c, _, d := b.graph.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	state, err := n.solveContext(t.Context(), nil, make([]float64, len(n.caps)), 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
	if err != nil {
		t.Fatal("biased junction constraints", err)
	}
	if len(state.result.Transistors) != 1 {
		t.Fatal("missing profiled transistor state")
	}
	q := state.result.Transistors[0]
	want := .5*v.CBE*.63*.63 + .5*v.CBC*(.63-4)*(.63-4)
	if math.Abs(q.JunctionEnergy-want) > 1e-20 || q.JunctionBECurrent != 0 || q.JunctionBCCurrent != 0 {
		t.Fatal("startup bias/charge", q)
	}
	g := DiscreteClassABExampleGraph()
	for i, p := range g.Components {
		if IsBipolarKind(p.Kind) {
			v := DefaultTransistorProfile()
			g.Components[i].Device = &DeviceParameters{Bipolar: &v}
		}
	}
	if _, _, d := g.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("existing storage budget weakened", d)
	}
}

func TestTransistorProfileBiasIgnoresExperimentCapacitorInitialVoltage(t *testing.T) {
	d, ok := findExample("transistor-profile")
	if !ok {
		t.Fatal("missing profile fixture")
	}
	c, _, diagnostic := d.graph.Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	var reference *MixedResult
	for _, initial := range []float64{0, 12, -12} {
		settings := d.settings
		settings.InitialVoltage = initial
		k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: settings, mode: "auto", run: 1, revision: 1})
		if err != nil {
			t.Fatalf("global capacitor initial %g affected biased intrinsic startup: %v", initial, err)
		}
		if reference == nil {
			reference = k.mixed.result
		} else {
			a, b := reference.Transistors[0], k.mixed.result.Transistors[0]
			if math.Abs(a.VBE-b.VBE) > 1e-8 || math.Abs(a.VCE-b.VCE) > 1e-8 {
				t.Fatal("intrinsic bias depended on an ignored global capacitor setting", initial, a, b)
			}
		}
	}
}
