package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func sweepImportFixture(t *testing.T) (string, *SweepResult) {
	t.Helper()
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "curve")
	state := awaitSweep(t, s, "completed")
	path := filepath.Join(t.TempDir(), "curve.json")
	call(t, s, "sweep.export", VerificationExportParams{ID: state.DCSweep.Selected, Path: path, Format: "json"}, "export")
	return path, state.DCSweep.Result
}
func blockSweepImport(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.sweepImport = func(ctx context.Context, path string, progress func(string, int)) (*SweepRecord, [32]byte, error) {
		close(entered)
		<-release
		return prepareSweepImport(ctx, path, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func TestSweepImportPreservesNewerLiveCircuitAndRoundTrips(t *testing.T) {
	path, want := sweepImportFixture(t)
	s := pausedRC(t)
	defer s.Close()
	entered, release := blockSweepImport(t, s)
	defer release()
	promptLiveCommand(t, s, "sweep.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	call(t, s, "simulation.step", StepParams{Duration: .005}, "step")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	awaitTestInitialization(t, s)
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || before.RunID != after.RunID || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.Document.Path != after.Document.Path {
		t.Fatal("import changes live source, time, history or file association")
	}
	if after.Document.RecordingCount != 1 || !after.Document.RecordingsDirty || !reflect.DeepEqual(after.DCSweep.Result, want) || !after.DCSweep.Visible {
		t.Fatal("missing captured data or membership")
	}
	op := call(t, s, "sweep.inspect", IDParams{ID: after.DCSweep.Selected}, "inspect").Result.(SweepInspection)
	data, _ := os.ReadFile(path)
	digest := sha256.Sum256(data)
	if op.Operation.Kind != "sweep-import" || op.Operation.SourceUnchanged || op.Operation.SourcePath != path || op.Operation.SourceSHA256 != hex.EncodeToString(digest[:]) || op.Operation.Actor != "test" {
		t.Fatal("import provenance/association incorrect", op.Operation)
	}
	op.Result.Points[0].Values[0] = 123
	if s.Snapshot().DCSweep.Result.Points[0].Values[0] == 123 {
		t.Fatal("import inspection aliases stored data")
	}
	call(t, s, "sweep.cursor.set", SweepCursorParams{Index: ptrSweepIndex(5)}, "cursor")
	saved := filepath.Join(t.TempDir(), "imported.elab")
	call(t, s, "project.save", SaveParams{Path: saved}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: saved}, "load")
	restored := settled(t, fresh)
	if restored.Document.Dirty || restored.Document.FormatVersion != 5 || !reflect.DeepEqual(restored.DCSweep.Result, want) || *restored.DCSweep.Cursor != 5 || restored.DCSweep.Operations[0].SourceUnchanged || restored.DCSweep.Operations[0].Kind != "project-sweep" {
		t.Fatal("imported curve does not round trip")
	}
}
func TestSweepImportOnBlankAndLaterViewerChoices(t *testing.T) {
	path, want := sweepImportFixture(t)
	s := New()
	defer s.Close()
	response := call(t, s, "sweep.import", RecordingImportParams{Path: path}, "blank").Result.(Snapshot)
	if response.Components == nil || response.Wires == nil {
		t.Fatal("waiting import response loses empty graph arrays")
	}
	duplicate := cloneMutationResponse(Response{Result: response})
	if got := duplicate.Result.(Snapshot); got.Components == nil || got.Wires == nil {
		t.Fatal("retry response loses empty graph arrays")
	}
	if len(s.Snapshot().Components) != 0 || !reflect.DeepEqual(s.Snapshot().DCSweep.Result, want) {
		t.Fatal("blank import replaces circuit")
	}
	for _, command := range []string{"sweep.visibility.set", "sweep.cursor.set", "sweep.trace.select", "recording.view.set"} {
		t.Run(command, func(t *testing.T) {
			live := pausedRC(t)
			defer live.Close()
			call(t, live, "sweep.import", RecordingImportParams{Path: path}, "first")
			entered, release := blockSweepImport(t, live)
			defer release()
			promptLiveCommand(t, live, "sweep.import.start", RecordingImportParams{Path: path})
			awaitLiveSampleBlock(t, entered)
			var p any
			switch command {
			case "sweep.visibility.set":
				p = SweepVisibilityParams{Visible: false}
			case "sweep.cursor.set":
				p = SweepCursorParams{Index: ptrSweepIndex(4)}
			case "sweep.trace.select":
				p = IDParams{ID: "v:D1"}
			case "recording.view.set":
				p = RecordingViewParams{Mode: "live"}
			}
			promptLiveCommand(t, live, command, p)
			before := live.Snapshot().DCSweep
			release()
			after := awaitProjectStatus(t, live, "completed")
			if *after.ProjectWork.ViewerSelected || after.DCSweep.Selected != before.Selected || after.DCSweep.Visible != before.Visible || after.DCSweep.TraceID != before.TraceID || !reflect.DeepEqual(after.DCSweep.Cursor, before.Cursor) || len(after.DCSweep.Operations) != 2 {
				t.Fatal("import steals later viewer choice")
			}
		})
	}
}
func TestSweepImportCancellationReplacementAndCapacity(t *testing.T) {
	path, _ := sweepImportFixture(t)
	for _, command := range []string{"project.work.cancel", "project.clear", "app.close"} {
		t.Run(command, func(t *testing.T) {
			s := NewExample()
			defer s.Close()
			settled(t, s)
			entered, release := blockSweepImport(t, s)
			defer release()
			admitted := promptLiveCommand(t, s, "sweep.import.start", RecordingImportParams{Path: path}).Result.(Snapshot)
			awaitLiveSampleBlock(t, entered)
			var p any = ReplaceParams{DiscardChanges: true}
			status := "superseded"
			if command == "project.work.cancel" {
				p = IDParams{ID: admitted.ProjectWork.ID}
				status = "canceled"
			}
			if command == "app.close" {
				status = "canceled"
			}
			promptLiveCommand(t, s, command, p)
			before := s.Snapshot()
			release()
			after := awaitProjectStatus(t, s, status)
			if after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || after.ProjectWork.ResultOperationID != "" {
				t.Fatal("discarded import attaches data")
			}
		})
	}
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	for i := 0; i < MaxDCSweepOperations-1; i++ {
		call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, fmt.Sprint(i))
		awaitSweep(t, s, "completed")
	}
	entered, release := blockSweepImport(t, s)
	defer release()
	promptLiveCommand(t, s, "sweep.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "last")
	before := awaitSweep(t, s, "completed")
	release()
	after := awaitProjectStatus(t, s, "failed")
	if after.ProjectWork.Error.Code != "record-limit" || !reflect.DeepEqual(after.DCSweep, before.DCSweep) {
		t.Fatal("import displaces kept curve at publication")
	}
	refuse(t, s, "sweep.import.start", RecordingImportParams{Path: path}, "record-limit")
}
func TestSweepImportRefusesMalformedAndIncompleteRecords(t *testing.T) {
	path, _ := sweepImportFixture(t)
	data, _ := os.ReadFile(path)
	for _, mutation := range []string{"missing-input", "null-residual", "missing-start", "axis", "engine", "unknown", "duplicate", "time-record"} {
		t.Run(mutation, func(t *testing.T) {
			var raw map[string]any
			_ = json.Unmarshal(data, &raw)
			m := raw["metadata"].(map[string]any)
			r := raw["result"].(map[string]any)
			p := r["points"].([]any)[0].(map[string]any)
			switch mutation {
			case "missing-input":
				delete(p, "input")
			case "null-residual":
				p["kcl_residual_a"] = nil
			case "missing-start":
				delete(r, "start_si")
			case "axis":
				p["input"] = .1
			case "engine":
				m["engine"] = "other"
			case "unknown":
				r["extra"] = true
			case "time-record":
				m["format"] = "elab.recording"
			}
			encoded, _ := json.Marshal(raw)
			if mutation == "duplicate" {
				encoded = append([]byte(`{"result":{},`), encoded[1:]...)
			}
			project, err := json.Marshal(map[string]any{"entries": []json.RawMessage{encoded}, "selected_index": 0, "trace_id": "i:D1", "viewing": true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := validateProjectSweeps(project); err == nil {
				t.Fatal("project load supplies omitted/corrupt captured values")
			}
			invalid := filepath.Join(t.TempDir(), "invalid.json")
			if err := os.WriteFile(invalid, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			s := New()
			defer s.Close()
			before := s.Snapshot()
			refuse(t, s, "sweep.import", RecordingImportParams{Path: invalid}, "invalid-sweep")
			if s.Snapshot().Document != before.Document || s.Snapshot().DCSweep != nil {
				t.Fatal("failed import mutates document")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := prepareSweepImport(ctx, path, func(string, int) {}); err != context.Canceled {
		t.Fatal("reader ignores cancellation", err)
	}
}
func TestTimeImportPreservesLaterSweepView(t *testing.T) {
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "sweep.visibility.set", SweepVisibilityParams{Visible: true})
	release()
	after := awaitProjectStatus(t, s, "completed")
	if *after.ProjectWork.ViewerSelected || !after.DCSweep.Visible {
		t.Fatal("time import ignores later sweep view")
	}
}
