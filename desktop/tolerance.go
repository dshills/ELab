package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) toleranceInspector(state workbench.Snapshot) []ui.Node {
	if state.Tolerance == nil {
		return nil
	}
	v := state.Tolerance
	p := v.Settings
	busy := v.Status == "running" || v.Status == "canceling"
	nodes := []ui.Node{section("tolerance-title", "Component tolerances"), quiet("tolerance-help", "Independent seeded trials · uniform ranges\nAC gain exposes R/C and profile-beta variation.\nDC opens capacitors; generic transistor gain stays fixed.")}
	apply := func(next workbench.ToleranceSettings) {
		s.invokeAtCommandRevision("tolerance.settings.set", next, state.CommandRevision, state.Document.Generation)
	}
	nodes = append(nodes, ui.Node{ID: "tolerance-mode", Kind: ui.Selector, SelectedID: p.Mode, Disabled: busy, Options: []ui.Option{{ID: "ac-gain", Label: "AC gain at frequency"}, {ID: "dc", Label: "DC operating point"}}, Accessibility: ui.Accessibility{Name: "Tolerance measurement mode"}, Change: func(e ui.ChangeEvent) {
		if e.SelectedID == p.Mode {
			return
		}
		n := p
		n.Mode = e.SelectedID
		if n.Mode == "ac-gain" {
			if n.Frequency < .01 {
				n.Frequency = 1000
			}
			if !strings.HasPrefix(n.ProbeID, "v:") {
				for _, q := range state.Scope.Available {
					if strings.HasPrefix(q.ID, "v:") {
						n.ProbeID = q.ID
						break
					}
				}
			}
		}
		apply(n)
	}})
	sources := []ui.Option{}
	for _, c := range state.Components {
		if workbench.IsVoltageSource(c.Kind) && c.Kind != "battery" {
			sources = append(sources, ui.Option{ID: c.ID, Label: c.ID})
		}
	}
	sourceSelected, sourceValid := p.SourceID, false
	for _, option := range sources {
		if option.ID == sourceSelected {
			sourceValid = true
		}
	}
	if !sourceValid {
		sourceSelected = ""
	}
	if p.Mode == "ac-gain" {
		nodes = append(nodes, quiet("tolerance-source-label", "AC excitation"), ui.Node{ID: "tolerance-source", Kind: ui.Selector, SelectedID: sourceSelected, Disabled: busy, Options: sources, Accessibility: ui.Accessibility{Name: "Tolerance AC source"}, Change: func(e ui.ChangeEvent) { n := p; n.SourceID = e.SelectedID; apply(n) }})
	}
	probes := []ui.Option{}
	for _, q := range state.Scope.Available {
		if p.Mode == "ac-gain" && !strings.HasPrefix(q.ID, "v:") {
			continue
		}
		probes = append(probes, ui.Option{ID: q.ID, Label: q.Label})
	}
	probeSelected, probeValid := p.ProbeID, false
	for _, option := range probes {
		if option.ID == probeSelected {
			probeValid = true
		}
	}
	if !probeValid {
		probeSelected = ""
	}
	nodes = append(nodes, quiet("tolerance-probe-label", "Measurement"), ui.Node{ID: "tolerance-probe", Kind: ui.Selector, SelectedID: probeSelected, Disabled: busy, Options: probes, Accessibility: ui.Accessibility{Name: "Tolerance measurement probe"}, Change: func(e ui.ChangeEvent) { n := p; n.ProbeID = e.SelectedID; apply(n) }})
	field := func(id, label string, value, min, max, step float64, update func(float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Disabled: busy, Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) {
			if e.Number != float32(value) {
				decimal, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
				if err == nil {
					update(decimal)
				}
			}
		}})
	}
	if p.Mode == "ac-gain" {
		field("tolerance-frequency", "Frequency · Hz", p.Frequency, .01, 100000, 10, func(f float64) { n := p; n.Frequency = f; apply(n) })
	}
	nodes = append(nodes, quiet("tolerance-trials-label", "Trials"), ui.Node{ID: "tolerance-trials", Kind: ui.TextField, Text: strconv.Itoa(p.Trials), Disabled: busy, Accessibility: ui.Accessibility{Name: "Tolerance trial count"}, Change: func(e ui.ChangeEvent) {
		value, err := strconv.Atoi(e.Text)
		if err != nil {
			s.invokeAtCommandRevision("tolerance.settings.set", textFieldParams(p, "trials", e.Text), state.CommandRevision, state.Document.Generation)
			return
		}
		if value != p.Trials {
			n := p
			n.Trials = value
			apply(n)
		}
	}})
	nodes = append(nodes, quiet("tolerance-seed-label", "Seed"), ui.Node{ID: "tolerance-seed", Kind: ui.TextField, Text: strconv.FormatUint(p.Seed, 10), Disabled: busy, Accessibility: ui.Accessibility{Name: "Tolerance deterministic seed"}, Change: func(e ui.ChangeEvent) {
		value, err := strconv.ParseUint(e.Text, 10, 64)
		if err != nil {
			s.invokeAtCommandRevision("tolerance.settings.set", textFieldParams(p, "seed", e.Text), state.CommandRevision, state.Document.Generation)
			return
		}
		if value != p.Seed {
			n := p
			n.Seed = value
			apply(n)
		}
	}})
	field("tolerance-resistance", "Resistance · ± %", p.Resistance*100, 0, 50, 1, func(f float64) { n := p; n.Resistance = f / 100; apply(n) })
	field("tolerance-capacitance", "Capacitance · ± %", p.Capacitance*100, 0, 50, 1, func(f float64) { n := p; n.Capacitance = f / 100; apply(n) })
	field("tolerance-beta", "Profile forward gain · ± %", p.Beta*100, 0, 50, 1, func(f float64) { n := p; n.Beta = f / 100; apply(n) })
	run := s.action("tolerance-run", "Run tolerance analysis", "tolerance.run", struct{}{})
	run.Disabled = busy || !probeValid || p.Mode == "ac-gain" && !sourceValid
	nodes = append(nodes, run)
	if busy {
		nodes = append(nodes, quiet("tolerance-progress", fmt.Sprintf("%s · %d / %d trials", v.Status, v.Completed, p.Trials)), s.action("tolerance-cancel", "Cancel tolerance run", "tolerance.cancel", workbench.IDParams{ID: v.ID}))
	}
	if v.Failure != "" {
		nodes = append(nodes, quiet("tolerance-failure", v.Status+": "+v.Failure))
	}
	if r := v.Result; r != nil {
		status := "Captured circuit"
		if !v.SourceCurrent {
			status = "Captured earlier circuit"
		}
		x := r.Statistics
		nodes = append(nodes, quiet("tolerance-summary", fmt.Sprintf("%s · seed %d\n%d successful · %d failed", status, r.Settings.Seed, x.Successful, x.Failed)))
		if x.Successful > 0 {
			nodes = append(nodes, quiet("tolerance-statistics", fmt.Sprintf("%s\nRange %.6g … %.6g %s\nMean %.6g · σ %.6g", r.Probe.Label, x.Minimum, x.Maximum, r.Probe.Unit, x.Mean, x.StandardDeviation)))
		}
		for _, t := range r.Trials {
			if t.Failure != "" {
				nodes = append(nodes, quiet(fmt.Sprintf("tolerance-failed-%d", t.Index), fmt.Sprintf("Trial %d: %s", t.Index, t.Failure)))
			}
		}
		for _, f := range []string{"json", "csv"} {
			format, id := f, r.ID
			nodes = append(nodes, buttonStyle(ui.Node{ID: "tolerance-export-" + f, Kind: ui.Button, Text: "Export trials " + strings.ToUpper(f), Accessibility: ui.Accessibility{Name: "Export tolerance " + f}, Activate: func() {
				s.beginFile("tolerance.export", true, workbench.VerificationExportParams{ID: id, Format: format}, nil)
			}}, false))
		}
	}
	return s.focusSimulationForm(state, "tolerances", nodes)
}
