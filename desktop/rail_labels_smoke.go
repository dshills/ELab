package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/core"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Exercise real Vellum label mounting, text shaping and final allocation. A
// source-tree max-width assertion alone cannot detect shaped text being clipped.
func nativeRailLabelsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	if err := nativeInspectorTextSmoke(ctx, a, s); err != nil {
		return err
	}
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id string
			p  any
		}{
			{"project.clear", workbench.ReplaceParams{DiscardChanges: true}},
			{"simulation.pause", struct{}{}},
			{"component.place", workbench.PlaceParams{Kind: "potentiometer"}},
		} {
			if r := s.invoke(action.id, action.p); r.Error != nil {
				done <- fmt.Errorf("rail label fixture: %s", r.Error.Message)
				return
			}
		}
		done <- smokeUI(s, "inspector-section-component", nil)
	}); err != nil {
		return err
	}
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	type geometry struct {
		ID           string  `json:"id"`
		Allocated    float32 `json:"allocated_width_px"`
		Actual       float32 `json:"actual_width_px"`
		Shaped       float32 `json:"shaped_right_px"`
		Lines        int     `json:"lines"`
		CompleteText bool    `json:"complete_text"`
		Truncated    bool    `json:"truncated"`
	}
	ids := []string{"wiper-label", "pot-actuation-help", "pot-model"}
	var measured []geometry
	_, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		if len(state.Components) != 1 || state.Components[0].Kind != "potentiometer" || state.Selected != state.Components[0].ID || s.drawing.widget == nil {
			return false
		}
		labels := map[string]*core.Text{}
		s.drawing.widget.Scene.WidgetWalkDown(func(w core.Widget, base *core.WidgetBase) bool {
			if text, ok := w.(*core.Text); ok {
				labels[base.Name] = text
			}
			return true
		})
		root := s.view()
		var sample []geometry
		for _, id := range ids {
			label := labels[id]
			node, ok := findNode(root, id)
			if !ok || label == nil || label.PaintText() == nil || label.Geom.Size.Actual.Content.X <= 0 {
				return false
			}
			paint := label.PaintText()
			same := strings.Join(strings.Fields(string(paint.Source.Join())), " ") == strings.Join(strings.Fields(node.Text), " ")
			sample = append(sample, geometry{ID: id, Allocated: label.Geom.Size.Alloc.Content.X, Actual: label.Geom.Size.Actual.Content.X, Shaped: paint.Bounds.Max.X, Lines: len(paint.Lines), CompleteText: same, Truncated: paint.Truncated})
		}
		measured = sample
		return true
	})
	if err != nil {
		return err
	}
	passed := true
	for _, sample := range measured {
		// One pixel accommodates the final integer box rounding; the defect
		// shapes to the parent width then clips twelve logical units away.
		passed = passed && sample.CompleteText && !sample.Truncated && sample.Lines > 0 && sample.Shaped <= sample.Actual+1
	}
	// Separate diagnostic: compare the public native Cocoa inspection with
	// the external CUA observation. Presence alone is not reader acceptance.
	probe := map[string]any{}
	if inspection, err := cogent.InspectAccessibility(ctx, "main"); err != nil {
		probe["error"] = err.Error()
	} else {
		probe["reachable"], probe["native_window"] = inspection.Reachable, inspection.NativeWindow
		var controls []cogent.AccessibilityElement
		for _, element := range inspection.Elements {
			switch element.DomainID {
			case "selection", "pot-wiper", "pot-wiper-exact", "value-entry", "part-assumptions", "pot-actuation-help":
				controls = append(controls, element)
			}
		}
		probe["selected_controls"] = controls
	}
	evidence, _ := json.Marshal(map[string]any{"event": "elab.rail-labels.smoke", "passed": passed, "labels": measured, "native_accessibility_diagnostic": probe})
	log.Print(string(evidence))
	if !passed {
		return fmt.Errorf("rail text exceeds its actual content box or lost source text")
	}
	// The newly placed part leaves the protected-close fixture dirty.
	return nil
}

