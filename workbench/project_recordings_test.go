package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProjectRecordingsSaveReopenHashesAndFreshPausedRun(t *testing.T) {
	for _, example := range []string{"example.reset", "example.rc", "example.npn", "example.npn-pulse", "example.flasher", "example.switch-npn"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, example, ReplaceParams{}, "example")
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			path := filepath.Join(t.TempDir(), "experiment.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save-clean")
			if s.Snapshot().Document.Dirty {
				t.Fatal("initial save stayed dirty")
			}
			duration := 0.
			if IsTransientAnalysis(s.Snapshot().Analysis) {
				duration = .02
				call(t, s, "simulation.step", StepParams{Duration: .01}, "live-time")
			}
			before := s.Snapshot()
			_, verified := verify(t, s, VerifyParams{Duration: &duration}, "verify")
			after := s.Snapshot()
			if !after.Document.Dirty || !after.Document.RecordingsDirty || after.Document.SchematicDirty || after.Document.RecordingCount != 1 || before.RunID != after.RunID {
				t.Fatal("completed asset did not mark project dirty independently")
			}
			refuse(t, s, "project.clear", ReplaceParams{}, "unsaved-changes")
			call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
			call(t, s, "project.save", SaveParams{}, "save-record")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			document, err := decodeDocument(data)
			if err != nil || document.Version != 4 || len(document.Recordings.Entries) != 1 || document.Paused == nil || !*document.Paused {
				t.Fatalf("bad saved experiment: %v", err)
			}
			if len(document.Recordings.Entries[0].ModelHashes) == 0 || strings.Contains(string(data), path) {
				t.Fatal("missing model hashes or leaked source-host path")
			}
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "load")
			loaded := settled(t, fresh)
			if !loaded.Paused || loaded.Document.Dirty || loaded.Document.RecordingCount != 1 || loaded.Recording == nil || !loaded.Recording.Viewing || loaded.Recording.Playing || len(loaded.Operations) != 1 || loaded.Operations[0].SourceUnchanged {
				t.Fatal("reopen confused recorded and live state")
			}
			if loaded.Transient != nil && (loaded.Transient.Sample.Time != 0 || loaded.Transient.Stale) {
				t.Fatal("reopen resumed physical history or left startup stale")
			}
			if loaded.Result != nil && loaded.Result.Stale || loaded.NPN != nil && loaded.NPN.Stale {
				t.Fatal("paused DC startup was stale")
			}
			metadata := inspectOperation(t, fresh, loaded.Recording.ID).Metadata
			if !reflect.DeepEqual(metadata, verified.Metadata) {
				t.Fatal("project reopen changed captured result")
			}
			call(t, fresh, "recording.keep.set", RecordingKeepParams{ID: loaded.Recording.ID, Keep: false}, "exclude")
			if !fresh.Snapshot().Document.RecordingsDirty || fresh.Snapshot().Document.RecordingCount != 0 || fresh.Snapshot().Recording == nil {
				t.Fatal("exclude erased session replay or failed to mark dirty")
			}
			call(t, fresh, "recording.keep.set", RecordingKeepParams{ID: loaded.Recording.ID, Keep: true}, "include-again")
			if fresh.Snapshot().Document.Dirty {
				t.Fatal("restoring saved membership stayed dirty")
			}
		})
	}
}
func TestProjectRecordingsCorruptEntryAndDependencyHashProtectCurrentWork(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .01
	verify(t, s, VerifyParams{Duration: &duration}, "verify")
	path := filepath.Join(t.TempDir(), "good.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	good, _ := os.ReadFile(path)
	for _, bad := range []string{"hash", "row", "selection", "missing-field"} {
		t.Run(bad, func(t *testing.T) {
			var document map[string]any
			_ = json.Unmarshal(good, &document)
			recordings := document["recordings"].(map[string]any)
			entry := recordings["entries"].([]any)[0].(map[string]any)
			switch bad {
			case "hash":
				entry["model_sha256"].(map[string]any)["ideal-resistor.v1"] = "damaged"
			case "row":
				delete(entry["data"].(map[string]any)["rows"].([]any)[0].(map[string]any), "time_s")
			case "selection":
				recordings["selected_index"] = 9
			case "missing-field":
				delete(document, "paused")
			}
			data, _ := json.Marshal(document)
			path := filepath.Join(t.TempDir(), "bad.elab")
			_ = os.WriteFile(path, data, 0600)
			refuse(t, s, "project.load", LoadParams{Path: path}, "invalid-document")
		})
	}
}
func TestKeptRecordingCapacityRefusesInsteadOfEvictingSavedWork(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path, id := exportedRecording(t, s, .01, []string{"v:C1"})
	for i := 0; i < 7; i++ {
		call(t, s, "verification.import", RecordingImportParams{Path: path}, "import-"+string(rune('a'+i)))
	}
	if s.Snapshot().Document.RecordingCount != 8 || len(s.operationOrder) != 8 {
		t.Fatal("record capacity setup failed")
	}
	refuse(t, s, "verification.import", RecordingImportParams{Path: path}, "record-limit")
	refuse(t, s, "simulation.verify", VerifyParams{}, "record-limit")
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: id, Keep: false}, "exclude")
	call(t, s, "verification.import", RecordingImportParams{Path: path}, "replace-excluded")
	if s.operations[id] != nil || s.Snapshot().Document.RecordingCount != 8 || len(s.operationOrder) != 8 {
		t.Fatal("retention evicted kept data or exceeded bound")
	}
}
func TestProjectRecoveryIncludesRecordingsAndOffersBoundedExactPages(t *testing.T) {
	s := pausedRC(t)
	directory := t.TempDir()
	configureRecovery(t, s, directory, false)
	duration := .02
	_, verified := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	s.Close()
	target := New()
	defer target.Close()
	configureRecovery(t, target, directory, false)
	response := call(t, target, "recovery.inspect", choice, "inspect")
	inspection := response.Result.(RecoveryInspection)
	if inspection.Candidate.Recordings != 1 || len(inspection.Document.Recordings) != 1 || inspection.Document.Recordings[0].SampleCount != 21 {
		t.Fatal("recovery inspection omitted recording metadata")
	}
	params, _ := json.Marshal(RecoverySamplesParams{ID: choice.ID, SHA256: choice.SHA256, Recording: 0, Offset: 10, Limit: 1})
	page := target.Dispatch(context.Background(), Request{Version: 1, ID: "page", Command: "recovery.samples", Params: params}, "test")
	if page.Error != nil || page.Result.(VerificationPage).Total != 21 || page.Result.(VerificationPage).Rows[0].Time != .01 {
		t.Fatal("recovery page failed")
	}
	call(t, target, "recovery.restore", choice, "restore")
	restored := settled(t, target)
	if restored.Document.Path != "" || !restored.Document.Dirty || !restored.Paused || restored.Recording == nil || restored.Document.RecordingCount != 1 {
		t.Fatal("recovery lost project assets/startup")
	}
	if !reflect.DeepEqual(inspectOperation(t, target, restored.Recording.ID).Metadata, verified.Metadata) {
		t.Fatal("recovery changed captured data")
	}
}
func TestProjectLegacySchematicsRemainReadable(t *testing.T) {
	s := NewExample()
	defer s.Close()
	path := filepath.Join(t.TempDir(), "current.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	original, _ := os.ReadFile(path)
	for _, version := range []int{1, 2, 3} {
		var document map[string]any
		_ = json.Unmarshal(original, &document)
		document["version"] = version
		delete(document, "recordings")
		delete(document, "example_id")
		delete(document, "paused")
		if version < 3 {
			delete(document, "scope_probes")
		}
		if version == 1 {
			delete(document, "transient_settings")
		}
		data, _ := json.Marshal(document)
		if _, err := decodeDocument(data); err != nil {
			t.Fatalf("version %d: %v", version, err)
		}
	}
}

func TestProjectReplacementCancelsOldVerificationAndSaveRefusesPendingWork(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	duration := 5.
	response := call(t, s, "simulation.verify", VerifyParams{Duration: &duration}, "long-job")
	id := response.Result.(Snapshot).LastOperation
	s.mu.Lock()
	operation := s.operations[id]
	s.mu.Unlock()
	path := filepath.Join(t.TempDir(), "pending.elab")
	raw, _ := json.Marshal(SaveParams{Path: path})
	revision := s.Snapshot().Revision
	save := s.Dispatch(context.Background(), Request{Version: 1, ID: "save-pending", Command: "project.save", Params: raw, ExpectedRevision: &revision, Key: "save-pending"}, "test")
	if save.Error != nil && save.Error.Code == "conflict" {
		revision = save.Revision
		save = s.Dispatch(context.Background(), Request{Version: 1, ID: "save-pending", Command: "project.save", Params: raw, ExpectedRevision: &revision, Key: "save-pending"}, "test")
	}
	if save.Error == nil || save.Error.Code != "busy" {
		t.Fatalf("pending Verify save: %+v", save.Error)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("busy save published a file")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "replace")
	before := s.Snapshot()
	select {
	case <-operation.done:
	case <-time.After(time.Second):
		t.Fatal("replaced verification did not stop")
	}
	after := s.Snapshot()
	if len(after.Operations) != 0 || after.Document.RecordingCount != 0 || after.Revision != before.Revision || len(after.Components) != 0 {
		t.Fatal("late old verification changed replacement project")
	}
}
