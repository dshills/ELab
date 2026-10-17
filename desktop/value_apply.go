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

func (s *screen) changeValueDraft(e ui.ChangeEvent) {
	pending := s.pendingValueApply
	result := s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text})
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
