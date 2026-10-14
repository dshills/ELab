package main

import "github.com/dshills/vellum/ui"

// Bind before platform selector adaptation so asynchronous native choices keep
// the original field callback's context too. All callbacks run on the owner.
func (s *screen) bindFieldContext(node *ui.Node, context actionContext) {
	if node.Change != nil {
		change := node.Change
		node.Change = func(event ui.ChangeEvent) {
			previous := s.fieldContext
			s.fieldContext = &context
			defer func() { s.fieldContext = previous }()
			change(event)
		}
	}
	for i := range node.Children {
		s.bindFieldContext(&node.Children[i], context)
	}
}
