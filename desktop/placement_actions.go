package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Retain the parameters from our accepted response, not a later agent snapshot.
// Explicit barriers also cover activation that does not move pointer focus.
type placementFieldContinuation struct {
	from, to actionContext
	params   workbench.PlaceParams
}

type placementActionWork struct {
	context actionContext
	params  workbench.PlaceParams
	failed  bool
	stage   int
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
		s.settlePlacement(pending, 0)
	}}, false)
}

func (s *screen) settlePlacement(pending *placementActionWork, stage int) {
	if s.pendingPlacement != pending || pending.stage != stage {
		return
	}
	if pending.failed {
		s.finishPlacement(pending, nil)
		return
	}
	fields := []string{"position-x", "position-y"}
	if pending.params.Kind == "net-label" {
		fields = append(fields, "place-label")
	}
	if stage == len(fields) {
		s.finishPlacement(pending, nil)
		return
	}
	settle := s.settleNumber
	if fields[stage] == "place-label" {
		settle = s.settleField
	}
	done := func(err error) {
		if s.pendingPlacement != pending || pending.stage != stage {
			return
		}
		if err != nil {
			s.finishPlacement(pending, err)
			return
		}
		pending.stage++
		s.settlePlacement(pending, stage+1)
	}
	if settle == nil {
		done(nil)
	} else if err := settle(fields[stage], done); err != nil {
		done(err)
	}
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
