package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// A native field can publish its final Change after the Apply click. Retain
// the rendered command context while settling it; only this field's accepted
// Change may advance that context. An intervening agent edit must still refuse.
type valueApply struct {
	context actionContext
	params  workbench.ValueParams
	failed  bool
}

// A focus loss can commit this field before its Apply click is delivered. Only
// consecutive accepted changes from that field may bridge the rendered context.
// Never use a later snapshot as evidence that another actor's edit was ours.
type valueDraftContinuation struct {
	from, to actionContext
	id, text string
}

func (s *screen) changeValueDraft(e ui.ChangeEvent) {
	pending := s.pendingValueApply
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	result := s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text})
	if result.Error != nil {
		s.valueDraftAccepted = nil
	}
	if result.Error == nil && origin != nil {
		accepted := result.Result.(workbench.Snapshot)
		from := *origin
		if previous := s.valueDraftAccepted; previous != nil && previous.to == from && previous.id == accepted.Selected {
			from = previous.from
		}
		s.valueDraftAccepted = &valueDraftContinuation{
			from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation},
			id: accepted.Selected, text: e.Text,
		}
	}
	if pending == nil {
		return
	}
	if result.Error != nil {
		pending.failed = true
		return
	}
	if s.fieldContext == nil || *s.fieldContext != pending.context {
		pending.failed = true
		s.errorMessage = "Value entry changed while Apply was pending. Review it and apply again."
		return
	}
	state := result.Result.(workbench.Snapshot)
	pending.context = actionContext{state.CommandRevision, state.Document.Generation}
	pending.params.Value = e.Text
}

func (s *screen) valueApplyAction(state workbench.Snapshot, id, text string) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	params := workbench.ValueParams{ID: id, Value: state.Drawing.Value}
	return buttonStyle(ui.Node{ID: "apply-value", Kind: ui.Button, Text: text, Disabled: s.pendingValueApply != nil, Accessibility: ui.Accessibility{Name: text}, Activate: func() {
		if s.pendingValueApply != nil {
			return
		}
		current := s.service.Snapshot()
		if accepted := s.valueDraftAccepted; accepted != nil && accepted.from.generation == context.generation &&
			accepted.from.revision <= context.revision && context.revision <= accepted.to.revision &&
			accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) &&
			accepted.id == id && current.Selected == id && current.Drawing.Value == accepted.text {
			context = accepted.to
			params.Value = accepted.text
		}
		if current.CommandRevision != context.revision || current.Document.Generation != context.generation {
			// Use the ordinary shared refusal without flushing an obsolete field.
			s.invokeAtCommandRevision("component.set", params, context.revision, context.generation)
			return
		}
		pending := &valueApply{context: context, params: params}
		s.pendingValueApply = pending
		done := func(err error) {
			if s.pendingValueApply != pending {
				return
			}
			s.pendingValueApply = nil
			if err != nil {
				s.errorMessage = fmt.Sprintf("Could not finish value entry: %v", err)
			} else if !pending.failed {
				s.invokeAtCommandRevision("component.set", pending.params, pending.context.revision, pending.context.generation)
			}
			s.app.Invalidate("main")
		}
		if s.settleField == nil {
			// Unmounted semantic controls already receive controlled Change events.
			done(nil)
		} else if err := s.settleField("value-entry", done); err != nil {
			done(err)
		}
	}}, false)
}
