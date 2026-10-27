package workbench

import (
	"encoding/json"
	"fmt"
)

const ExtendedModelID = "elab.hobbyist-devices.v1"

type ExtendedModel struct {
	ID       string `json:"id"`
	PWM      string `json:"pwm"`
	Dynamics string `json:"opamp_dynamics"`
	Shift    string `json:"shift_register"`
	Sound    string `json:"sound"`
	Speaker  string `json:"loudspeaker,omitempty"`
	JFET     string `json:"jfet,omitempty"`
	Current  string `json:"current_devices,omitempty"`
	Passive  string `json:"passive_losses,omitempty"`
}

func ExtendedDeviceModel() ExtendedModel {
	return ExtendedModel{ID: ExtendedModelID, PWM: "Generic powered PWM clock: configurable 10..1000 Hz and 10..90 percent duty. Rise/fall each occupy 5 percent of a period. Duty includes the rising phase. Output uses complementary 50 ohm paths, 1 Gohm off bleed, 1 Mohm supply load and smooth supply enable from 2 to 3 V. No feedback compensation, current limit, temporal dead time, switching loss, noise or hardware rating. Buck examples simulate external switched power stages, not an averaged converter.",
		Dynamics: "Optional dominant-pole and slew-limited compensation for the unchanged generic op-amp output stage. Gain-bandwidth and slew rate are saved settings; a private 1 nF compensation capacitor charges from the modeled rails, preserving KCL, energy and explicit dissipation. The output target remains bounded by the present rails even when stored compensation voltage outlasts a falling supply. Existing op-amps without dynamics retain their original equations. No fitted silicon identity, noise, offset, multiple poles, protection or thermal ratings.",
		Shift:    "Generic 8-bit serial-in parallel-out register with separate positive-edge shift and output-latch clocks, active-low shift clear, active-low parallel output enable, and serial cascade output independent of output enable. State starts cleared; power loss clears both registers. Sixteen internal D memories reuse sequential-v1 edge/guard laws; internal data uses ideal logic rails with explicit 1 Gohm loading, while finite 50 ohm drive is applied only at public Q/SERIAL outputs. A logic-v1 inverter maps active-low clear. Simultaneous shift/latch/cascade clocks sample pre-edge data. Parallel buffers use 50 ohm paths when enabled and 1 Gohm to their selected rail when disabled; OE interpolates across the declared 30/70 percent input interval. Serial has its own 50 ohm buffer. No package pinout, propagation or setup/hold timing, metastability, current/thermal ratings.",
		Sound:    "Generic capacitive piezo sounder and DC-powered active buzzer. Passive piezo uses explicit capacitance and leakage; acoustic preview is a normalized illustrative terminal-voltage signal, not calibrated SPL or a mechanical/room simulation. Active buzzer uses a passive voltage-dependent DC load and an ideal acoustic oscillator; no electrical oscillator ripple. Opening or simulating a project never plays audio. Preview rendering, playback, stopping and WAV export are explicit shared commands."}
}

type PWMConfig struct {
	Frequency float64 `json:"frequency_hz"`
	Duty      float64 `json:"duty_fraction"`
}
type OpAmpDynamics struct {
	GBW  float64 `json:"gain_bandwidth_hz"`
	Slew float64 `json:"slew_v_per_s"`
}
type SoundConfig struct {
	Capacitance float64 `json:"capacitance_f"`
	Frequency   float64 `json:"frequency_hz"`
}
type DeviceParameters struct {
	JFET     *JFETConfig        `json:"jfet,omitempty"`
	Current  *CurrentConfig     `json:"current_source,omitempty"`
	Load     *LoadConfig        `json:"electronic_load,omitempty"`
	Passive  *PassiveConfig     `json:"passive,omitempty"`
	Speaker  *SpeakerConfig     `json:"speaker,omitempty"`
	Bipolar  *TransistorProfile `json:"bipolar,omitempty"`
	PWM      *PWMConfig         `json:"pwm,omitempty"`
	Dynamics *OpAmpDynamics     `json:"opamp_dynamics,omitempty"`
	Sound    *SoundConfig       `json:"sound,omitempty"`
}

