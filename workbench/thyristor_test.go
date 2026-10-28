package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func thyristorTestPart(t *testing.T, kind string) Component {
	t.Helper()
	p, err := MakeComponent("DUT", kind, Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestThyristorMainGateJacobianKCLAndPower(t *testing.T) {
	for _, kind := range []string{"scr", "triac"} {
		p := thyristorTestPart(t, kind)
		for _, on := range []bool{false, true} {
			for _, vm := range []float64{-3, -.4, .4, 3} {
				for _, vg := range []float64{-2, 0, 2} {
					v := [5]float64{vm, 0, vg}
					m := ControlMemory{High: on}
					e := thyristorCurrents(p, v, m)
					sum, power := 0., 0.
					for k, i := range e.Current {
						sum += i
						power += v[k] * i
					}
					if math.Abs(sum) > 1e-12 || power < 0 {
						t.Fatal("thyristor KCL/passivity", kind, e, power)
					}
					for col := 0; col < 3; col++ {
						h := 1e-7
						vp, vm := v, v
						vp[col] += h
						vm[col] -= h
						ep, em := thyristorCurrents(p, vp, m), thyristorCurrents(p, vm, m)
						for row := 0; row < 3; row++ {
							want := (ep.Current[row] - em.Current[row]) / (2 * h)
							if math.Abs(e.Jacobian[row][col]-want) > 1e-7 {
								t.Fatal("thyristor Jacobian", kind, on, row, col, e.Jacobian[row][col], want)
							}
						}
					}
				}
			}
		}
	}
}
func TestThyristorTriggerLatchHoldingAndDirection(t *testing.T) {
	for _, kind := range []string{"scr", "triac"} {
		p := thyristorTestPart(t, kind)
		for _, sign := range []float64{1, -1} {
			if kind == "scr" && sign < 0 {
				continue
			}
			m, err := thyristorDecision(p, [5]float64{sign * 5, 0, 2}, ControlMemory{})
			if err != nil || !m.High || m.Latched {
				t.Fatal("tentative trigger", kind, m, err)
			}
			// Actual solved on-current below Il cannot latch merely from a large blocking voltage.
			v := [5]float64{sign * (.8 + .003), 0, 2}
			m, err = thyristorDecision(p, v, m)
			if err != nil || m.Latched || !m.High {
				t.Fatal("premature latch", m, err)
			}
			v[2] = 0
			m, err = thyristorDecision(p, v, m)
			if err != nil || m.High {
				t.Fatal("low-load false latch", m, err)
			}
			m, _ = thyristorDecision(p, [5]float64{sign * 5, 0, 2}, m)
			m, err = thyristorDecision(p, [5]float64{sign * .81, 0, 2}, m)
			if err != nil || !m.Latched {
				t.Fatal("actual current latch", kind, m, err)
			}
			m, err = thyristorDecision(p, [5]float64{sign * .803, 0, 0}, m)
			if err != nil || !m.Latched {
				t.Fatal("hold between Ih and Il", m, err)
			}
			reversed, err := thyristorDecision(p, [5]float64{-sign * 2, 0, 0}, m)
			if err != nil || reversed.High {
				t.Fatal("signed commutation", reversed, err)
			}
			m, err = thyristorDecision(p, [5]float64{sign * .801, 0, 0}, m)
			if err != nil || m.High {
				t.Fatal("below holding", m, err)
			}
		}
	}
	p := thyristorTestPart(t, "scr")
	m, err := thyristorDecision(p, [5]float64{-5, 0, 2}, ControlMemory{})
	if err != nil || m.High {
		t.Fatal("SCR reverse triggered", m, err)
	}
}
func TestThyristorSCRLatchActualCircuitAndReset(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "scr-latch"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	get := func(snap Snapshot) ControlResult {
		for _, c := range snap.Mixed.Controls {
			if c.ID == "SCR1" {
				return c
			}
		}
		t.Fatal("SCR result absent")
		return ControlResult{}
	}
	if get(before).Memory.High {
		t.Fatal("SCR starts latched")
	}
	call(t, s, "button.set", ButtonParams{ID: "TRIGGER", Pressed: true}, "trigger")
	on := settled(t, s)
	if !get(on).Memory.Latched || get(on).Current < .005 {
		t.Fatal("SCR did not latch", on.Diagnostics, get(on))
	}
	call(t, s, "button.set", ButtonParams{ID: "TRIGGER", Pressed: false}, "release")
	held := settled(t, s)
	if !get(held).Memory.Latched {
		t.Fatal("gate release lost latch", get(held))
	}
	call(t, s, "switch.set", SwitchParams{ID: "RESET", Closed: false}, "reset")
	call(t, s, "simulation.step", StepParams{Duration: .0001}, "apply-reset")
	off := settled(t, s)
	if get(off).Memory.High {
		t.Fatal("current interruption did not reset", get(off))
	}
	if before.RunID != on.RunID || on.RunID != held.RunID || held.Mixed.Time != on.Mixed.Time {
		t.Fatal("manual gate changed run/time")
	}
}
func TestThyristorTRIACBothHalfCyclesAndCommutation(t *testing.T) {
	g := TRIACPhaseExampleGraph()
	s, tr := initializeLossRC(t, g)
	positive, negative, offAfter := false, false, false
	for k := 1; k <= 400; k++ {
		next, err := s.advance(tr, float64(k)*.0001)
		if err != nil {
			t.Fatal(k, err)
		}
		s = next
		for _, c := range s.result.Controls {
			if c.ID != "TR1" {
				continue
			}
			if c.Memory.Latched && c.Current > .005 {
				positive = true
			}
			if c.Memory.Latched && c.Current < -.005 {
				negative = true
			}
			if positive && s.result.Time > .0099 && s.result.Time < .014 && !c.Memory.High {
				offAfter = true
			}
			if c.Memory.Latched && c.Memory.Direction == 0 {
				t.Fatal("missing latched polarity")
			}
		}
		if s.result.KCLResidual > 1e-8 || math.Abs(s.result.PowerResidual) > 1e-7 {
			t.Fatal("TRIAC conservation", s.result)
		}
	}
	if !positive || !negative || !offAfter {
		t.Fatal("phase control/commutation", positive, negative, offAfter)
	}
	if _, err := newSweepPlan(g, SweepSettings{ComponentID: "VAC", Parameter: "voltage", Start: "0", Stop: "5", Points: 3}); err == nil {
		t.Fatal("public sweep accepted stateful thyristor")
	}
}

func thyristorPulseFixture(kind string, load string, reverse bool) Graph {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -250, -100, 0)
	b.part("RL", "resistor", load, 0, -100, 0)
	b.part("DUT", kind, "", 200, 100, 0)
	b.part("VG", "finite-pulse-source", "5", -250, 200, 0)
	source, reference := "V1.positive", "V1.negative"
	if reverse {
		source, reference = reference, source
	}
	b.ground("GND", reference, -250, 400)
	pins := thyristorPins(kind)
	b.wire(source, "RL.1")
	b.wire("RL.2", "DUT."+pins[0])
	b.wire("DUT."+pins[1], "GND.reference")
	b.wire("VG.positive", "DUT.gate")
	b.wire("VG.negative", "GND.reference")
	for k, p := range b.graph.Components {
		if p.ID == "VG" {
			b.graph.Components[k].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .0001, LowSeconds: .0009, RiseSeconds: .00001, FallSeconds: .00001}
		}
	}
	return b.graph
}
func TestThyristorSubsampleGatePulseAndActualLoadThreshold(t *testing.T) {
	for _, tc := range []struct {
		kind, load       string
		reverse, latched bool
	}{{"scr", "100", false, true}, {"scr", "1k", false, false}, {"scr", "100", true, false}, {"triac", "100", true, true}} {
		g := thyristorPulseFixture(tc.kind, tc.load, tc.reverse)
		s, tr := initializeLossRC(t, g)
		next, err := s.advance(tr, .0002)
		if err != nil {
			t.Fatal(tc, err)
		}
		c := next.result.Controls[0]
		if c.Memory.Latched != tc.latched || c.Memory.High != tc.latched {
			t.Fatal("gate pulse/load latch reference", tc, c)
		}
		if tc.latched {
			want := (5 - .8) / 101.
			if tc.reverse {
				want = -want
			}
			if math.Abs(c.Current-want) > 1e-9 {
				t.Fatal("independent on-current", tc, c.Current, want)
			}
		}
	}
}
func TestThyristorEventTimingAndStepRefinement(t *testing.T) {
	integrate := func(step float64) []float64 {
		s, tr := initializeLossRC(t, TRIACPhaseExampleGraph())
		for k := 1; k <= int(math.Round(.02/step)); k++ {
			var err error
			s, err = s.advance(tr, float64(k)*step)
			if err != nil {
				t.Fatal(err)
			}
		}
		events := []float64{}
		probe := Probe{ID: "i:TR1", Unit: "A"}
		for k := 1; k < len(tr.Samples); k++ {
			left, right := tr.Samples[k-1], tr.Samples[k]
			if left.Time != right.Time {
				continue
			}
			a, aok := frameValue(left, probe)
			b, bok := frameValue(right, probe)
			if aok && bok && math.Abs(a-b) > .001 {
				events = append(events, right.Time)
			}
		}
		return events
	}
	coarse, fine := integrate(.0002), integrate(.0001)
	if len(coarse) < 4 || len(coarse) != len(fine) {
		t.Fatal("bracketed firing/holding transitions absent", coarse, fine)
	}
	for k, x := range coarse {
		if math.Abs(x-fine[k]) > 3e-9 {
			t.Fatal("event refinement", coarse, fine)
		}
	}
	// Holding cutoff with a resistive load: |Vs|=Vdrop+(Rload+Ron)*Ih.
	cutoff := .01 - math.Asin((.8+101*.002)/8)/(2*math.Pi*50)
	found := false
	for _, at := range fine {
		if math.Abs(at-cutoff) < 3e-8 {
			found = true
		}
	}
	if !found {
		t.Fatal("independent holding-current time", fine, cutoff)
	}
}

