package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Rejected time text is private editor state. Its owner and accepted cursor
// value must still match before it can replace the rendered field's text.
type cursorTimeRejection struct {
	generation, runID         uint64
	cursor, recordingID, text string
	requested                 float64
	placed                    bool
}

func cursorTimeFieldKey(cursor, recordingID string) string {
	return recordingID + ":" + cursor
}

func cursorTimeValue(state workbench.Snapshot, cursor, recordingID string) (float64, bool, bool) {
	scope := state.Scope
	if recordingID != "" {
		if state.Recording == nil || !state.Recording.Viewing || state.Recording.ID != recordingID {
			return 0, false, false
		}
		scope = state.Recording.Scope
	} else if state.Transient == nil || state.Transient.Stale {
		return 0, false, false
	}
	value := scope.A
	if cursor == "b" {
		value = scope.B
	}
	if value == nil {
		return 0, false, true
	}
	return value.Requested, true, true
}

func (s *screen) retainCursorTimeRejection(state workbench.Snapshot, cursor, text, recordingID string, origin *actionContext, response workbench.Response) {
	current := s.service.Snapshot()
	if origin == nil || response.Error.Code != "invalid-params" ||
		*origin != (actionContext{current.CommandRevision, current.Document.Generation}) {
		return
	}
	requested, placed, valid := cursorTimeValue(state, cursor, recordingID)
	if !valid {
		return
	}
	if s.cursorTimeRejected == nil {
		s.cursorTimeRejected = make(map[string]cursorTimeRejection)
	}
	s.cursorTimeRejected[cursorTimeFieldKey(cursor, recordingID)] = cursorTimeRejection{
		generation: state.Document.Generation, runID: state.RunID,
		cursor: cursor, recordingID: recordingID, text: text, requested: requested,
		placed: placed,
	}
}

func (s *screen) pruneCursorTimeRejections(state workbench.Snapshot) {
	for id, rejected := range s.cursorTimeRejected {
		requested, placed, valid := cursorTimeValue(state, rejected.cursor, rejected.recordingID)
		if !valid || state.Document.Generation != rejected.generation ||
			rejected.recordingID == "" && state.RunID != rejected.runID ||
			requested != rejected.requested || placed != rejected.placed {
			delete(s.cursorTimeRejected, id)
		}
	}
}

func (s *screen) applyCursorTimeRejection(state workbench.Snapshot, cursor, recordingID string, field *ui.Node) {
	s.pruneCursorTimeRejections(state)
	if rejected, ok := s.cursorTimeRejected[cursorTimeFieldKey(cursor, recordingID)]; ok {
		field.Text = rejected.text
	}
}

// A native field has already delivered retained rejection text, so its ordinary
// settlement may be a no-op. An explicit Select/Clear still retries that field
// through the same command and original action guard. In particular, more live
// history can make a formerly future time valid. Never rebase across an actor.
func (s *screen) retryCursorTimeRejection(pending *cursorActionWork, index int) {
	state := s.service.Snapshot()
	if pending.context != (actionContext{state.CommandRevision, state.Document.Generation}) {
		return
	}
	s.pruneCursorTimeRejections(state)
	cursor := "a"
	if index == 1 {
		cursor = "b"
	}
	rejected, ok := s.cursorTimeRejected[cursorTimeFieldKey(cursor, pending.recordingID)]
	if !ok {
		return
	}
	origin := pending.context
	previous, previousAccepted := s.fieldContext, s.fieldAccepted
	s.fieldContext, s.fieldAccepted = &origin, nil
	defer func() { s.fieldContext, s.fieldAccepted = previous, previousAccepted }()
	s.changeCursorTimeFor(state, cursor, rejected.text, pending.recordingID)
}
