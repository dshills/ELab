package workbench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var recordedViewerCommands = []string{"recording.view.set", "recording.seek", "recording.step", "recording.speed.set", "recording.play", "recording.pause", "recording.window.set", "recording.probes.set", "recording.cursor.select", "recording.cursor.clear", "recording.keep.set"}

func TestRecordedViewerDiscoveryContractsAreCompleteAndQueryOwned(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, command := range recordedViewerCommands {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract, ok := schema["x-command-contract"].(map[string]any)
		if !ok || contract["scope"] != "recorded-viewer-and-kept-membership" || len(contract["effects"].([]any)) == 0 || len(contract["errors"].([]any)) == 0 || schema["x-recording-inventory"] == "" {
			t.Fatal("missing stateful viewer semantics", command)
		}
		caps[0].Input[0] = '!'
		fresh, err := detailedCapabilities(command)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("discovery result aliases later requests", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("on-demand viewer contracts changed default catalog/output schemas")
	}
}

func TestRecordedWindowRejectsIntervalsCollapsedByBoundaryNormalization(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "captured")
	for _, window := range []RecordingWindowParams{{Start: -1e-13, End: 0}, {Start: duration, End: duration + 1e-13}, {Start: -2e-13, End: -1e-13}, {Start: duration + 1e-13, End: duration + 2e-13}} {
		before := s.Snapshot()
		refuse(t, s, "recording.window.set", window, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("collapsed recorded window changed the viewer")
		}
	}
}

func TestRecordedViewerWindowBoundariesDCAndMissingSelection(t *testing.T) {
	empty := New()
	defer empty.Close()
	call(t, empty, "recording.view.set", RecordingViewParams{Mode: "live"}, "live-without-record")
	refuse(t, empty, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "not-ready")
	refuse(t, empty, "recording.seek", map[string]any{"unexpected": true}, "not-ready")
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "recorded")
	call(t, s, "recording.window.set", RecordingWindowParams{Start: -1e-13, End: duration + 1e-13}, "clamp-valid")
	window := s.Snapshot().Recording.Scope.Window
	if window.Start != 0 || window.End != duration {
		t.Fatal("valid roundoff interval did not clamp to complete duration")
	}
	refuse(t, s, "recording.window.set", map[string]any{"start_s": nil, "end_s": "10 ms"}, "invalid-params")
	call(t, s, "recording.window.set", map[string]any{"start_s": "0 s", "end_s": "10 ms"}, "quantity-window")
	if s.Snapshot().Recording.Scope.Window.Start != 0 || s.Snapshot().Recording.Scope.Window.End != .01 {
		t.Fatal("time-quantity behavior differs from discovery")
	}
	dc := NewExample()
	defer dc.Close()
	zero := 0.
	verify(t, dc, VerifyParams{Duration: &zero}, "point")
	call(t, dc, "recording.window.set", RecordingWindowParams{Start: 1e-13, End: 2e-13}, "dc-clamp")
	if dc.Snapshot().Recording.Scope.Window.Start != 0 || dc.Snapshot().Recording.Scope.Window.End != 0 {
		t.Fatal("DC roundoff window did not normalize to its only point")
	}
	call(t, dc, "recording.speed.set", RecordingPlayParams{Speed: .001}, "dc-speed")
	refuse(t, dc, "recording.play", RecordingPlayParams{Speed: .001}, "unavailable")
	refuse(t, dc, "recording.step", RecordingStepParams{Direction: "next"}, "unavailable")
}

func TestRecordedViewerControlsRetainExactRowsAndSeparateLiveAndKeptState(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .06
	_, captured := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:C1"}}, "captured")
	id := captured.Operation.ID
	path := filepath.Join(t.TempDir(), "saved.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "baseline")
	before := settled(t, s)
	s.mu.Lock()
	rows := append([]VerificationRow(nil), s.operations[id].record.rows...)
	s.mu.Unlock()
	edge := -1
	for i := 0; i+1 < len(rows); i++ {
		if rows[i].Time == rows[i+1].Time {
			edge = i
			break
		}
	}
	if edge < 0 {
		t.Fatal("reference capture has no retained edge pair")
	}
	right := edge + 1
	for right+1 < len(rows) && rows[right+1].Time == rows[edge].Time {
		right++
	}
	call(t, s, "recording.seek", map[string]any{"index": nil, "time_s": rows[edge].Time}, "seek-time")
	if s.Snapshot().Recording.Index != right {
		t.Fatal("time seek did not select right-hand edge row")
	}
	call(t, s, "recording.seek", map[string]any{"index": edge, "time_s": nil}, "seek-index")
	call(t, s, "recording.step", RecordingStepParams{Direction: "next"}, "actual-row")
	if s.Snapshot().Recording.Index != edge+1 || s.Snapshot().Recording.Time != rows[edge].Time {
		t.Fatal("Step did not retain distinct equal-time rows")
	}
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .005, End: .015}, "window")
	call(t, s, "recording.probes.set", ProbeParams{IDs: []string{}}, "hide")
	call(t, s, "recording.cursor.select", CursorSelectParams{Cursor: "b"}, "select-b")
	call(t, s, "recording.cursor.set", RecordingCursorParams{Cursor: "a", Time: .01}, "a")
	call(t, s, "recording.cursor.clear", CursorSelectParams{Cursor: "a"}, "clear-a")
	if s.Snapshot().Recording.Scope.A != nil || s.Snapshot().Recording.Scope.ActiveCursor != "b" || len(s.Snapshot().Recording.Scope.Traces) != 0 {
		t.Fatal("recorded probe/cursor subset did not match its contract")
	}
	last := len(rows) - 1
	call(t, s, "recording.seek", RecordingSeekParams{Index: &last}, "end")
	playing := call(t, s, "recording.play", RecordingPlayParams{Speed: .001}, "play-hidden").Result.(Snapshot)
	if !playing.Recording.Playing || playing.Recording.Viewing || playing.Recording.Index != 0 || playing.Recording.Scope.Window.Start != .005 {
		t.Fatal("Play did not restart whole capture independently of window/live view")
	}
	call(t, s, "recording.speed.set", RecordingPlayParams{Speed: .002}, "speed")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "live"}, "stop-by-live")
	if s.Snapshot().Recording.Playing {
		t.Fatal("live view did not stop recorded playback")
	}
	refuse(t, s, "recording.keep.set", map[string]any{"id": id, "keep": nil}, "invalid-params")
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: id, Keep: false}, "exclude")
	if !s.Snapshot().Document.RecordingsDirty || s.Snapshot().Recording.ID != id {
		t.Fatal("excluded membership did not retain selected replay asset")
	}
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: id, Keep: true}, "restore-membership")
	after := s.Snapshot()
	if after.Document != before.Document || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("viewer/membership restoration changed live source or dirty baseline")
	}
	s.mu.Lock()
	unchanged := reflect.DeepEqual(rows, s.operations[id].record.rows)
	s.mu.Unlock()
	if !unchanged {
		t.Fatal("viewer controls changed immutable captured rows")
	}
	stored, err := os.ReadFile(path)
	if err != nil || !json.Valid(stored) {
		t.Fatal("saved baseline became invalid", err)
	}
}
