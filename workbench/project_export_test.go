package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func blockExportPreparation(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.projectExport = func(ctx context.Context, capture exportCapture, progress func(string, int)) (*preparedFile, error) {
		progress("encoding", 0)
		close(entered)
		<-release
		return prepareProjectExport(ctx, capture, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func TestBlockedSVGExportCapturesOriginalDrawingAndPreservesNewEdits(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockExportPreparation(t, s)
	defer release()
	path := filepath.Join(t.TempDir(), "captured.svg")
	admitted := promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: path}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "R1", Position: Point{300, -200}})
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if !strings.Contains(string(data), "R1 · 10 kΩ") || strings.Contains(string(data), "20 kΩ") {
		t.Fatal("SVG includes later source value edit")
	}
	if after.LastExport.RunID != before.RunID || after.LastExport.ElectricalRevision != before.ElectricalRevision || after.LastExport.DocumentGeneration != before.Document.Generation || after.LastExport.SHA256 != hex.EncodeToString(digest[:]) || after.ProjectWork.ID != admitted.ProjectWork.ID {
		t.Fatal("captured artifact loses source identity")
	}
	if after.Document != edited.Document || !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.Transient, edited.Transient) || !reflect.DeepEqual(after.UndoEntry, edited.UndoEntry) || after.View != edited.View {
		t.Fatal("export overwrites new editing/physics/history")
	}
}
func TestBlockedScopeExportKeepsCapturedWindowProbesAndSamples(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step")
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1", "i:C1"}}, "probes")
	call(t, s, "scope.window.set", ScopeWindow{Mode: "fixed", Start: .004, End: .01}, "window")
	before := s.Snapshot()
	entered, release := blockExportPreparation(t, s)
	defer release()
	path := filepath.Join(t.TempDir(), "scope.csv")
	promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "scope-csv", Path: path})
	awaitLiveSampleBlock(t, entered)
	call(t, s, "simulation.step", StepParams{Duration: .005}, "later-step")
	promptLiveCommand(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:R1"}})
	promptLiveCommand(t, s, "scope.window.set", ScopeWindow{Mode: "fixed", Start: .01, End: .015})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	start, _ := strconv.ParseFloat(rows[1][3], 64)
	if len(rows) != 8 || rows[0][7] != "v:C1 [V]" || rows[0][8] != "i:C1 [A]" || start != .004 {
		t.Fatal("CSV replaces captured window/probes or includes new samples", rows)
	}
	last := rows[len(rows)-1]
	timeValue, _ := strconv.ParseFloat(last[6], 64)
	voltage, _ := strconv.ParseFloat(last[7], 64)
	if timeValue != .01 || voltage != before.Transient.Sample.Voltage || !reflect.DeepEqual(after.Transient, edited.Transient) || !reflect.DeepEqual(after.Scope, edited.Scope) {
		t.Fatal("CSV loses retained endpoint or mutates current scope/physics")
	}
}
func TestBlockedRecordedExportRetainsAssetAfterLiveRecordRemoval(t *testing.T) {
	for _, kind := range []string{"verification-json", "verification-csv"} {
		t.Run(kind, func(t *testing.T) {
			s := pausedRC(t)
			defer s.Close()
			duration := .01
			_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify")
			entered, release := blockExportPreparation(t, s)
			defer release()
			path := filepath.Join(t.TempDir(), kind)
			promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: kind, ID: record.Operation.ID, Path: path})
			awaitLiveSampleBlock(t, entered)
			promptLiveCommand(t, s, "recording.keep.set", RecordingKeepParams{ID: record.Operation.ID, Keep: false})
			short := .001
			for i := 0; i < MaxVerificationOperations; i++ {
				verify(t, s, VerifyParams{Duration: &short}, "new-record-"+strconv.Itoa(i))
			}
			s.mu.Lock()
			evicted := s.operations[record.Operation.ID] == nil
			s.mu.Unlock()
			if !evicted {
				t.Fatal("fixture did not evict the original unkept asset")
			}
			promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
			before := s.Snapshot()
			release()
			after := awaitProjectStatus(t, s, "completed")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "verification-json" {
				var d VerificationDocument
				if err := json.Unmarshal(data, &d); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(d.Metadata.Graph, record.Metadata.Graph) || len(d.Rows) != 11 {
					t.Fatal("recorded JSON substitutes current input")
				}
			} else {
				rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
				if err != nil || len(rows) != 12 || rows[1][0] != record.Operation.ID {
					t.Fatal("recorded CSV loses original asset")
				}
			}
			if after.LastExport.OperationID != record.Operation.ID || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
				t.Fatal("record export changes current project or source identity")
			}
		})
	}
}
func TestExportCancellationPreservesDestinationAndLastArtifact(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path := filepath.Join(t.TempDir(), "existing.svg")
	original := []byte("existing artifact\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	entered, release := blockExportPreparation(t, s)
	defer release()
	before := s.Snapshot()
	admitted := promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: path, Overwrite: true}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
	release()
	after := awaitProjectStatus(t, s, "canceled")
	data, _ := os.ReadFile(path)
	temps, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".elab-export-*"))
	if string(data) != string(original) || len(temps) != 0 || after.LastExport != before.LastExport || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("canceled export replaces destination/current artifact or changes project")
	}
}
func TestExportPublishingFencesCancellationAndPreservesReplacement(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	path := filepath.Join(t.TempDir(), "captured.svg")
	entered, release := blockSavePublication(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: path}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID}, "operation-committing")
	refuseStepWork(t, s, "project.save.start", SaveParams{Path: filepath.Join(t.TempDir(), "other.elab")}, "busy")
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	replaced := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.Associated == nil || *after.ProjectWork.Associated || after.Document != replaced.Document || !reflect.DeepEqual(after.Components, replaced.Components) || after.LastExport.DocumentGeneration != before.Document.Generation {
		t.Fatal("publishing export changes/relabels replacement")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("captured artifact missing", err)
	}
}

func TestPreparedExportRejectsDestinationChangedBeforePublication(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	path := filepath.Join(t.TempDir(), "existing.svg")
	if err := os.WriteFile(path, []byte("first artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.mu.Lock()
	s.projectExport = func(ctx context.Context, capture exportCapture, progress func(string, int)) (*preparedFile, error) {
		prepared, err := prepareProjectExport(ctx, capture, progress)
		close(entered)
		<-release
		return prepared, err
	}
	s.mu.Unlock()
	before := s.Snapshot()
	promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: path, Overwrite: true})
	awaitLiveSampleBlock(t, entered)
	outside := []byte("second outside artifact")
	if err := os.WriteFile(path, outside, 0600); err != nil {
		t.Fatal(err)
	}
	unblock()
	after := awaitProjectStatus(t, s, "failed")
	data, _ := os.ReadFile(path)
	if after.ProjectWork.Error.Code != "file-conflict" || string(data) != string(outside) || after.Document != before.Document || after.LastExport != nil {
		t.Fatal("prepared export replaces a newer destination or changes project")
	}
	temps, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".elab-export-*"))
	if len(temps) != 0 {
		t.Fatal("failed prepared export leaks temporary")
	}
}
