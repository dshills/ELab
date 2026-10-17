package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

var projectContractCommands = []string{"project.inspect", "project.clear", "project.save", "project.save.start", "project.load", "project.load.start", "project.work.inspect", "project.work.cancel"}
var projectContractChecks = []string{"eight-detailed-contracts", "discovery-preserves-project", "captured-save-and-later-dirty-edit", "external-file-conflict-preserves-source", "explicit-overwrite-and-clean-baseline", "terminal-cancel-refusal", "failed-open-preserves-source", "clear-preserves-file-and-detaches", "background-open-fresh-paused-with-recording", "historical-job-inspection-owned", "waiting-open-compatible"}

type ProjectContractResult struct {
	Schemas              map[string]json.RawMessage    `json:"detailed_inputs"`
	Checks               []string                      `json:"checks"`
	Calls                []Call                        `json:"calls"`
	States               map[string]workbench.Snapshot `json:"states"`
	Captured             workbench.Document            `json:"admitted_save_document"`
	Historical           workbench.ProjectWork         `json:"historical_save"`
	Failures             map[string]string             `json:"failures"`
	ExternalAfterRefusal string                        `json:"external_file_after_refusal"`
}

func projectContractProtected(s workbench.Snapshot) any {
	return []any{workbench.Graph{Components: s.Components, Wires: s.Wires}, s.Document, s.RunID, s.ElectricalRevision, s.ActuationRevision, s.UndoEntry, s.RedoEntry, s.Transient, s.Result, s.Mixed, s.NPN, s.Paused, s.TransientSettings, s.PinnedReadings, s.Display, s.Example, s.Operations, s.Recording, s.Scope.Selected}
}

