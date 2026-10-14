package workbench

import (
	"context"
	"fmt"
	"math"
	"time"
)

const RCEngineID = "elab.ideal-rc-exact.v1"
const MaxTraceSamples = 512

type TransientSettings struct {
	StepSeconds    float64 `json:"step_s"`
	HighSeconds    float64 `json:"high_s"`
	LowSeconds     float64 `json:"low_s"`
	InitialVoltage float64 `json:"initial_voltage_v"`
	PlaybackSpeed  float64 `json:"seconds_per_wall_second"`
}

func DefaultTransientSettings() TransientSettings { return TransientSettings{.001, .05, .05, 0, .02} }
func (c TransientSettings) validateRanges() error {
	if !finite(c.StepSeconds) || c.StepSeconds < 1e-6 || c.StepSeconds > 1 {
		return fmt.Errorf("sample step must be 1 µs–1 s")
	}
	if !finite(c.HighSeconds) || !finite(c.LowSeconds) || c.HighSeconds < 1e-6 || c.LowSeconds < 1e-6 || c.HighSeconds > 10 || c.LowSeconds > 10 {
		return fmt.Errorf("legacy pulse phase lengths must be 1 µs–10 s")
	}
	if !finite(c.InitialVoltage) || c.InitialVoltage < 0 || c.InitialVoltage > 12 {
		return fmt.Errorf("initial capacitor voltage must be 0–12 V")
	}
	return nil
}
func (c TransientSettings) validateNumerics() error {
	if err := c.validateRanges(); err != nil {
		return err
	}
	if c.HighSeconds < c.StepSeconds || c.LowSeconds < c.StepSeconds {
		return fmt.Errorf("legacy pulse phases must be at least one sample step")
	}
	return nil
}
func (c TransientSettings) validateForGraph(g Graph, playback bool) error {
	if err := c.validateRanges(); err != nil {
		return err
	}
	for _, p := range g.Components {
		if p.Kind == "pulse-source" {
			if err := c.validateNumerics(); err != nil {
				return err
			}
			break
		}
	}
	if playback {
		return c.validatePlayback()
	}
	return nil
}
func (c TransientSettings) validatePlayback() error {
	if !finite(c.PlaybackSpeed) || c.PlaybackSpeed < .001 || c.PlaybackSpeed > 1 || c.PlaybackSpeed/c.StepSeconds > 1000 {
		return fmt.Errorf("playback must be 0.001–1 simulated s/wall s, at most 1000 samples/wall s")
	}
	return nil
}
func (c TransientSettings) Validate() error {
	if err := c.validateNumerics(); err != nil {
		return err
	}
	return c.validatePlayback()
}

type TraceSample struct {
	frame            *probeFrame
	CapacitorVoltage float64 `json:"capacitor_voltage_v,omitempty"`
	BaseCurrent      float64 `json:"base_current_a,omitempty"`
	VBE              float64 `json:"vbe_v,omitempty"`
	State            string  `json:"state,omitempty"`
	Time             float64 `json:"time_s"`
	Input            float64 `json:"input_v"`
	Voltage          float64 `json:"voltage_v"`
	Current          float64 `json:"current_a"`
	Energy           float64 `json:"energy_j"`
}
type InputEvent struct {
	Time float64 `json:"time_s"`
	Mode string  `json:"mode"`
}
type Transient struct {
	TransistorID       string            `json:"transistor_id,omitempty"`
	LEDID              string            `json:"led_id,omitempty"`
	SupplyID           string            `json:"supply_id,omitempty"`
	BaseResistorID     string            `json:"base_resistor_id,omitempty"`
	Engine             string            `json:"engine"`
	RunID              uint64            `json:"run_id"`
	ElectricalRevision uint64            `json:"electrical_revision"`
	SourceID           string            `json:"source_id"`
	ResistorID         string            `json:"resistor_id"`
	CapacitorID        string            `json:"capacitor_id"`
	OutputNet          string            `json:"output_net"`
	TimeConstant       float64           `json:"time_constant_s"`
	Settings           TransientSettings `json:"settings"`
	Mode               string            `json:"input_mode"`
	PendingMode        string            `json:"pending_input_mode,omitempty"`
	Sample             TraceSample       `json:"sample"`
	Samples            []TraceSample     `json:"samples"`
	Events             []InputEvent      `json:"events"`
	Stale              bool              `json:"stale"`
}
type InputParams struct {
	Mode string `json:"mode"`
}
type StepParams struct {
	Duration float64 `json:"duration_s"`
}
type historyEntry struct {
	ExampleID       string
	Display         DisplayPreferences
	Pins            []PinSpec
	Actor           string
	CommandRevision uint64
	Name            string
	Graph           Graph
	Settings        TransientSettings
}