func TestThyristorCaptureSaveHistoryAndFailureOwnership(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "triac-phase"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	duration, step := .016, .0001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:RL", "i:TR1", "i:TR1.gate"}}, "capture")
	s.mu.Lock()
	rec := s.operations[op.Operation.ID].record
	doc := VerificationDocument{Metadata: cloneVerificationMetadata(rec.metadata), Rows: append([]VerificationRow{}, rec.rows...)}
	s.mu.Unlock()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(data); err != nil {
		t.Fatal("valid thyristor capture", err)
	}
	original := cloneVerificationMetadata(doc.Metadata)
	for _, mutate := range []func(*ControlResult){func(c *ControlResult) { c.Memory.Direction = 0 }, func(c *ControlResult) { c.Current += .001 }, func(c *ControlResult) { c.PinCurrents[2] += .001 }, func(c *ControlResult) { c.Power += 1 }, func(c *ControlResult) { c.Model.Thyristor = "other" }} {
		doc.Metadata = cloneVerificationMetadata(original)
		for k, c := range doc.Metadata.Mixed.Controls {
			if c.ID == "TR1" {
				mutate(&doc.Metadata.Mixed.Controls[k])
			}
		}
		data, err = json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRecording(data); err == nil {
			t.Fatal("forged thyristor capture accepted")
		}
	}
	path := filepath.Join(t.TempDir(), "thyristor.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	after := settled(t, fresh)
	if !reflect.DeepEqual(before.Components, after.Components) || after.Document.RecordingCount != 1 {
		t.Fatal("thyristor project")
	}
	cfg := DefaultThyristorConfig()
	cfg.Holding = .001
	call(t, s, "thyristor.configure", ThyristorParams{ID: "TR1", Parameters: cfg}, "configure")
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("thyristor undo")
	}
	cfg.Holding = cfg.Latching
	unchanged := s.Snapshot()
	refuse(t, s, "thyristor.configure", ThyristorParams{ID: "TR1", Parameters: cfg}, "invalid-params")
	if !reflect.DeepEqual(unchanged.Components, s.Snapshot().Components) || !reflect.DeepEqual(unchanged.UndoEntry, s.Snapshot().UndoEntry) {
		t.Fatal("refused thyristor edit changed source/history")
	}
	g := thyristorPulseFixture("scr", "100", false)
	state, tr := initializeLossRC(t, g)
	old := append([]ControlMemory{}, state.net.controlMemory...)
	_, err = state.advanceBudget(tr, 1, newIntegrationBudget())
	if err == nil {
		t.Fatal("huge pulse request unexpectedly escaped original work guard")
	}
	if !reflect.DeepEqual(old, state.net.controlMemory) || state.result.Time != 0 {
		t.Fatal("rejected trial mutated accepted memory")
	}
}
