package workbench

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const SensorModelID = "elab.resistive-sensors.v1"

type SensorModel struct {
	ID                     string  `json:"id"`
	ReferenceTemperature   float64 `json:"reference_temperature_c"`
	Beta                   float64 `json:"beta_k"`
	ReferenceLight         float64 `json:"reference_illuminance_lux"`
	Gamma                  float64 `json:"light_exponent"`
	DarkResistance         float64 `json:"dark_resistance_ohm"`
	MinimumLightResistance float64 `json:"minimum_light_resistance_ohm"`
	MinimumTemperature     float64 `json:"minimum_temperature_c"`
	MaximumTemperature     float64 `json:"maximum_temperature_c"`
	MaximumLight           float64 `json:"maximum_illuminance_lux"`
	Limitations            string  `json:"limitations"`
}

func ResistiveSensorModel() SensorModel {
	return SensorModel{SensorModelID, 25, 3950, 10, .7, 1e6, 100, -40, 125, 10000, "Generic educational resistive sensors. NTC uses the beta equation with fixed 3950 K beta and value R25. LDR uses R10*(lux/10)^(-0.7), clipped to 100 Ω–1 MΩ; zero lux is 1 MΩ. Environment is externally prescribed, uniform and instantaneous. No self-heating, thermal/optical response delay, tolerance, spectral dependence, noise, ageing or physical part rating. Not fitted to a manufacturer device."}
}
func IsSensorKind(kind string) bool { return kind == "thermistor" || kind == "ldr" }
func sensorEnvironment(p Component) float64 {
	if p.Kind == "thermistor" && p.TemperatureC != nil {
		return *p.TemperatureC
	}
	if p.Kind == "ldr" && p.IlluminanceLux != nil {
		return *p.IlluminanceLux
	}
	return 0
}
func sensorResistance(p Component) float64 {
	m := ResistiveSensorModel()
	v := sensorEnvironment(p)
	if p.Kind == "thermistor" {
		return p.Value * math.Exp(m.Beta*(1/(v+273.15)-1/(m.ReferenceTemperature+273.15)))
	}
	if v <= 0 {
		return m.DarkResistance
	}
	return math.Max(m.MinimumLightResistance, math.Min(m.DarkResistance, p.Value*math.Pow(v/m.ReferenceLight, -m.Gamma)))
}
func validateSensorEnvironment(kind string, value float64) error {
	m := ResistiveSensorModel()
	if !finite(value) {
		return fmt.Errorf("sensor environment must be finite")
	}
	if kind == "thermistor" && (value < m.MinimumTemperature || value > m.MaximumTemperature) {
		return fmt.Errorf("temperature must be −40–125 °C")
	}
	if kind == "ldr" && (value < 0 || value > m.MaximumLight) {
		return fmt.Errorf("illuminance must be 0–10000 lux")
	}
	return nil
}
func setSensorEnvironment(p *Component, value float64) {
	v := value
	if p.Kind == "thermistor" {
		p.TemperatureC = &v
	} else {
		p.IlluminanceLux = &v
	}
}
func sensorModelForGraph(g Graph) *SensorModel {
	for _, p := range g.Components {
		if IsSensorKind(p.Kind) {
			m := ResistiveSensorModel()
			return &m
		}
	}
	return nil
}
func validateSensorModel(g Graph, m *SensorModel) error {
	if m != nil && *m != ResistiveSensorModel() {
		return fmt.Errorf("unsupported resistive sensor model")
	}
	if sensorModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("resistive sensors require sensor_model parameters")
	}
	return nil
}

type TemperatureParams struct {
	ID          string  `json:"id"`
	Temperature float64 `json:"temperature_c"`
}
type LightParams struct {
	ID          string  `json:"id"`
	Illuminance float64 `json:"illuminance_lux"`
}