func cloneDeviceParameters(p *DeviceParameters) *DeviceParameters {
	if p == nil {
		return nil
	}
	o := *p
	if p.JFET != nil {
		v := *p.JFET
		o.JFET = &v
	}
	if p.Current != nil {
		v := *p.Current
		o.Current = &v
	}
	if p.Load != nil {
		v := *p.Load
		o.Load = &v
	}
	if p.Passive != nil {
		v := *p.Passive
		o.Passive = &v
	}
	if p.Speaker != nil {
		v := *p.Speaker
		o.Speaker = &v
	}
	if p.Bipolar != nil {
		v := *p.Bipolar
		o.Bipolar = &v
	}
	if p.PWM != nil {
		v := *p.PWM
		o.PWM = &v
	}
	if p.Dynamics != nil {
		v := *p.Dynamics
		o.Dynamics = &v
	}
	if p.Sound != nil {
		v := *p.Sound
		o.Sound = &v
	}
	return &o
}
func hasDynamics(p Component) bool {
	return p.Kind == "opamp" && p.Device != nil && p.Device.Dynamics != nil
}
func isSoundKind(k string) bool { return k == "piezo" || k == "active-buzzer" }
func isExtendedKind(k string) bool {
	return IsJFETKind(k) || IsCurrentDevice(k) || k == "speaker" || k == "pwm-controller" || k == "shift-register" || isSoundKind(k)
}
func extendedModelForGraph(g Graph) *ExtendedModel {
	for _, p := range g.Components {
		if isExtendedKind(p.Kind) || hasDynamics(p) || hasPassiveLoss(p) {
			m := ExtendedDeviceModel()
			if hasJFETs(g) {
				m.JFET = jfetDescription
			}
			if hasCurrentDevices(g) {
				m.Current = currentDeviceDescription
			}
			if hasPassiveLosses(g) {
				m.Passive = passiveLossDescription
			}
			if hasSpeakers(g) {
				m.Speaker = speakerDescription
			}
			return &m
		}
	}
	return nil
}
func validateExtendedModel(g Graph, m *ExtendedModel) error {
	expected := ExtendedDeviceModel()
	if hasJFETs(g) {
		expected.JFET = jfetDescription
	}
	if hasCurrentDevices(g) {
		expected.Current = currentDeviceDescription
	}
	if hasPassiveLosses(g) {
		expected.Passive = passiveLossDescription
	}
	if hasSpeakers(g) {
		expected.Speaker = speakerDescription
	}
	if m != nil && *m != expected {
		return fmt.Errorf("unsupported extended device model")
	}
	if extendedModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("extended devices require device_model parameters")
	}
	return nil
}
func defaultDeviceParameters(p *Component) {
	if IsJFETKind(p.Kind) {
		v := DefaultJFETConfig(p.Kind)
		p.Device = &DeviceParameters{JFET: &v}
	}
	if p.Kind == "current-source" {
		v := DefaultCurrentConfig()
		p.Device = &DeviceParameters{Current: &v}
	}
	if p.Kind == "electronic-load" {
		v := DefaultLoadConfig()
		p.Device = &DeviceParameters{Load: &v}
	}
	if p.Kind == "speaker" {
		v := DefaultSpeakerConfig()
		p.Device = &DeviceParameters{Speaker: &v}
	}
	if p.Kind == "pwm-controller" {
		p.Device = &DeviceParameters{PWM: &PWMConfig{500, .45}}
	}
	if isSoundKind(p.Kind) {
		p.Device = &DeviceParameters{Sound: &SoundConfig{20e-9, 1000}}
	}
}
func validateDeviceParameters(p Component) error {
	d := p.Device
	if d == nil {
		if IsJFETKind(p.Kind) || IsCurrentDevice(p.Kind) || p.Kind == "speaker" || p.Kind == "pwm-controller" || isSoundKind(p.Kind) {
			return fmt.Errorf("device requires explicit parameters")
		}
		return nil
	}
	count := 0
	if d.JFET != nil {
		count++
	}
	if d.Current != nil {
		count++
	}
	if d.Load != nil {
		count++
	}
	if d.Passive != nil {
		count++
	}
	if d.Speaker != nil {
		count++
	}
	if d.Bipolar != nil {
		count++
	}
	if d.PWM != nil {
		count++
	}
	if d.Dynamics != nil {
		count++
	}
	if d.Sound != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("device parameters must match one component model")
	}
	if IsJFETKind(p.Kind) && d.JFET != nil {
		return validateJFET(p.Kind, *d.JFET)
	}
	if p.Kind == "current-source" && d.Current != nil {
		return validateCurrent(*d.Current)
	}
	if p.Kind == "electronic-load" && d.Load != nil {
		return validateLoad(*d.Load)
	}
	if IsPassiveStorageKind(p.Kind) && d.Passive != nil {
		return validatePassiveConfig(p.Kind, *d.Passive)
	}
	if p.Kind == "speaker" && d.Speaker != nil {
		return validateSpeaker(*d.Speaker)
	}
	if IsBipolarKind(p.Kind) && d.Bipolar != nil {
		return validateTransistorProfile(*d.Bipolar)
	}
	if p.Kind == "pwm-controller" && d.PWM != nil {
		v := *d.PWM
		if !finite(v.Frequency) || v.Frequency < 10 || v.Frequency > 1000 || !finite(v.Duty) || v.Duty < .1 || v.Duty > .9 {
			return fmt.Errorf("PWM frequency must be 10–1000 Hz and duty 0.1–0.9")
		}
		return nil
	}
	if p.Kind == "opamp" && d.Dynamics != nil {
		v := *d.Dynamics
		if !finite(v.GBW) || v.GBW < 100 || v.GBW > 1e6 || !finite(v.Slew) || v.Slew < 100 || v.Slew > 1e7 {
			return fmt.Errorf("op-amp gain-bandwidth must be 100 Hz–1 MHz and slew 100–10000000 V/s")
		}
		return nil
	}
	if isSoundKind(p.Kind) && d.Sound != nil {
		v := *d.Sound
		if !finite(v.Capacitance) || v.Capacitance < 1e-9 || v.Capacitance > 1e-6 || !finite(v.Frequency) || v.Frequency < 100 || v.Frequency > 4000 {
			return fmt.Errorf("sound capacitance must be 1 nF–1 µF and frequency 100–4000 Hz")
		}
		return nil
	}
	return fmt.Errorf("device parameters do not match component kind")
}
func validateDeviceFields(raw json.RawMessage, p Component) error {
	if p.Device == nil {
		return nil
	}
	var f map[string]json.RawMessage
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	switch {
	case p.Device.JFET != nil:
		return paramsNested(f["device_parameters"], "jfet", &JFETConfig{}, jfetFields()...)
	case p.Device.Current != nil:
		if err := paramsNested(f["device_parameters"], "current_source", &CurrentConfig{}, currentFields()...); err != nil {
			return err
		}
		return validateCurrentTimingFields(f["device_parameters"], "current_source")
	case p.Device.Load != nil:
		if err := paramsNested(f["device_parameters"], "electronic_load", &LoadConfig{}, loadFields()...); err != nil {
			return err
		}
		return validateCurrentTimingFields(f["device_parameters"], "electronic_load")
	case p.Device.Passive != nil:
		return paramsNested(f["device_parameters"], "passive", &PassiveConfig{}, passiveRequiredFields()...)
	case p.Device.Speaker != nil:
		return paramsNested(f["device_parameters"], "speaker", &SpeakerConfig{}, speakerRequiredFields()...)
	case p.Device.Bipolar != nil:
		return paramsNested(f["device_parameters"], "bipolar", &TransistorProfile{}, "version", "forward_beta", "reverse_beta", "saturation_current_a", "early_voltage_v", "base_emitter_capacitance_f", "base_collector_capacitance_f")
	case p.Device.PWM != nil:
		return paramsNested(f["device_parameters"], "pwm", &PWMConfig{}, "frequency_hz", "duty_fraction")
	case p.Device.Dynamics != nil:
		return paramsNested(f["device_parameters"], "opamp_dynamics", &OpAmpDynamics{}, "gain_bandwidth_hz", "slew_v_per_s")
	case p.Device.Sound != nil:
		return paramsNested(f["device_parameters"], "sound", &SoundConfig{}, "capacitance_f", "frequency_hz")
	}
	return nil
}

