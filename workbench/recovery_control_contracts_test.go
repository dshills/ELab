package workbench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var recoveryControlCommands = []string{"recovery.configure", "recovery.configure.start", "recovery.work.start", "recovery.scan", "recovery.checkpoint", "recovery.select"}

func TestRecoveryControlContractsKeepDiscoveryAndExistingDisableBounds(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range recoveryControlCommands {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		c := schema["x-command-contract"].(map[string]any)
		if c["scope"] != "recovery-controls-and-work" || len(c["effects"].([]any)) == 0 || schema["x-recovery-inventory"] == nil {
			t.Fatal("missing recovery control contract", id)
		}
		if id == "recovery.configure" || id == "recovery.configure.start" {
			p := inputProperty(schema, "interval_s")
			if p["$ref"] != TimeInputSchemaID || p["minimum"] != float64(1) || p["maximum"] != float64(60) || schema["x-existing-store-disable"].(map[string]any)["folder_access"] != false {
				t.Fatal("new contract dropped existing quantity/disable semantics", id)
			}
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller changed future recovery discovery", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("recovery details changed ordinary catalog/shared output schemas")
	}
}

func TestRecoverySelectionUsesCachedCatalogWithoutStorage(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "12k"}, "edit-template")
	call(t, s, "recovery.select", IDParams{ID: ""}, "clear-unconfigured")
	refuse(t, s, "recovery.select", IDParams{ID: "unknown"}, "invalid-params")
	directory := recoverySetupDirectory(t)
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 60}, "configure")
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	raw, err := os.ReadFile(filepath.Join(directory, choice.ID+".elab-recovery"))
	if err != nil {
		t.Fatal(err)
	}
	moved := directory + "-preserved"
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directory, []byte("owned unavailable folder"), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(directory); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(moved, directory); err != nil {
			t.Fatal(err)
		}
	}()
	before := s.Snapshot()
	call(t, s, "recovery.select", IDParams{ID: choice.ID}, "cached")
	call(t, s, "recovery.select", IDParams{ID: ""}, "clear")
	after := s.Snapshot()
	if after.Recovery.Selected != "" || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.ProjectWork, after.ProjectWork) || !reflect.DeepEqual(before.Recovery.Candidates, after.Recovery.Candidates) {
		t.Fatal("cached selection accessed storage or changed source/work/catalog")
	}
	stored, err := os.ReadFile(filepath.Join(moved, choice.ID+".elab-recovery"))
	if err != nil || !bytes.Equal(raw, stored) {
		t.Fatal("selection changed the preserved capsule", err)
	}
}

func TestDisabledManualCheckpointDeduplicatesAndRetiresRestoredCleanContent(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "manual.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	manual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	directory := recoverySetupDirectory(t)
	call(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 60}, "configure")
	awaitProjectStatus(t, s, "completed")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty")
	call(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "checkpoint"}, "checkpoint")
	first := awaitProjectStatus(t, s, "completed")
	if first.Recovery.Enabled || first.ProjectWork.RecoveryID == "" || len(first.Recovery.Candidates) != 1 || !first.Document.Dirty {
		t.Fatal("manual checkpoint did not publish while automatic scheduling was disabled")
	}
	call(t, s, "recovery.select", IDParams{ID: ""}, "clear")
	call(t, s, "recovery.checkpoint", struct{}{}, "same")
	if state := s.Snapshot(); state.ProjectWork.RecoveryID != "" || len(state.Recovery.Candidates) != 1 || state.Recovery.Selected != "" {
		t.Fatal("unchanged checkpoint duplicated data or overrode clear intent")
	}
	call(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "scan"}, "scan")
	scan := awaitProjectStatus(t, s, "completed")
	if scan.ProjectWork.Command != "recovery.scan" || scan.Recovery.Selected != "" || !*scan.ProjectWork.CatalogApplied {
		t.Fatal("scan did not retain explicit empty selection")
	}
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 60}, "enable")
	if s.Snapshot().Recovery.Selected != "" {
		t.Fatal("same-store refresh forgot explicit clear intent")
	}
	// Return to the saved contents without saving again; clean checkpoint must
	// retire the obsolete dirty recovery while leaving the manual file alone.
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "1k"}, "restore-content")
	if s.Snapshot().Document.Dirty {
		t.Fatal("fixture did not return to its actual saved content")
	}
	call(t, s, "recovery.checkpoint", struct{}{}, "clean")
	after := s.Snapshot()
	if after.ProjectWork.RecoveryID != "" || len(after.Recovery.Candidates) != 0 || after.Recovery.Selected != "" {
		t.Fatal("clean checkpoint failed to retire owned dirty capsules")
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(manual, actual) {
		t.Fatal("manual recovery work changed the explicit saved project", err)
	}
}
