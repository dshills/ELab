package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lossRCGraph() Graph {
	g := ACRCExampleGraph()
	for i, p := range g.Components {
		if p.ID == "V1" {
			g.Components[i].Value = 5
		}
		if p.ID == "C1" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, ESR: 100, Leakage: 1e5}}
		}
	}
	return g
}
func initializeLossRC(t *testing.T, g Graph) (*mixedState, *Transient) {
	t.Helper()
	c, _, d := g.Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	tr := &Transient{Mode: "auto", Settings: DefaultTransientSettings()}
	state, err := n.solve(nil, make([]float64, len(n.caps)), 0, tr, 0)
	if err != nil {
		t.Fatal(err)
	}
	return state, tr
}
func TestPassiveCapacitorAgainstIndependentTheveninResponse(t *testing.T) {
	peaks := []float64{}
	for _, step := range []float64{.001, .00002} {
		state, tr := initializeLossRC(t, lossRCGraph())
		rth := 1000 * 1e5 / (1000 + 1e5)
		target := 5 * 1e5 / (1000 + 1e5)
		peak := 0.
		for sample := 0; sample <= int(math.Round(.04/step)); sample++ {
			at := float64(sample) * step
			if sample > 0 {
				var err error
				state, err = state.advance(tr, at)
				if err != nil {
					t.Fatal(sample, err)
				}
			}
			cap := state.result.Capacitors[0]
			stored := target * (1 - math.Exp(-at/((rth+100)*1e-6)))
			charging := (target - stored) / (rth + 100)
			terminal := stored + 100*charging
			total := charging + terminal/1e5
			peak = math.Max(peak, math.Abs(cap.Voltage-terminal))
			if cap.Loss == nil || math.Abs(cap.Voltage-terminal) > 1e-4 || math.Abs(cap.Current-total) > 1e-7 || math.Abs(cap.Loss.StoredVoltage-stored) > 1e-4 || math.Abs(cap.Energy-.5e-6*stored*stored) > 1e-9 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
				t.Fatal("independent lossy RC response", step, sample, cap, stored, total, state.result.PowerResidual)
			}
			if cap.Loss.ESRHeat < 0 || cap.Loss.LeakageHeat < 0 {
				t.Fatal("negative passive heat")
			}
		}
		peaks = append(peaks, peak)
	}
	if peaks[1] > 1e-5 || peaks[1] >= peaks[0]/2 {
		t.Fatal("independent lossy RC refinement", peaks)
	}
}

