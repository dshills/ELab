package main

import (
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type verificationContinuation struct {
	from, to actionContext
	settings workbench.VerificationSettings
}

type verificationFieldRejection struct {
	generation uint64
	value      float64
	text       string
	message    string
	help       *workbench.DiagnosticHelp
}

type verificationField struct {
	id     string
	number bool
}

type verificationActionWork struct {
	context   actionContext
	settings  workbench.VerificationSettings
	fields    []verificationField
	field     int
	reset     bool
	transient bool
	failed    bool
	details   *workbench.WorkspaceDetailsParams
}

// Only accepted responses from this form can continue a related action. Keep
// their settings with the context; a later snapshot only proves freshness.
func (s *screen) changeVerificationSettings(state workbench.Snapshot, id string, params any) {
	s.changeVerificationField(state, id, params, "")
}

func (s *screen) changeVerificationTime(state workbench.Snapshot, id, name, text string) {
	s.changeVerificationField(state, id, textFieldParams(state.VerificationSettings, name, text), text)
}

func (s *screen) changeVerificationField(state workbench.Snapshot, id string, params any, text string) {
	pending := s.pendingVerificationAction
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || pending.field >= len(pending.fields) || pending.fields[pending.field].id != id) {
		pending.failed = true
		s.verificationAccepted = nil
		s.errorMessage = "Verification settings changed. Review them and try again."
		s.app.Invalidate("main")
		return
	}
	response := s.invokeAtCommandRevision("verification.settings.set", params, state.CommandRevision, state.Document.Generation)
	if response.Error != nil {
		s.verificationAccepted = nil
		if s.fieldContextCurrent(origin) && response.Help != nil && response.Help.Code == "invalid-params" && (id == "verify-duration" || id == "verify-step") {
			if s.verificationRejected == nil {
				s.verificationRejected = make(map[string]verificationFieldRejection)
			}
			s.verificationRejected[id] = verificationFieldRejection{generation: origin.generation, value: verificationTextValue(state.VerificationSettings, id), text: text, message: response.Error.Message, help: response.Help}
		}
		if pending != nil {
			pending.failed = true
		}
		return
	}
	delete(s.verificationRejected, id)
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.verificationAccepted; previous != nil && previous.to == from {
		from = previous.from
	}
	s.verificationAccepted = &verificationContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, settings: accepted.VerificationSettings}
	if pending != nil {
		pending.context = s.verificationAccepted.to
		pending.settings = accepted.VerificationSettings
	}
}

// Showing/clearing field feedback can resize the native canvas. Continue only
// an exact owner-published viewport response after our own accepted settings;
// agent or other field commands cannot supply that context.
func (s *screen) continueVerificationViewport(from actionContext, accepted workbench.Snapshot) {
	previous := s.verificationAccepted
	if previous == nil || previous.to != from || accepted.Document.Generation != from.generation || !reflect.DeepEqual(previous.settings, accepted.VerificationSettings) {
		return
	}
	previous.to = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	if pending := s.pendingVerificationAction; pending != nil && pending.context == from {
		pending.context = previous.to
	}
}

func (s *screen) verificationAction(state workbench.Snapshot, reset bool) ui.Node {
	return s.verificationRelatedAction(state, reset, nil)
}

func (s *screen) verificationRelatedAction(state workbench.Snapshot, reset bool, details *workbench.WorkspaceDetailsParams) ui.Node {
	id, text := "verify-start", "Verify"
	if reset {
		id, text = "verify-project-startup", "Use project startup"
	}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Disabled: s.pendingVerificationAction != nil, Activate: func() {
		if s.pendingVerificationAction != nil {
			return
		}
		context := actionContext{state.CommandRevision, state.Document.Generation}
		pending := &verificationActionWork{context: context, settings: state.VerificationSettings, reset: reset, details: details, transient: workbench.IsTransientAnalysis(state.Analysis)}
		current := s.service.Snapshot()
		if accepted := s.verificationAccepted; accepted != nil && accepted.from.generation == context.generation && accepted.from.revision <= context.revision && context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			pending.context, pending.settings = accepted.to, accepted.settings
		}
		if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation {
			s.finishVerificationRequest(pending)
			return
		}
		if pending.transient {
			pending.fields = []verificationField{{id: "verify-duration"}}
			if state.WorkspaceDetails.Recordings {
				pending.fields = append(pending.fields, verificationField{id: "verify-step"}, verificationField{id: "verify-initial", number: true})
				for _, part := range state.Components {
					if part.Kind == "capacitor" {
						pending.fields = append(pending.fields, verificationField{id: "verify-cap-" + part.ID, number: true})
					}
				}
			}
		}
		s.pendingVerificationAction = pending
		s.settleVerificationAction(pending, 0)
	}}, false)
}

func (s *screen) settleVerificationAction(pending *verificationActionWork, index int) {
	if s.pendingVerificationAction != pending || pending.field != index {
		return
	}
	if pending.failed || index == len(pending.fields) {
		s.pendingVerificationAction = nil
		if pending.failed {
			s.app.Invalidate("main")
			return
		}
		s.finishVerificationRequest(pending)
		return
	}
	field := pending.fields[index]
	done := func(err error) {
		if s.pendingVerificationAction != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			pending.failed = true
			s.errorMessage = err.Error()
		}
		s.settleVerificationAction(pending, index+1)
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

func (s *screen) finishVerificationRequest(pending *verificationActionWork) {
	current := s.service.Snapshot()
	s.pruneVerificationRejections(current)
	if pending.details == nil && current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		// A settled field need not emit Change again for previously delivered
		// invalid text. Another field's success must not turn that draft into
		// a capture using the old accepted value. Check after settlement so a
		// corrected accessibility activation can first commit its new draft.
		for _, id := range []string{"verify-duration", "verify-step"} {
			if rejected, ok := s.verificationRejected[id]; ok {
				s.errorMessage, s.requestHelp = rejected.message, rejected.help
				s.requestHelpRevision = current.CommandRevision
				s.app.Invalidate("main")
				return
			}
		}
	}
	if pending.details != nil {
		s.invokeAtCommandRevision("workspace.details.set", *pending.details, pending.context.revision, pending.context.generation)
		return
	}
	if pending.reset {
		settings := workbench.DefaultVerificationSettings()
		settings.Duration = pending.settings.Duration
		s.invokeAtCommandRevision("verification.settings.set", settings, pending.context.revision, pending.context.generation)
		return
	}
	duration := 0.
	if pending.transient {
		duration = pending.settings.Duration
	}
	s.invokeAtCommandRevision("simulation.verify", workbench.VerifyParams{Duration: &duration}, pending.context.revision, pending.context.generation)
}

func verificationTextValue(settings workbench.VerificationSettings, id string) float64 {
	if id == "verify-step" {
		return settings.Step
	}
	return settings.Duration
}

// Controlled value or document changes replace private native text. Refusals
// otherwise belong to their individual fields, independent of the visible
// message and other accepted settings or agent edits.
func (s *screen) pruneVerificationRejections(state workbench.Snapshot) {
	for id, rejected := range s.verificationRejected {
		if rejected.generation != state.Document.Generation || !workbench.IsTransientAnalysis(state.Analysis) || rejected.value != verificationTextValue(state.VerificationSettings, id) {
			delete(s.verificationRejected, id)
		}
	}
}
