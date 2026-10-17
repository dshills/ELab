package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestRecordingImportContractsAreOwnedAndKeepDiscovery(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range []string{"verification.import", "verification.import.start"} {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		c := schema["x-command-contract"].(map[string]any)
		if c["scope"] != "captured-recording-import" || len(c["effects"].([]any)) == 0 || len(c["errors"].([]any)) == 0 || schema["x-import-format"] == nil || schema["x-import-capacity"] == nil {
			t.Fatal("missing import state/format/capacity contract", id)
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller corrupted later import schema", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("import details changed ordinary catalog or output definitions")
	}
}

func TestFailedWaitingImportRetryDoesNotReadRepairedFileUntilNewKey(t *testing.T) {
	fixture := importFixture(t)
	valid, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "repair.json")
	if err := os.WriteFile(path, []byte("not a recorded capture"), 0600); err != nil {
		t.Fatal(err)
	}
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	raw, _ := json.Marshal(RecordingImportParams{Path: path})
	request := Request{Version: 1, ID: "import", Command: "verification.import", Params: raw, ExpectedCommandRevision: &before.CommandRevision, Key: "failed"}
	failed := s.Dispatch(context.Background(), request, "importer")
	if failed.Error == nil || failed.Error.Code != "invalid-recording" {
		t.Fatal("expected admitted validation failure", failed.Error)
	}
	after := s.Snapshot()
	if after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Operations, before.Operations) || !reflect.DeepEqual(after.Recording, before.Recording) {
		t.Fatal("failed import changed live data or membership")
	}
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	retry := s.Dispatch(context.Background(), request, "importer")
	if !retry.Duplicate || retry.Error == nil || retry.Error.Code != "invalid-recording" || len(s.Snapshot().Operations) != 0 {
		t.Fatal("owned failed retry reread corrected file or attached a duplicate")
	}
	retry.Result.(Snapshot).ProjectWork.Error.Message = "caller mutation"
	if inspectProjectWork(t, s, after.ProjectWork.ID).Error.Message == "caller mutation" {
		t.Fatal("retry result aliases live job error")
	}
	request.Key, request.ID = "corrected", "corrected"
	current := s.Snapshot().CommandRevision
	request.ExpectedCommandRevision = &current
	completed := s.Dispatch(context.Background(), request, "importer")
	if completed.Error != nil || s.Snapshot().Document.RecordingCount != 1 || s.Snapshot().ProjectWork.ID == after.ProjectWork.ID {
		t.Fatal("new intent did not read and attach the corrected capture", completed.Error)
	}
	view := inspectOperation(t, s, s.Snapshot().ProjectWork.ResultOperationID)
	if view.Operation.SourceUnchanged || !s.Snapshot().Document.RecordingsDirty {
		t.Fatal("import claimed live source association or missed kept membership")
	}
}

func TestRecordingImportRespectsPreviewIntentDuringValidation(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	duration := .003
	_, existing := verify(t, s, VerifyParams{Duration: &duration}, "existing")
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	preview := awaitPreview(t, s, "completed").PrintPreview
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ViewerSelected == nil || *after.ProjectWork.ViewerSelected || after.SelectedOperation != existing.Operation.ID || after.Document.RecordingCount != 2 || !reflect.DeepEqual(after.PrintPreview, preview) || !reflect.DeepEqual(after.Recording, before.Recording) || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("pending import overrode later captured-preview/viewer intent")
	}
}