type PWMParams struct {
	ID         string    `json:"id"`
	Parameters PWMConfig `json:"parameters"`
}
type DynamicsParams struct {
	ID         string        `json:"id"`
	Enabled    bool          `json:"enabled"`
	Parameters OpAmpDynamics `json:"parameters"`
}
type SoundParams struct {
	ID         string      `json:"id"`
	Parameters SoundConfig `json:"parameters"`
}

func (s *Service) configureDeviceLocked(id, kind string, d *DeviceParameters) error {
	for i, p := range s.graph.Components {
		if p.ID == id && p.Kind == kind {
			p.Device = cloneDeviceParameters(d)
			if err := validateDeviceParameters(p); err != nil {
				return err
			}
			s.graph.Components[i] = p
			return nil
		}
	}
	return fmt.Errorf("choose an existing %s", kind)
}
func pwmWaveform(p Component) PulseWaveform {
	c := p.Device.PWM
	period := 1 / c.Frequency
	return PulseWaveform{HighVoltage: 1, LowVoltage: 0, HighSeconds: period * c.Duty, LowSeconds: period * (1 - c.Duty), RiseSeconds: period * .05, FallSeconds: period * .05}
}
func withDeviceWaveform(p Component) Component {
	if IsCurrentDevice(p.Kind) {
		return currentWaveform(p)
	}
	if p.Kind == "pwm-controller" {
		v := pwmWaveform(p)
		p.Pulse = &v
	}
	return p
}
func (n *mixedNetwork) waveformComponents() []Component {
	result := n.sources
	for _, p := range n.c.MixedGraph.Components {
		if p.Kind == "pwm-controller" || IsCurrentDevice(p.Kind) {
			if len(result) == len(n.sources) {
				result = append([]Component(nil), result...)
			}
			result = append(result, withDeviceWaveform(p))
		}
	}
	return result
}

