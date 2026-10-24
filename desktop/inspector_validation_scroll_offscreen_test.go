//go:build offscreen

package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorValidationKeepsTheFocusedIntervalVisible(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, start, end := "live", "window-start", "window-end"
		if recorded {
			name, start, end = "recorded", "recorded-window-start", "recorded-window-end"
		}
		for _, enterEnd := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/end=%t", name, enterEnd), func(t *testing.T) {
				s, cleanup := scopeContextRC(t)
				t.Cleanup(cleanup)
				if recorded {
					recordedCursorSource(t, s.service)
				} else {
					agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", End: .02})
				}
				agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
				view := func() ui.Node {
					rail, ok := findNode(s.view(), "inspector-rail")
					if !ok {
						t.Fatal("missing production inspector")
					}
					return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
				}
				theme := inspectorLayoutTestTheme()
				theme.Default.Background = ui.Some(color.NRGBA{})
				m := newMountedControlWindowSize(t, s, view, 320, 380, theme)
				railID := inspectorScrollNode(t, s.view()).ID
				for attempt := range 120 {
					rail := mountedControlElement(t, m, railID)
					first, second := mountedControlElement(t, m, start), mountedControlElement(t, m, end)
					if !first.Bounds.Empty() && !second.Bounds.Empty() && first.Bounds.In(rail.Bounds) && second.Bounds.In(rail.Bounds) {
						break
					}
					if attempt == 119 {
						t.Fatal("could not expose both interval fields")
					}
					m.input(cogent.ManualInput{Type: events.Scroll, Target: rail.ID, TargetIncarnation: rail.Incarnation, Delta: math32.Vec2(0, 32)})
					m.frame()
				}
				edited := start
				if enterEnd {
					edited = end
				}
				typeMountedCursor(t, m, edited, "oops")
				before := s.service.Snapshot()
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				m.frame()
				focused := m.snapshot().FocusedID
				target := mountedControlElement(t, m, focused)
				rail := mountedControlElement(t, m, railID)
				t.Logf("focused=%s field=%v rail=%v", focused, target.Bounds, rail.Bounds)
				if !enterEnd && focused != end || target.Bounds.Empty() || !target.Bounds.In(rail.Bounds) {
					t.Error("validation guidance moved the focused interval out of view")
				}
				if field := mountedControlElement(t, m, edited); field.Value != "oops" {
					t.Fatal("validation lost the rejected private text", field.Value)
				}
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("validation or scroll reveal changed accepted circuit state")
				}
				if artifacts := os.Getenv("ELAB_VALIDATION_SCROLL_ARTIFACTS"); artifacts != "" {
					im, err := m.manual.Capture(m.ctx, "main")
					if err != nil {
						t.Fatal(err)
					}
					file, err := os.OpenFile(filepath.Join(artifacts, fmt.Sprintf("%s-end-%t.png", name, enterEnd)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if err != nil {
						t.Fatal(err)
					}
					encodeErr, closeErr := png.Encode(file, im), file.Close()
					if encodeErr != nil || closeErr != nil {
						t.Fatal(encodeErr, closeErr)
					}
				}
			})
		}
	}
}
