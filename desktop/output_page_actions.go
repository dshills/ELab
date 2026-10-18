package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Output spinners commit on focus loss before a page action reaches the owner. Only
// this page form's consecutive accepted responses may continue that action.
type outputPageContinuation struct {
	from, to actionContext
	page     workbench.SchematicPage
}

func (s *screen) changeOutputPage(page workbench.SchematicPage) {
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	response := s.invoke("export.page.set", page)
	if response.Error != nil {
		s.outputPageAccepted = nil
		return
	}
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.outputPageAccepted; previous != nil && previous.to == from {
		from = previous.from
	}
	s.outputPageAccepted = &outputPageContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, page: accepted.SchematicPage}
}

// A later snapshot can prove the endpoint is still current, but only the
// accepted response supplies a continued action's settings and context.
func (s *screen) continuedOutputPage(context actionContext) *outputPageContinuation {
	current := s.service.Snapshot()
	accepted := s.outputPageAccepted
	if accepted != nil && accepted.from.generation == context.generation && accepted.from.revision <= context.revision && context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
		return accepted
	}
	return nil
}

func (s *screen) outputPreviewAction(state workbench.Snapshot, id, text string) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: func() {
		if accepted := s.continuedOutputPage(context); accepted != nil {
			context = accepted.to
		}
		s.invokeAtCommandRevision("print.preview.start", workbench.PreviewParams{}, context.revision, context.generation)
	}}, false)
}

func (s *screen) outputFitAction(state workbench.Snapshot) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	page := state.SchematicPage
	return buttonStyle(ui.Node{ID: "pdf-fit", Kind: ui.Button, Text: "Fit to page", Accessibility: ui.Accessibility{Name: "Fit to page"}, Activate: func() {
		if accepted := s.continuedOutputPage(context); accepted != nil {
			context = accepted.to
			page = accepted.page
		}
		page.Scale = nil
		page.Tile = false
		page.OverlapMM = nil
		s.invokeAtCommandRevision("export.page.set", page, context.revision, context.generation)
	}}, false)
}
