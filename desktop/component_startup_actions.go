package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type componentStartupContinuation struct {
	from, to actionContext
	id, kind string
}
type componentStartupRejection struct {
	generation        uint64
	id, text, message string
	help              *workbench.DiagnosticHelp
}
type componentStartupWork struct {
	context       actionContext
	part          workbench.Component
	reset, failed bool
}

func componentStartupField(part workbench.Component) string {
	if part.Kind == "inductor" {
		return "inductor-initial"
	}
	return "cap-initial"
}

func startupCurrentText(part workbench.Component) string {
	value := 0.
	if part.InitialCurrent != nil {
		value = *part.InitialCurrent
	}
	return workbench.FormatValue(value, "A")
}

func (s *screen) pruneComponentStartup(state workbench.Snapshot) {
	rejected := s.componentStartupRejected
	if rejected == nil {
		return
	}
	part, ok := (workbench.Graph{Components: state.Components}).Component(state.Selected)
	if !ok || part.Kind != "inductor" || part.ID != rejected.id || state.Document.Generation != rejected.generation || startupCurrentText(part) != rejected.text {
		s.componentStartupRejected = nil
	}
}

func (s *screen) changeComponentStartup(state workbench.Snapshot, part workbench.Component, value string) {
	pending := s.pendingComponentStartup
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context || pending.part.ID != part.ID || pending.part.Kind != part.Kind) {
		pending.failed = true
		s.componentStartupAccepted = nil
		s.errorMessage = "Startup entry changed. Review it and try again."
		s.app.Invalidate("main")
		return
	}
	params := workbench.ValueParams{ID: part.ID, Value: value}
	var response workbench.Response
	if part.Kind == "inductor" {
		response = s.invokeAtCommandRevision("inductor.initial.set", params, state.CommandRevision, state.Document.Generation)
	} else {
		response = s.invokeAtCommandRevision("capacitor.initial.set", params, state.CommandRevision, state.Document.Generation)
	}
	if response.Error != nil {
		s.componentStartupAccepted = nil
		if origin != nil && part.Kind == "inductor" && response.Error.Code == "invalid-params" {
			s.componentStartupRejected = &componentStartupRejection{generation: origin.generation, id: part.ID, text: startupCurrentText(part), message: response.Error.Message, help: response.Help}
		}
		if pending != nil && (!pending.reset || response.Error.Code != "invalid-params") {
			pending.failed = true
		}
		return
	}
	s.componentStartupRejected = nil
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.componentStartupAccepted; previous != nil && previous.to == from && previous.id == part.ID && previous.kind == part.Kind {
		from = previous.from
	}
	s.componentStartupAccepted = &componentStartupContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, id: part.ID, kind: part.Kind}
	if pending != nil {
		pending.context = s.componentStartupAccepted.to
	}
}

func (s *screen) componentStartupDefault(state workbench.Snapshot, part workbench.Component) ui.Node {
	id, text := "inductor-initial-default", "Reset startup to zero"
	if part.Kind == "capacitor" {
		id, text = "cap-initial-default", "Use experiment initial voltage"
	}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Disabled: s.pendingComponentStartup != nil, Activate: func() { s.beginComponentStartup(state, part, true) }}, false)
}

func (s *screen) beginComponentStartup(state workbench.Snapshot, part workbench.Component, reset bool) {
	if s.pendingComponentStartup != nil {
		return
	}
	pending := &componentStartupWork{context: actionContext{state.CommandRevision, state.Document.Generation}, part: part, reset: reset}
	previousMenu := s.menuCommandRevision
	if previousMenu != nil {
		pending.context.revision = *previousMenu
	}
	s.menuCommandRevision = nil
	defer func() { s.menuCommandRevision = previousMenu }()
	current := s.service.Snapshot()
	if accepted := s.componentStartupAccepted; accepted != nil && accepted.id == part.ID && accepted.kind == part.Kind && accepted.from.generation == pending.context.generation && accepted.from.revision <= pending.context.revision && pending.context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) && current.Selected == part.ID {
		pending.context = accepted.to
	}
	if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation || current.Selected != part.ID {
		s.finishComponentStartup(pending)
		return
	}
	s.pendingComponentStartup = pending
	done := func(err error) {
		if s.pendingComponentStartup != pending {
			return
		}
		s.pendingComponentStartup = nil
		// The pinned legacy NumberField reports this validation refusal without
		// an exported sentinel. Reset explicitly discards it, while ownership,
		// composition, cancellation and every other barrier error still refuse.
		discard := reset && part.Kind == "capacitor" && err != nil && err.Error() == "cogent: invalid number field entry"
		if err != nil && !discard {
			s.errorMessage = err.Error()
			s.app.Invalidate("main")
			return
		}
		if pending.failed {
			s.app.Invalidate("main")
			return
		}
		s.finishComponentStartup(pending)
	}
	field := componentStartupField(part)
	settle := s.settleField
	if part.Kind == "capacitor" {
		settle = s.settleNumber
	}
	if settle == nil {
		done(nil)
	} else if err := settle(field, done); err != nil {
		done(err)
	}
}

func (s *screen) finishComponentStartup(pending *componentStartupWork) {
	current := s.service.Snapshot()
	s.pruneComponentStartup(current)
	if !pending.reset && current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		if rejected := s.componentStartupRejected; rejected != nil && rejected.id == pending.part.ID {
			s.errorMessage, s.requestHelp = rejected.message, rejected.help
			s.requestHelpRevision = current.CommandRevision
			s.app.Invalidate("main")
			return
		}
	}
	var response workbench.Response
	if pending.reset {
		params := workbench.ValueParams{ID: pending.part.ID, Value: "default"}
		if pending.part.Kind == "inductor" {
			response = s.invokeAtCommandRevision("inductor.initial.set", params, pending.context.revision, pending.context.generation)
		} else {
			response = s.invokeAtCommandRevision("capacitor.initial.set", params, pending.context.revision, pending.context.generation)
		}
	} else {
		response = s.invokeAtCommandRevision("simulation.restart", struct{}{}, pending.context.revision, pending.context.generation)
	}
	if pending.reset && response.Error == nil {
		// Also retire invalid private text when the controlled value was already
		// default, so it cannot reappear or publish a late Change after Reset.
		s.componentStartupEpoch++
		s.componentStartupRejected = nil
		s.app.Invalidate("main")
	}
}

func (s *screen) continueComponentStartupViewport(from actionContext, accepted workbench.Snapshot) {
	previous := s.componentStartupAccepted
	if previous == nil || previous.to != from || accepted.Document.Generation != from.generation || accepted.Selected != previous.id {
		return
	}
	previous.to = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	if pending := s.pendingComponentStartup; pending != nil && pending.context == from {
		pending.context = previous.to
	}
}

func (s *screen) startupControlStateKey(part string) string {
	return fmt.Sprintf("%s:startup:%d", part, s.componentStartupEpoch)
}
