package main

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func captureForControls(t *testing.T, f *webFixture, duration float64, probes []string) string {
	t.Helper()
	f.command("simulation.verify", map[string]any{"duration_s": duration, "sample_step_s": .001, "probe_ids": probes})
	id := f.state().LastOperation
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, op := range f.state().Operations {
			if op.ID == id && op.Status == "completed" {
				return id
			}
			if op.ID == id && (op.Status == "failed" || op.Status == "canceled") {
				t.Fatalf("capture %s: %+v", id, op)
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("capture did not finish")
	return ""
}

func TestBrowserRecordedControlsPreserveExactRowsAndLiveCircuit(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.restart", struct{}{})
	f.wait()
	f.command("simulation.step", workbench.StepParams{Duration: .02})
	id := captureForControls(t, f, .1, []string{"v:C1", "i:C1"})
	f.command("operation.select", workbench.IDParams{ID: id})
	f.command("recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	before := f.state()
	rows := f.command("operation.samples", workbench.VerificationPageParams{ID: id, Limit: 200}).Result
	index := 20
	f.command("recording.seek", workbench.RecordingSeekParams{Index: &index})
	r := f.state().Recording
	if r.Index != index || math.Abs(r.Time-.02) > 1e-12 || len(r.Readings) != 2 {
		t.Fatal("exact index seek did not select captured sample", r)
	}
	f.command("recording.step", workbench.RecordingStepParams{Direction: "previous"})
	if f.state().Recording.Index != index-1 {
		t.Fatal("previous did not move exactly one stored row")
	}
	f.command("recording.seek", map[string]any{"time_s": "30 ms"})
	f.command("recording.speed.set", workbench.RecordingPlayParams{Speed: .05})
	f.command("recording.window.set", map[string]any{"start_s": "10.1 ms", "end_s": "20.9 ms"})
	f.command("recording.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}})
	f.command("recording.cursor.set", map[string]any{"operation_id": id, "cursor": "a", "time_s": "15 ms"})
	f.command("recording.statistics.set", map[string]any{"id": id, "probe_id": "v:C1", "start_s": "10 ms", "end_s": "20 ms"})
	r = f.state().Recording
	if r.Index != 30 || r.Speed != .05 || len(r.Scope.Traces) != 1 || r.Scope.Window.Start != .0101 || r.Scope.Window.End != .0209 || r.Statistics == nil || r.Scope.A == nil || !r.Scope.A.Valid {
		t.Fatal("recorded interval, probes or exact cursor lost", r)
	}
	if r.Scope.Traces[0].Points[0].Time != .011 || r.Scope.Traces[0].Points[len(r.Scope.Traces[0].Points)-1].Time != .02 {
		t.Fatal("plot borrowed rows outside the requested margins")
	}
	f.command("recording.play", workbench.RecordingPlayParams{Speed: 10})
	deadline := time.Now().Add(time.Second)
	for f.state().Recording.Playing && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	r = f.state().Recording
	if r.Playing || r.Index != r.Total-1 {
		t.Fatal("playback did not stop at the final captured row")
	}
	// The Play action replays from the start after reaching the end.
	f.command("recording.play", workbench.RecordingPlayParams{Speed: .001})
	f.command("recording.pause", struct{}{})
	if f.state().Recording.Index != 0 {
		t.Fatal("replay did not restart at first row")
	}
	after := f.state()
	if !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID || before.Document.Dirty != after.Document.Dirty {
		t.Fatal("recorded viewer altered live simulation or dirtied the document")
	}
	if !reflect.DeepEqual(rows, f.command("operation.samples", workbench.VerificationPageParams{ID: id, Limit: 200}).Result) {
		t.Fatal("viewer altered saved rows")
	}
	// A draft from an older tab must not seek a newly selected capture.
	draft := f.state()
	f.command("recording.view.set", workbench.RecordingViewParams{Mode: "live"})
	raw, _ := json.Marshal(workbench.Request{Version: 1, ID: token(), Key: token(), Command: "recording.seek", Params: json.RawMessage(`{"index":10}`), ExpectedCommandRevision: &draft.CommandRevision, ExpectedDocumentGeneration: &draft.Document.Generation})
	var reply workbench.Response
	response := f.request("POST", "/api/command", raw, "")
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil || reply.Error == nil || reply.Error.Code != "conflict" {
		t.Fatal("stale recorded position accepted", response.Body.String())
	}
}

func TestBrowserDCRecordingExposesOneExactPoint(t *testing.T) {
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "led", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.wait()
	id := captureForControls(t, f, 0, []string{"v:R1"})
	f.command("operation.select", workbench.IDParams{ID: id})
	f.command("recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	r := f.state().Recording
	if r.Duration != 0 || r.Total != 1 || r.Index != 0 || len(r.Scope.Traces) != 1 || len(r.Scope.Traces[0].Points) != 1 || len(r.Readings) != 1 {
		t.Fatal("DC capture did not expose one drawable exact point", r)
	}
}
