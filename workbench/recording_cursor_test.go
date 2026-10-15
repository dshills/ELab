package workbench

import (
	"reflect"
	"testing"
)

func TestRecordedCursorPinsSelectionAndKeepsLegacyInputs(t *testing.T) {
	s, a, b := comparisonFixture(t)
	defer s.Close()
	before := s.Snapshot()
	refuse(t, s, "recording.cursor.set", RecordingCursorParams{OperationID: a, Cursor: "a", Time: .01}, "conflict")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("wrong capture cursor changed state")
	}
	call(t, s, "recording.cursor.set", RecordingCursorParams{OperationID: b, Cursor: "a", Time: .01}, "pinned")
	if s.Snapshot().Recording.Scope.A.Requested != .01 {
		t.Fatal("pinned cursor not applied")
	}
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "b", Time: .02}, "legacy")
	if s.Snapshot().Recording.Scope.B.Requested != .02 {
		t.Fatal("legacy current-selection cursor refused")
	}
	schema := sweepDetailSchema(t, "recording.cursor.set")
	if inputProperty(schema, "operation_id")["description"] == nil || schema["x-command-contract"] == nil {
		t.Fatal("capture guard not discoverable")
	}
	call(t, s, "operation.select", IDParams{ID: a}, "select")
	refuse(t, s, "recording.cursor.set", RecordingCursorParams{OperationID: b, Cursor: "a", Time: 0}, "conflict")
}
