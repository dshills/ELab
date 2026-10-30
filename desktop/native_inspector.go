package main

import (
	"fmt"

	"cogentcore.org/core/icons"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) nativeInspectorGroups(state workbench.Snapshot, groups []ui.Node) []ui.Node {
	if state.InspectorSection == "simulation" && state.SimulationTool == "live" {
		var basic, advanced []ui.Node
		basic = append(basic, inspectorGroup("native-engine", "Engine", []ui.Node{section("native-engine-title", "Engine"), quiet("native-engine-value", state.Analysis)})...)
		for _, g := range groups {
			switch g.ID {
			case "inspector-group-probes", "inspector-group-experiment", "inspector-group-live-sampling":
				advanced = append(advanced, g)
			case "inspector-group-transient":
				// The selected engine determines the solver. These are the existing
				// output-sample and playback controls, never invented solver/quality knobs.
				var kept []ui.Node
				for _, n := range g.Children {
					if n.ID == "transient-title" || n.ID == "transient-help" {
						advanced = append(advanced, n)
					} else {
						kept = append(kept, n)
					}
				}
				g.Children = kept
				basic = append(basic, g)
			default:
				basic = append(basic, g)
			}
		}
		if len(advanced) > 0 {
			d := s.detailsSection(state, "live", "Probes, controls & experiment", advanced)
			d.ID = "workspace-details-native-live"
			basic = append(basic, inspectorGroup("native-live-more", "More live tools", []ui.Node{d})...)
		}
		return basic
	}
	if state.InspectorSection == "document" {
		var actions, settings, extra []ui.Node
		for _, g := range groups {
			switch g.ID {
			case "inspector-group-document-actions":
				actions = append(actions, g)
			case "inspector-group-display", "inspector-group-output", "inspector-group-appearance":
				settings = append(settings, g)
			default:
				extra = append(extra, g)
			}
		}
		storage := []ui.Node{section("native-storage-title", "Storage"), quiet("native-storage-location", "Local schematic file")}
		if state.Document.Path != "" {
			storage = append(storage, quiet("native-storage-path", state.Document.Path))
		}
		status := "Not saved yet"
		if state.Document.Path != "" {
			status = "Saved"
		}
		if state.Document.Dirty {
			status = "Edited · Save to keep your changes"
		}
		if s.fileBusy {
			status = "Working on file…"
		}
		storage = append(storage, quiet("native-storage-status", status), ui.Node{ID: "native-save-actions", Kind: ui.Row, Children: []ui.Node{s.button("file.save", "Save"), s.button("file.save-as", "Save as…")}}, s.themeMenu(state.Theme))
		if state.Example != nil {
			storage = append(storage, quiet("native-storage-origin", "Based on "+state.Example.Info.Title))
		}
		out := inspectorGroup("native-storage", "Storage", storage)
		out = append(out, actions...)
		// Existing output/display/recovery/block controls are retained under one
		// task disclosure rather than extending Document into several long pages.
		all := append(settings, extra...)
		d := s.detailsSection(state, "circuit", "Sheet, appearance & recovery", all)
		d.ID = "workspace-details-native-document"
		return append(out, inspectorGroup("native-document-more", "Document settings", []ui.Node{d})...)
	}
	if state.InspectorSection == "component" {
		graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
		if part, ok := graph.Component(state.Selected); ok {
			for i := range groups {
				if groups[i].ID != "inspector-group-object" {
					continue
				}
				for j, n := range groups[i].Children {
					if n.ID != "inspector" {
						continue
					}
					n.Text = part.ID
					n.Style.Typography = ui.Some(ui.Typography{Size: 17, Weight: 600})
					symbol := browserSymbols()[part.Kind]
					if symbol == "" {
						symbol = icons.Memory
					}
					thumb := nativeArtwork(symbol, s.colorsForTheme(state.Theme).Ink)
					image := ui.Node{ID: "native-inspector-symbol", Kind: ui.Image, Image: thumb, Fit: ui.ImageContain, Accessibility: ui.Accessibility{Name: part.Name + " schematic symbol"}, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 36, Height: 36}), MaxSize: ui.Some(ui.Size{Width: 36, Height: 36}), Padding: ui.Some(ui.UniformInsets(4)), Background: ui.Some(paper), Border: ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken}), CornerRadius: ui.Some(float32(8))}}
					description := ui.Node{ID: "native-object-description", Kind: ui.Column, Children: []ui.Node{n, quiet("native-object-type", part.Name)}}
					groups[i].Children[j] = ui.Node{ID: "native-object-header", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(10)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{image, description}}
				}
			}
		}
	}
	// Compact repeated section headings without changing the explicit open/closed
	// pinned editor or any existing control/quantity identity.
	for i := range groups {
		if groups[i].ID == "inspector-group-pinned" {
			var rows []ui.Node
			for _, reading := range state.PinnedReadings {
				caption := quiet("native-pin-label-"+reading.Definition.ID, reading.Definition.ID)
				value := quiet("native-pin-value-"+reading.Definition.ID, reading.Display)
				value.Style.Typography = ui.Some(ui.Typography{Family: "Menlo", Size: 11})
				rows = append(rows, ui.Node{ID: "native-pin-row-" + reading.Definition.ID, Kind: ui.Row, Children: []ui.Node{caption, value}})
			}
			groups[i].Children = append(rows, groups[i].Children...)
		}
	}
	return groups
}

func nativeStepSummary(state workbench.Snapshot) string {
	if state.Transient == nil {
		return "No transient samples yet"
	}
	return fmt.Sprintf("Run %d · %.4f s · %d retained samples", state.RunID, state.Transient.Sample.Time, len(state.Transient.Samples))
}
