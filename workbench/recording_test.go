package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func exportedRecording(t *testing.T, s *Service, duration float64, ids []string) (string, string) {
	t.Helper()
	_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: ids}, "record-for-import")
	path := filepath.Join(t.TempDir(), "record.json")
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: path, Format: "json"}, "record-export")
	return path, view.Operation.ID
}
func TestRecordingImportRoundTripAcrossSupportedAnalyses(t *testing.T) {
	for _, example := range []string{"example.reset", "example.rc", "example.npn", "example.npn-pulse", "example.flasher"} {
		t.Run(example, func(t *testing.T) {
			source := New()
			defer source.Close()
			call(t, source, example, ReplaceParams{}, "example")
			call(t, source, "simulation.pause", struct{}{}, "pause")
			settled(t, source)
			duration := 0.
			if IsTransientAnalysis(source.Snapshot().Analysis) {
				duration = .02
			}
			path, id := exportedRecording(t, source, duration, nil)
			target := pausedRC(t)
			defer target.Close()
			call(t, target, "simulation.step", StepParams{Duration: .015}, "live-time")
			before := target.Snapshot()
			call(t, target, "verification.import", RecordingImportParams{Path: path}, "import")
			imported := target.Snapshot()
			record := imported.Recording
			if record == nil || !record.Imported || !record.Viewing || record.OriginID != id || record.Index != 0 || record.Playing {
				t.Fatal("imported viewer metadata missing")
			}
			if schematicDocumentState(before.Document) != schematicDocumentState(imported.Document) || before.Transient.Sample != imported.Transient.Sample || before.ElectricalRevision != imported.ElectricalRevision || before.RunID != imported.RunID || !reflect.DeepEqual(before.Components, imported.Components) || before.UndoName != imported.UndoName {
				t.Fatal("import changed live circuit/state")
			}
			view := inspectOperation(t, target, record.ID)
			if view.Operation.SourceUnchanged || view.Operation.SourcePath != path || view.Operation.SourceSHA256 == "" {
				t.Fatal("import claimed current source or lost provenance")
			}
			original := inspectOperation(t, source, id)
			if !reflect.DeepEqual(original.Metadata, view.Metadata) {
				t.Fatal("import changed captured metadata")
			}
			copyPath := filepath.Join(t.TempDir(), "copy.json")
			call(t, target, "verification.export", VerificationExportParams{ID: record.ID, Path: copyPath, Format: "json"}, "copy")
			a, _ := os.ReadFile(path)
			b, _ := os.ReadFile(copyPath)
			if string(a) != string(b) {
				t.Fatal("reexport changed record or leaked host provenance")
			}
			call(t, target, "verification.import", RecordingImportParams{Path: path}, "import")
			if len(target.operationOrder) != 1 {
				t.Fatal("retry duplicated record")
			}
		})
	}
}
func TestRecordingInvalidFilesRefuseWithoutEditorMutation(t *testing.T) {
	source := pausedRC(t)
	defer source.Close()
	path, _ := exportedRecording(t, source, .01, []string{"v:C1"})
	good, _ := os.ReadFile(path)
	mutations := []struct {
		name   string
		change func(map[string]any)
	}{
		{"future-version", func(d map[string]any) { d["metadata"].(map[string]any)["version"] = 99 }},
		{"missing-zero-time", func(d map[string]any) { delete(d["rows"].([]any)[0].(map[string]any), "time_s") }},
		{"null-value", func(d map[string]any) { d["rows"].([]any)[0].(map[string]any)["values"] = []any{nil} }},
		{"incomplete-row", func(d map[string]any) { d["rows"].([]any)[0].(map[string]any)["values"] = []any{} }},
		{"out-of-order", func(d map[string]any) { d["rows"].([]any)[2].(map[string]any)["time_s"] = 0 }},
		{"missing-end", func(d map[string]any) { d["rows"] = d["rows"].([]any)[:10] }},
		{"wrong-unit", func(d map[string]any) {
			d["metadata"].(map[string]any)["probes"].([]any)[0].(map[string]any)["unit"] = "A"
		}},
		{"wrong-statistics", func(d map[string]any) {
			d["metadata"].(map[string]any)["statistics"].([]any)[0].(map[string]any)["sample_rms"] = 999
		}},
		{"unbounded-reading", func(d map[string]any) { d["rows"].([]any)[0].(map[string]any)["values"] = []any{1e308} }},
		{"wrong-model", func(d map[string]any) { d["metadata"].(map[string]any)["led_model"].(map[string]any)["id"] = "other" }},
		{"missing-wire-state", func(d map[string]any) {
			delete(d["metadata"].(map[string]any)["graph"].(map[string]any)["wires"].([]any)[0].(map[string]any), "connected")
		}},
	}
	target := pausedRC(t)
	defer target.Close()
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			var d map[string]any
			_ = json.Unmarshal(good, &d)
			mutation.change(d)
			data, _ := json.Marshal(d)
			bad := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(bad, data, 0600); err != nil {
				t.Fatal(err)
			}
			refuse(t, target, "verification.import", RecordingImportParams{Path: bad}, "invalid-recording")
		})
	}
	refuse(t, target, "simulation.verify", map[string]any{"probe_ids": nil}, "invalid-params")
	generation := target.Snapshot().Document.Generation + 1
	refuse(t, target, "verification.import", RecordingImportParams{Path: path, ExpectedGeneration: &generation}, "conflict")
	link := filepath.Join(t.TempDir(), "linked.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	refuse(t, target, "verification.import", RecordingImportParams{Path: link}, "file-error")
}
func TestRecordingPulseEdgesCursorsAndPlaybackRemainIndependent(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	cfg := s.Snapshot().TransientSettings
	cfg.HighSeconds = .005
	cfg.LowSeconds = .005
	call(t, s, "simulation.transient.set", cfg, "short-pulses")
	_, id := exportedRecording(t, s, .02, []string{"i:D1", "v:Q1.collector"})
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show-record")
	rows := s.operations[id].record.rows
	left := -1
	for i := 0; i+1 < len(rows); i++ {
		if rows[i].Time == rows[i+1].Time && !reflect.DeepEqual(rows[i].Values, rows[i+1].Values) {
			left = i
			break
		}
	}
	if left < 0 {
		t.Fatal("pulse edge sides missing")
	}
	before := s.Snapshot()
	call(t, s, "recording.seek", RecordingSeekParams{Index: &left}, "left-side")
	a := s.Snapshot().Recording
	call(t, s, "recording.step", RecordingStepParams{Direction: "next"}, "right-side")
	b := s.Snapshot().Recording
	if a.Time != b.Time || a.Readings[0].Value == b.Readings[0].Value {
		t.Fatal("row stepping discarded edge sides")
	}
	at := a.Time
	call(t, s, "recording.seek", RecordingSeekParams{Time: &at}, "time-seek")
	if s.Snapshot().Recording.Index != nearestRecordingIndex(rows, at) {
		t.Fatal("time seek did not use latest right side")
	}
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "a", Time: .005}, "cursor-a")
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "b", Time: .015}, "cursor-b")
	scope := s.Snapshot().Recording.Scope
	if scope.DeltaSeconds == nil || !recordingNear(*scope.DeltaSeconds, .01) || len(scope.Readings) != 2 {
		t.Fatal("recorded timing/readings missing")
	}
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .006, End: .02}, "window")
	if s.Snapshot().Recording.Scope.A.Valid {
		t.Fatal("hidden cursor presented as visible")
	}
	refuse(t, s, "recording.seek", RecordingSeekParams{}, "invalid-params")
	refuse(t, s, "recording.probes.set", ProbeParams{IDs: []string{"v:missing"}}, "invalid-params")
	call(t, s, "recording.play", RecordingPlayParams{Speed: 10}, "play")
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Recording.Playing && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	after := s.Snapshot()
	if after.Recording.Playing || after.Recording.Index != len(rows)-1 || before.Transient.Sample != after.Transient.Sample || before.Document != after.Document || before.Scope.ActiveCursor != after.Scope.ActiveCursor || after.Scope.A != nil {
		t.Fatal("recorded playback did not end independently")
	}
	clone := after
	clone.Recording = cloneRecordingView(after.Recording)
	clone.Recording.Scope.Traces[0].Points[0].Value = 999
	if s.Snapshot().Recording.Scope.Traces[0].Points[0].Value == 999 {
		t.Fatal("recorded plot aliases snapshots")
	}
}
func TestRecordingPlotReductionPreservesExtremaAndExactStatistics(t *testing.T) {
	points := make([]ScopePoint, 4096)
	for i := range points {
		points[i] = ScopePoint{Time: float64(i), Value: float64(i % 11)}
	}
	points[107].Value = -999
	points[1000].Value = 999
	reduced := reduceRecordedPoints(points)
	if len(reduced) > 2*RecordingPlotBuckets+2 {
		t.Fatal("plot exceeds bound")
	}
	a, b := scopeStatistics(points, "p"), scopeStatistics(reduced, "p")
	if a.Minimum != b.Minimum || a.Maximum != b.Maximum || reduced[0] != points[0] || reduced[len(reduced)-1] != points[len(points)-1] {
		t.Fatal("reduction dropped extrema/endpoints")
	}
	s := pausedRC(t)
	defer s.Close()
	_, id := exportedRecording(t, s, 1, []string{"v:C1"})
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "plot")
	view := s.Snapshot().Recording
	if !view.PlotReduced || view.Scope.Statistics[0].Count != s.operations[id].record.metadata.SampleCount || len(view.Scope.Traces[0].Points) > 258 || view.Scope.Statistics[0] != s.operations[id].record.metadata.Statistics[0] {
		t.Fatal("plot reduction changed exact measurements")
	}
	data, err := json.Marshal(s.Snapshot())
	if err != nil || len(data) > 1024*1024 {
		t.Fatal("recorded snapshot exceeds transport bound")
	}
}

