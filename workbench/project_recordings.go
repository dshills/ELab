package workbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
)

const MaxProjectBytes = 8 * 1024 * 1024

type ProjectRecording struct {
	Data        VerificationDocument `json:"data"`
	ModelHashes map[string]string    `json:"model_sha256"`
}
type ProjectRecordings struct {
	Entries  []ProjectRecording `json:"entries"`
	Selected int                `json:"selected_index"`
	Viewing  bool               `json:"viewing"`
}
type RecordingKeepParams struct {
	ID   string `json:"id"`
	Keep bool   `json:"keep"`
}

// Device parameters are hashed canonically; ideal model dependencies have
// immutable versioned IDs and no additional embedded parameters.
func recordingModelHashes(m VerificationMetadata) map[string]string {
	out := map[string]string{}
	for _, part := range m.Graph.Components {
		var data []byte
		switch part.Model {
		case ModelID:
			data, _ = json.Marshal(m.LEDModel)
		case SwitchModelID:
			data, _ = json.Marshal(m.SwitchModel)
		case NPNModelID:
			data, _ = json.Marshal(m.NPNModel)
		default:
			data, _ = json.Marshal(struct {
				ID string `json:"id"`
			}{part.Model})
		}
		digest := sha256.Sum256(data)
		out[part.Model] = hex.EncodeToString(digest[:])
	}
	return out
}
func (s *Service) projectRecordingsLocked() *ProjectRecordings {
	out := &ProjectRecordings{Entries: []ProjectRecording{}, Selected: -1}
	for _, id := range s.operationOrder {
		operation := s.operations[id]
		if !operation.keep || operation.record == nil {
			continue
		}
		record := operation.record
		if s.selectedOperation == id {
			out.Selected = len(out.Entries)
			out.Viewing = s.recording.viewing
		}
		out.Entries = append(out.Entries, ProjectRecording{Data: VerificationDocument{Metadata: cloneVerificationMetadata(record.metadata), Rows: record.rows}, ModelHashes: recordingModelHashes(record.metadata)})
	}
	return out
}
func validateProjectRecordings(raw json.RawMessage) (*ProjectRecordings, error) {
	var project ProjectRecordings
	if err := params(raw, &project, "entries", "selected_index", "viewing"); err != nil {
		return nil, err
	}
	if project.Entries == nil || len(project.Entries) > MaxVerificationOperations || project.Selected < -1 || project.Selected >= len(project.Entries) || project.Viewing && project.Selected < 0 {
		return nil, fmt.Errorf("invalid embedded recording count or selection")
	}
	var fields struct {
		Entries []json.RawMessage `json:"entries"`
	}
	_ = json.Unmarshal(raw, &fields)
	for i, entry := range fields.Entries {
		var object struct {
			Data   json.RawMessage   `json:"data"`
			Hashes map[string]string `json:"model_sha256"`
		}
		if err := params(entry, &object, "data", "model_sha256"); err != nil {
			return nil, err
		}
		record, err := decodeRecording(object.Data)
		if err != nil {
			return nil, fmt.Errorf("recording %d: %w", i, err)
		}
		if !reflect.DeepEqual(object.Hashes, recordingModelHashes(record.metadata)) {
			return nil, fmt.Errorf("recording %d model dependency hashes do not match", i)
		}
		project.Entries[i] = ProjectRecording{Data: VerificationDocument{Metadata: record.metadata, Rows: record.rows}, ModelHashes: object.Hashes}
	}
	return &project, nil
}
func (s *Service) recordingsDirtyLocked() bool {
	return s.recordingMembershipHashLocked() != s.savedRecordings
}
func (s *Service) recordingCountLocked() int {
	count := 0
	for _, operation := range s.operations {
		if operation.keep && operation.record != nil {
			count++
		}
	}
	return count
}
func (s *Service) setRecordingKeepLocked(p RecordingKeepParams) error {
	operation := s.operations[p.ID]
	if operation == nil || operation.record == nil {
		return &actionError{"not-ready", "Choose a completed recording"}
	}
	operation.keep = p.Keep
	return nil
}
func (s *Service) clearProjectOperationsLocked() {
	for _, operation := range s.operations {
		if operationActive(operation.summary.Status) {
			operation.cancel()
		}
	}
	s.operations = map[string]*verificationOperation{}
	s.operationOrder = nil
	s.lastOperation = ""
	s.selectRecordingLocked("")
	s.savedRecordings = s.recordingMembershipHashLocked()
}
func (s *Service) installProjectRecordingsLocked(project *ProjectRecordings) {
	s.clearProjectOperationsLocked()
	if project == nil {
		return
	}
	for i, entry := range project.Entries {
		s.operationSerial++
		id := fmt.Sprintf("record-%d", s.operationSerial)
		record := &verificationRecord{metadata: cloneVerificationMetadata(entry.Data.Metadata), rows: entry.Data.Rows}
		operation := &verificationOperation{keep: true, record: record, inputMetadata: cloneVerificationMetadata(record.metadata), summary: OperationSummary{ID: id, Kind: "project-record", Status: "completed", Actor: "project", InputRevision: record.metadata.InputRevision, ElectricalRevision: record.metadata.ElectricalRevision, DocumentGeneration: record.metadata.DocumentGeneration, RequestedDuration: record.metadata.Duration, SimulatedTime: record.metadata.Duration, CompletedSteps: len(record.rows), TotalSteps: len(record.rows), Progress: 1}}
		s.operations[id] = operation
		s.operationOrder = append(s.operationOrder, id)
		s.lastOperation = id
		if project.Selected == i {
			s.selectRecordingLocked(id)
			s.recording.viewing = project.Viewing
		}
	}
}
func (s *Service) requireRecordingSlotLocked() error {
	if len(s.operationOrder) < MaxVerificationOperations {
		return nil
	}
	for _, id := range s.operationOrder {
		operation := s.operations[id]
		if !operation.keep && !operationActive(operation.summary.Status) {
			return nil
		}
	}
	return &actionError{"record-limit", "The project keeps eight recordings. Exclude one from the project before creating or importing another."}
}
func (s *Service) verificationActiveLocked() bool {
	for _, operation := range s.operations {
		if operationActive(operation.summary.Status) {
			return true
		}
	}
	return false
}

func (s *Service) recordingMembershipHashLocked() [32]byte {
	ids := []string{}
	for _, id := range s.operationOrder {
		operation := s.operations[id]
		if operation.keep && operation.record != nil {
			ids = append(ids, id)
		}
	}
	data, _ := json.Marshal(ids)
	return sha256.Sum256(data)
}
