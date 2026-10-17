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

func projectDetailedSchema(t *testing.T, s *Service, id string) (map[string]any, commandContractDetail) {
	t.Helper()
	raw, _ := json.Marshal(CapabilityQuery{Command: id})
	response := s.Dispatch(t.Context(), Request{Version: 1, ID: "discover", Command: "capabilities.list", Params: raw}, "agent")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	caps := response.Result.(map[string]any)["capabilities"].([]Capability)
	if len(caps) != 1 || caps[0].ID != id {
		t.Fatal("wrong detailed capability")
	}
	var schema map[string]any
	if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
		t.Fatal(err)
	}
	var contract commandContractDetail
	encoded, _ := json.Marshal(schema["x-command-contract"])
	if err := json.Unmarshal(encoded, &contract); err != nil {
		t.Fatal(err)
	}
	return schema, contract
}
func projectAdvertisesError(t *testing.T, s *Service, id, code, phase string) {
	t.Helper()
	_, contract := projectDetailedSchema(t, s, id)
	for _, failure := range contract.Errors {
		if failure.Code == code && failure.Phase == phase {
			return
		}
	}
	t.Fatal(id, "does not advertise observed error", code, phase)
}

func TestProjectDiscoveryOwnedPureBoundsAndCompletion(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	catalog, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range []string{"project.inspect", "project.clear", "project.save", "project.save.start", "project.load", "project.load.start", "project.work.inspect", "project.work.cancel"} {
		schema, contract := projectDetailedSchema(t, s, id)
		if contract.Version != 1 || contract.Scope != "project-files-and-work" || contract.Completion == nil || len(contract.Effects) == 0 || len(contract.Errors) == 0 || schema["$ref"] != nil {
			t.Fatal(id, "missing expanded contract")
		}
		jobs := schema["x-project-work"].(map[string]any)
		if jobs["retained_jobs"] != float64(MaxProjectOperations) || jobs["max_workers"] != float64(MaxProjectOperations) {
			t.Fatal(id, "job limits differ")
		}
		if strings.HasPrefix(id, "project.save") || strings.HasPrefix(id, "project.load") {
			p := inputProperty(schema, "path")
			f := schema["x-file-format"].(map[string]any)
			if p["x-max-utf8-bytes"] != float64(4096) || f["max_bytes"] != float64(MaxProjectBytes) || f["legacy_max_bytes"] != float64(MaxDocumentBytes) || f["default_write_version"] != float64(DocumentVersion) || f["read_version_maximum"] != float64(SweepDocumentVersion) || contract.Completion.Cancel != "project.work.cancel" || contract.Completion.CooperativeDeadlineSeconds != 30 {
				t.Fatal(id, "file bounds/completion drift")
			}
			for _, observer := range contract.Completion.Observe {
				if _, ok := capabilityIndex[observer]; !ok {
					t.Fatal("unknown observer", observer)
				}
			}
		}
		// A detailed response owns every mutable layer and cannot poison discovery.
		schema["x-project-work"].(map[string]any)["retained_jobs"] = 0
		againSchema, again := projectDetailedSchema(t, s, id)
		if again.Scope != contract.Scope || againSchema["x-project-work"].(map[string]any)["retained_jobs"] != float64(MaxProjectOperations) {
			t.Fatal("detailed schema alias")
		}
	}
	afterCatalog, _ := json.Marshal(Catalog())
	afterRegistry, _ := json.Marshal(SchemaRegistry())
	if string(catalog) != string(afterCatalog) || string(registry) != string(afterRegistry) || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery changes default catalog, registry or state")
	}
}

func TestProjectContractPathAndNullAdmissionMatchesRuntime(t *testing.T) {
	s := New()
	defer s.Close()
	for _, command := range []string{"project.load.start", "project.save.start"} {
		for _, path := range []string{" ", "bad\x00path", strings.Repeat("é", 2049)} {
			before := s.Snapshot()
			refuse(t, s, command, map[string]any{"path": path}, "invalid-params")
			projectAdvertisesError(t, s, command, "invalid-params", "admission")
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("invalid path admits a job")
			}
		}
	}
	path := filepath.Join(t.TempDir(), " literal name ")
	call(t, s, "project.save", map[string]any{"path": path, "overwrite": nil}, "save-literal")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("path spaces changed", err)
	}
	call(t, s, "project.save", map[string]any{"path": nil, "overwrite": nil, "expected_document_generation": nil}, "save-null-current")
	fresh := New()
	defer fresh.Close()
	refuse(t, fresh, "project.save.start", map[string]any{"path": nil}, "unavailable")
	projectAdvertisesError(t, fresh, "project.save.start", "unavailable", "admission")
	refuse(t, fresh, "project.load.start", map[string]any{"path": nil}, "invalid-params")
	call(t, fresh, "component.place", PlaceParams{ID: "R1", Kind: "resistor", Position: Point{}}, "place")
	refuse(t, fresh, "project.clear", map[string]any{"discard_changes": nil}, "unsaved-changes")
	projectAdvertisesError(t, fresh, "project.clear", "unsaved-changes", "admission")
	call(t, fresh, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	refuse(t, fresh, "component.place", PlaceParams{ID: "R1", Kind: "resistor", Position: Point{}}, "invalid-params")
	call(t, fresh, "component.place", PlaceParams{Kind: "resistor", Position: Point{}}, "automatic-id")
	if fresh.Snapshot().Selected == "R1" {
		t.Fatal("clear released lifetime ID")
	}
}

