package workbench

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecordedTransitionRC1090BoundsAndOwnership(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .04
	_, capture := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "rise-capture")
	p := RecordingTimingParams{ID: capture.Operation.ID, ProbeID: "v:C1", Level: "500 mV", SecondLevel: "4.5 V", Direction: "falling", End: duration, Limit: 1}
	before := s.Snapshot()
	r := timingQuery(t, s, p)
	tr := r.Transitions
	expected := .01 * math.Log(9)
	if tr == nil || tr.Rise == nil || tr.Fall != nil || tr.Lower != .5 || tr.Upper != 4.5 || r.Total != 0 {
		t.Fatal("missing RC traversal", tr)
	}
	if !(tr.Rise.Minimum < expected && expected < tr.Rise.Maximum) || tr.Rise.First.Before.Index != 1 || tr.Rise.Second.Before.Index != 23 {
		t.Fatal("RC 10–90% outside saved brackets", tr.Rise, expected)
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("query mutated state")
	}
	// Reversing threshold entry order does not reverse rise/fall meaning.
	reversed := p
	reversed.Level, reversed.SecondLevel = p.SecondLevel, p.Level
	if !reflect.DeepEqual(tr, timingQuery(t, s, reversed).Transitions) {
		t.Fatal("threshold entry order changed traversal")
	}
	call(t, s, "recording.timing.set", p, "set-transition")
	cached := s.Snapshot().Recording.Timing
	if !reflect.DeepEqual(cached, r) {
		t.Fatal("configured/query mismatch")
	}
	cached.Transitions.Rise.First.Before.Value = 999
	r.Transitions.Rise.Second.After.Time = 999
	if s.Snapshot().Recording.Timing.Transitions.Rise.First.Before.Value == 999 || timingQuery(t, s, p).Transitions.Rise.Second.After.Time == 999 {
		t.Fatal("transition brackets alias stored data")
	}
	clipped := p
	clipped.Start = .002
	if timingQuery(t, s, clipped).Transitions.Rise != nil {
		t.Fatal("invented low arrival at interval boundary")
	}
	path := filepath.Join(t.TempDir(), "transition.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-transition")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "load-transition")
	if loaded.Snapshot().Recording.Timing != nil {
		t.Fatal("derived session settings persisted")
	}
	reopened := p
	reopened.ID = loaded.Snapshot().Recording.ID
	if !reflect.DeepEqual(timingQuery(t, s, p).Transitions, timingQuery(t, loaded, reopened).Transitions) {
		t.Fatal("saved raw capture changed timing")
	}
	initial := 5.
	_, capture = verify(t, s, VerifyParams{Duration: &duration, InputMode: "discharge", InitialVoltage: &initial, ProbeIDs: []string{"v:C1"}}, "fall-capture")
	p.ID = capture.Operation.ID
	p.Direction = "rising"
	tr = timingQuery(t, s, p).Transitions
	if tr.Fall == nil || tr.Rise != nil || tr.Fall.Minimum > expected || tr.Fall.Maximum < expected {
		t.Fatal("RC falling 90–10%", tr)
	}
}

func TestRecordedTransitionSampleSequences(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		times, values                            []float64
		riseFirst, riseLast, fallFirst, fallLast int
	}{
		{"one-row", []float64{0, .001, .002}, []float64{0, 5, 0}, 0, 0, 1, 1},
		{"ideal-duplicate-times", []float64{0, 0, .001, .001}, []float64{0, 5, 5, 0}, 0, 0, 2, 2},
		{"aborted-rise", []float64{0, .001, .002, .003, .004, .005}, []float64{0, 1, 0, 1, 2, 5}, 2, 4, -1, -1},
		{"aborted-fall", []float64{0, .001, .002, .003, .004, .005}, []float64{5, 4, 5, 4, 3, 0}, -1, -1, 2, 4},
		{"start-level-plateau", []float64{0, .001, .002, .003}, []float64{0, .5, .5, 4.5}, 0, 2, -1, -1},
		{"started-between-levels", []float64{0, .001, .002, .003}, []float64{2, 5, 2, 0}, -1, -1, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p := timingFixture(t)
			defer s.Close()
			pulseRows(t, s, p.ID, tc.times, tc.values)
			p.Level, p.SecondLevel, p.Direction = ".5 V", "4.5 V", "both"
			result := timingQuery(t, s, p)
			for _, check := range []struct {
				spacing     *TimingSpacing
				first, last int
			}{{result.Transitions.Rise, tc.riseFirst, tc.riseLast}, {result.Transitions.Fall, tc.fallFirst, tc.fallLast}} {
				if check.first < 0 {
					if check.spacing != nil {
						t.Fatal("invented traversal", check.spacing)
					}
					continue
				}
				if check.spacing == nil || check.spacing.First.Before.Index != check.first || check.spacing.Second.Before.Index != check.last {
					t.Fatal("wrong traversal sequence", check.spacing, check.first, check.last)
				}
				expectedMin := math.Max(0, tc.times[check.last]-tc.times[check.first+1])
				expectedMax := tc.times[check.last+1] - tc.times[check.first]
				nearTiming(t, "minimum", check.spacing.Minimum, expectedMin)
				nearTiming(t, "maximum", check.spacing.Maximum, expectedMax)
			}
			// Filtering and arrival paging do not remove transition summaries.
			p.Direction, p.Offset, p.Limit = "falling", 0, 1
			if !reflect.DeepEqual(result.Transitions, timingQuery(t, s, p).Transitions) {
				t.Fatal("filter/page changes traversal")
			}
		})
	}
}

func TestRecordedTransitionValidationAndDisable(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	p.SecondLevel = "4.5 V"
	call(t, s, "recording.timing.set", p, "valid-transition")
	for _, level := range []string{"2500 mV", "1 A", "NaN", "1e-400 V", " ", "1e13 V"} {
		bad := p
		bad.SecondLevel = level
		before := s.Snapshot()
		refuse(t, s, "recording.timing", bad, "invalid-params")
		refuse(t, s, "recording.timing.set", bad, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid second level changed state", level)
		}
	}
	p.SecondLevel = ""
	legacy := timingQuery(t, s, p)
	if legacy.Transitions != nil {
		t.Fatal("disabled transition remained")
	}
	call(t, s, "recording.timing.set", p, "disable-transition")
	if !reflect.DeepEqual(legacy, s.Snapshot().Recording.Timing) {
		t.Fatal("disabled path differs from original single-threshold timing")
	}
	badClear := RecordingTimingParams{ID: p.ID, Level: "0", SecondLevel: "1", Direction: "both"}
	refuse(t, s, "recording.timing.set", badClear, "invalid-params")
}
