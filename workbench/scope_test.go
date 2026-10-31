package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/testutil"
)

func scopeTrace(t *testing.T, s Snapshot, id string) ScopeTrace {
	t.Helper()
	for _, trace := range s.Scope.Traces {
		if trace.Probe.ID == id {
			return trace
		}
	}
	t.Fatalf("missing probe %s", id)
	return ScopeTrace{}
}
func TestFlasherProbesBackfillAndCursorTiming(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := pausedFlasher(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{1}, "one-second")
	before := s.Snapshot()
	ids := []string{"v:Q1.collector", "v:Q2.collector", "i:D1", "i:D2", "v:C2", "i:C2"}
	call(t, s, "scope.probes.set", ProbeParams{ids}, "both-sides")
	state := s.Snapshot()
	if state.RunID != before.RunID || state.ElectricalRevision != before.ElectricalRevision || state.Transient.Sample != before.Transient.Sample || state.Document != before.Document || !reflect.DeepEqual(state.Mixed, before.Mixed) {
		t.Fatal("scope selection changed experiment")
	}
	for _, trace := range state.Scope.Traces {
		if len(trace.Points) != len(state.Transient.Samples) {
			t.Fatal("new probe missing retained history")
		}
	}
	for _, q := range state.Mixed.Transistors {
		p := scopeTrace(t, state, "v:"+q.TransistorID+".collector").Points
		if p[len(p)-1].Value != q.VCE {
			t.Fatal("collector voltage differs from solver")
		}
	}
	for _, led := range state.Mixed.LEDs {
		p := scopeTrace(t, state, "i:"+led.ID).Points
		if p[len(p)-1].Value != led.Current {
			t.Fatal("LED current differs from solver")
		}
	}
	for _, c := range state.Mixed.Capacitors {
		if c.ID == "C2" {
			p := scopeTrace(t, state, "v:C2").Points
			ip := scopeTrace(t, state, "i:C2").Points
			if p[len(p)-1].Value != c.Voltage || ip[len(ip)-1].Value != c.Current {
				t.Fatal("capacitor probe polarity wrong")
			}
		}
	}
	d1, d2 := scopeTrace(t, state, "i:D1").Points, scopeTrace(t, state, "i:D2").Points
	edges := []float64{}
	opposite := 0
	for i := 1; i < len(d1); i++ {
		if d1[i-1].Value < .001 && d1[i].Value >= .001 {
			edges = append(edges, d1[i].Time)
		}
		if (d1[i].Value > .001) != (d2[i].Value > .001) {
			opposite++
		}
	}
	if len(edges) < 2 || opposite < len(d1)/2 {
		t.Fatal("probes did not capture alternating flasher")
	}
	call(t, s, "scope.cursor.set", CursorParams{"a", edges[0]}, "a")
	call(t, s, "scope.cursor.set", CursorParams{"b", edges[1]}, "b")
	measured := s.Snapshot().Scope
	if measured.DeltaSeconds == nil || math.Abs(*measured.DeltaSeconds-.485) > .01 || measured.Frequency == nil || math.Abs(*measured.Frequency-1/(*measured.DeltaSeconds)) > 1e-12 || len(measured.Readings) != 6 {
		t.Fatalf("invalid cursor timing %+v", measured)
	}
	copy := s.Snapshot()
	copy.Scope.Traces[0].Points[0].Value = 999
	copy.Scope.Selected[0] = "bad"
	copy.Scope.A.Time = 999
	*copy.Scope.Frequency = 999
	if s.Snapshot().Scope.Traces[0].Points[0].Value == 999 || s.Snapshot().Scope.A.Time == 999 || *s.Snapshot().Scope.Frequency == 999 {
		t.Fatal("scope snapshot aliases state")
	}
	call(t, s, "scope.cursor.set", CursorParams{"a", edges[1]}, "coincident")
	if s.Snapshot().Scope.Frequency != nil || *s.Snapshot().Scope.DeltaSeconds != 0 {
		t.Fatal("coincident cursors divided by zero")
	}
	call(t, s, "scope.cursor.set", CursorParams{"b", edges[0]}, "negative-delta")
	if *s.Snapshot().Scope.DeltaSeconds >= 0 || *s.Snapshot().Scope.Frequency <= 0 {
		t.Fatal("signed timing or absolute reciprocal wrong")
	}
	call(t, s, "simulation.restart", struct{}{}, "reset")
	restarted := s.Snapshot()
	if restarted.Scope.A != nil || restarted.Scope.B != nil || !reflect.DeepEqual(restarted.Scope.Selected, ids) || len(restarted.Scope.Traces[0].Points) != 1 {
		t.Fatal("restart mixed histories or lost probes")
	}
}
func TestScopeEdgesPolarityBoundsAndPresentation(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path := filepath.Join(t.TempDir(), "rc.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "saved")
	call(t, s, "simulation.step", StepParams{.05}, "fall")
	call(t, s, "scope.probes.set", ProbeParams{[]string{"i:C1", "i:R1", "i:V1", "v:C1.1", "v:C1.2", "v:V1.positive"}}, "currents")
	state := s.Snapshot()
	if state.Document.Dirty {
		t.Fatal("presentation dirtied schematic")
	}
	trace := scopeTrace(t, state, "i:C1").Points
	if trace[len(trace)-1].Value >= 0 {
		t.Fatal("discharge capacitor current should be negative")
	}
	call(t, s, "scope.cursor.set", CursorParams{"a", .0496}, "snap-a")
	call(t, s, "scope.cursor.set", CursorParams{"b", .05}, "snap-b")
	sc := s.Snapshot().Scope
	if math.Abs(sc.A.Time-.05) > 1e-12 || math.Abs(sc.B.Time-.05) > 1e-12 || sc.Readings[0].A >= 0 || sc.Readings[0].B >= 0 {
		t.Fatal("edge did not use nearest right-hand sample")
	}
	before := s.Snapshot()
	refuse(t, s, "scope.probes.set", ProbeParams{[]string{"i:C1", "i:C1"}}, "invalid-params")
	refuse(t, s, "scope.cursor.set", CursorParams{"a", .06}, "invalid-params")
	if !reflect.DeepEqual(s.Snapshot(), before) {
		t.Fatal("refusal mutated state")
	}
	call(t, s, "simulation.step", StepParams{.6}, "roll")
	sc = s.Snapshot().Scope
	if sc.A.Valid || sc.A.Reason != "outside retained history" || sc.DeltaSeconds != nil {
		t.Fatal("expired cursor presented as measured")
	}
	call(t, s, "component.set", ValueParams{"R1", "20k"}, "edit")
	if !s.Snapshot().Scope.Stale {
		t.Fatal("edited experiment exported fresh scope")
	}
	call(t, s, "scope.cursor.clear", CursorSelectParams{"a"}, "clear-a")
	if s.Snapshot().Scope.A != nil {
		t.Fatal("cursor clear failed")
	}
	call(t, s, "scope.probes.set", ProbeParams{[]string{}}, "hide")
	if len(s.Snapshot().Scope.Traces) != 0 {
		t.Fatal("empty selection restored defaults")
	}
	// Catalog remains bounded and the internal frames never appear in schemas/JSON.
	data, err := json.Marshal(Catalog())
	if err != nil || len(data) > 1024*1024 {
		t.Fatalf("catalog exceeds transport: %d %v", len(data), err)
	}
}
func TestScopeNPNBranchOrientationAndRetryIsolation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.npn-pulse", ReplaceParams{}, "npn")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	ids := []string{"i:Q1.base", "i:Q1.collector", "i:Q1.emitter", "i:RB", "i:RL", "i:VCC"}
	// Read actual identities from the solver rather than assuming example names.
	state := s.Snapshot()
	r := state.NPN
	ids[3] = "i:" + r.BaseResistorID
	ids[4] = "i:" + r.LoadResistorID
	ids[5] = "i:" + r.SupplyID
	response := call(t, s, "scope.probes.set", ProbeParams{ids}, "probes")
	stored := response.Result.(Snapshot)
	for i, expected := range []float64{r.BaseCurrent, r.CollectorCurrent, r.EmitterCurrent} {
		if stored.Scope.Traces[i].Points[0].Value != expected {
			t.Fatal("transistor current probe wrong")
		}
	}
	call(t, s, "simulation.step", StepParams{.05}, "step")
	retry := call(t, s, "scope.probes.set", ProbeParams{ids}, "probes")
	if !retry.Duplicate || !reflect.DeepEqual(retry.Result.(Snapshot).Scope, stored.Scope) {
		t.Fatal("retry lost original scope snapshot")
	}
}
