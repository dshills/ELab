package workbench

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRecordingImportRequiresTinyFinalSample(t *testing.T) {
	for _, duration := range []float64{1e-20, 1e-12, .001 + .5e-12, math.Nextafter(.001, math.Inf(1))} {
		t.Run(number(duration), func(t *testing.T) {
			testRecordingMissingFinal(t, duration)
		})
	}
}

func testRecordingMissingFinal(t *testing.T, duration float64) {
	t.Helper()
	s := pausedRC(t)
	defer s.Close()
	_, view := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "tiny-grid")
	s.mu.Lock()
	r := s.operations[view.Operation.ID].record
	d := VerificationDocument{Metadata: cloneVerificationMetadata(r.metadata), Rows: append([]VerificationRow{}, r.rows...)}
	s.mu.Unlock()
	if len(d.Rows) < 2 || d.Rows[len(d.Rows)-1].Time != duration {
		t.Fatal("expected actual startup and final rows", d.Rows)
	}
	complete, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(complete); err != nil {
		t.Fatal("complete tiny recording refused", err)
	}
	d.Rows = d.Rows[:len(d.Rows)-1]
	d.Metadata.SampleCount = len(d.Rows)
	points := make([]ScopePoint, len(d.Rows))
	for i, row := range d.Rows {
		points[i] = ScopePoint{Time: row.Time, Value: row.Values[0]}
	}
	d.Metadata.Statistics = []ScopeStatistics{scopeStatistics(points, "v:C1")}
	incomplete, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRecording(incomplete); err == nil {
		t.Fatal("recording missing its final sample accepted as complete")
	}
	project, err := json.Marshal(ProjectRecordings{Entries: []ProjectRecording{{Data: d, ModelHashes: recordingModelHashes(d.Metadata)}}, Selected: 0, Viewing: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateProjectRecordings(project); err == nil {
		t.Fatal("embedded incomplete recording accepted")
	}
}

func TestRecordingGridToleranceDoesNotMergeNeighbors(t *testing.T) {
	for _, spacing := range []float64{1e-20, 1e-12, 5e-13, math.Nextafter(.001, math.Inf(1)) - .001, 1e-6, .001} {
		if !recordingGridTimeMatches(.001, .001, spacing) {
			t.Fatal("exact grid row refused", spacing)
		}
		if recordingGridTimeMatches(.001-spacing, .001, spacing) {
			// At spacings below one ULP the subtraction is the same timestamp;
			// there is no distinct binary64 neighbor to distinguish.
			if .001-spacing != .001 {
				t.Fatal("distinct neighboring grid times merged", spacing)
			}
		}
	}
	if !recordingGridTimeMatches(.02-5e-13, .02, .001) {
		t.Fatal("ordinary decimal timestamp compatibility lost")
	}
}

func TestRecordingGridBoundaryRoundTripAcrossEngines(t *testing.T) {
	for _, test := range []struct {
		example      string
		duration     float64
		boundedError bool
	}{
		{"example.rc", math.Nextafter(.001, math.Inf(1)), false},
		{"example.npn-pulse", math.Nextafter(.05, math.Inf(1)), false},
		{"example.rectifier", math.Nextafter(.001, math.Inf(1)), true},
		{"example.rectifier", .001001, false},
		{"example.flasher", math.Nextafter(.001, math.Inf(1)), false},
	} {
		t.Run(test.example+"/"+number(test.duration), func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, test.example, ReplaceParams{}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			settled(t, s)
			before := s.Snapshot()
			response := call(t, s, "simulation.verify", VerifyParams{Duration: &test.duration}, "boundary")
			view := awaitOperation(t, s, response.Result.(Snapshot).LastOperation)
			if test.boundedError {
				s.mu.Lock()
				hasRecord := s.operations[view.Operation.ID].record != nil
				s.mu.Unlock()
				if view.Operation.Status != "failed" || view.Operation.Error == nil || view.Operation.Error.Code != "solve-failed" || !strings.Contains(view.Operation.Error.Message, "mixed timestep too small") || hasRecord || view.Operation.KeptInProject {
					t.Fatal("expected existing integration-bound refusal without a partial record", view.Operation.Error)
				}
				after := s.Snapshot()
				if before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
					t.Fatal("failed verification changed live source/history")
				}
				return
			}
			if view.Operation.Status != "completed" {
				t.Fatal("verification failed", view.Operation.Error)
			}
			s.mu.Lock()
			encoded, err := encodeVerificationExport(s.operations[view.Operation.ID].record, "json")
			s.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeRecording(encoded); err != nil {
				t.Fatal("engine's complete export refused", err)
			}
		})
	}
}

func TestVerificationOutputGridUsesRepresentedEndpoints(t *testing.T) {
	step := 1e-6
	for _, test := range []struct {
		name     string
		duration float64
		steps    int
	}{
		{"limit", .001, 1000},
		{"below-limit", math.Nextafter(.001, 0), 1000},
		{"short-grid", .000063, 63},
		{"distinct-tail", math.Nextafter(.000063, math.Inf(1)), 64},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := pausedRC(t)
			defer s.Close()
			before := s.Snapshot()
			_, view := verify(t, s, VerifyParams{Duration: &test.duration, Step: &step, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "grid")
			if view.Operation.TotalSteps != test.steps || view.Operation.CompletedSteps != test.steps || view.Metadata.SampleCount != test.steps+1 {
				t.Fatalf("wrong output grid: %+v, samples %d", view.Operation, view.Metadata.SampleCount)
			}
			s.mu.Lock()
			record := s.operations[view.Operation.ID].record
			encoded, err := encodeVerificationExport(record, "json")
			s.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeRecording(encoded)
			if err != nil {
				t.Fatal("complete recording refused on import", err)
			}
			for i, row := range decoded.rows {
				at := math.Min(test.duration, float64(i)*step)
				want := 5 * (1 - math.Exp(-at/.01))
				if row.Time != at || math.Abs(row.Values[0]-want) > 1e-12 {
					t.Fatalf("row %d: got %+v, want time %.17g voltage %.17g", i, row, at, want)
				}
			}
			var document VerificationDocument
			if err := json.Unmarshal(encoded, &document); err != nil {
				t.Fatal(err)
			}
			project, err := json.Marshal(ProjectRecordings{Entries: []ProjectRecording{{Data: document, ModelHashes: recordingModelHashes(document.Metadata)}}, Selected: 0, Viewing: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := validateProjectRecordings(project); err != nil {
				t.Fatal("complete embedded recording refused", err)
			}
			after := s.Snapshot()
			if !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID || before.Paused != after.Paused {
				t.Fatal("offline recording changed live physics or circuit")
			}
		})
	}
}

func TestVerificationOutputGridStillRefusesDistinctIntervalAboveLimit(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	duration, step := math.Nextafter(.001, math.Inf(1)), 1e-6
	refuse(t, s, "simulation.verify", VerifyParams{Duration: &duration, Step: &step}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("over-limit recording request changed state")
	}
}
