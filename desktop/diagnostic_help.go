package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) diagnosticControls(state workbench.Snapshot) []ui.Node {
	items := append([]workbench.DiagnosticHelp{}, state.DiagnosticPresentation.Items...)
	if s.requestHelp != nil && s.requestHelp.Details == s.errorMessage {
		items = append(items, *s.requestHelp)
	}
	if len(items) == 0 {
		return nil
	}
	var nodes []ui.Node
	for i, help := range items {
		id := fmt.Sprintf("diagnostic-%d", i)
		nodes = append(nodes, quiet(id+"-summary", help.Summary), quiet(id+"-action", help.Action))
		refs := append(append(append([]string{}, help.Objects...), help.Terminals...), help.Wires...)
		if len(refs) > 0 {
			nodes = append(nodes, quiet(id+"-related", "Highlighted: "+strings.Join(refs, ", ")))
		}
		nodes = append(nodes, quiet(id+"-technical", help.Source+" · "+help.Code+"\n"+help.Details))
	}
	return []ui.Node{{ID: "diagnostic-details", Kind: ui.Disclosure, Text: fmt.Sprintf("All guidance & technical details (%d)", len(items)), Checked: state.DiagnosticPresentation.DetailsVisible, Accessibility: ui.Accessibility{Name: "Show diagnostic technical details"}, Children: nodes, Change: func(e ui.ChangeEvent) {
		s.invoke("diagnostics.details.set", workbench.DiagnosticDetailsParams{Visible: e.Checked})
	}}}
}
