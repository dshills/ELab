package conformance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
)

func validateProjectContractEvidence(r ProjectContractResult) error {
	names := []string{"before-discovery", "after-discovery", "original", "save-admission", "captured-save", "before-conflict", "after-conflict", "overwritten", "before-invalid-open", "after-invalid-open", "cleared", "open-admission", "loaded", "waiting-loaded"}
	if len(r.States) != len(names) {
		return fmt.Errorf("project evidence state coverage differs")
	}
	for _, name := range names {
		if _, ok := r.States[name]; !ok {
			return fmt.Errorf("missing project state %s", name)
		}
	}
	for _, pair := range [][2]string{{"before-discovery", "after-discovery"}, {"before-conflict", "after-conflict"}, {"before-invalid-open", "after-invalid-open"}} {
		if !reflect.DeepEqual(projectContractProtected(r.States[pair[0]]), projectContractProtected(r.States[pair[1]])) {
			return fmt.Errorf("project source preservation differs at %s", pair[1])
		}
	}
	if r.ExternalAfterRefusal != "outside writer\n" {
		return fmt.Errorf("external destination preservation differs")
	}
	if len(r.Failures) != 3 || r.Failures["external-save"] != "file-conflict" || r.Failures["terminal-cancel"] != "operation-unavailable" || r.Failures["invalid-open"] != "invalid-document" {
		return fmt.Errorf("project refusal evidence differs")
	}
	saved := r.States["captured-save"]
	admitted := r.States["save-admission"]
	raw, err := json.MarshalIndent(r.Captured, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	part, ok := r.Captured.Graph.Component("R1")
	current, hasCurrent := (workbench.Graph{Components: saved.Components, Wires: saved.Wires}).Component("R1")
	if !ok || !hasCurrent || part.Value != 10000 || current.Value != 20000 || !saved.Document.Dirty || saved.ProjectWork == nil || admitted.ProjectWork == nil || admitted.ProjectWork.Status != "queued" || saved.ProjectWork.Status != "completed" || saved.ProjectWork.ID != admitted.ProjectWork.ID || saved.ProjectWork.ArtifactSHA256 != hash || saved.ProjectWork.Associated == nil || !*saved.ProjectWork.Associated || r.Historical.ID != admitted.ProjectWork.ID || r.Historical.Status != "completed" || r.Historical.ArtifactSHA256 != hash {
		return fmt.Errorf("captured save/history evidence differs")
	}
	overwritten := r.States["overwritten"]
	failed := r.States["after-invalid-open"]
	if overwritten.ProjectWork != nil && overwritten.ProjectWork.ID == admitted.ProjectWork.ID {
		return fmt.Errorf("distinct saves reused one job")
	}
	if overwritten.Document.Dirty || overwritten.ProjectWork == nil || overwritten.ProjectWork.Status != "completed" || failed.ProjectWork == nil || failed.ProjectWork.Status != "failed" || failed.ProjectWork.Error == nil || failed.ProjectWork.Error.Code != "invalid-document" {
		return fmt.Errorf("overwrite/failed open evidence differs")
	}
	cleared := r.States["cleared"]
	opened := r.States["open-admission"]
	loaded := r.States["loaded"]
	waited := r.States["waiting-loaded"]
	loadPart, ok := (workbench.Graph{Components: loaded.Components, Wires: loaded.Wires}).Component("R1")
	if len(cleared.Components) != 0 || len(cleared.Wires) != 0 || cleared.Document.Path != "" || cleared.Document.Generation == failed.Document.Generation || opened.ProjectWork == nil || opened.ProjectWork.Status != "queued" || len(opened.Components) != 0 || !ok || loadPart.Value != 20000 || !loaded.Paused || loaded.Document.Dirty || loaded.Document.RecordingCount != 1 || loaded.ProjectWork == nil || loaded.ProjectWork.Status != "completed" || loaded.Document.Generation == cleared.Document.Generation || loaded.Transient == nil || loaded.Transient.Stale || loaded.Transient.Sample.Time != 0 || !reflect.DeepEqual(loaded.Scope.Selected, r.States["original"].Scope.Selected) || waited.ProjectWork == nil || waited.ProjectWork.Status != "completed" || waited.Document.Generation == loaded.Document.Generation {
		return fmt.Errorf("clear/fresh open evidence differs")
	}
	return nil
}
