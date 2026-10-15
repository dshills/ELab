package workbench

import "math"

type TimingDuration struct {
	Minimum float64 `json:"minimum_s"`
	Maximum float64 `json:"maximum_s"`
}

type TimingFraction struct {
	Minimum float64 `json:"minimum"`
	Maximum float64 `json:"maximum"`
}

// TimingCycle identifies three consecutive observed arrivals, not a guarantee
// of periodicity or of unobserved waveform behavior between retained rows.
type TimingCycle struct {
	Rising     ThresholdCrossing `json:"rising"`
	Falling    ThresholdCrossing `json:"falling"`
	NextRising ThresholdCrossing `json:"next_rising"`
	High       TimingDuration    `json:"high"`
	Low        TimingDuration    `json:"low"`
	Period     TimingDuration    `json:"period"`
	Duty       *TimingFraction   `json:"high_fraction,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

type timingPulseCollector struct {
	previous, earlier ThresholdCrossing
	count             int
}

func timingDuration(first, second ThresholdCrossing) TimingDuration {
	return TimingDuration{Minimum: math.Max(0, second.Before.Time-first.After.Time), Maximum: second.After.Time - first.Before.Time}
}

func (p *timingPulseCollector) observe(out *RecordingTiming, crossing ThresholdCrossing) {
	if p.count > 0 && crossing.Direction != p.previous.Direction {
		width := timingDuration(p.previous, crossing)
		pulse := &TimingSpacing{First: p.previous, Second: crossing, Minimum: width.Minimum, Maximum: width.Maximum}
		if p.previous.Direction == "rising" && out.HighPulse == nil {
			out.HighPulse = pulse
		} else if p.previous.Direction == "falling" && out.LowPulse == nil {
			out.LowPulse = pulse
		}
	}
	if out.Cycle == nil && p.count >= 2 && p.earlier.Direction == "rising" && p.previous.Direction == "falling" && crossing.Direction == "rising" {
		rise, fall, next := p.earlier, p.previous, crossing
		cycle := &TimingCycle{Rising: rise, Falling: fall, NextRising: next, High: timingDuration(rise, fall), Low: timingDuration(fall, next), Period: timingDuration(rise, next)}
		if cycle.Period.Minimum > 0 {
			// The two durations share the first rising time. For x<=y<=z,
			// (y-x)/(z-x) decreases with x and z, and increases with y.
			// Preserve that dependence instead of dividing unrelated extrema.
			cycle.Duty = &TimingFraction{
				Minimum: math.Max(0, (fall.Before.Time-rise.After.Time)/(next.After.Time-rise.After.Time)),
				Maximum: math.Min(1, (fall.After.Time-rise.Before.Time)/(next.Before.Time-rise.Before.Time)),
			}
		} else {
			cycle.Reason = "Saved brackets do not resolve a strictly positive cycle duration"
		}
		out.Cycle = cycle
	}
	p.earlier, p.previous = p.previous, crossing
	p.count++
}

func cloneTimingSpacing(in *TimingSpacing) *TimingSpacing {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneTimingCycle(in *TimingCycle) *TimingCycle {
	if in == nil {
		return nil
	}
	out := *in
	if in.Duty != nil {
		v := *in.Duty
		out.Duty = &v
	}
	return &out
}
