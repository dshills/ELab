package workbench

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestNarrowLiveWindowDoesNotBorrowOutsideSamples(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: 1e-12}, "one-picosecond")
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1"}}, "probe")
	before := s.Snapshot()
	call(t, s, "scope.window.set", ScopeWindow{Mode: "fixed", Start: .4e-12, End: .6e-12}, "between-samples")
	after := s.Snapshot()
	if after.Scope.WindowValid || after.Scope.WindowReason == "" || len(after.Scope.Traces[0].Points) != 0 || len(after.Scope.Statistics) != 0 {
		t.Fatal("window borrowed saved rows outside its interval", after.Scope)
	}
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "a", Time: .5e-12}, "a")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "b", Time: .5e-12}, "b")
	scope := s.Snapshot().Scope
	if scope.A.Valid || scope.B.Valid || scope.DeltaSeconds != nil || len(scope.Readings) != 0 {
		t.Fatal("empty visible interval invented measurements", scope)
	}
	path := filepath.Join(t.TempDir(), "untouched.csv")
	if err := os.WriteFile(path, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "export.scope.csv", ExportParams{Path: path, Overwrite: true}, "unavailable")
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "untouched" {
		t.Fatal("unavailable export replaced destination")
	}
	call(t, s, "scope.window.set", ScopeWindow{Mode: "fixed", Start: .75e-12, End: 1e-12}, "one-saved-row")
	scope = s.Snapshot().Scope
	if !scope.WindowValid || len(scope.Traces[0].Points) != 1 || scope.Statistics[0].Count != 1 {
		t.Fatal("valid narrow interval included startup row", scope)
	}
	path = filepath.Join(t.TempDir(), "one-row.csv")
	call(t, s, "export.scope.csv", ExportParams{Path: path}, "csv")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rows) != 2 {
		t.Fatal("CSV disagrees with visible scope", len(rows), err)
	}
	at, err := strconv.ParseFloat(rows[1][6], 64)
	if err != nil || at != 1e-12 {
		t.Fatal("CSV exported an outside row", rows[1])
	}
	after = s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("window/cursor/export changed physical source or history")
	}
}

func TestNarrowRecordedWindowAndReferenceStayEmptyBetweenRows(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := 1e-12
	_, first := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "reference")
	_, second := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "selected")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show-recorded")
	before := s.Snapshot()
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .4e-12, End: .6e-12}, "empty-window")
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: second.Operation.ID, ReferenceID: first.Operation.ID, Time: .5e-12, ShowOverlay: true}, "reference-overlay")
	view := s.Snapshot().Recording
	if view.Scope.WindowValid || view.WindowSamples != 0 || len(view.Scope.Traces[0].Points) != 0 || len(view.Scope.Statistics) != 0 || len(view.Readings) != 0 {
		t.Fatal("recorded interval fabricated rows or zero statistics", view.Scope, view.WindowSamples, view.Readings)
	}
	overlay := view.ReferenceOverlay
	if overlay == nil || overlay.Scope.WindowValid || overlay.WindowSamples != 0 || len(overlay.Scope.Traces[0].Points) != 0 {
		t.Fatal("reference escaped selected interval", overlay)
	}
	after := s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("recorded window changed live source/history")
	}
}

func TestFullRecordedViewKeepsLegacyRoundedEndpoint(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	s.mu.Lock()
	record := s.operations[p.ID].record
	actualEnd := record.rows[len(record.rows)-1].Time
	record.metadata.Duration = actualEnd - 5e-13
	s.mu.Unlock()
	// The unchanged file-grid tolerance accepts this legacy endpoint roundoff.
	if !scopeTimeInside(actualEnd, actualEnd-5e-13, actualEnd-5e-13) {
		t.Fatal("legacy grid validation tolerance changed")
	}
	view := s.Snapshot().Recording
	if view.Scope.Window.End != actualEnd || view.WindowSamples != view.Total || len(view.Scope.Traces[0].Points) != view.Total {
		t.Fatal("full plot hid a retained legacy endpoint", view.Scope.Window, view.WindowSamples, view.Total)
	}
}