func (s *Service) historyLocked() historyEntry {
	return historyEntry{ExampleID: s.exampleID, Display: s.displayPreferences, Pins: clonePins(s.pins), Graph: s.graph.Clone(), Settings: s.transientSettings}
}
func (s *Service) restoreLocked(h historyEntry) {
	s.exampleID = h.ExampleID
	s.displayPreferences = h.Display
	s.pins = clonePins(h.Pins)
	s.graph = h.Graph.Clone()
	s.transientSettings = h.Settings
}
func RCExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
	}{
		{"V1", "pulse-source", "5 V", Point{-200, 0}}, {"R1", "resistor", "10k", Point{0, -110}}, {"C1", "capacitor", "1uF", Point{200, 0}}, {"GND", "ground", "", Point{0, 110}},
	} {
		c, _ := MakeComponent(p.id, p.kind, p.position, p.value)
		g.Components = append(g.Components, c)
	}
	g.Wires = []Wire{{"W1", "V1.positive", "R1.1", true}, {"W2", "R1.2", "C1.1", true}, {"W3", "C1.2", "GND.reference", true}, {"W4", "GND.reference", "V1.negative", true}}
	return g
}
func cloneTransient(t *Transient) *Transient {
	if t == nil {
		return nil
	}
	c := *t
	c.Samples = append([]TraceSample{}, t.Samples...)
	c.Events = append([]InputEvent{}, t.Events...)
	return &c
}
func rcInput(c Compiled, t *Transient, at float64) float64 {
	if t.Mode == "discharge" {
		return 0
	}
	if t.Mode == "charge" || c.Source.Kind == "dc-source" {
		return c.Source.Value
	}
	if pulseHigh(t.Settings, at) {
		return c.Source.Value
	}
	return 0
}

// Exact first-order response; split at pulse edges even when they lie between
// samples. This is an ideal RC solver, not a general SPICE/transient engine.
func appendTrace(t *Transient, p TraceSample) {
	t.Samples = append(t.Samples, p)
	if len(t.Samples) > MaxTraceSamples {
		copy(t.Samples, t.Samples[len(t.Samples)-MaxTraceSamples:])
		t.Samples = t.Samples[:MaxTraceSamples]
	}
}
func rcSample(c Compiled, t *Transient, at, input, v float64) TraceSample {
	p := TraceSample{Time: at, Input: input, Voltage: v, Current: (input - v) / c.Resistor.Value, Energy: .5 * c.Capacitor.Value * v * v}
	nodes := map[string]float64{c.Ground: 0, c.Supply: input, c.Output: v}
	currents := map[string]float64{c.Source.ID: p.Current, c.Resistor.ID: p.Current, c.Capacitor.ID: p.Current}
	if c.TerminalNet[c.Resistor.ID+".1"] != c.Supply {
		currents[c.Resistor.ID] = -p.Current
	}
	if c.TerminalNet[c.Capacitor.ID+".1"] != c.Output {
		currents[c.Capacitor.ID] = -p.Current
	}
	return captureFrame(c, p, nodes, currents)
}
func advanceRCSample(c Compiled, t *Transient, end float64) {
	at, v := t.Sample.Time, t.Sample.Voltage
	for at < end {
		stop := end
		input := rcInput(c, t, at)
		edgeHit := false
		if t.Mode == "auto" && c.Source.Kind == "pulse-source" {
			edge := nextPulseEdge(t.Settings, at)
			if edge > at && sourceCornerReached(edge, stop) {
				stop = math.Min(edge, stop)
				edgeHit = true
			}
		}
		v += (input - v) * (-math.Expm1(-(stop - at) / t.TimeConstant))
		at = stop
		if edgeHit {
			nextInput := rcInput(c, t, at)
			appendTrace(t, rcSample(c, t, at, input, v))
			appendTrace(t, rcSample(c, t, at, nextInput, v))
			mode := "pulse-low"
			if nextInput > 0 {
				mode = "pulse-high"
			}
			t.Events = append(t.Events, InputEvent{at, mode})
			if len(t.Events) > 64 {
				t.Events = append([]InputEvent{}, t.Events[len(t.Events)-64:]...)
			}
		}
	}
	t.Sample = rcSample(c, t, end, rcInput(c, t, end), v)
	appendTrace(t, t.Sample)
}

