package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) stepResponseControls(state workbench.Snapshot) []ui.Node {
	r := state.Recording
	if r == nil {
		return nil
	}
	nodes := []ui.Node{section("step-response-title", "Step response")}
	if r.Duration <= 0 || r.LastSampleTime <= 0 || r.Total < 2 {
		return append(nodes, quiet("step-response-unavailable", "No positive saved step-response interval is available."))
	}
	options := []ui.Option{{ID: "__none", Label: "Step response off"}}
	for _, p := range r.Scope.Available {
		options = append(options, ui.Option{ID: p.ID, Label: p.Label + " · " + p.Unit})
	}
	selected := "__none"
	if r.StepResponse != nil {
		selected = r.StepResponse.Settings.ProbeID
	}
	nodes = append(nodes, ui.Node{ID: "step-response-probe", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Captured step-response probe"}, Change: func(e ui.ChangeEvent) {
		p := workbench.StepResponseParams{ID: r.ID}
		if e.SelectedID != "__none" {
			p.ProbeID = e.SelectedID
			p.End = math.Min(r.Duration, r.LastSampleTime)
			p.Initial, p.Final, p.Tolerance, p.Hold = 0, 1, .02, math.Min(.002, p.End/10)
		}
		s.invokeAtCommandRevision("recording.step-response.set", p, state.CommandRevision, state.Document.Generation)
	}})
	if r.StepResponse == nil {
		return nodes
	}
	result := r.StepResponse
	cfg := workbench.StepResponseParams(result.Settings)
	nodes = append(nodes, quiet("step-response-help", "Enter the known initial and final references.\nFinal reference defaults to 1; it is not inferred from the last sample.\nObserved linear segments only; hidden peaks/future behavior are unknown."))
	for _, f := range []struct {
		id, label             string
		value, min, max, step float64
		set                   func(*workbench.StepResponseParams, float64)
	}{
		{"initial", "Initial reference · " + result.Probe.Unit, cfg.Initial, -1e6, 1e6, .1, func(p *workbench.StepResponseParams, v float64) { p.Initial = v }},
		{"final", "Final reference · " + result.Probe.Unit, cfg.Final, -1e6, 1e6, .1, func(p *workbench.StepResponseParams, v float64) { p.Final = v }},
		{"start", "Step begins · ms", cfg.Start * 1000, 0, r.Duration * 1000, .1, func(p *workbench.StepResponseParams, v float64) { p.Start = decimalControlSI(v, -3) }},
		{"end", "Observe through · ms", cfg.End * 1000, 0, r.Duration * 1000, .1, func(p *workbench.StepResponseParams, v float64) { p.End = decimalControlSI(v, -3) }},
		{"tolerance", "Final band · % of step", cfg.Tolerance * 100, .01, 50, .1, func(p *workbench.StepResponseParams, v float64) { p.Tolerance = decimalControlSI(v, -2) }},
		{"hold", "Required continuous hold · ms", cfg.Hold * 1000, .000001, r.Duration * 1000, .1, func(p *workbench.StepResponseParams, v float64) { p.Hold = decimalControlSI(v, -3) }},
	} {
		nodes = append(nodes, quiet("step-response-"+f.id+"-label", f.label), ui.Node{ID: "step-response-" + f.id, Kind: ui.NumberField, Number: float32(f.value), Numeric: ui.Numeric{Min: float32(f.min), Max: float32(f.max), Step: float32(f.step)}, Accessibility: ui.Accessibility{Name: f.label}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(f.value) {
				return
			}
			value, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			next := cfg
			f.set(&next, value)
			s.invokeAtCommandRevision("recording.step-response.set", next, state.CommandRevision, state.Document.Generation)
		}})
	}
	nodes = append(nodes, quiet("step-response-peak", fmt.Sprintf("Overshoot %.5g %%\nPeak %.5g %s at %.6g ms", result.Overshoot, result.PeakValue, result.Probe.Unit, result.PeakTime*1000)))
	if result.Settling == nil {
		nodes = append(nodes, quiet("step-response-settling", result.Reason))
	} else {
		nodes = append(nodes, quiet("step-response-settling", fmt.Sprintf("Observed settling delay %.6g ms\nRemains in band through %.6g ms\nObserved hold %.6g ms", *result.Settling*1000, cfg.End*1000, result.Held*1000)))
	}
	return nodes
}
