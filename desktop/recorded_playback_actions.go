package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type recordedPlaybackValues struct {
	index       int
	time, speed float64
}

type recordedPlaybackContinuation struct {
	from, to    actionContext
	recordingID string
	values      recordedPlaybackValues
	playing     bool
}

type recordedPlaybackRejection struct {
	generation  uint64
	recordingID string
	value       float64
	text        string
	number      float32
	message     string
	help        *workbench.DiagnosticHelp
}

type recordedPlaybackWork struct {
	context     actionContext
	recordingID string
	values      recordedPlaybackValues
	field       int
	failed      bool
	direction   string
}

// Managed time can publish focus-loss Change while the first native barrier is
// pending. Settle it before legacy numeric controls so its own callback belongs
// to that barrier. Later-field, stale-origin and unrelated-work guards stay exact.
var recordedPlaybackFields = []struct {
	id     string
	number bool
}{{"recorded-time", false}, {"recorded-index", true}, {"recorded-speed", true}}

func recordedPlaybackState(r *workbench.RecordingView) recordedPlaybackValues {
	return recordedPlaybackValues{r.Index, r.Time, r.Speed}
}

func recordedPlaybackFieldValue(r *workbench.RecordingView, id string) float64 {
	switch id {
	case "recorded-index":
		return float64(r.Index)
	case "recorded-time":
		return r.Time
	default:
		return r.Speed
	}
}

// Only this capture's own field responses can continue a related viewer action.
// Rejections retain the input and its accepted value until correction or owner replacement;
// another field's successful edit must not play with the old rejected value.
func (s *screen) changeRecordedPlaybackField(state workbench.Snapshot, id string, entry ui.ChangeEvent) {
	r := state.Recording
	pending := s.pendingRecordedPlayback
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || r.ID != pending.recordingID || pending.field >= len(recordedPlaybackFields) || recordedPlaybackFields[pending.field].id != id) {
		pending.failed = true
		s.recordedPlaybackAccepted = nil
		s.errorMessage = "Recorded playback entries changed. Review them and try again."
		s.requestHelp = nil
		s.app.Invalidate("main")
		return
	}
	var params any
	switch id {
	case "recorded-index":
		params = map[string]any{"index": entry.Number}
	case "recorded-time":
		params = map[string]any{"time_s": entry.Text}
	default:
		params = workbench.RecordingPlayParams{Speed: float64(entry.Number)}
	}
	var response workbench.Response
	if id == "recorded-speed" {
		response = s.invokeAtCommandRevision("recording.speed.set", params, state.CommandRevision, state.Document.Generation)
	} else {
		response = s.invokeAtCommandRevision("recording.seek", params, state.CommandRevision, state.Document.Generation)
	}
	if response.Error != nil {
		s.recordedPlaybackAccepted = nil
		if origin != nil && response.Error.Code == "invalid-params" {
			if s.recordedPlaybackRejected == nil {
				s.recordedPlaybackRejected = make(map[string]recordedPlaybackRejection)
			}
			s.recordedPlaybackRejected[id] = recordedPlaybackRejection{
				generation: origin.generation, recordingID: r.ID,
				value: recordedPlaybackFieldValue(r, id), text: entry.Text, number: entry.Number,
				message: response.Error.Message, help: response.Help,
			}
			// A refusal itself is not an accepted edit. Its unchanged context
			// can only advance through our explicitly observed viewport response.
			s.recordedPlaybackAccepted = &recordedPlaybackContinuation{from: *origin, to: *origin, recordingID: r.ID, values: recordedPlaybackState(r), playing: r.Playing}
		}
		if pending != nil {
			pending.failed = true
		}
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	delete(s.recordedPlaybackRejected, id)
	s.pruneRecordedPlaybackRejections(accepted)
	if origin == nil {
		return
	}
	from := *origin
	if previous := s.recordedPlaybackAccepted; previous != nil && previous.to == from && previous.recordingID == r.ID {
		from = previous.from
	}
	s.recordedPlaybackAccepted = &recordedPlaybackContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, recordingID: r.ID, values: recordedPlaybackState(accepted.Recording), playing: accepted.Recording.Playing}
	if pending != nil {
		pending.context, pending.values = s.recordedPlaybackAccepted.to, s.recordedPlaybackAccepted.values
	}
}

