package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInductorStartupCommandsVerifyAndProjectReopen(t *testing.T) {
	g := inductorTestGraph(t, 5, 100, 1, nil)
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "inductor.initial.set", ValueParams{ID: "L1", Value: "-20 mA"}, "initial")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	state := settled(t, s)
	if state.Mixed == nil || state.Mixed.Inductors[0].Current != -.02 || state.Transient.Engine != InductorEngineID {
		t.Fatal("shared startup command/engine", state.Diagnostics)
	}
	copy := s.Snapshot()
	*copy.Components[2].InitialCurrent = .7
	copy.Mixed.Inductors[0].Current = .8
	if s.Snapshot().Mixed.Inductors[0].Current != -.02 || *s.Snapshot().Components[2].InitialCurrent != -.02 {
		t.Fatal("snapshot aliases current memory or startup intent")
	}
	before := s.Snapshot()
	call(t, s, "component.move", MoveParams{ID: "L1", Position: Point{X: 400, Y: 120}}, "move")
	if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) || !reflect.DeepEqual(before.Transient, s.Snapshot().Transient) {
		t.Fatal("geometry changed current/charge/time")
	}
	duration := .005
	_, view := verify(t, s, VerifyParams{Duration: &duration, Inductors: []InductorInitial{{ID: "L1", Current: .03}}, ProbeIDs: []string{"i:L1", "v:L1"}}, "verify")
	page := call(t, s, "operation.samples", VerificationPageParams{ID: view.Operation.ID, Limit: 100}, "samples").Result.(VerificationPage)
	if len(page.Rows) != 6 || page.Rows[0].Values[0] != .03 || !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) {
		t.Fatal("Verify scenario current or live-state isolation")
	}
	for _, row := range page.Rows {
		want := .05 + (.03-.05)*math.Exp(-row.Time/.01)
		if math.Abs(row.Values[0]-want) > 1e-5 || math.Abs(row.Values[1]-(5-100*want)) > .001 {
			t.Fatal("captured RL reference", row, want)
		}
	}
	raw, err := encodeVerificationExport(s.operations[view.Operation.ID].record, "json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("valid inductor capture refused", err)
	}
	path := filepath.Join(t.TempDir(), "rl.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	reopened := settled(t, fresh)
	if !reopened.Paused || reopened.Transient.Sample.Time != 0 || reopened.Mixed.Inductors[0].Current != -.02 || reopened.Document.RecordingCount != 1 {
		t.Fatal("fresh reopen lost startup intent/capture or trusted prior current", reopened.Diagnostics)
	}
	for _, mutate := range []func(*VerificationDocument){
		func(d *VerificationDocument) { d.Metadata.Mixed.Inductors = nil },
		func(d *VerificationDocument) { d.Metadata.Mixed.Inductors[0].ID = "foreign" },
		func(d *VerificationDocument) { d.Metadata.Mixed.Inductors[0].Energy = .5 },
		func(d *VerificationDocument) { d.Metadata.Mixed.Inductors[0].Current = 1e308 },
		func(d *VerificationDocument) { d.Metadata.Graph.Components[0].InitialCurrent = new(float64) },
	} {
		var document VerificationDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		mutate(&document)
		bad, _ := json.Marshal(document)
		if _, err := decodeRecording(bad); err == nil {
			t.Fatal("invalid or foreign inductor capture accepted")
		}
	}
	refuse(t, s, "inductor.initial.set", ValueParams{ID: "R1", Value: "1 mA"}, "invalid-params")
	refuse(t, s, "inductor.initial.set", ValueParams{ID: "L1", Value: "1.1 A"}, "invalid-params")
	refuse(t, s, "simulation.verify", map[string]any{"inductor_initials": nil}, "invalid-params")
	refuse(t, s, "simulation.verify", map[string]any{"inductor_initials": []any{map[string]any{"id": "L1"}}}, "invalid-params")
	refuse(t, s, "simulation.verify", VerifyParams{Inductors: []InductorInitial{{ID: "L1"}, {ID: "L1"}}}, "invalid-params")
	call(t, s, "inductor.initial.set", ValueParams{ID: "L1", Value: "default"}, "default")
	if s.Snapshot().Components[2].InitialCurrent != nil {
		t.Fatal("reset did not restore implicit zero startup")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo-default")
	if *s.Snapshot().Components[2].InitialCurrent != -.02 {
		t.Fatal("startup intent is not undoable")
	}
}

func TestInductorSteadyDCIsSeparateFromConfiguredStartup(t *testing.T) {
	g := inductorTestGraph(t, 5, 100, 1, nil)
	initial := -.02
	g.Components[2].InitialCurrent = &initial
	r, _, err := solveDCOperatingPoint(t.Context(), g, dcSourceLevels(g))
	if err != nil || len(r.Inductors) != 1 || math.Abs(r.Inductors[0].Current-.05) > 1e-12 || math.Abs(r.Inductors[0].Voltage) > 1e-12 {
		t.Fatal("DC must short L and solve its steady current", r, err)
	}
	// A voltage applied directly to L is a valid current ramp in transient,
	// but no finite steady current exists. Keep those analyses distinct.
	g.Wires[1].From = "V1.positive"
	g.Wires = append(g.Wires, Wire{ID: "WR", From: "R1.2", To: "GND1.reference", Connected: true})
	state, trace := initializeInductorTest(t, g)
	state, err = state.advance(trace, .01)
	if err != nil || math.Abs(state.result.Inductors[0].Current-(initial+.05)) > 1e-12 {
		t.Fatal("ideal source across L did not integrate current ramp", err)
	}
	if _, _, err := solveDCOperatingPoint(t.Context(), g, dcSourceLevels(g)); err == nil {
		t.Fatal("inconsistent source/inductor DC constraint accepted")
	}
}
