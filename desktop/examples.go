package main

import (
	"fmt"

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
	nodes := []ui.Node{section("example-guidance-title", info.Title), quiet("example-origin-status", status), quiet("example-question", info.Question), section("example-try-title", "Try this"), quiet("example-try", info.Try), section("example-observe-title", "What to look for"), quiet("example-expected", "Original template expectation: "+info.Expected)}
	if len(state.Example.Changes) > 0 {
		nodes = append(nodes, section("example-changes-title", "Your changes"), quiet("example-change-help", "Compared with the original template. The expectation above describes that template; your edited circuit may behave differently."))
		for i, change := range state.Example.Changes {
			nodes = append(nodes, quiet(fmt.Sprintf("example-change-%d", i), change.Summary))
			for j, detail := range change.Details {
				nodes = append(nodes, quiet(fmt.Sprintf("example-change-%d-detail-%d", i, j), detail))
			}
		}
	}
	return append(nodes, quiet("example-limitation", info.Limitation), reset)
}