var pwmPins = []string{"out", "vcc", "gnd"}

type PWMResult struct {
	ID          string     `json:"id"`
	Model       string     `json:"model"`
	Phase       float64    `json:"drive_fraction"`
	Voltage     float64    `json:"voltage_v"`
	PinCurrents [3]float64 `json:"pin_currents_a"`
	Power       float64    `json:"power_w"`
}

func pwmCurrents(v [3]float64, q float64) (i [3]float64, j [3][3]float64) {
	branch := func(a, b int, g float64, dg [3]float64) {
		drop := v[a] - v[b]
		i[a] += g * drop
		i[b] -= g * drop
		for k := range dg {
			d := drop * dg[k]
			if k == a {
				d += g
			}
			if k == b {
				d -= g
			}
			j[a][k] += d
			j[b][k] -= d
		}
	}
	enable, de := smoothLevel(v[1]-v[2], 2, 3)
	branch(1, 2, 1e-6, [3]float64{})
	for _, s := range []struct {
		rail   int
		weight float64
	}{{1, q}, {2, 1 - q}} {
		d := de * s.weight / 50
		branch(0, s.rail, 1e-9+enable*s.weight/50, [3]float64{0, d, -d})
	}
	return
}
func (n *mixedNetwork) pwmResult(p Component, x []float64, at float64) PWMResult {
	var v [3]float64
	for k, pin := range pwmPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	clock := withDeviceWaveform(p)
	q := sourceVoltageAt(clock, &Transient{Mode: "auto"}, at, false)
	i, _ := pwmCurrents(v, q)
	power := 0.
	for k, c := range i {
		power += (v[k] - v[2]) * c
	}
	return PWMResult{p.ID, ExtendedModelID, q, v[0] - v[2], i, power}
}
func (n *mixedNetwork) stampPWM(p Component, x []float64, at float64, add func(int, int, float64), f []float64) {
	v := [3]float64{}
	for k, pin := range pwmPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	q := sourceVoltageAt(withDeviceWaveform(p), &Transient{Mode: "auto"}, at, false)
	i, j := pwmCurrents(v, q)
	for row, pin := range pwmPins {
		node := n.pin(p, pin)
		if node >= 0 {
			f[node] += i[row]
		}
		for col, other := range pwmPins {
			add(node, n.pin(p, other), j[row][col])
		}
	}
}
func validateRecordedPWM(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, p := range r.PWM {
		ids = append(ids, p.ID)
	}
	if err := recordedObjectIDs(g, "pwm-controller", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, u := range r.PWM {
		p, _ := g.Component(u.ID)
		e := n.pwmResult(p, x, r.Time)
		if u.Model != e.Model || !recordingNear(u.Phase, e.Phase) || !recordingNear(u.Voltage, e.Voltage) || !recordingNear(u.Power, e.Power) {
			return fmt.Errorf("PWM readings disagree with clock/terminal voltages")
		}
		for k, i := range u.PinCurrents {
			if !recordingNear(i, e.PinCurrents[k]) {
				return fmt.Errorf("invalid PWM terminal current")
			}
		}
	}
	return nil
}

// Composite storage normally uses the same independent startup voltage
// constraints as authored capacitors. Profile junctions explicitly initialize at
// solved bias with zero current and therefore add no voltage constraint.
func validateExtendedConstraints(c Compiled) error {
	if extendedModelForGraph(c.MixedGraph) == nil {
		return nil
	}
	lowered, _ := lowerPowerDevices(c)
	parent := map[string]string{}
	for _, node := range lowered.Nets {
		parent[node.ID] = node.ID
	}
	var root func(string) string
	root = func(a string) string {
		if parent[a] != a {
			parent[a] = root(parent[a])
		}
		return parent[a]
	}
	for _, p := range lowered.MixedGraph.Components {
		// Profile junctions start at solved bias with zero current, not an
		// imposed startup voltage. ESR also removes the ideal voltage constraint.
		if isJunctionCap(p) || (p.Kind == "capacitor" && capacitorESR(p) > 0) {
			continue
		}
		if p.Kind != "dc-source" && p.Kind != "pulse-source" && !IsWaveformSource(p.Kind) && p.Kind != "capacitor" {
			continue
		}
		a, b := "positive", "negative"
		if p.Kind == "capacitor" {
			a, b = "1", "2"
		}
		u, v := root(lowered.TerminalNet[p.ID+"."+a]), root(lowered.TerminalNet[p.ID+"."+b])
		if u == v {
			return fmt.Errorf("source/capacitor constraints are redundant; insert resistance before the piezo or other storage")
		}
		parent[u] = v
	}
	return nil
}
