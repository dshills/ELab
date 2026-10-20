package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type recordedPlayValues struct {
	index       int
	time, speed float64
}

type recordedPlayContinuation struct {
	from, to    actionContext
	recordingID string
	values      recordedPlayValues
}

type recordedPlayRejection struct {
	generation  uint64
	recordingID string
	value       float64
	message     string
	help        *workbench.DiagnosticHelp
}

type recordedPlayWork struct {
	context     actionContext
	recordingID string
	values      recordedPlayValues
	field       int
	failed      bool
}

var recordedPlayFields = []struct {
	id     string
	number bool
}{{"recorded-index", true}, {"recorded-time", false}, {"recorded-speed", true}}

func recordedPlayState(r *workbench.RecordingView) recordedPlayValues {
	return recordedPlayValues{r.Index, r.Time, r.Speed}
}

func recordedPlayFieldValue(r *workbench.RecordingView, id string) float64 {
	switch id {
	case "recorded-index":
		return float64(r.Index)
	case "recorded-time":
		return r.Time
	default:
		return r.Speed
	}
}

// Only this capture's own field responses can continue a rendered Play action.
// Rejections retain their canonical value until correction or owner replacement;
// another field's successful edit must not play with the old rejected value.
func (s *screen) changeRecordedPlayField(state workbench.Snapshot, id string, params any) {
	r := state.Recording
	pending := s.pendingRecordedPlay
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || r.ID != pending.recordingID || pending.field >= len(recordedPlayFields) || recordedPlayFields[pending.field].id != id) {
		pending.failed = true
		s.recordedPlayAccepted = nil
		s.errorMessage = "Recorded playback entries changed. Review them and try again."
		s.requestHelp = nil
		s.app.Invalidate("main")
		return
	}
	var response workbench.Response
	if id == "recorded-speed" {
		response = s.invokeAtCommandRevision("recording.speed.set", params, state.CommandRevision, state.Document.Generation)
	} else {
		response = s.invokeAtCommandRevision("recording.seek", params, state.CommandRevision, state.Document.Generation)
	}
	if response.Error != nil {
		s.recordedPlayAccepted = nil
		if origin != nil && response.Error.Code == "invalid-params" {
			if s.recordedPlayRejected == nil {
				s.recordedPlayRejected = make(map[string]recordedPlayRejection)
			}
			s.recordedPlayRejected[id] = recordedPlayRejection{origin.generation, r.ID, recordedPlayFieldValue(r, id), response.Error.Message, response.Help}
			// A refusal itself is not an accepted edit. Its unchanged context
			// can only advance through our explicitly observed viewport response.
			s.recordedPlayAccepted = &recordedPlayContinuation{*origin, *origin, r.ID, recordedPlayState(r)}
		}
		if pending != nil {
			pending.failed = true
		}
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	delete(s.recordedPlayRejected, id)
	s.pruneRecordedPlayRejections(accepted)
	if origin == nil {
		return
	}
	from := *origin
	if previous := s.recordedPlayAccepted; previous != nil && previous.to == from && previous.recordingID == r.ID {
		from = previous.from
	}
	s.recordedPlayAccepted = &recordedPlayContinuation{from, actionContext{accepted.CommandRevision, accepted.Document.Generation}, r.ID, recordedPlayState(accepted.Recording)}
	if pending != nil {
		pending.context, pending.values = s.recordedPlayAccepted.to, s.recordedPlayAccepted.values
	}
}

func (s *screen) recordedPlayAction(state workbench.Snapshot) ui.Node {
	r := state.Recording
	return buttonStyle(ui.Node{ID: "recorded-play", Kind: ui.Button, Text: "Play", Accessibility: ui.Accessibility{Name: "Play"}, Disabled: r.Duration == 0 || r.Playing || s.pendingRecordedPlay != nil, Activate: func() {
		if s.pendingRecordedPlay != nil {
			return
		}
		pending := &recordedPlayWork{context: actionContext{state.CommandRevision, state.Document.Generation}, recordingID: r.ID, values: recordedPlayState(r)}
		current := s.service.Snapshot()
		if accepted := s.recordedPlayAccepted; accepted != nil && accepted.recordingID == r.ID && accepted.from.generation == pending.context.generation && accepted.from.revision <= pending.context.revision && pending.context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) && current.Recording != nil && recordedPlayState(current.Recording) == accepted.values {
			pending.context, pending.values = accepted.to, accepted.values
		}
		if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation || current.Recording == nil || current.Recording.ID != r.ID {
			s.finishRecordedPlay(pending)
			return
		}
		s.pendingRecordedPlay = pending
		s.settleRecordedPlay(pending, 0)
	}}, false)
}

func (s *screen) settleRecordedPlay(pending *recordedPlayWork, index int) {
	if s.pendingRecordedPlay != pending || pending.field != index {
		return
	}
	if pending.failed || index == len(recordedPlayFields) {
		s.pendingRecordedPlay = nil
		if pending.failed {
			s.app.Invalidate("main")
			return
		}
		s.finishRecordedPlay(pending)
		return
	}
	field := recordedPlayFields[index]
	done := func(err error) {
		if s.pendingRecordedPlay != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			pending.failed = true
			s.errorMessage = fmt.Sprintf("Could not finish recorded playback entry: %v", err)
			s.requestHelp = nil
		}
		s.settleRecordedPlay(pending, index+1)
	}
	settle := s.settleField
	if field.number {
		settle = s.settleNumber
	}
	if settle == nil {
		done(nil)
	} else if err := settle(field.id, done); err != nil {
		done(err)
	}
}

func (s *screen) finishRecordedPlay(pending *recordedPlayWork) {
	current := s.service.Snapshot()
	s.pruneRecordedPlayRejections(current)
	if current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		if current.Recording == nil || current.Recording.ID != pending.recordingID || !current.Recording.Viewing {
			s.errorMessage = "The recording changed. Review it before starting playback."
			s.requestHelp = nil
			s.app.Invalidate("main")
			return
		}
		for _, field := range recordedPlayFields {
			if rejected, ok := s.recordedPlayRejected[field.id]; ok {
				s.errorMessage, s.requestHelp = rejected.message, rejected.help
				s.requestHelpRevision = current.CommandRevision
				s.app.Invalidate("main")
				return
			}
		}
	}
	speed := pending.values.speed
	if speed == 0 {
		speed = .1
	}
	s.invokeAtCommandRevision("recording.play", workbench.RecordingPlayParams{Speed: speed}, pending.context.revision, pending.context.generation)
}

func (s *screen) pruneRecordedPlayRejections(state workbench.Snapshot) {
	for id, rejected := range s.recordedPlayRejected {
		if state.Recording == nil || !state.Recording.Viewing || state.Document.Generation != rejected.generation || state.Recording.ID != rejected.recordingID || recordedPlayFieldValue(state.Recording, id) != rejected.value {
			delete(s.recordedPlayRejected, id)
		}
	}
}

func (s *screen) continueRecordedPlayViewport(from actionContext, accepted workbench.Snapshot) {
	previous := s.recordedPlayAccepted
	if previous == nil || previous.to != from || accepted.Document.Generation != from.generation || accepted.Recording == nil || accepted.Recording.ID != previous.recordingID || recordedPlayState(accepted.Recording) != previous.values {
		return
	}
	previous.to = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	if pending := s.pendingRecordedPlay; pending != nil && pending.context == from {
		pending.context = previous.to
	}
}
