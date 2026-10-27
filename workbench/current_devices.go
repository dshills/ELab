package workbench

import (
	"encoding/json"
	"fmt"
)

const CurrentDeviceModelID = "elab.current-devices.v1"
const currentDeviceDescription = CurrentDeviceModelID + ": Educational Norton current source and one-quadrant CC/CP electronic load. Explicit 1 Gohm terminal bleed. Source delivers programmed current below compliance minus 1 V and smoothly reduces to zero at compliance. Load draws CC or CP above minimum voltage, linear current below it, limited by maximum current; reverse voltage draws only bleed. Finite linear pulse ramps are optional and independent of the primary voltage-source mode. Constant parameters; no control-loop bandwidth, thermal protection, ratings or manufacturer identity."

type CurrentTiming struct {
	Pulsed      bool    `json:"pulsed"`
	HighSeconds float64 `json:"high_s"`
	LowSeconds  float64 `json:"low_s"`
	RiseSeconds float64 `json:"rise_s"`
	FallSeconds float64 `json:"fall_s"`
}
type CurrentConfig struct {
	Version    string        `json:"version"`
	Current    float64       `json:"current_a"`
	Low        float64       `json:"low_current_a"`
	Compliance float64       `json:"compliance_v"`
	Timing     CurrentTiming `json:"timing"`
}
type LoadConfig struct {
	Version        string        `json:"version"`
	Mode           string        `json:"mode"`
	Level          float64       `json:"level"`
	Low            float64       `json:"low_level"`
	MinimumVoltage float64       `json:"minimum_voltage_v"`
	MaximumCurrent float64       `json:"maximum_current_a"`
	Timing         CurrentTiming `json:"timing"`
}
type CurrentParams struct {
	ID         string        `json:"id"`
	Parameters CurrentConfig `json:"parameters"`
}
type ElectronicLoadParams struct {
	ID         string     `json:"id"`
	Parameters LoadConfig `json:"parameters"`
}

func DefaultCurrentTiming() CurrentTiming { return CurrentTiming{false, .05, .05, .001, .001} }
func DefaultCurrentConfig() CurrentConfig {
	return CurrentConfig{CurrentDeviceModelID, .01, 0, 12, DefaultCurrentTiming()}
}
func DefaultLoadConfig() LoadConfig {
	return LoadConfig{CurrentDeviceModelID, "cc", .01, 0, 1, .1, DefaultCurrentTiming()}
}
func IsCurrentDevice(k string) bool { return k == "current-source" || k == "electronic-load" }
func currentFields() []string {
	return []string{"version", "current_a", "low_current_a", "compliance_v", "timing"}
}
func loadFields() []string {
	return []string{"version", "mode", "level", "low_level", "minimum_voltage_v", "maximum_current_a", "timing"}
}
func timingFields() []string { return []string{"pulsed", "high_s", "low_s", "rise_s", "fall_s"} }
func validateCurrentTiming(v CurrentTiming) error {
	return (PulseWaveform{HighVoltage: 1, LowVoltage: 0, HighSeconds: v.HighSeconds, LowSeconds: v.LowSeconds, RiseSeconds: v.RiseSeconds, FallSeconds: v.FallSeconds}).Validate()
}
func validateCurrent(v CurrentConfig) error {
	if v.Version != CurrentDeviceModelID || !finite(v.Current) || !finite(v.Low) || v.Low < 0 || v.Current < v.Low || v.Current > 1 || !finite(v.Compliance) || v.Compliance < 1 || v.Compliance > 24 {
		return fmt.Errorf("current source requires version %s, 0 ≤ low ≤ current ≤ 1 A and compliance 1–24 V", CurrentDeviceModelID)
	}
	return validateCurrentTiming(v.Timing)
}
func validateLoad(v LoadConfig) error {
	max := 1.
	if v.Mode == "cp" {
		max = 24
	}
	if v.Version != CurrentDeviceModelID || (v.Mode != "cc" && v.Mode != "cp") || !finite(v.Level) || !finite(v.Low) || v.Low < 0 || v.Level < v.Low || v.Level > max || !finite(v.MinimumVoltage) || v.MinimumVoltage < .1 || v.MinimumVoltage > 12 || !finite(v.MaximumCurrent) || v.MaximumCurrent < .001 || v.MaximumCurrent > 1 {
		return fmt.Errorf("load requires supported version/mode cc or cp, 0 ≤ low ≤ level ≤ 1 A (CC) or 24 W (CP), minimum voltage 0.1–12 V and current limit 0.001–1 A")
	}
	return validateCurrentTiming(v.Timing)
}
func currentWaveform(p Component) Component {
	if p.Device == nil || (p.Kind == "current-source" && p.Device.Current == nil) || (p.Kind == "electronic-load" && p.Device.Load == nil) {
		p.Pulse = nil
		return p
	}
	var high, low float64
	var t CurrentTiming
	if p.Kind == "current-source" {
		v := p.Device.Current
		high, low, t = v.Current, v.Low, v.Timing
	} else {
		v := p.Device.Load
		high, low, t = v.Level, v.Low, v.Timing
	}
	if t.Pulsed {
		p.Pulse = &PulseWaveform{high, low, t.HighSeconds, t.LowSeconds, t.RiseSeconds, t.FallSeconds}
	} else {
		p.Pulse = nil
		p.Value = high
	}
	return p
}
func currentLevel(p Component, at float64) float64 {
	return sourceVoltageAt(currentWaveform(p), &Transient{Mode: "auto"}, at, false)
}

