package workbench

import (
	"math"
	"reflect"
	"testing"
)

func pulseRows(t *testing.T, s *Service, id string, times, values []float64) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.operations[id].record
	record.rows = nil
	for i, value := range values {
		record.rows = append(record.rows, VerificationRow{Time: times[i], Values: []float64{value}})
	}
}

func nearTiming(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-14 {
		t.Fatalf("%s: got %.17g, want %.17g", name, got, want)
	}
}

func TestRecordedPulseBoundsRetainSharedEdgeDependence(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	// Controlled rows isolate sampled-bracket arithmetic, not device behavior.
	pulseRows(t, s, p.ID, []float64{.001, .002, .004, .005, .008, .009}, []float64{-1, 1, 1, -1, -1, 1})
	p.Level, p.Direction = "0", "both"
	before := s.Snapshot()
	r := timingQuery(t, s, p)
	if !reflect.DeepEqual(before, s.Snapshot()) || r.HighPulse == nil || r.LowPulse == nil || r.Cycle == nil || r.Cycle.Duty == nil {
		t.Fatal("pulse query changed state or omitted complete cycle")
	}
	c := r.Cycle
	nearTiming(t, "high minimum", c.High.Minimum, .002)
	nearTiming(t, "high maximum", c.High.Maximum, .004)
	nearTiming(t, "low minimum", c.Low.Minimum, .003)
	nearTiming(t, "low maximum", c.Low.Maximum, .005)
	nearTiming(t, "period minimum", c.Period.Minimum, .006)
	nearTiming(t, "period maximum", c.Period.Maximum, .008)
	nearTiming(t, "duty minimum", c.Duty.Minimum, 2./7)
	nearTiming(t, "duty maximum", c.Duty.Maximum, 4./7)
	for _, x := range []float64{.001, .0015, .002} {
		for _, y := range []float64{.004, .0045, .005} {
			for _, z := range []float64{.008, .0085, .009} {
				fraction := (y - x) / (z - x)
				if fraction < c.Duty.Minimum-1e-14 || fraction > c.Duty.Maximum+1e-14 {
					t.Fatal("feasible edge combination outside fraction bounds")
				}
			}
		}
	}
	// Arrival direction and page choices do not change interval pulse summaries.
	p.Direction, p.Offset, p.Limit = "rising", 2, 1
	filtered := timingQuery(t, s, p)
	if len(filtered.Crossings) != 0 || filtered.Total != 2 || !reflect.DeepEqual(filtered.Cycle, c) || !reflect.DeepEqual(filtered.HighPulse, r.HighPulse) || !reflect.DeepEqual(filtered.LowPulse, r.LowPulse) {
		t.Fatal("arrival filter/page changes pulse summary")
	}
	call(t, s, "recording.timing.set", p, "pulse-set")
	copy := s.Snapshot().Recording.Timing
	copy.Cycle.Duty.Minimum = 999
	copy.Cycle.Rising.Before.Value = 999
	copy.HighPulse.First.After.Value = 999
	copy.LowPulse.Second.Before.Value = 999
	fresh := s.Snapshot().Recording.Timing
	if !reflect.DeepEqual(fresh, filtered) {
		t.Fatal("cycle/pulse pointers alias snapshot consumer")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "live-edit")
	if !reflect.DeepEqual(timingQuery(t, s, p), filtered) {
		t.Fatal("later live value replaced captured pulse summary")
	}
	p.Offset, p.Start = 0, .0015
	clipped := timingQuery(t, s, p)
	if clipped.HighPulse != nil || clipped.LowPulse == nil || clipped.Cycle != nil {
		t.Fatal("interval invents a starting rising bracket")
	}
	p.Start, p.End = 0, .0085
	clipped = timingQuery(t, s, p)
	if clipped.HighPulse == nil || clipped.LowPulse != nil || clipped.Cycle != nil {
		t.Fatal("interval invents an ending rising bracket")
	}
}