// RunProjectContracts exercises real owned files and the public contract that
// clients discover. In-flight cancellation/publication barriers stay in the
// controlled race tests, rather than relying on filesystem timing races.
func RunProjectContracts(ctx context.Context, factory Factory, dir string) (out ProjectContractResult, err error) {
	out.Schemas = map[string]json.RawMessage{}
	out.States = map[string]workbench.Snapshot{}
	out.Failures = map[string]string{}
	err = electricalSession(ctx, factory, dir, func(s *session) error {
		mark := func(name string) { out.Checks = append(out.Checks, name) }
		if err := s.ask("simulation.pause", nil, true, nil); err != nil {
			return err
		}
		before, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["before-discovery"] = before
		for _, id := range projectContractCommands {
			var catalog struct {
				Capabilities []workbench.Capability `json:"capabilities"`
			}
			if err := s.ask("capabilities.list", workbench.CapabilityQuery{Command: id}, false, &catalog); err != nil {
				return err
			}
			if len(catalog.Capabilities) != 1 {
				return fmt.Errorf("project contract query missing capability")
			}
			out.Schemas[id] = append(json.RawMessage(nil), catalog.Capabilities[0].Input...)
		}
		if err := validateProjectContractSchemas(out.Schemas); err != nil {
			return err
		}
		mark("eight-detailed-contracts")
		after, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["after-discovery"] = after
		if !reflect.DeepEqual(projectContractProtected(before), projectContractProtected(after)) {
			return fmt.Errorf("project discovery changed schematic state")
		}
		mark("discovery-preserves-project")
		if err := s.ask("example.rc", nil, true, nil); err != nil {
			return err
		}
		if err := s.ask("simulation.pause", nil, true, nil); err != nil {
			return err
		}
		if _, err := restart(s); err != nil {
			return err
		}
		if _, err := s.state("simulation.step", workbench.StepParams{Duration: .012}, true); err != nil {
			return err
		}
		if _, _, _, err := electricalVerify(s, .02, .001, []string{"v:C1", "i:C1"}); err != nil {
			return err
		}
		original, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["original"] = original
		path := filepath.Join(dir, "captured.elab")
		admitted, err := s.state("project.save.start", workbench.SaveParams{Path: path}, true)
		if err != nil {
			return err
		}
		if admitted.ProjectWork == nil || admitted.ProjectWork.Status != "queued" {
			return fmt.Errorf("save lost queued admission")
		}
		out.States["save-admission"] = admitted
		firstID := admitted.ProjectWork.ID
		if err := s.ask("component.set", workbench.ValueParams{ID: "R1", Value: "20k"}, true, nil); err != nil {
			return err
		}
		saved, err := s.state("simulation.wait", nil, false)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var document workbench.Document
		if err := json.Unmarshal(raw, &document); err != nil {
			return err
		}
		out.Captured = document
		out.States["captured-save"] = saved
		if err := os.WriteFile(filepath.Join(dir, "admitted-save.elab"), raw, 0600); err != nil {
			return err
		}
		r, _ := document.Graph.Component("R1")
		if r.Value != 10000 || !saved.Document.Dirty || saved.ProjectWork.Status != "completed" || saved.ProjectWork.ArtifactSHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
			return fmt.Errorf("save did not preserve captured input/later edit/hash")
		}
		mark("captured-save-and-later-dirty-edit")
		capturedHash := saved.ProjectWork.ArtifactSHA256
		if err := os.WriteFile(path, []byte("outside writer\n"), 0600); err != nil {
			return err
		}
		current, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["before-conflict"] = current
		refused, err := s.request("project.save", workbench.SaveParams{}, true)
		if err != nil {
			return err
		}
		if refused.Error == nil || refused.Error.Code != "file-conflict" {
			return fmt.Errorf("save failed to protect external change")
		}
		out.Failures["external-save"] = refused.Error.Code
		outside, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out.ExternalAfterRefusal = string(outside)
		if out.ExternalAfterRefusal != "outside writer\n" {
			return fmt.Errorf("failed save modified outside writer")
		}

		next, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["after-conflict"] = next
		if !reflect.DeepEqual(projectContractProtected(current), projectContractProtected(next)) {
			return fmt.Errorf("failed save changed source")
		}
		mark("external-file-conflict-preserves-source")
		saved, err = s.state("project.save", workbench.SaveParams{Overwrite: true}, true)
		if err != nil {
			return err
		}
		if saved.Document.Dirty || saved.ProjectWork.Status != "completed" {
			return fmt.Errorf("overwrite did not save current baseline")
		}
		out.States["overwritten"] = saved
		mark("explicit-overwrite-and-clean-baseline")
		disk, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		refused, err = s.request("project.work.cancel", workbench.IDParams{ID: saved.ProjectWork.ID}, true)
		if err != nil {
			return err
		}
		if refused.Error == nil || refused.Error.Code != "operation-unavailable" {
			return fmt.Errorf("terminal project job accepted cancel")
		}
		out.Failures["terminal-cancel"] = refused.Error.Code
		mark("terminal-cancel-refusal")
		bad := filepath.Join(dir, "invalid.elab")
		if err := os.WriteFile(bad, []byte("invalid JSON\n"), 0600); err != nil {
			return err
		}
		current, err = s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["before-invalid-open"] = current
		refused, err = s.request("project.load", workbench.LoadParams{Path: bad}, true)
		if err != nil {
			return err
		}
		if refused.Error == nil || refused.Error.Code != "invalid-document" {
			return fmt.Errorf("invalid project did not fail as discovered")
		}
		out.Failures["invalid-open"] = refused.Error.Code
		next, err = s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		out.States["after-invalid-open"] = next
		if next.ProjectWork.Status != "failed" || !reflect.DeepEqual(projectContractProtected(current), projectContractProtected(next)) {
			return fmt.Errorf("failed open changed source")
		}
		mark("failed-open-preserves-source")
		cleared, err := s.state("project.clear", workbench.ReplaceParams{ExpectedGeneration: &next.Document.Generation}, true)
		if err != nil {
			return err
		}
		unchanged, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(cleared.Components) != 0 || cleared.Document.Path != "" || cleared.Document.Generation == next.Document.Generation || !bytes.Equal(disk, unchanged) {
			return fmt.Errorf("clear rewrote saved file or retained association")
		}
		out.States["cleared"] = cleared
		mark("clear-preserves-file-and-detaches")
		opened, err := s.state("project.load.start", workbench.LoadParams{Path: path, ExpectedGeneration: &cleared.Document.Generation}, true)
		if err != nil {
			return err
		}
		if opened.ProjectWork == nil || opened.ProjectWork.Status != "queued" || len(opened.Components) != 0 {
			return fmt.Errorf("open admission replaced source")
		}
		out.States["open-admission"] = opened
		loaded, err := s.state("simulation.wait", nil, false)
		if err != nil {
			return err
		}
		out.States["loaded"] = loaded
		part, _ := (workbench.Graph{Components: loaded.Components, Wires: loaded.Wires}).Component("R1")
		if !loaded.Paused || loaded.Document.Dirty || loaded.Document.RecordingCount != 1 || loaded.Transient == nil || loaded.Transient.Sample.Time != 0 || part.Value != 20000 || !reflect.DeepEqual(loaded.Scope.Selected, original.Scope.Selected) {
			return fmt.Errorf("open lost fresh paused state, kept recording, graph or probes")
		}
		mark("background-open-fresh-paused-with-recording")
		var historical workbench.ProjectWork
		if err := s.ask("project.work.inspect", workbench.IDParams{ID: firstID}, false, &historical); err != nil {
			return err
		}
		if historical.Status != "completed" || historical.ArtifactSHA256 != capturedHash {
			return fmt.Errorf("historical completion changed after replacement")
		}
		out.Historical = historical
		mark("historical-job-inspection-owned")
		waited, err := s.state("project.load", workbench.LoadParams{Path: path}, true)
		if err != nil {
			return err
		}
		if waited.ProjectWork.Status != "completed" || waited.Document.Generation == loaded.Document.Generation {
			return fmt.Errorf("waiting alias did not join current open")
		}
		out.States["waiting-loaded"] = waited
		mark("waiting-open-compatible")
		out.Calls = append([]Call(nil), s.calls...)
		return nil
	})
	return out, err
}