// Positive current is into positive. The source's public delivered reading is its negative.
func currentDeviceLaw(p Component, v, at float64) (current, derivative float64) {
	level := currentLevel(p, at)
	if p.Kind == "current-source" {
		c := p.Device.Current
		q, dq := smoothLevel(v, c.Compliance-1, c.Compliance)
		return v*1e-9 - level*(1-q), 1e-9 + level*dq
	}
	c := p.Device.Load
	current, derivative = v*1e-9, 1e-9
	if v <= 0 {
		return
	}
	i, di := level, 0.
	if c.Mode == "cp" {
		if v < c.MinimumVoltage {
			i = level * v / (c.MinimumVoltage * c.MinimumVoltage)
			di = level / (c.MinimumVoltage * c.MinimumVoltage)
		} else {
			i = level / v
			di = -level / (v * v)
		}
	} else if v < c.MinimumVoltage {
		i = level * v / c.MinimumVoltage
		di = level / c.MinimumVoltage
	}
	if i >= c.MaximumCurrent {
		i = c.MaximumCurrent
		di = 0
	}
	return current + i, derivative + di
}
func (n *mixedNetwork) currentDeviceResult(p Component, x []float64, at float64) BranchResult {
	v := voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative"))
	i, _ := currentDeviceLaw(p, v, at)
	if p.Kind == "current-source" {
		i = -i
		if n.startupSourceScale != nil {
			i = *n.startupSourceScale*(i+v*1e-9) - v*1e-9
		}
	}
	return BranchResult{p.ID, i, v, v * i}
}
func validateRecordedCurrentDevices(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, q := range r.Loads {
		ids = append(ids, q.ID)
	}
	if err := recordedObjectIDs(g, "electronic-load", ids); err != nil {
		return err
	}
	for _, q := range append(append([]BranchResult{}, r.Loads...), r.Sources...) {
		p, ok := g.Component(q.ID)
		if !ok || !IsCurrentDevice(p.Kind) {
			continue
		}
		v := r.Nodes[c.TerminalNet[p.ID+".positive"]] - r.Nodes[c.TerminalNet[p.ID+".negative"]]
		i, _ := currentDeviceLaw(p, v, r.Time)
		if p.Kind == "current-source" {
			i = -i
		}
		if !finite(q.Current) || !finite(q.Voltage) || !finite(q.Power) || !recordingNear(q.Voltage, v) || !recordingNear(q.Current, i) || !recordingNear(q.Power, v*i) {
			return fmt.Errorf("recorded current device law mismatch")
		}
	}
	return nil
}
func currentDeviceName(p Component) string {
	if p.Device == nil || (p.Kind == "current-source" && p.Device.Current == nil) || (p.Kind == "electronic-load" && p.Device.Load == nil) {
		return "unconfigured current device"
	}
	if p.Kind == "current-source" {
		c := p.Device.Current
		return fmt.Sprintf("%.4g A · %.4g V compliance", c.Current, c.Compliance)
	}
	c := p.Device.Load
	unit := "A"
	if c.Mode == "cp" {
		unit = "W"
	}
	return fmt.Sprintf("%s %.4g %s · ≤ %.4g A", c.Mode, c.Level, unit, c.MaximumCurrent)
}
func hasCurrentDevices(g Graph) bool {
	for _, p := range g.Components {
		if IsCurrentDevice(p.Kind) {
			return true
		}
	}
	return false
}
func validateCurrentTimingFields(raw json.RawMessage, key string) error {
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil {
		return err
	}
	return paramsNested(outer[key], "timing", &CurrentTiming{}, timingFields()...)
}
