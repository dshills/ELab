package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/abilities"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// The plot suppresses MeshCanvas zoom. In a short dock, pass its wheel to the
// containing managed scroll frame so cursor controls remain reachable. Forward
// only to this named host; never pan/zoom the scope or scroll an unrelated pane.
func forwardScopeScroll(widget *core.WidgetBase, event events.Event) {
	for parent := widget.Parent; parent != nil; parent = parent.AsTree().Parent {
		if parent.AsTree().Name != "scope-scroll" {
			continue
		}
		if host, ok := parent.(core.Widget); ok {
			w := host.AsWidget()
			previous := w.Styles.Abilities
			w.Styles.SetAbilities(true, abilities.ScrollableUnattended)
			w.HandleEvent(event)
			w.Styles.Abilities = previous
		}
		break
	}
	event.SetHandled()
}

func (s *screen) workspaceDock(workspace, scope ui.Node, state workbench.Snapshot) ui.Node {
	// Let the scroll viewport constrain allocation, not the scope's intrinsic
	// control height. Small split panes retain every control through scrolling.
	scope.Grow = true
	scope.Style.Grow = ui.Some(float32(1))
	scope.Style.MaxSize = ui.Some(ui.Size{})
	return ui.Node{ID: "workspace-split", Kind: ui.SplitPane, Grow: true,
		Orientation: ui.Vertical, Number: float32(state.WorkspaceLayout.SchematicFraction),
		Numeric:       ui.Numeric{Min: .35, Max: .85, Step: .02},
		Accessibility: ui.Accessibility{Name: "Schematic and scope divider", Help: "Drag to resize; Up/Down adjusts the schematic share, Home/End selects its limits. Scope controls scroll when space is limited."},
		Change: func(e ui.ChangeEvent) {
			fraction := float64(e.Number)
			if e.Number == float32(.35) {
				fraction = .35
			} else if e.Number == float32(.85) {
				fraction = .85
			}
			s.invokeAtCommandRevision("workspace.layout.set", workbench.WorkspaceLayout{SchematicFraction: fraction}, state.CommandRevision, state.Document.Generation)
		},
		Children: []ui.Node{workspace, {ID: "scope-scroll", Kind: ui.Scroll, Grow: true,
			Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), MinSize: ui.Some(ui.Size{Height: 64}), Background: ui.Some(paper)}, Children: []ui.Node{scope}}},
	}
}
