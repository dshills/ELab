package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// A spinner commits on focus loss before its Place click reaches the owner.
// Retain the parameters from our accepted response, not a later agent snapshot.
// The ordinary label TextField can finish later and uses explicit settlement.
type placementFieldContinuation struct {
	from, to actionContext
	params   workbench.PlaceParams
}

type placementActionWork struct {
	context actionContext
	params  workbench.PlaceParams
	failed  bool
}

func placementParams(state workbench.Snapshot) workbench.PlaceParams {
	params := workbench.PlaceParams{Kind: state.Drawing.Kind, Position: state.Drawing.Position}
	if params.Kind == "net-label" {
		params.Value = state.Drawing.Value
	}
	return params
}

func (s *screen) changePlacementField(command string, params any) {
	pending := s.pendingPlacement
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	if pending != nil && (origin == nil || *origin != pending.context) {
		pending.failed = true
		s.placementFieldAccepted = nil
		s.errorMessage = "Placement entry changed while Place was pending. Review it and try again."
		s.app.Invalidate("main")
		return
	}
	response := s.invoke(command, params)
	if response.Error != nil {
		s.placementFieldAccepted = nil
		if pending != nil {
			pending.failed = true
		}
		return
	}
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	acceptedParams := placementParams(accepted)
	from := *origin
	if previous := s.placementFieldAccepted; previous != nil && previous.to == from && previous.params.Kind == acceptedParams.Kind {
		from = previous.from
	}
	s.placementFieldAccepted = &placementFieldContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, params: acceptedParams}
	if pending != nil {
		pending.context = s.placementFieldAccepted.to
		pending.params = acceptedParams
	}
}

func (s *screen) placementAction(state workbench.Snapshot) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	params := placementParams(state)
	return buttonStyle(ui.Node{ID: "place-at-position", Kind: ui.Button, Text: "Place at X / Y", Accessibility: ui.Accessibility{Name: "Place at X / Y"}, Disabled: s.pendingPlacement != nil, Activate: func() {
		if s.pendingPlacement != nil {
			return
		}
		current := s.service.Snapshot()
		if accepted := s.placementFieldAccepted; accepted != nil && accepted.params.Kind == params.Kind && accepted.from.generation == context.generation && accepted.from.revision <= context.revision && context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			context = accepted.to
			params = accepted.params
		}
		if current.CommandRevision != context.revision || current.Document.Generation != context.generation {
			s.invokeAtCommandRevision("component.place", params, context.revision, context.generation)
			return
		}
		pending := &placementActionWork{context: context, params: params}
		s.pendingPlacement = pending
		if params.Kind != "net-label" || s.settleField == nil {
			s.finishPlacement(pending, nil)
		} else if err := s.settleField("place-label", func(err error) { s.finishPlacement(pending, err) }); err != nil {
			s.finishPlacement(pending, err)
		}
	}}, false)
}

func (s *screen) finishPlacement(pending *placementActionWork, err error) {
	if s.pendingPlacement != pending {
		return
	}
	s.pendingPlacement = nil
	if err != nil {
		s.errorMessage = fmt.Sprintf("Could not finish placement entry: %v", err)
	} else if !pending.failed {
		result := s.invokeAtCommandRevision("component.place", pending.params, pending.context.revision, pending.context.generation)
		if result.Error == nil {
			s.requestKeyboardFocus("part-search")
		}
	}
	s.app.Invalidate("main")
}
