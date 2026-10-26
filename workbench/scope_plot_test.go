package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestPlotLiveProbeRevealsAndFocusesWithoutChangingPhysics(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{}}, "empty")
	call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: false}, "hide")
	before := s.Snapshot()
	call(t, s, "scope.probe.plot", IDParams{ID: "v:C1"}, "plot")
	after := s.Snapshot()
	if !after.ScopeVisible || !reflect.DeepEqual(after.Scope.Selected, []string{"v:C1"}) || after.TraceFocus == nil || after.TraceFocus.ProbeID != "v:C1" || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("plot failed to reveal/highlight or changed simulation")
	}
	call(t, s, "scope.probe.plot", IDParams{ID: "v:C1"}, "same")
	if len(s.Snapshot().Scope.Selected) != 1 {
		t.Fatal("plotting the same source duplicated a trace")
	}
}

func TestPlotProbeFullScopeAndUnknownSourceAreAtomicRefusals(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	ids := []string{}
	for _, probe := range s.Snapshot().Scope.Available {
		ids = append(ids, probe.ID)
	}
	if len(ids) <= MaxScopeProbes {
		t.Fatal("fixture lacks a seventh source")
	}
	call(t, s, "scope.probes.set", ProbeParams{IDs: ids[:MaxScopeProbes]}, "six")
	call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: false}, "hide")
	before := s.Snapshot()
	raw, _ := json.Marshal(IDParams{ID: ids[MaxScopeProbes]})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "full", Command: "scope.probe.plot", Params: raw, ExpectedCommandRevision: &before.CommandRevision, Key: "full"}, "test")
	if r.Error == nil || r.Error.Code != "invalid-params" || r.Help == nil || !strings.Contains(r.Help.Summary, "six") || !strings.Contains(r.Help.Action, "Remove a trace") {
		t.Fatal("full scope lacks actionable shared guidance", r)
	}
	refuse(t, s, "scope.probe.plot", IDParams{ID: "missing"}, "not-found")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("refused plot evicted a trace or changed view")
	}
}

func TestPlotLiveProbePreservesRecordingAndReturnsToLive(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .003
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "record")
	rows := call(t, s, "operation.samples", VerificationPageParams{ID: record.Operation.ID}, "rows-before").Result
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show-record")
	before := s.Snapshot()
	viewerRevision := s.viewerRevision
	call(t, s, "scope.probe.plot", IDParams{ID: "i:C1"}, "live-plot")
	after := s.Snapshot()
	if before.Recording == nil || after.Recording == nil || after.Recording.Viewing || after.Recording.Playing || after.Recording.ID != before.Recording.ID || s.viewerRevision <= viewerRevision || !reflect.DeepEqual(rows, call(t, s, "operation.samples", VerificationPageParams{ID: record.Operation.ID}, "rows-after").Result) {
		t.Fatal("live plot lost captured data or retained recorded mode")
	}
}
