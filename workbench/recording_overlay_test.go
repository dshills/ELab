package workbench

import (
	"reflect"
	"testing"
)

func TestRecordingOverlayOwnsExactReferenceWindowAndChannelOrder(t *testing.T) {
	s, reference, selected := comparisonFixture(t)
	defer s.Close()
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .004, End: .018}, "window")
	before := s.Snapshot()
	p := RecordingCompareParams{ID: selected, ReferenceID: reference, Time: .01, ShowOverlay: true}
	response := call(t, s, "recording.compare.set", p, "overlay")
	view := response.Result.(Snapshot).Recording
	overlay := view.ReferenceOverlay
	if overlay == nil || overlay.ID != reference || overlay.TimeBasis != "captured-seconds-from-startup" || !overlay.Scope.WindowValid || overlay.WindowSamples != 15 || overlay.PlotReduced {
		t.Fatal("reference window identity/count", overlay)
	}
	if !reflect.DeepEqual(overlay.Scope.Selected, view.Scope.Selected) {
		t.Fatal("reference does not follow A channel order")
	}
	for _, trace := range overlay.Scope.Traces {
		var expected []ScopePoint
		record := s.operations[reference].record
		for i, probe := range record.metadata.Probes {
			if sameComparisonProbe(probe, trace.Probe) {
				for _, row := range record.rows {
					if scopeTimeInside(row.Time, .004, .018) {
						expected = append(expected, ScopePoint{Time: row.Time, Value: row.Values[i]})
					}
				}
			}
		}
		if !reflect.DeepEqual(trace.Points, expected) {
			t.Fatal("reference values/timestamps resampled or reordered", trace.Probe.ID)
		}
	}
	after := s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("overlay changes live circuit/history")
	}
	view.ReferenceOverlay.Scope.Traces[0].Points[0].Value = 999
	view.ReferenceOverlay.Scope.Selected[0] = "corrupt"
	fresh := s.Snapshot().Recording.ReferenceOverlay
	if fresh.Scope.Traces[0].Points[0].Value == 999 || fresh.Scope.Selected[0] == "corrupt" {
		t.Fatal("reference view aliases caller")
	}
	retry := call(t, s, "recording.compare.set", p, "overlay")
	if !retry.Duplicate || retry.Result.(Snapshot).Recording.ReferenceOverlay.Scope.Traces[0].Points[0].Value == 999 {
		t.Fatal("reference aliases retry cache")
	}
	call(t, s, "recording.probes.set", ProbeParams{IDs: []string{"v:C1"}}, "voltage")
	if o := s.Snapshot().Recording.ReferenceOverlay; len(o.Scope.Traces) != 1 || o.Scope.Traces[0].Probe.ID != "v:C1" {
		t.Fatal("reference selection differs")
	}
	call(t, s, "recording.probes.set", ProbeParams{IDs: []string{}}, "hidden")
	if o := s.Snapshot().Recording.ReferenceOverlay; o.Scope.WindowValid || len(o.Scope.Traces) != 0 || o.Scope.WindowReason == "" {
		t.Fatal("hidden primary probes still show reference")
	}
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "live"}, "live")
	if s.Snapshot().Recording.ReferenceOverlay != nil {
		t.Fatal("reference leaked into live plot")
	}
}

func TestRecordingOverlayNeverExtrapolatesShorterReference(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .01
	_, a := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "short")
	duration = .02
	_, b := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "long")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b.Operation.ID, ReferenceID: a.Operation.ID, ShowOverlay: true}, "compare")
	o := s.Snapshot().Recording.ReferenceOverlay
	points := o.Scope.Traces[0].Points
	if o.Scope.Window.End != .02 || points[len(points)-1].Time != .01 || o.Duration != .01 {
		t.Fatal("short reference stretched to A duration")
	}
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .012, End: .02}, "no-overlap")
	o = s.Snapshot().Recording.ReferenceOverlay
	if o.Scope.WindowValid || o.WindowSamples != 0 || len(o.Scope.Traces[0].Points) != 0 || o.Scope.WindowReason == "" {
		t.Fatal("missing reference interval fabricated samples")
	}
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b.Operation.ID}, "clear")
	if s.Snapshot().Recording.ReferenceOverlay != nil {
		t.Fatal("clear comparison left overlay")
	}
}

func TestRecordingOverlayUsesBoundedCapturedExtremaAndMatchingIdentity(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .9
	_, a := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1", "i:R1"}}, "reference")
	_, b := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1", "i:R1"}}, "selected")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b.Operation.ID, ReferenceID: a.Operation.ID, ShowOverlay: true}, "compare")
	o := s.Snapshot().Recording.ReferenceOverlay
	if !o.PlotReduced || o.WindowSamples != 901 {
		t.Fatal("reference reduction not declared")
	}
	for i, trace := range o.Scope.Traces {
		if len(trace.Points) > 2*RecordingPlotBuckets+2 {
			t.Fatal("reference plot unbounded")
		}
		var exact []ScopePoint
		for _, row := range s.operations[a.Operation.ID].record.rows {
			exact = append(exact, ScopePoint{Time: row.Time, Value: row.Values[i]})
		}
		if !reflect.DeepEqual(trace.Points, reduceRecordedPoints(exact)) {
			t.Fatal("reference reduced independently of captured extrema")
		}
	}
	// Matching IDs/labels alone must not compare different polarity or units.
	s.mu.Lock()
	s.operations[a.Operation.ID].record.metadata.Probes[0].Reference = "different-terminal"
	s.operations[a.Operation.ID].record.metadata.Probes[1].Unit = "V"
	s.mu.Unlock()
	o = s.Snapshot().Recording.ReferenceOverlay
	if o.Scope.WindowValid || len(o.Scope.Traces) != 0 {
		t.Fatal("reference matched an electrically different probe identity")
	}
}

func TestRecordingOverlayRetainsUnreducedPulseEdgeSides(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.npn-pulse", ReplaceParams{}, "pulse")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration := .1
	_, a := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}}, "reference")
	_, b := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}}, "selected")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b.Operation.ID, ReferenceID: a.Operation.ID, ShowOverlay: true}, "compare")
	o := s.Snapshot().Recording.ReferenceOverlay
	left, right := false, false
	for _, p := range o.Scope.Traces[0].Points {
		if p.Time == .05 {
			left = left || p.Value == 5
			right = right || p.Value == 0
		}
	}
	if o.PlotReduced || !left || !right {
		t.Fatal("reference ideal edge sides lost or interpolated")
	}
}
