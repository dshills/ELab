package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type recordedTimingContinuation struct {
	from, to    actionContext
	recordingID string
	settings    workbench.RecordingTimingParams
	next, total int
}

type recordedTimingFieldRejection struct {
	context     actionContext
	recordingID string
	message     string
}

type recordedTimingPageWork struct {
	context     actionContext
	recordingID string
	direction   string
	arrival     string
	menuSerial  uint64
	settings    workbench.RecordingTimingParams
	next, total int
	field       int
	failed      bool
}

// Only this recording's accepted threshold/interval field responses can
// continue a page or direction choice. Never adopt a later agent's settings.
func (s *screen) changeRecordedTimingField(state workbench.Snapshot, field, text string) {
	r := state.Recording
	pending := s.pendingRecordedTimingPage
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || r.ID != pending.recordingID) {
		pending.failed = true
		s.recordedTimingAccepted = nil
		s.errorMessage = "Threshold settings changed while the timing action was pending. Review them and try again."
		s.app.Invalidate("main")
		return
	}
	cfg := workbench.RecordingTimingParams(r.Timing.Settings)
	cfg.Offset = 0
	response := s.invokeAtCommandRevision("recording.timing.set", textFieldParams(cfg, field, text), state.CommandRevision)
	if response.Error != nil {
		s.recordedTimingAccepted = nil
		if origin != nil {
			s.recordedTimingRejected = &recordedTimingFieldRejection{context: *origin, recordingID: r.ID, message: response.Error.Message}
		}
		if pending != nil {
			pending.failed = true
		}
		return
	}
	s.recordedTimingRejected = nil
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	result := accepted.Recording.Timing
	from := *origin
	if previous := s.recordedTimingAccepted; previous != nil && previous.to == from && previous.recordingID == r.ID {
		from = previous.from
	}
	s.recordedTimingAccepted = &recordedTimingContinuation{
		from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, recordingID: r.ID,
		settings: workbench.RecordingTimingParams(result.Settings), next: result.NextOffset, total: result.Total,
	}
	if pending != nil {
		pending.context = s.recordedTimingAccepted.to
		pending.settings, pending.next, pending.total = s.recordedTimingAccepted.settings, result.NextOffset, result.Total
	}
}

// A native popup can choose an option while the threshold still has focus.
// Settle the related drafts before dispatching the choice, using only this
// action's captured context and its own accepted field continuations.
func (s *screen) changeRecordedTimingDirection(state workbench.Snapshot, direction string) {
	previousField, previousAccepted, previousMenu := s.fieldContext, s.fieldAccepted, s.menuCommandRevision
	s.fieldContext, s.fieldAccepted, s.menuCommandRevision = nil, nil, nil
	defer func() {
		s.fieldContext, s.fieldAccepted, s.menuCommandRevision = previousField, previousAccepted, previousMenu
	}()
	s.beginRecordedTimingAction(state, "", direction)
}

func (s *screen) recordedTimingPageAction(state workbench.Snapshot, direction string) ui.Node {
	text := "Next arrivals"
	if direction == "previous" {
		text = "Previous arrivals"
	}
	return buttonStyle(ui.Node{ID: "recorded-timing-" + direction, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Disabled: s.pendingRecordedTimingPage != nil, Activate: func() {
		s.beginRecordedTimingAction(state, direction, "")
	}}, false)
}

func (s *screen) beginRecordedTimingAction(state workbench.Snapshot, direction, arrival string) {
	if s.pendingRecordedTimingPage != nil {
		return
	}
	r := state.Recording
	result := r.Timing
	context := actionContext{state.CommandRevision, state.Document.Generation}
	pending := &recordedTimingPageWork{context: context, recordingID: r.ID, direction: direction, arrival: arrival, menuSerial: s.menuSerial, settings: workbench.RecordingTimingParams(result.Settings), next: result.NextOffset, total: result.Total}
	current := s.service.Snapshot()
	if rejected := s.recordedTimingRejected; arrival == "" && rejected != nil && rejected.recordingID == r.ID && rejected.context == (actionContext{current.CommandRevision, current.Document.Generation}) {
		s.errorMessage = rejected.message
		s.app.Invalidate("main")
		return
	}
	if accepted := s.recordedTimingAccepted; accepted != nil && accepted.recordingID == r.ID && accepted.from.generation == context.generation &&
		accepted.from.revision <= context.revision && context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
		pending.context = accepted.to
		pending.settings, pending.next, pending.total = accepted.settings, accepted.next, accepted.total
	}
	if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation {
		s.invokeAtCommandRevision("recording.timing.set", pending.settings, pending.context.revision, pending.context.generation)
		return
	}
	s.pendingRecordedTimingPage = pending
	s.settleRecordedTimingPage(pending, 0)
}

func (s *screen) settleRecordedTimingPage(pending *recordedTimingPageWork, index int) {
	if s.pendingRecordedTimingPage != pending || pending.field != index {
		return
	}
	if pending.failed || index == 4 {
		s.finishRecordedTimingPage(pending)
		return
	}
	fields := [...]string{"recorded-timing-level", "recorded-timing-second-level", "recorded-timing-start", "recorded-timing-end"}
	done := func(err error) {
		if s.pendingRecordedTimingPage != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			pending.failed = true
			s.errorMessage = err.Error()
		}
		s.settleRecordedTimingPage(pending, index+1)
	}
	if s.settleField == nil {
		done(nil)
	} else if err := s.settleField(fields[index], done); err != nil {
		done(err)
	}
}

func (s *screen) finishRecordedTimingPage(pending *recordedTimingPageWork) {
	if s.pendingRecordedTimingPage != pending {
		return
	}
	s.pendingRecordedTimingPage = nil
	if pending.failed {
		s.app.Invalidate("main")
		return
	}
	if pending.arrival != "" && (s.menuSerial != pending.menuSerial || s.fileBusy || s.pendingCommand != "") {
		s.errorMessage = "The timing menu or file operation changed. Review the settings and choose again."
		s.app.Invalidate("main")
		return
	}
	current := s.service.Snapshot()
	if rejected := s.recordedTimingRejected; rejected != nil && rejected.recordingID == pending.recordingID && rejected.context == (actionContext{current.CommandRevision, current.Document.Generation}) {
		s.errorMessage = rejected.message
		s.app.Invalidate("main")
		return
	}
	if current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		if current.Recording == nil || current.Recording.ID != pending.recordingID || !current.Recording.Viewing {
			s.errorMessage = "The selected recording changed. Review it before changing threshold timing."
			s.app.Invalidate("main")
			return
		}
		if pending.arrival == "" && pending.direction == "next" && pending.next >= pending.total {
			s.errorMessage = "There are no more arrivals with these threshold settings."
			s.app.Invalidate("main")
			return
		}
	}
	if pending.arrival != "" {
		pending.settings.Direction = pending.arrival
		pending.settings.Offset = 0
	} else if pending.direction == "previous" {
		pending.settings.Offset = max(0, pending.settings.Offset-pending.settings.Limit)
	} else {
		pending.settings.Offset = pending.next
	}
	s.invokeAtCommandRevision("recording.timing.set", pending.settings, pending.context.revision, pending.context.generation)
}
