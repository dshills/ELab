package workbench

import (
	"fmt"
	"strings"
)

// MeasurementReading reports only an accepted live result, with provenance and
// honest stale/unavailable state. Pin JSON remains compatible with older clients.
type MeasurementReading struct {
	Probe              Probe    `json:"probe"`
	Value              *float64 `json:"value,omitempty"`
	Time               float64  `json:"time_s"`
	RunID              uint64   `json:"run_id"`
	ElectricalRevision uint64   `json:"electrical_revision"`
	State              string   `json:"state"`
	Reason             string   `json:"reason,omitempty"`
	Display            string   `json:"display"`
	ReferenceDisplay   string   `json:"reference_display"`
}
type readingContext struct {
	available     map[string]Probe
	sample        TraceSample
	run, revision uint64
	stale         bool
}

func (s *Service) readingContextLocked() readingContext {
	sample, run, revision, stale := s.readingFrameLocked()
	c := readingContext{available: map[string]Probe{}, sample: sample, run: run, revision: revision, stale: stale || run != s.run || revision != s.electrical || s.status == "error"}
	for _, p := range probeInventory(s.graph) {
		c.available[p.ID] = p
	}
	return c
}
func (c readingContext) reading(id string) MeasurementReading {
	r := MeasurementReading{Time: c.sample.Time, RunID: c.run, ElectricalRevision: c.revision, State: "unavailable"}
	p, exists := c.available[id]
	if !exists {
		r.Probe = Probe{ID: id, Label: id, Unit: "V"}
		if strings.HasPrefix(id, "i:") {
			r.Probe.Unit = "A"
		}
		r.Reason = "Source removed; undo restores it or choose another measurement"
	} else {
		r.Probe = p
		if v, ok := frameValue(c.sample, p); ok {
			r.Value = &v
			r.State = "live"
			if c.stale {
				r.State = "stale"
				r.Reason = "Previous accepted live result; resume or restart a valid circuit"
			}
		} else {
			r.Reason = "No accepted live result for this probe"
		}
	}
	r.ReferenceDisplay = "Source unavailable"
	if exists {
		r.ReferenceDisplay = "Voltage: " + p.Positive + " - " + p.Reference
		if p.Unit == "A" {
			r.ReferenceDisplay = "Current: " + p.Positive + " -> " + p.Reference
		}
	}
	r.Display = "Live " + id + " · unavailable"
	if r.Value != nil {
		value, unit := *r.Value, p.Unit
		if unit == "A" {
			value *= 1000
			unit = "mA"
		}
		r.Display = fmt.Sprintf("Live %s · %.5g %s", id, value, unit)
		if r.State == "stale" {
			r.Display += " · stale"
		}
	}
	return r
}
func (r MeasurementReading) pinned(pin PinSpec) PinnedReading {
	return PinnedReading{Definition: pin, Probe: r.Probe, Value: r.Value, Time: r.Time, RunID: r.RunID, ElectricalRevision: r.ElectricalRevision, State: r.State, Reason: r.Reason, Display: r.Display, ReferenceDisplay: r.ReferenceDisplay}
}
func cloneMeasurementReadings(in []MeasurementReading) []MeasurementReading {
	out := append([]MeasurementReading{}, in...)
	for i := range out {
		if out[i].Value != nil {
			v := *out[i].Value
			out[i].Value = &v
		}
	}
	return out
}

// The compiled graph paired with a DC result owns its node associations. Never
// interpret old node voltages using a new/incomplete graph after an edit.
func (s *Service) readingFrameLocked() (TraceSample, uint64, uint64, bool) {
	if s.transient != nil {
		t := s.transient
		return t.Sample, t.RunID, t.ElectricalRevision, t.Stale
	}
	if s.mixed != nil && s.mixedState != nil {
		r := s.mixed
		return mixedSample(s.mixedState.net.c, r, 0), r.RunID, r.ElectricalRevision, r.Stale
	}
	if s.dcCompiled != nil {
		c := *s.dcCompiled
		if s.npn != nil {
			r := s.npn
			return npnTraceSample(c, r, r, 0, c.Input.Value), r.RunID, r.ElectricalRevision, r.Stale
		}
		if s.result != nil {
			r := s.result
			return ledTraceSample(c, *r), r.RunID, r.ElectricalRevision, r.Stale
		}
	}
	return TraceSample{}, 0, 0, true
}
func ledTraceSample(c Compiled, r Result) TraceSample {
	currents := map[string]float64{c.Source.ID: r.Current, c.LED.ID: r.Current, c.Resistor.ID: (r.Nodes[c.TerminalNet[c.Resistor.ID+".1"]] - r.Nodes[c.TerminalNet[c.Resistor.ID+".2"]]) / c.Resistor.Value}
	return captureFrame(c, TraceSample{Input: c.Source.Value, Voltage: r.LEDVoltage, Current: r.Current}, r.Nodes, currents)
}
