package main

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func motorLoadPosition(load float64) float64 {
	return 100 * math.Log1p(load/1e-6) / math.Log1p(.01/1e-6)
}
func motorLoadAt(position float64) float64 {
	if position <= 0 {
		return 0
	}
	if position >= 100 {
		return .01
	}
	return 1e-6 * math.Expm1(math.Log1p(.01/1e-6)*position/100)
}
func (s *screen) motorControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	config := *part.Power.Motor
	value := config.Load
	pending := false
	if activeInputWork(state) && state.InputWork.MotorLoad != nil && state.InputWork.MotorLoad.ID == part.ID {
		value = state.InputWork.MotorLoad.Load
		pending = true
	}
	setLoad := func(v float64) { s.invoke("motor.load.set", workbench.MotorLoadParams{ID: part.ID, Load: v}) }
	unchanged := func() {
		current := s.service.Snapshot()
		if activeInputWork(current) && current.InputWork.MotorLoad != nil && current.InputWork.MotorLoad.ID == part.ID && current.InputWork.MotorLoad.Load != value {
			setLoad(value)
		}
	}
	nodes := []ui.Node{section("motor-load-title", "Mechanical load"), quiet("motor-load-value", fmt.Sprintf("Drag %.6g µN·m·s/rad", value*1e6)), {ID: "motor-load-slider", Kind: ui.Slider, Number: float32(motorLoadPosition(value)), Numeric: ui.Numeric{Min: 0, Max: 100, Step: 1}, Accessibility: ui.Accessibility{Name: "Motor mechanical load", Help: "Logarithmic viscous drag control. Higher load slows the shaft. Live changes preserve winding current and shaft speed."}, Change: func(e ui.ChangeEvent) {
		if e.Number == float32(motorLoadPosition(value)) {
			unchanged()
			return
		}
		setLoad(motorLoadAt(float64(e.Number)))
	}}, {ID: "motor-load-exact", Kind: ui.NumberField, Number: float32(value * 1e6), Numeric: ui.Numeric{Min: 0, Max: 10000, Step: 1}, Accessibility: ui.Accessibility{Name: "Motor load in micro-newton metre seconds per radian"}, Change: func(e ui.ChangeEvent) {
		if e.Number == float32(value*1e6) {
			unchanged()
			return
		}
		setLoad(float64(e.Number) / 1e6)
	}}}
	if state.Mixed != nil {
		for _, r := range state.Mixed.Motors {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				} else if pending {
					status = "Committed · load pending"
				}
				nodes = append(nodes, section("motor-reading-title", status+" motor"), quiet("motor-rpm", fmt.Sprintf("%.1f RPM · %s", r.RPM, r.State)), quiet("motor-current", fmt.Sprintf("%.6g A · stall reference %.6g A", r.Current, r.StallCurrent)), quiet("motor-emf", fmt.Sprintf("Back EMF %.6g V · torque %.6g mN·m", r.BackEMF, r.Torque*1000)), quiet("motor-energy", fmt.Sprintf("Rotor %.6g mJ · winding %.6g mJ", r.RotationalEnergy*1000, r.MagneticEnergy*1000)), quiet("motor-loss", fmt.Sprintf("Winding loss %.6g W · drag %.6g W", r.WindingHeat, r.DragPower)))
			}
		}
	}
	apply := func(v workbench.MotorConfig) {
		s.invoke("motor.configure", workbench.MotorParams{ID: part.ID, Parameters: v})
	}
	nodes = append(nodes, section("motor-startup-title", "Motor startup parameters"), quiet("motor-startup-help", "Parameter edits restart a running circuit; paused readings wait for Restart. Load above changes live."))
	for _, f := range []struct {
		id, title              string
		value, min, max, scale float64
		set                    func(*workbench.MotorConfig, float64)
	}{
		{"resistance", "Winding resistance · Ω", config.Resistance, .1, 100, 1, func(m *workbench.MotorConfig, v float64) { m.Resistance = v }},
		{"inductance", "Winding inductance · mH", config.Inductance * 1000, .01, 1000, 1000, func(m *workbench.MotorConfig, v float64) { m.Inductance = v }},
		{"constant", "Back EMF / torque constant · mV·s/rad", config.Constant * 1000, 1, 1000, 1000, func(m *workbench.MotorConfig, v float64) { m.Constant = v }},
		{"inertia", "Rotor inertia · µkg·m²", config.Inertia * 1e6, .1, 1000, 1e6, func(m *workbench.MotorConfig, v float64) { m.Inertia = v }},
		{"friction", "Friction · µN·m·s/rad", config.Friction * 1e6, .1, 10000, 1e6, func(m *workbench.MotorConfig, v float64) { m.Friction = v }},
	} {
		nodes = append(nodes, quiet("motor-"+f.id+"-title", f.title), ui.Node{ID: "motor-" + f.id, Kind: ui.NumberField, Number: float32(f.value), Numeric: ui.Numeric{Min: float32(f.min), Max: float32(f.max), Step: .1}, Accessibility: ui.Accessibility{Name: f.title}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(f.value) {
				return
			}
			v := float64(e.Number)
			if e.Number == float32(f.min) {
				v = f.min
			}
			if e.Number == float32(f.max) {
				v = f.max
			}
			next := config
			f.set(&next, v/f.scale)
			apply(next)
		}})
	}
	nodes = append(nodes, ui.Node{ID: "motor-locked", Kind: ui.Checkbox, Text: "Lock the rotor for stall testing", Checked: config.Locked, Change: func(e ui.ChangeEvent) { v := config; v.Locked = e.Checked; apply(v) }})
	return nodes
}
