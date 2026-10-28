package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func scopeAvailable(state workbench.Snapshot) bool {
	return state.ScopeView == "spectrum" || state.Recording != nil && state.Recording.Viewing || workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil
}
func (s *screen) scopeVisibilityButton(state workbench.Snapshot) ui.Node {
	text := "Show scope"
	if state.ScopeVisible {
		text = "Hide scope"
	}
	if !scopeAvailable(state) {
		text = "Scope"
	}
	n := s.action("scope-visibility", text, "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: !state.ScopeVisible})
	n.Disabled = !scopeAvailable(state)
	n.Accessibility.Help = "Expand or collapse the dock without changing simulation or recorded measurements"
	return n
}
func (s *screen) collapsedScope(state workbench.Snapshot) ui.Node {
	caption := "Scope collapsed · measurements retained"
	if record := state.Recording; record != nil && record.Viewing {
		caption = fmt.Sprintf("Recorded scope collapsed · sample %d · %.4g s", record.Index, record.Time)
	} else if state.Transient != nil {
		status := "live"
		if state.Paused {
			status = "paused"
		}
		if state.Transient.Stale {
			status = "stale"
		}
		caption = fmt.Sprintf("Scope collapsed · %.4g s · %s", state.Transient.Sample.Time, status)
	}
	style := chromeStyle()
	style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 10, Right: 10})
	return ui.Node{ID: "scope-panel", Kind: ui.Toolbar, Style: style, Children: []ui.Node{
		s.action("scope-expand", "Show scope", "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: true}), quiet("scope-collapsed-caption", caption),
	}}
}
