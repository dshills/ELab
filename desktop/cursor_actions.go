package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// A native focus loss can commit either cursor's time before its Select/Clear
// click arrives. Only consecutive accepted cursor-field responses may advance
// that rendered action; an agent or any other command still breaks the chain.
type cursorFieldContinuation struct {
	from, to    actionContext
	recordingID string
}

type cursorActionWork struct {
	context         actionContext
	command, cursor string
	recordingID     string
	field           int
	failed          bool
}

func (s *screen) changeCursorTime(state workbench.Snapshot, cursor, text string) {
	s.changeCursorTimeFor(state, cursor, text, "")
}

func (s *screen) changeCursorTimeFor(state workbench.Snapshot, cursor, text, recordingID string) {
	pending := s.pendingCursorAction
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || pending.recordingID != recordingID) {
		pending.failed = true
		s.cursorFieldAccepted = nil
		s.errorMessage = "Cursor time changed while the action was pending. Review it and try again."
		s.app.Invalidate("main")
		return
	}
	var response workbench.Response
	if recordingID == "" {
		response = s.invokeAtCommandRevision("scope.cursor.set", map[string]any{"cursor": cursor, "time_s": text}, state.CommandRevision)
	} else {
		response = s.invokeAtCommandRevision("recording.cursor.set", map[string]any{"operation_id": recordingID, "cursor": cursor, "time_s": text}, state.CommandRevision)
	}
	if response.Error != nil {
		s.cursorFieldAccepted = nil
		if pending != nil {
			pending.failed = true
		}
		return
	}
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.cursorFieldAccepted; previous != nil && previous.to == from && previous.recordingID == recordingID {
		from = previous.from
	}
	s.cursorFieldAccepted = &cursorFieldContinuation{
		from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation},
		recordingID: recordingID,
	}
	if pending != nil {
		pending.context = s.cursorFieldAccepted.to
	}
}

func (s *screen) cursorAction(state workbench.Snapshot, id, text, command, cursor string) ui.Node {
	return s.cursorActionFor(state, id, text, command, cursor, "")
}

func (s *screen) recordedCursorAction(state workbench.Snapshot, id, text, command, cursor string) ui.Node {
	return s.cursorActionFor(state, id, text, command, cursor, state.Recording.ID)
}

func (s *screen) cursorActionFor(state workbench.Snapshot, id, text, command, cursor, recordingID string) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Disabled: s.pendingCursorAction != nil, Accessibility: ui.Accessibility{Name: text}, Activate: func() {
		if s.pendingCursorAction != nil {
			return
		}
		current := s.service.Snapshot()
		if accepted := s.cursorFieldAccepted; accepted != nil && accepted.recordingID == recordingID && accepted.from.generation == context.generation &&
			accepted.from.revision <= context.revision && context.revision <= accepted.to.revision &&
			accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			context = accepted.to
		}
		if current.CommandRevision != context.revision || current.Document.Generation != context.generation {
			s.invokeAtCommandRevision(command, workbench.CursorSelectParams{Cursor: cursor}, context.revision, context.generation)
			return
		}
		pending := &cursorActionWork{context: context, command: command, cursor: cursor, recordingID: recordingID}
		s.pendingCursorAction = pending
		if recordingID == "" && (current.Transient == nil || current.Transient.Stale) {
			// Disabled time fields cannot commit. Selecting/clearing existing
			// cursors must remain available on stale measurements.
			s.finishCursorAction(pending, nil)
		} else {
			s.settleCursorActionField(pending, 0)
		}
	}}, false)
}

func (s *screen) settleCursorActionField(pending *cursorActionWork, index int) {
	if s.pendingCursorAction != pending || pending.field != index {
		return
	}
	if pending.failed || index == 2 {
		s.finishCursorAction(pending, nil)
		return
	}
	id := "cursor-time-a"
	if index == 1 {
		id = "cursor-time-b"
	}
	if pending.recordingID != "" {
		id = "recorded-cursor-a"
		if index == 1 {
			id = "recorded-cursor-b"
		}
	}
	done := func(err error) {
		if s.pendingCursorAction != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			s.finishCursorAction(pending, err)
		} else {
			s.settleCursorActionField(pending, index+1)
		}
	}
	if s.settleField == nil {
		done(nil)
	} else if err := s.settleField(id, done); err != nil {
		done(err)
	}
}

func (s *screen) finishCursorAction(pending *cursorActionWork, err error) {
	if s.pendingCursorAction != pending {
		return
	}
	s.pendingCursorAction = nil
	if err != nil {
		s.errorMessage = fmt.Sprintf("Could not finish cursor time entry: %v", err)
	} else if !pending.failed {
		s.invokeAtCommandRevision(pending.command, workbench.CursorSelectParams{Cursor: pending.cursor}, pending.context.revision, pending.context.generation)
	}
	s.app.Invalidate("main")
}
