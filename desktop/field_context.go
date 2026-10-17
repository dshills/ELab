package main

import "github.com/dshills/vellum/ui"

// Bind before platform selector adaptation so asynchronous native choices keep
// the original field callback's context too. All callbacks run on the owner.
func (s *screen) bindFieldContext(node *ui.Node, context actionContext) {
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