func (s *screen) recordedPlaybackAction(state workbench.Snapshot, direction string) ui.Node {
	r := state.Recording
	id, label, help := "recorded-play", "Play", ""
	disabled := r.Duration == 0 || r.Playing
	if direction != "" {
		id, label = "recorded-"+direction, map[string]string{"previous": "Previous", "next": "Next"}[direction]
		// The accepted index can still be at a boundary while a new position
		// is private in a native field. Let settlement precede shared boundary
		// validation; single-row DC recordings still have no step available.
		disabled = r.Total <= 1
		help = "Applies your position entry, then moves one captured sample. Moving beyond either end reports the recording boundary."
	}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: label, Accessibility: ui.Accessibility{Name: label, Help: help}, Disabled: disabled || s.pendingRecordedPlayback != nil, Activate: func() {
		if s.pendingRecordedPlayback != nil {
			return
		}
		pending := &recordedPlaybackWork{context: actionContext{state.CommandRevision, state.Document.Generation}, recordingID: r.ID, values: recordedPlaybackState(r), direction: direction}
		current := s.service.Snapshot()
		if accepted := s.recordedPlaybackAccepted; accepted != nil && accepted.recordingID == r.ID && accepted.from.generation == pending.context.generation && accepted.from.revision <= pending.context.revision && pending.context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) && current.Recording != nil && recordedPlaybackValuesMatch(accepted, current.Recording, direction != "") {
			pending.context, pending.values = accepted.to, accepted.values
		}
		if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation || current.Recording == nil || current.Recording.ID != r.ID {
			s.finishRecordedPlayback(pending)
			return
		}
		s.pendingRecordedPlayback = pending
		s.settleRecordedPlayback(pending, 0)
	}}, false)
}

func (s *screen) settleRecordedPlayback(pending *recordedPlaybackWork, index int) {
	if s.pendingRecordedPlayback != pending || pending.field != index {
		return
	}
	if pending.failed || index == len(recordedPlaybackFields) {
		s.pendingRecordedPlayback = nil
		if pending.failed {
			s.app.Invalidate("main")
			return
		}
		s.finishRecordedPlayback(pending)
		return
	}
	field := recordedPlaybackFields[index]
	done := func(err error) {
		if s.pendingRecordedPlayback != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			pending.failed = true
			s.errorMessage = fmt.Sprintf("Could not finish recorded playback entry: %v", err)
			s.requestHelp = nil
		}
		s.settleRecordedPlayback(pending, index+1)
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

func (s *screen) finishRecordedPlayback(pending *recordedPlaybackWork) {
	current := s.service.Snapshot()
	s.pruneRecordedPlaybackRejections(current)
	if current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		if current.Recording == nil || current.Recording.ID != pending.recordingID || !current.Recording.Viewing {
			s.errorMessage = "The recording changed. Review it before continuing recorded playback."
			s.requestHelp = nil
			s.app.Invalidate("main")
			return
		}
		for _, field := range recordedPlaybackFields {
			if rejected, ok := s.recordedPlaybackRejected[field.id]; ok {
				s.errorMessage, s.requestHelp = rejected.message, rejected.help
				s.requestHelpRevision = current.CommandRevision
				s.app.Invalidate("main")
				return
			}
		}
	}
	if pending.direction != "" {
		s.invokeAtCommandRevision("recording.step", workbench.RecordingStepParams{Direction: pending.direction}, pending.context.revision, pending.context.generation)
		return
	}
	speed := pending.values.speed
	if speed == 0 {
		speed = .1
	}
	s.invokeAtCommandRevision("recording.play", workbench.RecordingPlayParams{Speed: speed}, pending.context.revision, pending.context.generation)
}

func (s *screen) pruneRecordedPlaybackRejections(state workbench.Snapshot) {
	for id, rejected := range s.recordedPlaybackRejected {
		if state.Recording == nil || !state.Recording.Viewing || state.Document.Generation != rejected.generation || state.Recording.ID != rejected.recordingID || recordedPlaybackFieldValue(state.Recording, id) != rejected.value {
			delete(s.recordedPlaybackRejected, id)
		}
	}
}

func (s *screen) continueRecordedPlaybackViewport(from actionContext, accepted workbench.Snapshot) {
	previous := s.recordedPlaybackAccepted
	if previous == nil || previous.to != from || accepted.Document.Generation != from.generation || accepted.Recording == nil || accepted.Recording.ID != previous.recordingID || !recordedPlaybackValuesMatch(previous, accepted.Recording, true) {
		return
	}
	previous.to = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	if pending := s.pendingRecordedPlayback; pending != nil && pending.context == from {
		pending.context = previous.to
	}
}

func recordedPlaybackValuesMatch(accepted *recordedPlaybackContinuation, current *workbench.RecordingView, allowTicks bool) bool {
	// After our own speed response, the viewer ticker can advance (or reach
	// the end) without a command revision change. Stepping uses the current
	// stored row. It must not mistake that presentation progress for an agent
	// edit, or borrow another speed. Play retains its stricter paused values.
	return recordedPlaybackState(current) == accepted.values || allowTicks && accepted.playing && current.Speed == accepted.values.speed
}