func TestRecordedPulsePlateausAndUnresolvedCycles(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	p.Level, p.Direction = "0", "both"
	pulseRows(t, s, p.ID, []float64{0, .001, .002, .003, .004, .005, .006, .007}, []float64{-1, 0, -1, 0, 1, 0, -1, 1})
	r := timingQuery(t, s, p)
	if r.Total != 4 || r.HighPulse.First.After.Index != 3 || r.Cycle.Rising.After.Index != 3 || r.Cycle.Falling.After.Index != 5 || r.Cycle.NextRising.After.Index != 7 {
		t.Fatal("same-direction plateau touches were bridged as an alternating pulse")
	}
	for _, times := range [][]float64{{0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, .001}} {
		pulseRows(t, s, p.ID, times, []float64{-1, 1, 1, -1, -1, 1})
		r = timingQuery(t, s, p)
		if r.Cycle == nil || r.Cycle.Duty != nil || r.Cycle.Reason == "" || r.Cycle.Period.Minimum != 0 {
			t.Fatal("unresolved positive cycle fabricates duty fraction")
		}
	}
	// Exact zero-width high pulse in a positive cycle is a valid zero fraction.
	pulseRows(t, s, p.ID, []float64{0, 0, 0, 0, .001, .001}, []float64{-1, 1, 1, -1, -1, 1})
	r = timingQuery(t, s, p)
	if r.Cycle.Duty == nil || *r.Cycle.Duty != (TimingFraction{}) || r.Cycle.Reason != "" {
		t.Fatal("valid sampled zero fraction became unavailable")
	}
}

func TestRecordedNPNPulseHasKnownHighLowAndDutyBounds(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	duration := .25
	_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}}, "pulse")
	r := timingQuery(t, s, RecordingTimingParams{ID: view.Operation.ID, ProbeID: "v:VIN.positive", Level: "2.5 V", Direction: "both", End: duration})
	if r.Cycle == nil || r.Cycle.Duty == nil {
		t.Fatal("NPN pulse lacks a complete cycle")
	}
	nearTiming(t, "period", r.Cycle.Period.Minimum, .1)
	nearTiming(t, "exact period", r.Cycle.Period.Minimum, r.Cycle.Period.Maximum)
	// The default example uses an ideal 50 ms high / 50 ms low drive.
	if view.Metadata.Settings.HighSeconds != .05 || view.Metadata.Settings.LowSeconds != .05 {
		t.Fatal("known source fixture changed")
	}
	nearTiming(t, "high", r.Cycle.High.Minimum, .05)
	nearTiming(t, "low", r.Cycle.Low.Minimum, .05)
	nearTiming(t, "fraction", r.Cycle.Duty.Minimum, .5)
	nearTiming(t, "exact fraction", r.Cycle.Duty.Minimum, r.Cycle.Duty.Maximum)
}

func TestRecordedFiniteRampPulseContainsAnalyticDuty(t *testing.T) {
	g := waveformRCGraph(t, "finite-pulse-source")
	g.Components[0].Pulse = &PulseWaveform{HighVoltage: 5, HighSeconds: .007, LowSeconds: .013, RiseSeconds: .002, FallSeconds: .003}
	s := newGraphService(SolveLED, g)
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration, step := .05, .0005
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:V1.positive"}}, "ramps")
	r := timingQuery(t, s, RecordingTimingParams{ID: op.Operation.ID, ProbeID: "v:V1.positive", Level: "2.5 V", Direction: "rising", End: duration})
	if r.Cycle == nil || r.Cycle.Duty == nil {
		t.Fatal("finite pulse cycle absent")
	}
	c := r.Cycle
	// Linear-ramp midpoint arrivals: 1 ms, 8.5 ms, 21 ms. The high
	// fraction is 37.5%, unlike the programmed high_s/period (35%).
	for _, tc := range []struct {
		name         string
		lo, hi, want float64
	}{{"high", c.High.Minimum, c.High.Maximum, .0075}, {"low", c.Low.Minimum, c.Low.Maximum, .0125}, {"period", c.Period.Minimum, c.Period.Maximum, .02}, {"fraction", c.Duty.Minimum, c.Duty.Maximum, .375}} {
		if tc.want < tc.lo-1e-12 || tc.want > tc.hi+1e-12 {
			t.Fatal("analytic finite-ramp timing outside sampled bounds", tc)
		}
	}
}