func TestProjectContractFailedWaitRetryNeedsNewKeyAfterRepair(t *testing.T) {
	original, err := os.ReadFile(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	s := New()
	defer s.Close()
	path := filepath.Join(t.TempDir(), "later.elab")
	params, _ := json.Marshal(LoadParams{Path: path})
	revision := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "missing", Command: "project.load", Params: params, ExpectedCommandRevision: &revision, Key: "original"}
	failed := s.Dispatch(context.Background(), request, "agent")
	if failed.Error == nil || failed.Error.Code != "file-error" {
		t.Fatal("missing file did not fail as advertised", failed.Error)
	}
	snapshot := failed.Result.(Snapshot)
	if snapshot.ProjectWork == nil || snapshot.ProjectWork.Status != "failed" {
		t.Fatal("waiting failure lost terminal job")
	}
	projectAdvertisesError(t, s, "project.load", "file-error", "project_work")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	duplicate := s.Dispatch(context.Background(), request, "agent")
	if !duplicate.Duplicate || duplicate.Error == nil || duplicate.Error.Code != "file-error" || duplicate.Result.(Snapshot).ProjectWork.ID != snapshot.ProjectWork.ID {
		t.Fatal("failed admitted retry reread repaired file")
	}
	current := s.Snapshot().CommandRevision
	request.ID, request.Key, request.ExpectedCommandRevision = "repaired", "new-attempt", &current
	loaded := s.Dispatch(context.Background(), request, "agent")
	if loaded.Error != nil || loaded.Result.(Snapshot).Document.Path != path || loaded.Result.(Snapshot).Document.Generation == snapshot.Document.Generation {
		t.Fatal("new key did not load repaired file", loaded.Error)
	}
}

func TestProjectContractPublishingBoundaryAndOwnedInspection(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	entered, release := blockSavePublication(t, s)
	defer release()
	started := time.Now()
	admitted := promptLiveCommand(t, s, "project.save.start", SaveParams{}).Result.(Snapshot)
	returned := time.Now()
	awaitLiveSampleBlock(t, entered)
	work := inspectProjectWork(t, s, admitted.ProjectWork.ID)
	if work.Phase != "publishing" || work.Cancelable {
		t.Fatal("publication claims cancellation is possible")
	}
	_, contract := projectDetailedSchema(t, s, "project.save.start")
	s.mu.Lock()
	deadline, bounded := s.projectJob.ctx.Deadline()
	s.mu.Unlock()
	duration := time.Duration(contract.Completion.CooperativeDeadlineSeconds) * time.Second
	if !bounded || deadline.Before(started.Add(duration)) || deadline.After(returned.Add(duration)) {
		t.Fatal("discovered cooperative deadline differs from worker context")
	}
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: work.ID}, "operation-committing")
	projectAdvertisesError(t, s, "project.work.cancel", "operation-committing", "admission")
	raw, _ := json.Marshal(IDParams{ID: "unknown"})
	unknown := s.Dispatch(t.Context(), Request{Version: 1, ID: "unknown-job", Command: "project.work.inspect", Params: raw}, "agent")
	if unknown.Error == nil || unknown.Error.Code != "operation-unavailable" {
		t.Fatal("unknown job error differs", unknown.Error)
	}
	projectAdvertisesError(t, s, "project.work.inspect", "operation-unavailable", "admission")
	promptLiveCommand(t, s, "project.clear", ReplaceParams{})
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.Document.Path != "" || len(after.Components) != 0 || after.ProjectWork.Associated == nil || *after.ProjectWork.Associated || after.ProjectWork.ResultGeneration == after.Document.Generation {
		t.Fatal("prior captured save is associated with the replacement document")
	}
	work = inspectProjectWork(t, s, work.ID)
	*work.Associated = true
	if *inspectProjectWork(t, s, work.ID).Associated {
		t.Fatal("inspection aliases retained completion metadata")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("completed captured artifact missing", err)
	}
}
