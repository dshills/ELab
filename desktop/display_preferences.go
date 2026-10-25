package main

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) displayControls(state workbench.Snapshot) []ui.Node {
	prefs := state.Display.Preferences
	toggle := func(id, text string, active bool, change func(*workbench.DisplayPreferences)) ui.Node {
		p := prefs
		change(&p)
		word := "Off"
		if active {
			word = "On"
		}
		return buttonStyle(s.action(id, text+" · "+word, "display.preferences.set", p), active)
	}
	nodes := []ui.Node{section("display-title", "Circuit display"), quiet("display-help", "Color shows signed voltage relative to ground. Arrows use uniquely determined wire currents. Dots and LED brightness are illustrative, not physical speed or calibrated light.")}
	nodes = append(nodes,
		toggle("display-voltage", "Voltage colors", prefs.VoltageColor, func(p *workbench.DisplayPreferences) { p.VoltageColor = !p.VoltageColor }),
		toggle("display-current", "Current arrows", prefs.CurrentCues, func(p *workbench.DisplayPreferences) {
			p.CurrentCues = !p.CurrentCues
			if !p.CurrentCues {
				p.CurrentMotion = false
			}
		}),
		toggle("display-motion", "Moving current dots", prefs.CurrentMotion, func(p *workbench.DisplayPreferences) {
			p.CurrentMotion = !p.CurrentMotion
			if p.CurrentMotion {
				p.CurrentCues = true
			}
		}),
		toggle("display-reduced-motion", "Reduced motion", prefs.ReducedMotion, func(p *workbench.DisplayPreferences) { p.ReducedMotion = !p.ReducedMotion }),
		ui.Node{ID: "display-direction", Kind: ui.Selector, Options: []ui.Option{{ID: "conventional", Label: "Conventional current"}, {ID: "electron", Label: "Electron-flow illustration"}}, SelectedID: prefs.Direction, Accessibility: ui.Accessibility{Name: "Illustrative current direction"}, Change: func(e ui.ChangeEvent) { p := prefs; p.Direction = e.SelectedID; s.invoke("display.preferences.set", p) }},
		quiet("display-reference", "Signed amperes always use conventional current. Electron mode reverses visual cues only."),
		quiet("display-voltage-range-label", "Color range · ±V"),
		ui.Node{ID: "display-voltage-range", Kind: ui.NumberField, Number: float32(prefs.VoltageRange), Numeric: ui.Numeric{Min: .01, Max: 30, Step: .5}, Accessibility: ui.Accessibility{Name: "Voltage color range in volts"}, Change: func(e ui.ChangeEvent) {
			p := prefs
			p.VoltageRange = float64(e.Number)
			if e.Number == float32(.01) {
				p.VoltageRange = .01
			}
			s.invoke("display.preferences.set", p)
		}})
	legend := "Voltage color off"
	if prefs.VoltageColor {
		legend = fmt.Sprintf("Blue −%.4g V · neutral 0 V · orange +%.4g V\nGray means unavailable/stale; larger magnitudes saturate.", prefs.VoltageRange, prefs.VoltageRange)
	}
	if prefs.ReducedMotion {
		legend += "\nReduced motion: static arrows; no moving dots."
	} else if s.reduceMotion.Load() {
		legend += "\nmacOS Reduce Motion: static arrows; no moving dots."
	} else if prefs.CurrentMotion && !state.Display.Animate {
		legend += "\nMotion stopped: paused, unavailable, zero or ambiguous current."
	}
	return append(nodes, quiet("display-legend", legend))
}
func displayColor(hex string) color.NRGBA {
	if len(hex) != 7 {
		return color.NRGBA{R: 56, G: 125, B: 147, A: 255}
	}
	n, err := strconv.ParseUint(hex[1:], 16, 24)
	if err != nil {
		return color.NRGBA{R: 56, G: 125, B: 147, A: 255}
	}
	return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}
}