func (s *Service) setSensorLocked(id, kind string, value float64) error {
	p, ok := s.graph.Component(id)
	if !ok || p.Kind != kind {
		return fmt.Errorf("choose a %s", kind)
	}
	if err := validateSensorEnvironment(kind, value); err != nil {
		return err
	}
	if sensorEnvironment(p) == value {
		if s.activeInputWorkLocked() && sensorInputID(&s.inputJob.work) == id {
			s.cancelInputWorkLocked("superseded")
		}
		return nil
	}
	g := s.graph.Clone()
	for k := range g.Components {
		if g.Components[k].ID == id {
			setSensorEnvironment(&g.Components[k], value)
		}
	}
	if s.transient != nil && !s.transient.Stale && s.mixedState != nil && IsNetworkAnalysis(s.mixedState.net.c.Analysis) {
		return s.startInputWorkLocked("", nil, g, fmt.Sprintf("sensor:%s:%s:%.17g", kind, id, value))
	}
	s.graph = g
	return nil
}
func sensorInputID(w *InputWork) string {
	if w.Temperature != nil {
		return w.Temperature.ID
	}
	if w.Light != nil {
		return w.Light.ID
	}
	return ""
}
func setSensorWork(w *InputWork, event string) {
	v := strings.Split(event, ":")
	if len(v) != 4 || v[0] != "sensor" {
		return
	}
	value, _ := strconv.ParseFloat(v[3], 64)
	if v[1] == "thermistor" {
		w.Command = "sensor.temperature.set"
		w.Temperature = &TemperatureParams{v[2], value}
	} else {
		w.Command = "sensor.light.set"
		w.Light = &LightParams{v[2], value}
	}
}
func applySensorWork(g *Graph, w *InputWork) {
	id := sensorInputID(w)
	for k := range g.Components {
		if g.Components[k].ID == id {
			if w.Temperature != nil {
				setSensorEnvironment(&g.Components[k], w.Temperature.Temperature)
			} else {
				setSensorEnvironment(&g.Components[k], w.Light.Illuminance)
			}
		}
	}
}

type SensorResult struct {
	ID              string      `json:"id"`
	Kind            string      `json:"kind"`
	Model           SensorModel `json:"model"`
	Environment     float64     `json:"environment_value"`
	EnvironmentUnit string      `json:"environment_unit"`
	Resistance      float64     `json:"resistance_ohm"`
	Voltage         float64     `json:"voltage_v"`
	Current         float64     `json:"current_a"`
	Power           float64     `json:"power_w"`
}

func (n *mixedNetwork) sensorResult(p Component, x []float64) SensorResult {
	v := voltage(x, n.pin(p, "1")) - voltage(x, n.pin(p, "2"))
	r := sensorResistance(p)
	unit := "lx"
	if p.Kind == "thermistor" {
		unit = "°C"
	}
	return SensorResult{p.ID, p.Kind, ResistiveSensorModel(), sensorEnvironment(p), unit, r, v, v / r, v * v / r}
}
func validateRecordedSensors(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, s := range r.Sensors {
		p, ok := g.Component(s.ID)
		if !ok || !IsSensorKind(p.Kind) || p.Kind != s.Kind {
			return fmt.Errorf("invalid sensor identity")
		}
		ids[s.Kind] = append(ids[s.Kind], s.ID)
	}
	for _, kind := range []string{"thermistor", "ldr"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.Sensors) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, a := range r.Sensors {
		p, _ := g.Component(a.ID)
		e := n.sensorResult(p, x)
		if a.Model != e.Model || a.EnvironmentUnit != e.EnvironmentUnit {
			return fmt.Errorf("invalid sensor model/units")
		}
		for _, v := range [][2]float64{{a.Environment, e.Environment}, {a.Resistance, e.Resistance}, {a.Voltage, e.Voltage}, {a.Current, e.Current}, {a.Power, e.Power}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("recorded sensor readings disagree with environment and terminal voltage")
			}
		}
	}
	return nil
}
