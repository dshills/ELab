package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) schematicPageControls(state workbench.Snapshot) []ui.Node {
	page := state.SchematicPage
	nodes := []ui.Node{section("pdf-page-title", "Output page"), quiet("pdf-page-help", "PDF or PNG · white paper · full schematic")}
	for _, choice := range []struct {
		id, label, value string
		options          []ui.Option
	}{
		{"paper", "Paper", page.Paper, []ui.Option{{ID: "letter", Label: "Letter"}, {ID: "a4", Label: "A4"}}},
		{"orientation", "Orientation", page.Orientation, []ui.Option{{ID: "landscape", Label: "Landscape"}, {ID: "portrait", Label: "Portrait"}}},
	} {
		choice := choice
		nodes = append(nodes, ui.Node{ID: "pdf-" + choice.id, Kind: ui.Selector, Options: choice.options, SelectedID: choice.value, Accessibility: ui.Accessibility{Name: "Output " + choice.label}, Change: func(e ui.ChangeEvent) {
			next := page
			if choice.id == "paper" {
				next.Paper = e.SelectedID
			} else {
				next.Orientation = e.SelectedID
			}
			s.invoke("export.page.set", next)
		}})
	}
	nodes = append(nodes, quiet("pdf-margin-label", "Margin · mm"), ui.Node{ID: "pdf-margin", Kind: ui.NumberField, Number: float32(*page.MarginMM), Numeric: ui.Numeric{Min: 0, Max: 40, Step: 1}, Accessibility: ui.Accessibility{Name: "Output margin millimeters"}, Change: func(e ui.ChangeEvent) {
		next := page
		v := float64(e.Number)
		next.MarginMM = &v
		s.invoke("export.page.set", next)
	}})
	scale := 1.
	label := "Fit complete schematic"
	if page.Scale != nil {
		scale = *page.Scale
		label = fmt.Sprintf("Explicit scale · %.0f%%", scale*100)
	}
	nodes = append(nodes, quiet("pdf-scale-mode", label), s.action("pdf-fit", "Fit to page", "export.page.set", func() workbench.SchematicPage { p := page; p.Scale = nil; return p }()), ui.Node{ID: "pdf-scale", Kind: ui.NumberField, Number: float32(scale * 100), Numeric: ui.Numeric{Min: 10, Max: 400, Step: 5}, Accessibility: ui.Accessibility{Name: "Output explicit scale percent"}, Change: func(e ui.ChangeEvent) {
		next := page
		v := float64(e.Number) / 100
		next.Scale = &v
		s.invoke("export.page.set", next)
	}})
	for _, toggle := range []struct {
		id, label string
		checked   bool
	}{{"color", "Color output", page.Color}, {"measurements", "Live readings", page.Measurements}} {
		toggle := toggle
		name := toggle.label
		if toggle.id == "measurements" {
			name = "Include live measurement overlays"
		}
		nodes = append(nodes, ui.Node{ID: "pdf-" + toggle.id, Kind: ui.Checkbox, Text: toggle.label, Checked: toggle.checked, Accessibility: ui.Accessibility{Name: name}, Change: func(e ui.ChangeEvent) {
			next := page
			if toggle.id == "color" {
				next.Color = e.Checked
			} else {
				next.Measurements = e.Checked
			}
			s.invoke("export.page.set", next)
		}})
	}
	return nodes
}
