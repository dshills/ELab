package main

import (
	"strconv"

	"github.com/dshills/vellum/ui"
)

// Bind before platform selector adaptation so asynchronous native choices keep
// the original field callback's context too. All callbacks run on the owner.
func (s *screen) bindFieldContext(node *ui.Node, context actionContext) {
	if ui.SupportsControlStateKey(node.Kind) {
		// A new document must not inherit private edits or a pressed control,
		// even when its public ID and controlled value return in one redraw.
		node.ControlStateKey = strconv.FormatUint(context.generation, 10) + ":" + node.ControlStateKey
	}
	if node.Change != nil {
		change := node.Change
		node.Change = func(event ui.ChangeEvent) {
			previous, previousAccepted := s.fieldContext, s.fieldAccepted
			s.fieldContext = &context
			s.fieldAccepted = nil
			defer func() { s.fieldContext, s.fieldAccepted = previous, previousAccepted }()
			change(event)
			// Key repeats can outpace rendering. Only this field's own accepted
			// result may continue its binding; another field or agent still
			// invalidates the captured revision and document generation.
			if s.fieldAccepted != nil {
				context = *s.fieldAccepted
			}
		}
	}
	for i := range node.Children {
		s.bindFieldContext(&node.Children[i], context)
	}
}

// The same inspector control IDs serve different selected objects. Their private
// native state belongs to that object rather than the reusable public control ID.
func scopeComponentControlState(node *ui.Node, id string) {
	if ui.SupportsControlStateKey(node.Kind) {
		node.ControlStateKey = id
	}
	for i := range node.Children {
		scopeComponentControlState(&node.Children[i], id)
	}
}
