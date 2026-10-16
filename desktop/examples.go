package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) exampleGuidance(state workbench.Snapshot) []ui.Node {
	if state.Example == nil {
		return nil
	}
	info := state.Example.Info
	status := "Original template circuit"
	if state.Example.Modified {
		status = "Based on this example · circuit/experiment edited"
	}
	generation := state.Document.Generation
	reset := s.button("example.current.reset", "Reset example")
	reset.Accessibility.Name = "Reset " + info.Title
	reset.Activate = func() {
		s.requestReplace("example.current.reset", workbench.ExampleResetParams{ExpectedID: info.ID, ExpectedGeneration: &generation})
	}
	return []ui.Node{section("example-guidance-title", info.Title), quiet("example-origin-status", status), quiet("example-question", info.Question), quiet("example-expected", "Template expectation: "+info.Expected), quiet("example-try", info.Try), quiet("example-limitation", info.Limitation), reset}
}
