package main

import "encoding/json"

// A decision belongs to one pending intent, not merely a reusable button ID.
// Its rendered context must also survive through shared command admission.
type fileDecision struct {
	serial     uint64
	command    string
	context    actionContext
	generation uint64
}

func (s *screen) setPendingFileAction(command string, params json.RawMessage) {
	s.pendingFileSerial++
	s.pendingCommand = command
	s.pendingParams = append(json.RawMessage(nil), params...)
}

func (s *screen) captureFileDecision() fileDecision {
	state := s.service.Snapshot()
	decision := fileDecision{serial: s.pendingFileSerial, command: s.pendingCommand, context: actionContext{state.CommandRevision, state.Document.Generation}, generation: state.Document.Generation}
	var params struct {
		Generation *uint64 `json:"expected_document_generation"`
	}
	if json.Unmarshal(s.pendingParams, &params) == nil && params.Generation != nil {
		decision.generation = *params.Generation
	}
	return decision
}

func (s *screen) withFileDecision(decision fileDecision, needsDocument bool, action func()) {
	if s.fileBusy || decision.command == "" || s.pendingFileSerial != decision.serial || s.pendingCommand != decision.command {
		return
	}
	state := s.service.Snapshot()
	if state.CommandRevision != decision.context.revision || state.Document.Generation != decision.context.generation || needsDocument && state.Document.Generation != decision.generation {
		s.errorMessage = "Schematic changed. Review the current project before choosing again."
		s.requestHelp = nil
		s.app.Invalidate("main")
		return
	}
	previous := s.fileDecisionContext
	s.fileDecisionContext = &decision.context
	defer func() { s.fileDecisionContext = previous }()
	action()
}
