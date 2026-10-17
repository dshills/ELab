package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

func storageProjectInvariant(state workbench.Snapshot) any {
	return []any{state.Document, state.Components, state.Wires, state.RunID, state.ElectricalRevision, state.Mixed, state.Transient, state.NPN, state.Result, state.Scope, state.Operations, state.SelectedOperation, state.Recording, state.UndoEntry, state.RedoEntry, state.Paused, state.TransientSettings, state.ActuationRevision, state.SwitchStates, state.Drawing, state.Selected, state.Nets, state.Example, state.OpAmpModel, state.PMOSModel, state.NMOSModel, state.PinnedReadings, state.MeasurementInspection, state.TraceFocus, state.VerificationSettings, state.SchematicPage, state.Display}
}

func refuseStorageFile(s *session, command, path, code string) error {
	before, err := s.state("project.inspect", nil, false)
	if err != nil {
		return err
	}
	var params any = workbench.RecordingImportParams{Path: path}
	if command == "project.load" {
		params = workbench.LoadParams{Path: path, DiscardChanges: true}
	}
	response, err := s.request(command, params, true)
	if err != nil {
		return err
	}
	if response.Error == nil || response.Error.Code != code {
		return fmt.Errorf("corrupted %s: want %s, got %+v", command, code, response.Error)
	}
	after, err := s.state("project.inspect", nil, false)
	if err != nil {
		return err
	}
	if after.ProjectWork == nil || after.ProjectWork.Status != "failed" || after.ProjectWork.Error == nil || after.ProjectWork.Error.Code != code || !reflect.DeepEqual(storageProjectInvariant(before), storageProjectInvariant(after)) {
		return fmt.Errorf("failed %s changed project/physics/history/records or lost job outcome", command)
	}
	return nil
}

func storageCorruptionNames(name string) []string {
	names := []string{"missing-opamp-model", "opamp-model", "opamp-power", "opamp-id", "missing-ground", "foreign-node", "capacitor-voltage", "capacitor-energy"}
	if name == "opamp-buffered-rlc" {
		return append(names, "inductor-energy", "inductor-voltage", "inductor-current")
	}
	return append(names, "missing-pmos-model", "mosfet-model", "mosfet-id", "gate-current", "drain-current", "source-current", "gate-voltage", "drain-voltage", "mosfet-power", "mosfet-region")
}

func corruptStorageRecording(document *workbench.VerificationDocument, mutation string) {
	m := &document.Metadata
	r := m.Mixed
	switch mutation {
	case "missing-opamp-model":
		m.OpAmpModel = nil
	case "opamp-model":
		r.OpAmps[0].Model.OpenLoopGain = 42
	case "opamp-power":
		r.OpAmps[0].Power += .001
	case "opamp-id":
		r.OpAmps[0].ID = "Foreign"
	case "missing-ground":
		c, _, _ := m.Graph.Compile()
		delete(r.Nodes, c.Ground)
	case "foreign-node":
		r.Nodes["foreign-node"] = 0
	case "capacitor-voltage":
		r.Capacitors[0].Voltage += .25
	case "capacitor-energy":
		r.Capacitors[0].Energy += .001
	case "inductor-energy":
		r.Inductors[0].Energy += .001
	case "inductor-voltage":
		r.Inductors[0].Voltage += .25
	case "inductor-current":
		r.Inductors[0].Current += .001
		p, _ := m.Graph.Component(r.Inductors[0].ID)
		r.Inductors[0].Energy = .5 * p.Value * r.Inductors[0].Current * r.Inductors[0].Current
	case "missing-pmos-model":
		m.PMOSModel = nil
	case "mosfet-model":
		r.MOSFETs[0].Model = workbench.NMOSDeviceModel()
	case "mosfet-id":
		r.MOSFETs[0].ID = "Foreign"
	case "gate-current":
		r.MOSFETs[0].GateCurrent = .001
	case "drain-current":
		r.MOSFETs[0].DrainCurrent += .001
	case "source-current":
		r.MOSFETs[0].SourceCurrent += .001
	case "gate-voltage":
		r.MOSFETs[0].VGS += .25
	case "drain-voltage":
		r.MOSFETs[0].VDS += .25
	case "mosfet-power":
		r.MOSFETs[0].Power += .001
	case "mosfet-region":
		r.MOSFETs[0].State = "foreign"
	}
}

func refuseOpAmpStorageCorruptions(s *session, name string, recordingBytes, projectBytes []byte) ([]StorageRefusal, error) {
	var results []StorageRefusal
	for _, mutation := range storageCorruptionNames(name) {
		var document workbench.VerificationDocument
		if err := json.Unmarshal(recordingBytes, &document); err != nil {
			return nil, err
		}
		corruptStorageRecording(&document, mutation)
		raw, err := json.Marshal(document)
		if err != nil {
			return nil, err
		}
		recordingPath := filepath.Join(s.directory, "bad-"+mutation+".json")
		if err = os.WriteFile(recordingPath, raw, 0600); err != nil {
			return nil, err
		}
		if err = refuseStorageFile(s, "verification.import", recordingPath, "invalid-recording"); err != nil {
			return results, fmt.Errorf("%s %s: %w", name, mutation, err)
		}
		results = append(results, StorageRefusal{mutation, "verification.import", "invalid-recording", true})
		var project workbench.Document
		if err = json.Unmarshal(projectBytes, &project); err != nil {
			return nil, err
		}
		if project.Recordings == nil || len(project.Recordings.Entries) != 1 {
			return nil, fmt.Errorf("saved storage fixture lacks one kept recording")
		}
		project.Recordings.Entries[0].Data = document
		raw, err = json.Marshal(project)
		if err != nil {
			return nil, err
		}
		projectPath := filepath.Join(s.directory, "bad-"+mutation+".elab")
		if err = os.WriteFile(projectPath, raw, 0600); err != nil {
			return nil, err
		}
		if err = refuseStorageFile(s, "project.load", projectPath, "invalid-document"); err != nil {
			return results, fmt.Errorf("%s %s embedded: %w", name, mutation, err)
		}
		results = append(results, StorageRefusal{mutation, "project.load", "invalid-document", true})
	}
	return results, nil
}