// Text-only sections must use the rail's available wrapping width as well as
// contain their shaped text. Check native allocations, independent of font/DPI.
func nativeInspectorTextSmoke(ctx context.Context, a *app.Application, s *screen) error {
	type geometry struct {
		Example      string  `json:"example"`
		ID           string  `json:"id"`
		Group        float32 `json:"group_width_px"`
		Frame        float32 `json:"frame_width_px"`
		Allocated    float32 `json:"allocated_width_px"`
		Actual       float32 `json:"actual_width_px"`
		Shaped       float32 `json:"shaped_right_px"`
		CompleteText bool    `json:"complete_text"`
		Truncated    bool    `json:"truncated"`
	}
	var measured []geometry
	for _, example := range []string{"rc", "nmos", "pmos", "cmos"} {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() {
			for _, action := range []struct {
				id string
				p  any
			}{
				{"example.load", workbench.ExampleLoadParams{ID: example, DiscardChanges: true}},
				{"simulation.pause", struct{}{}},
				{"simulation.restart", struct{}{}},
			} {
				if r := s.invoke(action.id, action.p); r.Error != nil {
					done <- fmt.Errorf("inspector width fixture %s: %s", example, r.Error.Message)
					return
				}
			}
			if err := smokeUI(s, "inspector-section-simulation", nil); err != nil {
				done <- err
				return
			}
			done <- smokeUI(s, "simulation-tool", &ui.ChangeEvent{SelectedID: "circuit"})
		}); err != nil {
			return err
		}
		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		var sample []geometry
		_, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
			if state.Status == "updating" || s.drawing.widget == nil {
				return false
			}
			labels := map[string]*core.Text{}
			frames := map[string]float32{}
			var group float32
			s.drawing.widget.Scene.WidgetWalkDown(func(w core.Widget, base *core.WidgetBase) bool {
				if base.Name == "inspector-group-circuit" {
					group = base.Geom.Size.Actual.Content.X
				}
				if text, ok := w.(*core.Text); ok {
					labels[base.Name] = text
				}
				if strings.HasPrefix(base.Name, railLabelBoundsPrefix) {
					frames[base.Name] = base.Geom.Size.Actual.Content.X
				}
				return true
			})
			if group <= 0 {
				return false
			}
			ids := []string{"circuit-capacity", "selection-navigation-help"}
			if example != "rc" {
				ids = append(ids, "mixed-help")
			}
			root := s.view()
			sample = nil
			for _, id := range ids {
				label := labels[id]
				frame := frames[railLabelBoundsPrefix+id]
				node, ok := findNode(root, id)
				if !ok || label == nil || frame <= 0 || label.PaintText() == nil || label.Geom.Size.Actual.Content.X <= 0 {
					return false
				}
				paint := label.PaintText()
				same := strings.Join(strings.Fields(string(paint.Source.Join())), " ") == strings.Join(strings.Fields(node.Text), " ")
				sample = append(sample, geometry{Example: example, ID: id, Group: group, Frame: frame, Allocated: label.Geom.Size.Alloc.Content.X, Actual: label.Geom.Size.Actual.Content.X, Shaped: paint.Bounds.Max.X, CompleteText: same, Truncated: paint.Truncated})
			}
			return true
		})
		if err != nil {
			return err
		}
		measured = append(measured, sample...)
	}
	passed := true
	for _, sample := range measured {
		passed = passed && sample.Frame >= sample.Group*0.9 && sample.Frame <= sample.Group+1 && sample.Actual <= sample.Frame+1 && sample.CompleteText && !sample.Truncated && sample.Shaped <= sample.Actual+1
	}
	evidence, _ := json.Marshal(map[string]any{"event": "elab.inspector-width.smoke", "passed": passed, "labels": measured})
	log.Print(string(evidence))
	if !passed {
		return fmt.Errorf("inspector text lacks its available wrapping width or clips source text")
	}
	return nil
}
