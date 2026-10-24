package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type intervalTimeRejection struct {
	generation, runID uint64
	recordingID, text string
	window            workbench.ScopeWindow
}

type intervalFieldContinuation struct {
	from, to    actionContext
	runID       uint64
	recordingID string
	window      workbench.ScopeWindow
}

func intervalTimeWindow(state workbench.Snapshot, recordingID string) (workbench.ScopeWindow, bool) {
	if recordingID != "" {
		if state.Recording == nil || !state.Recording.Viewing || state.Recording.ID != recordingID {
			return workbench.ScopeWindow{}, false
		}
		return state.Recording.Scope.Window, true
	}
	if state.Transient == nil || state.Transient.Stale || len(state.Transient.Samples) == 0 {
		return workbench.ScopeWindow{}, false
	}
	return state.Scope.Window, true
}

func sameIntervalTimeWindow(before, after workbench.ScopeWindow) bool {
	// Rolling history advances during private editing. It is still the same
	// live interval owner; an explicit fixed-window replacement is different.
	return before.Mode == after.Mode && (before.Mode == "live" || before == after)
}

func (s *screen) changeIntervalTime(state workbench.Snapshot, side, text, recordingID string) {
	window, valid := intervalTimeWindow(state, recordingID)
	if !valid {
		return
	}
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	var response workbench.Response
	if recordingID == "" {
		params := window
		params.Mode = "fixed"
		response = s.invokeAtCommandRevision("scope.window.set", textFieldParams(params, side+"_s", text), state.CommandRevision, state.Document.Generation)
	} else {
		params := workbench.RecordingWindowParams{Start: window.Start, End: window.End}
		response = s.invokeAtCommandRevision("recording.window.set", textFieldParams(params, side+"_s", text), state.CommandRevision, state.Document.Generation)
	}
	id := measurementTimeFieldKey(side, recordingID)
	if response.Error == nil {
		delete(s.intervalTimeRejected, id)
		accepted := response.Result.(workbench.Snapshot)
		if origin != nil {
			from := *origin
			if previous := s.intervalFieldAccepted; previous != nil && previous.to == from && previous.recordingID == recordingID &&
				(recordingID != "" || previous.runID == state.RunID) && sameIntervalTimeWindow(previous.window, window) {
				from = previous.from
			}
			next, _ := intervalTimeWindow(accepted, recordingID)
			s.intervalFieldAccepted = &intervalFieldContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation},
				runID: accepted.RunID, recordingID: recordingID, window: next}
		} else {
			s.intervalFieldAccepted = nil
		}
		return
	}
	s.intervalFieldAccepted = nil
	current := s.service.Snapshot()
	accepted, valid := intervalTimeWindow(current, recordingID)
	if origin == nil || response.Error.Code != "invalid-params" ||
		*origin != (actionContext{current.CommandRevision, current.Document.Generation}) ||
		!valid || !sameIntervalTimeWindow(window, accepted) ||
		recordingID == "" && state.RunID != current.RunID {
		return
	}
	if s.intervalTimeRejected == nil {
		s.intervalTimeRejected = make(map[string]intervalTimeRejection)
	}
	s.intervalTimeRejected[id] = intervalTimeRejection{
		generation: state.Document.Generation, runID: state.RunID,
		recordingID: recordingID, text: text, window: window,
	}
}

func (s *screen) intervalResetAction(state workbench.Snapshot, id, text string, params any, recordingID string) ui.Node {
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: func() {
		context := actionContext{state.CommandRevision, state.Document.Generation}
		current := s.service.Snapshot()
		window, valid := intervalTimeWindow(current, recordingID)
		if accepted := s.intervalFieldAccepted; accepted != nil && valid && accepted.recordingID == recordingID &&
			(recordingID != "" || accepted.runID == current.RunID) && sameIntervalTimeWindow(accepted.window, window) &&
			accepted.from.generation == context.generation && accepted.from.revision <= context.revision && context.revision <= accepted.to.revision &&
			accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			context = accepted.to
		}
		var response workbench.Response
		if recordingID == "" {
			response = s.invokeAtCommandRevision("scope.window.set", params, context.revision, context.generation)
		} else {
			response = s.invokeAtCommandRevision("recording.window.set", params, context.revision, context.generation)
		}
		// An explicit reset replaces private interval entries even when the
		// accepted bounds already match. Ordinary field correction owns only
		// that field; it must not erase its peer's unchanged rejected entry.
		if response.Error == nil {
			s.clearIntervalTimeRejections(recordingID)
			s.intervalFieldAccepted = nil
		}
	}}, false)
}

func (s *screen) clearIntervalTimeRejections(recordingID string) {
	for id, rejected := range s.intervalTimeRejected {
		if rejected.recordingID == recordingID {
			delete(s.intervalTimeRejected, id)
		}
	}
}

func (s *screen) pruneIntervalTimeRejections(state workbench.Snapshot) {
	for id, rejected := range s.intervalTimeRejected {
		window, valid := intervalTimeWindow(state, rejected.recordingID)
		if !valid || state.Document.Generation != rejected.generation ||
			rejected.recordingID == "" && state.RunID != rejected.runID ||
			!sameIntervalTimeWindow(rejected.window, window) {
			delete(s.intervalTimeRejected, id)
		}
	}
}

func (s *screen) applyIntervalTimeRejection(state workbench.Snapshot, side, recordingID string, field *ui.Node) {
	s.pruneIntervalTimeRejections(state)
	if rejected, ok := s.intervalTimeRejected[measurementTimeFieldKey(side, recordingID)]; ok {
		field.Text = rejected.text
	}
}
