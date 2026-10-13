package workbench

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRecordedExportsKeepCapturedDataAfterLiveEdits(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .01
	_, view := verify(t, s, VerifyParams{Duration: &duration}, "verified")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "changed-live")
	before := s.Snapshot()
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "record.json")
	csvPath := filepath.Join(directory, "record.csv")
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: jsonPath, Format: "json"}, "json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var document VerificationDocument
	if err := DecodeStrict(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.Metadata.Settings != view.Metadata.Settings || !reflect.DeepEqual(document.Metadata.Graph, view.Metadata.Graph) || len(document.Rows) != 11 || document.Metadata.Format != "elab.verification" || document.Metadata.Version != 1 {
		t.Fatal("export substituted current graph/settings/data")
	}
	call(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: csvPath, Format: "csv"}, "csv")
	data, err = os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rows) != 12 || rows[1][0] != view.Operation.ID || rows[1][3] != RCEngineID || rows[0][10] != "v:C1.1 [V]" {
		t.Fatal("record CSV missing identity, units or values")
	}
	after := s.Snapshot()
	if before.Transient.Sample != after.Transient.Sample || before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision {
		t.Fatal("record export changed editor or physical state")
	}
	if after.LastExport.OperationID != view.Operation.ID || after.LastExport.ElectricalRevision != view.Operation.ElectricalRevision {
		t.Fatal("artifact carries wrong source identity")
	}
	refuse(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: jsonPath, Format: "json"}, "file-exists")
	refuse(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: filepath.Join(directory, "bad"), Format: "exe"}, "invalid-params")
	saved := filepath.Join(directory, "project.elab")
	call(t, s, "project.save", SaveParams{Path: saved}, "save")
	refuse(t, s, "verification.export", VerificationExportParams{ID: view.Operation.ID, Path: saved, Format: "json", Overwrite: true}, "invalid-params")
	// JSON contains the source graph and model parameters, not source-host paths.
	data, _ = json.Marshal(document)
	if strings.Contains(string(data), directory) {
		t.Fatal("portable record leaked a host path")
	}
}