func TestRecordingDCMetadataStepAndEmptyCaptureRoundTrip(t *testing.T) {
	s := NewExample()
	defer s.Close()
	duration, step := 0., 1.
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{}}, "dc-step")
	path := filepath.Join(t.TempDir(), "dc.json")
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: path, Format: "json"}, "dc-export")
	call(t, s, "verification.import", RecordingImportParams{Path: path}, "dc-import")
	if r := s.Snapshot().Recording; r == nil || r.Total != 1 || len(r.Scope.Traces) != 0 || r.Duration != 0 {
		t.Fatal("valid DC override/empty capture did not import")
	}
	refuse(t, s, "recording.play", RecordingPlayParams{Speed: 1}, "unavailable")
}

func TestRecordingMissingInteriorOutputAndTinyCursorIntervals(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path, id := exportedRecording(t, s, .01, []string{"v:C1"})
	record := s.operations[id].record
	d := VerificationDocument{Metadata: cloneVerificationMetadata(record.metadata), Rows: append([]VerificationRow{}, record.rows...)}
	d.Rows = append(d.Rows[:5], d.Rows[6:]...)
	d.Metadata.SampleCount = len(d.Rows)
	points := []ScopePoint{}
	for _, row := range d.Rows {
		points = append(points, ScopePoint{Time: row.Time, Value: row.Values[0]})
	}
	d.Metadata.Statistics = []ScopeStatistics{scopeStatistics(points, "v:C1")}
	bytes, _ := json.Marshal(d)
	if err := os.WriteFile(path, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "verification.import", RecordingImportParams{Path: path}, "invalid-recording")
	// Finite, very close recorded timestamps must not produce an infinite Hz field.
	d.Rows = append([]VerificationRow{}, record.rows...)
	extra := VerificationRow{Time: 1e-320, Values: append([]float64{}, d.Rows[0].Values...)}
	d.Rows = append(d.Rows[:1], append([]VerificationRow{extra}, d.Rows[1:]...)...)
	d.Metadata.SampleCount = len(d.Rows)
	points = []ScopePoint{}
	for _, row := range d.Rows {
		points = append(points, ScopePoint{Time: row.Time, Value: row.Values[0]})
	}
	d.Metadata.Statistics = []ScopeStatistics{scopeStatistics(points, "v:C1")}
	bytes, _ = json.Marshal(d)
	if err := os.WriteFile(path, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	call(t, s, "verification.import", RecordingImportParams{Path: path}, "tiny-import")
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "a", Time: 0}, "tiny-a")
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "b", Time: 1e-320}, "tiny-b")
	if _, err := json.Marshal(s.Snapshot()); err != nil {
		t.Fatalf("recorded cursor made snapshot unserializable: %v", err)
	}
}