func TestPassiveWindingAgainstIndependentRLResponse(t *testing.T) {
	g := inductorTestGraph(t, 5, 100, 1, nil)
	for i, p := range g.Components {
		if p.Kind == "inductor" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, Winding: 20}}
		}
	}
	state, tr := initializeInductorTest(t, g)
	for sample := 1; sample <= 60; sample++ {
		at := float64(sample) * .001
		var err error
		state, err = state.advance(tr, at)
		if err != nil {
			t.Fatal(err)
		}
		got := state.result.Inductors[0]
		want := 5. / 120 * (1 - math.Exp(-120*at))
		if got.Loss == nil || math.Abs(got.Current-want) > 1e-6 || math.Abs(got.Voltage-(5-100*want)) > .0001 || math.Abs(got.Loss.WindingHeat-20*want*want) > 2e-6 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("independent lossy RL", sample, got, want, state.result.PowerResidual)
		}
	}
}
func TestPassiveACAgainstIndependentImpedanceAndDCLeakage(t *testing.T) {
	g := lossRCGraph()
	cfg := SweepSettings{ComponentID: "V1", Parameter: "frequency", Spacing: "logarithmic", Start: "1 Hz", Stop: "100 kHz", Points: 51, ProbeIDs: []string{"v:C1"}}
	plan, err := newSweepPlan(g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := computeACSweep(context.Background(), plan, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.Points {
		z := complex(100, 0) + 1/complex(0, 2*math.Pi*p.Input*1e-6)
		load := 1 / (1/z + complex(1e-5, 0))
		want := load / (1000 + load)
		got := p.AC[0]
		if cmplx.Abs(complex(got.Real, got.Imaginary)-want) > 1e-10 {
			t.Fatal("independent ESR/leak impedance", p.Input, got, want)
		}
	}
	dc, _, err := solveDCOperatingPoint(context.Background(), g, dcSourceLevels(g))
	if err != nil {
		t.Fatal(err)
	}
	c := dc.Capacitors[0]
	if c.Loss == nil || math.Abs(c.Voltage-5e5/101000) > 1e-10 || math.Abs(c.Current-c.Voltage/1e5) > 1e-12 || math.Abs(dc.PowerResidual) > 1e-8 {
		t.Fatal("DC leakage opens storage but not shunt", c, dc.PowerResidual)
	}
}
func TestPassiveSharedHistoryCaptureReopenAndCorruption(t *testing.T) {
	s := newGraphService(SolveLED, lossRCGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	cfg := PassiveConfig{Version: PassiveLossModelID, ESR: 150, Leakage: 1e5}
	call(t, s, "passive.configure", PassiveParams{ID: "C1", Enabled: true, Parameters: cfg}, "configure")
	call(t, s, "edit.undo", struct{}{}, "undo")
	call(t, s, "simulation.restart", struct{}{}, "restart-again")
	settled(t, s)
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("passive undo")
	}
	duration, step := .01, .001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1", "i:C1"}}, "capture")
	path := filepath.Join(t.TempDir(), "lossy.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	after := settled(t, fresh)
	if !reflect.DeepEqual(before.Components, after.Components) || after.Document.RecordingCount != 1 {
		t.Fatal("passive round trip")
	}
	s.mu.Lock()
	rec := s.operations[op.Operation.ID].record
	doc := VerificationDocument{Metadata: cloneVerificationMetadata(rec.metadata), Rows: append([]VerificationRow{}, rec.rows...)}
	s.mu.Unlock()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodeRecording(raw); err != nil {
		t.Fatal("valid passive capture", err)
	}
	doc.Metadata.Mixed.Capacitors[0].Loss.ESRHeat += 1
	raw, _ = json.Marshal(doc)
	if _, err = decodeRecording(raw); err == nil {
		t.Fatal("forged ESR heat accepted")
	}
	snap := s.Snapshot()
	snap.Mixed.Capacitors[0].Loss.StoredVoltage = 999
	if s.Snapshot().Mixed.Capacitors[0].Loss.StoredVoltage == 999 {
		t.Fatal("loss reading aliases snapshot")
	}
	call(t, s, "passive.configure", PassiveParams{ID: "C1", Parameters: cfg}, "restore-ideal")
	cap, _ := s.graph.Component("C1")
	if cap.Device != nil {
		t.Fatal("disable retained model")
	}
	invalid := cfg
	invalid.ESR = -1
	refuse(t, s, "passive.configure", PassiveParams{ID: "C1", Enabled: true, Parameters: invalid}, "invalid-params")
}

func TestPassiveRLCCombinedLossAgainstIndependentODEAndJacobian(t *testing.T) {
	g := LossyWindingExampleGraph()
	for i, p := range g.Components {
		if p.Kind == "capacitor" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, ESR: 10, Leakage: 1e5}}
		}
	}
	state, tr := initializeInductorTest(t, g)
	derivative := func(v, i float64) (float64, float64) {
		charge := (i - v/1e5) / (1 + 10./1e5)
		terminal := v + 10*charge
		return charge / 1e-5, (5 - 70*i - terminal) / .1
	}
	v, current := 0., 0.
	const h = 1e-6
	for sample := 1; sample <= 100; sample++ {
		for range 100 {
			a, b := derivative(v, current)
			c, d := derivative(v+h*a/2, current+h*b/2)
			e, f := derivative(v+h*c/2, current+h*d/2)
			g, k := derivative(v+h*e, current+h*f)
			v += h / 6 * (a + 2*c + 2*e + g)
			current += h / 6 * (b + 2*d + 2*f + k)
		}
		var err error
		state, err = state.advance(tr, float64(sample)*.0001)
		if err != nil {
			t.Fatal(sample, err)
		}
		if math.Abs(state.result.Capacitors[0].Loss.StoredVoltage-v) > 5e-6 || math.Abs(state.result.Inductors[0].Current-current) > 1e-7 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("independent lossy RLC RK4", sample, state.result.Capacitors[0], state.result.Inductors[0], v, current, state.result.PowerResidual)
		}
	}
	n := state.net
	size := len(state.x)
	x := make([]float64, 3*size)
	for k := range 3 {
		copy(x[k*size:], state.x)
	}
	old := n.reactiveHistory(state.x, state.capV)
	a, _, _ := n.radauEquations(x, old, .0001, tr, .0101)
	const delta = 1e-6
	for col := range x {
		plus, minus := append([]float64{}, x...), append([]float64{}, x...)
		plus[col] += delta
		minus[col] -= delta
		_, one, _ := n.radauEquations(plus, old, .0001, tr, .0101)
		_, two, _ := n.radauEquations(minus, old, .0001, tr, .0101)
		for row := range x {
			expected := (one[row] - two[row]) / (2 * delta)
			if math.Abs(a[row][col]-expected) > 1e-7+1e-5*math.Abs(expected) {
				t.Fatal("lossy collocation Jacobian", row, col, a[row][col], expected)
			}
		}
	}
}
func TestPassiveESRRemovesOnlyActualIdealConstraint(t *testing.T) {
	g := lossRCGraph()
	parts := g.Components[:0]
	for _, p := range g.Components {
		if p.Kind != "resistor" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	g.Wires = nil
	for i, ends := range [][2]string{{"V1.positive", "C1.1"}, {"V1.negative", "GND.reference"}, {"C1.2", "GND.reference"}} {
		// ACRC's actual ground ID is discovered rather than assumed here.
		ground := ""
		for _, p := range g.Components {
			if p.Kind == "ground" {
				ground = p.ID + ".reference"
				break
			}
		}
		if ends[1] == "GND.reference" {
			ends[1] = ground
		}
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprint("W", i), From: ends[0], To: ends[1], Connected: true})
	}
	// Remove the second, now unconnected ground from the original RC graph.
	parts = g.Components[:0]
	seenGround := false
	for _, p := range g.Components {
		if p.Kind == "ground" {
			if seenGround {
				continue
			}
			seenGround = true
		}
		parts = append(parts, p)
	}
	g.Components = parts
	state, _ := initializeLossRC(t, g)
	cap := state.result.Capacitors[0]
	if cap.Loss == nil || math.Abs(cap.Voltage-5) > 1e-10 || math.Abs(cap.Loss.StoredVoltage) > 1e-10 || math.Abs(cap.Loss.StorageCurrent-.05) > 1e-10 {
		t.Fatal("ESR startup clamp", cap)
	}
	for i, p := range g.Components {
		if p.Kind == "capacitor" {
			g.Components[i].Device.Passive.ESR = 0
		}
	}
	_, _, d := g.Compile()
	if d == nil || d.Code != "ideal-constraint-loop" && d.Code != "ideal-source-capacitor" {
		t.Fatal("leakage must not relax ideal series constraint", d)
	}
}

