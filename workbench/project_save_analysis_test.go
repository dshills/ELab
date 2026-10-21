package workbench

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveDuringVerificationKeepsCompletedCaptureAndLaterCompletionDirty(t *testing.T) {
	for _, finishBeforeSave := range []bool{false, true} {
		t.Run(map[bool]string{false: "completion-after-publication", true: "completion-before-publication"}[finishBeforeSave], func(t *testing.T) {
			s := pausedRC(t)
			defer s.Close()
			duration := .02
			_, completed := verify(t, s, VerifyParams{Duration: &duration}, "reference")
			before := s.Snapshot()
			// Retain a running operation deterministically; real worker overlap
			// is also exercised through the packaged public agent.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s.mu.Lock()
			reference := s.operations[completed.Operation.ID].record
			metadata := cloneVerificationMetadata(reference.metadata)
			metadata.OperationID = "pending-test"
			pending := &verificationOperation{ctx: ctx, cancel: cancel, inputMetadata: metadata, summary: OperationSummary{ID: "pending-test", Kind: "verify", Status: "running"}}
			s.operations[pending.summary.ID] = pending
			s.operationOrder = append(s.operationOrder, pending.summary.ID)
			s.selectedOperation = pending.summary.ID
			s.mu.Unlock()
			entered, release := blockSavePreparation(t, s)
			defer release()
			path := filepath.Join(t.TempDir(), "during-verify.elab")
			promptLiveCommand(t, s, "project.save.start", SaveParams{Path: path})
			awaitLiveSampleBlock(t, entered)
			if inspectOperation(t, s, pending.summary.ID).Operation.Status != "running" || ctx.Err() != nil {
				t.Fatal("saving canceled the independent verification")
			}
			finish := func() {
				s.mu.Lock()
				s.finishVerificationLocked(pending, &verificationRecord{metadata: metadata, rows: reference.rows}, nil)
				s.mu.Unlock()
			}
			if finishBeforeSave {
				finish()
			}
			release()
			afterSave := awaitProjectStatus(t, s, "completed")
			if !finishBeforeSave {
				if afterSave.Document.Dirty {
					t.Fatal("saved completed content stayed dirty solely for pending work")
				}
				finish()
			}
			after := s.Snapshot()
			if !after.Document.RecordingsDirty || !after.Document.Dirty || after.Document.RecordingCount != 2 || !reflect.DeepEqual(after.Transient, before.Transient) || after.RunID != before.RunID {
				t.Fatal("later completion was lost, marked saved, or changed live physics")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			document, err := decodeDocument(data)
			if err != nil || len(document.Recordings.Entries) != 1 || document.Recordings.Selected != -1 || document.Recordings.Viewing {
				t.Fatalf("saved a partial operation or invalid selection: %v", err)
			}
			if !reflect.DeepEqual(document.Recordings.Entries[0].Data.Metadata, *completed.Metadata) || !reflect.DeepEqual(document.Recordings.Entries[0].Data.Rows, reference.rows) {
				t.Fatal("saving during work changed the earlier complete recording")
			}
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
			if state := settled(t, fresh); state.Document.Dirty || state.Document.RecordingCount != 1 || !reflect.DeepEqual(state.Components, before.Components) {
				t.Fatal("saved in-flight snapshot does not reopen cleanly")
			}
		})
	}
}

func TestSaveDuringSweepOmitsPendingCurveAndKeepsLaterCompletionDirty(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	entered, release := make(chan struct{}), make(chan struct{})
	s.sweepCompute = func(ctx context.Context, plan sweepPlan, progress func(int)) (*SweepResult, error) {
		close(entered)
		select {
		case <-release:
			return computeDCSweep(ctx, plan, progress)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "sweep")
	awaitLiveSampleBlock(t, entered)
	path := filepath.Join(t.TempDir(), "during-sweep.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	if s.Snapshot().Document.Dirty {
		t.Fatal("pending curve alone marks the saved schematic dirty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := decodeDocument(data)
	if err != nil || document.Version != 4 || document.Sweeps != nil {
		t.Fatalf("saved a pending sweep: %v", err)
	}
	close(release)
	after := awaitSweep(t, s, "completed")
	if !after.Document.RecordingsDirty || !after.Document.Dirty || after.Document.RecordingCount != 1 || after.Document.FormatVersion != 5 {
		t.Fatal("completed curve was dropped or silently marked saved")
	}
	later, err := os.ReadFile(path)
	if err != nil || string(later) != string(data) {
		t.Fatal("later analysis completion rewrote the manual save")
	}
}
