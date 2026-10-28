package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) photodiodeControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if p.Kind != "photodiode" || p.Device == nil || p.Device.Photo == nil {
		return nil
	}
	cfg := *p.Device.Photo
	value := cfg.Light
	if activeInputWork(state) && state.InputWork.OpticalLight != nil && state.InputWork.OpticalLight.ID == p.ID {
		value = state.InputWork.OpticalLight.Light
	}
	light := func(e ui.ChangeEvent) {
		v, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
		if err != nil {
			return
		}
		s.invokeAtCommandRevision("photodiode.light.set", workbench.OpticalLightParams{ID: p.ID, Light: v}, state.CommandRevision, state.Document.Generation)
	}
	nodes := []ui.Node{
		section("photo-title", "Optical detector"),
		quiet("photo-light-label", "Incident light · µW"),
		{ID: "photo-light", Kind: ui.Slider, Number: float32(value), Numeric: ui.Numeric{Min: 0, Max: 1000, Step: 1}, Accessibility: ui.Accessibility{Name: "Incident optical power in microwatts", Help: "Preserves time and junction voltage; supersedes pending input."},
			Change: light,
		},
		{ID: "photo-light-exact", Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: 0, Max: 1000, Step: 1}, Accessibility: ui.Accessibility{Name: "Exact incident optical power in microwatts"},
			Change: light,
		},
	}
	field := func(id, label string, value, min, max, step float64, change func(*workbench.PhotodiodeConfig, float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label, Help: "Saved detector model setting; restarts the experiment."}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			v, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			next := cfg
			change(&next, v)
			s.invokeAtCommandRevision("photodiode.configure", workbench.PhotodiodeParams{ID: p.ID, Parameters: next}, state.CommandRevision, state.Document.Generation)
		}})
	}
	field("photo-responsivity", "Responsivity · A/W", cfg.Responsivity, .01, 1, .01, func(c *workbench.PhotodiodeConfig, v float64) { c.Responsivity = v })
	field("photo-dark", "Dark saturation · pA", cfg.DarkCurrent*1e12, .001, 1e6, 1, func(c *workbench.PhotodiodeConfig, v float64) { c.DarkCurrent = decimalControlSI(v, -12) })
	field("photo-capacitance", "Junction capacitance · pF", cfg.Capacitance*1e12, 0, 1e6, 10, func(c *workbench.PhotodiodeConfig, v float64) { c.Capacitance = decimalControlSI(v, -12) })
	nodes = append(nodes, quiet("photo-help", "Current positive anode → cathode. Light drives the opposite direction. Constant uncharged junction capacitance; no spectral/lux calibration or avalanche model."))
	if state.Mixed != nil {
		for _, r := range state.Mixed.Photodiodes {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("photo-reading", fmt.Sprintf("%s · %.5g V\nPhotocurrent %.5g µA\nTerminal %.5g µA · charging %.5g µA\nStored %.5g nJ", measurementState(state.Mixed.Stale), r.Voltage, r.Photocurrent*1e6, r.Current*1e6, r.StorageCurrent*1e6, r.Energy*1e9)))
			}
		}
	}
	return nodes
}