func (s *Service) startRCLocked(c Compiled) {
	s.result = nil
	s.npnHigh = nil
	s.npnLow = nil
	if s.pendingInputMode != "" {
		s.inputMode = s.pendingInputMode
		s.pendingInputMode = ""
	}
	if s.inputMode != "auto" && s.inputMode != "charge" && s.inputMode != "discharge" {
		s.inputMode = "auto"
	}
	t := &Transient{Engine: RCEngineID, RunID: s.run, ElectricalRevision: s.electrical, SourceID: c.Source.ID, ResistorID: c.Resistor.ID, CapacitorID: c.Capacitor.ID, OutputNet: c.Output, TimeConstant: c.Resistor.Value * c.Capacitor.Value, Settings: s.transientSettings, Mode: s.inputMode, Samples: []TraceSample{}, Events: []InputEvent{{0, "restart"}}}
	v := s.transientSettings.InitialVoltage
	if c.Capacitor.InitialVoltage != nil {
		v = *c.Capacitor.InitialVoltage
		if c.TerminalNet[c.Capacitor.ID+".1"] != c.Output {
			v = -v
		}
	}
	input := rcInput(c, t, 0)
	t.Sample = rcSample(c, t, 0, input, v)
	t.Samples = append(t.Samples, t.Sample)
	s.transient = t
	if s.paused {
		s.status = "paused"
	} else {
		s.status = "running"
		s.launchTransientLocked(c)
	}
}
func (s *Service) launchTransientLocked(c Compiled) {
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	run := s.run
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		last := time.Now()
		budget := 0.
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				elapsed := math.Min(.25, now.Sub(last).Seconds())
				last = now
				s.mu.Lock()
				if ctx.Err() != nil || s.closed || s.paused || s.run != run || s.transient == nil || s.transient.Stale {
					s.mu.Unlock()
					return
				}
				if s.activeInputWorkLocked() {
					budget = 0
					s.mu.Unlock()
					continue
				}
				budget += elapsed * s.transientSettings.PlaybackSpeed
				steps := int((budget + 1e-12) / s.transientSettings.StepSeconds)
				limit := 250
				if IsNetworkAnalysis(c.Analysis) {
					limit = 16
				}
				if steps > limit {
					steps = limit
				}
				budget = math.Min(budget, .25*s.transientSettings.PlaybackSpeed)
				budget -= float64(steps) * s.transientSettings.StepSeconds
				for i := 0; i < steps; i++ {
					end := s.transient.Sample.Time + s.transientSettings.StepSeconds
					var err error
					if IsNetworkAnalysis(c.Analysis) {
						var current bool
						current, err = s.liveNetworkSampleLocked(ctx, run, end)
						if !current {
							break // discarded work; capture current inputs on the next tick
						}
					} else {
						err = s.advanceTransientLocked(c, end)
					}
					if err != nil {
						s.status = "error"
						s.transient.Stale = true
						if s.mixed != nil {
							s.mixed.Stale = true
						}
						s.diagnostics = []Diagnostic{{simulationFailureCode(err, "mixed-integration-failed"), err.Error(), ""}}
						s.revision++
						s.signalLocked("simulation", "diagnostics")
						s.mu.Unlock()
						cancel()
						return
					}
				}
				if steps > 0 && !IsNetworkAnalysis(c.Analysis) {
					s.revision++
					s.signalLocked("simulation", "diagnostics")
				}
				s.mu.Unlock()
			}
		}
	}()
}
func (s *Service) applyRCInputLocked(c Compiled, mode string) {
	t := s.transient
	s.inputMode = mode
	s.pendingInputMode = ""
	t.Mode = mode
	t.PendingMode = ""
	t.Sample.Input = rcInput(c, t, t.Sample.Time)
	t.Sample.Current = (t.Sample.Input - t.Sample.Voltage) / c.Resistor.Value
	t.Events = append(t.Events, InputEvent{t.Sample.Time, mode})
	if len(t.Events) > 64 {
		t.Events = append([]InputEvent{}, t.Events[len(t.Events)-64:]...)
	}
	// Two samples at the same time render the input's vertical event edge while
	// the stored capacitor voltage remains continuous.
	t.Samples = append(t.Samples, t.Sample)
	if len(t.Samples) > MaxTraceSamples {
		t.Samples = append([]TraceSample{}, t.Samples[len(t.Samples)-MaxTraceSamples:]...)
	}
}
func (s *Service) transientCommandLocked(command string, raw []byte) error {
	switch command {
	case "simulation.transient.set":
		var p TransientSettings
		if err := params(raw, &p, "step_s", "high_s", "low_s", "initial_voltage_v", "seconds_per_wall_second"); err != nil {
			return err
		}
		if err := p.validateForGraph(s.graph, true); err != nil {
			return err
		}
		s.transientSettings = p
	case "simulation.input.set":
		var p InputParams
		if err := params(raw, &p, "mode"); err != nil {
			return err
		}
		c, _, d := s.graph.Compile()
		if d != nil || !IsTransientAnalysis(c.Analysis) || s.transient == nil || s.transient.Stale {
			return &actionError{"unavailable", "complete and simulate a supported transient circuit before changing its input"}
		}
		valid := p.Mode == "auto" || IsNPNTransientAnalysis(c.Analysis) && (p.Mode == "high" || p.Mode == "low") || c.Analysis == "RC transient" && (p.Mode == "charge" || p.Mode == "discharge")
		if !valid {
			return fmt.Errorf("RC input modes: auto/charge/discharge; NPN input modes: auto/high/low")
		}

		s.cancelStepLocked("superseded")
		s.cancelInputWorkLocked("superseded")
		if s.paused {
			s.pendingInputMode = p.Mode
			s.transient.PendingMode = p.Mode
		} else {
			if IsNetworkAnalysis(c.Analysis) {
				return s.startInputWorkLocked(p.Mode, nil, s.graph, "")
			}
			if err := s.applyTransientInputLocked(c, p.Mode); err != nil {
				return err
			}
		}
	case "simulation.step", "simulation.step.start":
		var p StepParams
		if err := params(raw, &p, "duration_s"); err != nil {
			return err
		}
		return s.startStepLocked(p)
	case "simulation.work.cancel":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		return s.cancelStepCommandLocked(p)
	}
	return nil
}
