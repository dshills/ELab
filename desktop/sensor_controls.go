package main

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func lightPosition(lux float64) float64 { return 100 * math.Log1p(lux) / math.Log1p(10000) }
func lightAtPosition(position float64) float64 {
	if position <= 0 {
		return 0
	}
	if position >= 100 {
		return 10000
	}
	return math.Expm1(math.Log1p(10000) * position / 100)
}
func (s *screen) sensorControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	temperature := part.Kind == "thermistor"
	value := 0.
	title, unit := "Light level", "lux"
	if temperature {
		value = *part.TemperatureC
		title = "Temperature"
		unit = "°C"
	} else {
		value = *part.IlluminanceLux
	}
	pending := false
	if activeInputWork(state) {
		w := state.InputWork
		if temperature && w.Temperature != nil && w.Temperature.ID == part.ID {
			value = w.Temperature.Temperature
			pending = true
		}
		if !temperature && w.Light != nil && w.Light.ID == part.ID {
			value = w.Light.Illuminance
			pending = true
		}
	}
	apply := func(v float64) {
		if temperature {
			s.invoke("sensor.temperature.set", workbench.TemperatureParams{ID: part.ID, Temperature: v})
		} else {
			s.invoke("sensor.light.set", workbench.LightParams{ID: part.ID, Illuminance: v})
		}
	}
	sliderValue, min, max, step := value, -40., 125., .5
	if !temperature {
		sliderValue = lightPosition(value)
		min, max, step = 0, 100, 1
	}
	// A held native callback may return to its starting display before a
	// previous staged drag publishes. Forward the precise committed value to
	// retire that request; a plain unchanged display still stays a no-op.
	unchanged := func() {
		current := s.service.Snapshot()
		if !activeInputWork(current) {
			return
		}
		w := current.InputWork
		if temperature && w.Temperature != nil && w.Temperature.ID == part.ID && w.Temperature.Temperature != value {
			apply(value)
		}
		if !temperature && w.Light != nil && w.Light.ID == part.ID && w.Light.Illuminance != value {
			apply(value)
		}
	}
	change := func(e ui.ChangeEvent) {
		if e.Number == float32(sliderValue) {
			unchanged()
			return
		}
		v := float64(e.Number)
		if !temperature {
			v = lightAtPosition(v)
		}
		apply(v)
	}
	exact := func(e ui.ChangeEvent) {
		if e.Number != float32(value) {
			apply(float64(e.Number))
		} else {
			unchanged()
		}
	}
	exactMin, exactMax, exactStep := -40., 125., .5
	if !temperature {
		exactMin, exactMax, exactStep = 0, 10000, 1
	}
	help := "Changes settle at the current simulation time, preserving capacitor voltage and coil current. Accepted settings are saved and undoable."
	if !temperature {
		help = "Light slider uses a logarithmic scale; zero is darkness. Enter lux below for an exact level. " + help
	}
	nodes := []ui.Node{section("sensor-title", title), quiet("sensor-environment", fmt.Sprintf("%.6g %s", value, unit)), {ID: "sensor-slider", Kind: ui.Slider, Number: float32(sliderValue), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: title, Help: help}, Change: change}, {ID: "sensor-exact", Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(exactMin), Max: float32(exactMax), Step: float32(exactStep)}, Accessibility: ui.Accessibility{Name: "Exact " + title + " · " + unit}, Change: exact}, quiet("sensor-help", help)}
	if state.Mixed != nil {
		for _, r := range state.Mixed.Sensors {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				} else if pending {
					status = "Committed · environment change pending"
				}
				nodes = append(nodes, quiet("sensor-reading", fmt.Sprintf("%s · %.6g %s\nResistance %.6g Ω\n%.6g V · %.6g mA · %.6g mW", status, r.Environment, r.EnvironmentUnit, r.Resistance, r.Voltage, r.Current*1000, r.Power*1000)))
			}
		}
	}
	return nodes
}