func TestPassiveDisableRestoresIdealGraphAndBoundsAreAtomic(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before, _ := json.Marshal(s.graph)
	cfg := PassiveConfig{Version: PassiveLossModelID, ESR: 20, Leakage: 1e6}
	call(t, s, "passive.configure", PassiveParams{ID: "C1", Enabled: true, Parameters: cfg}, "enable")
	explain := explanationQuery(t, s, "C1")
	if explain.Model != PassiveLossModelID {
		t.Fatal("explanation still claims ideal model", explain)
	}
	call(t, s, "passive.configure", PassiveParams{ID: "C1", Parameters: cfg}, "disable")
	after, _ := json.Marshal(s.graph)
	if string(before) != string(after) {
		t.Fatal("disable changed original ideal graph")
	}
	for _, bad := range []PassiveConfig{{Version: PassiveLossModelID, ESR: .0001}, {Version: PassiveLossModelID, Leakage: 999}, {Version: PassiveLossModelID, Leakage: 1e13}, {Version: PassiveLossModelID, Winding: 1}, {Version: "wrong"}} {
		prior, _ := json.Marshal(s.Snapshot())
		refuse(t, s, "passive.configure", PassiveParams{ID: "C1", Enabled: true, Parameters: bad}, "invalid-params")
		next, _ := json.Marshal(s.Snapshot())
		if string(prior) != string(next) {
			t.Fatal("invalid passive parameters mutated accepted state")
		}
	}
}
func TestPassiveExampleExportAndRecordingKinds(t *testing.T) {
	for _, name := range []string{"capacitor-loss", "winding-loss"} {
		t.Run(name, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: name}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if len(initial.Diagnostics) > 0 || initial.Document.Dirty || initial.DeviceModel == nil || initial.DeviceModel.Passive != passiveLossDescription {
				t.Fatal("passive example", initial.Diagnostics)
			}
			duration, step := .002, .0001
			probe := "v:C1"
			if name == "winding-loss" {
				probe = "i:L1"
			}
			_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{probe}}, "verify")
			s.mu.Lock()
			record := s.operations[op.Operation.ID].record
			doc := VerificationDocument{Metadata: cloneVerificationMetadata(record.metadata), Rows: append([]VerificationRow{}, record.rows...)}
			s.mu.Unlock()
			data, _ := json.Marshal(doc)
			if _, err := decodeRecording(data); err != nil {
				t.Fatal("passive example capture", err)
			}
			svg := string(schematicSVG(s.graph, 1, 1))
			if !strings.Contains(svg, "device_components") || !strings.Contains(svg, "passive-loss.v1") {
				t.Fatal("export lost passive intent")
			}
			if name == "winding-loss" {
				doc.Metadata.Mixed.Inductors[0].Loss.WindingHeat += 1
				bad, _ := json.Marshal(doc)
				if _, err := decodeRecording(bad); err == nil {
					t.Fatal("corrupt winding heat accepted")
				}
			}
		})
	}
}
func TestPassiveInternalVoltageUsesExistingNodeEnvelope(t *testing.T) {
	// Internal voltage can differ from the visible terminal when ESR carries
	// current, but remains subject to the same ±30V absolute node envelope.
	g := RLCExampleGraph()
	for i, part := range g.Components {
		if part.Kind == "capacitor" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, ESR: 100}}
		}
	}
	r, _ := initializeInductorTest(t, g)
	n := r.net
	size := len(r.x)
	x := make([]float64, 3*size)
	for stage := range 3 {
		copy(x[stage*size:], r.x)
		x[stage*size+n.capacitorIndex(0)] = 1
	}
	if err := n.radauNodeRange(x, size); err == nil {
		t.Fatal("hidden ESR storage node bypassed range")
	}
}
