package workbench

// TimingTransitions measures observed travel between two levels on one captured
// probe. Original brackets remain authoritative; no time is interpolated.
type TimingTransitions struct {
	Lower float64        `json:"lower_level_si"`
	Upper float64        `json:"upper_level_si"`
	Rise  *TimingSpacing `json:"first_rise,omitempty"`
	Fall  *TimingSpacing `json:"first_fall,omitempty"`
}

type timingTransitionCollector struct {
	rise, fall *ThresholdCrossing
}

func sampledArrival(index int, before, after VerificationRow, channel int, level float64) *ThresholdCrossing {
	a, b := before.Values[channel], after.Values[channel]
	direction := ""
	if a < level && b >= level {
		direction = "rising"
	} else if a > level && b <= level {
		direction = "falling"
	}
	if direction == "" {
		return nil
	}
	return &ThresholdCrossing{Direction: direction, Before: TimingSample{Index: index - 1, Time: before.Time, Value: a}, After: TimingSample{Index: index, Time: after.Time, Value: b}}
}

func (c *timingTransitionCollector) observe(out *TimingTransitions, low, high *ThresholdCrossing) {
	// A return arrival at the starting threshold aborts the unfinished traversal.
	// Process resets before starts/completions so one row can span both levels.
	if low != nil && low.Direction == "falling" {
		c.rise = nil
	}
	if high != nil && high.Direction == "rising" {
		c.fall = nil
	}
	if low != nil && low.Direction == "rising" {
		copy := *low
		c.rise = &copy
	}
	if high != nil && high.Direction == "falling" {
		copy := *high
		c.fall = &copy
	}
	spacing := func(a, b ThresholdCrossing) *TimingSpacing {
		duration := timingDuration(a, b)
		return &TimingSpacing{First: a, Second: b, Minimum: duration.Minimum, Maximum: duration.Maximum}
	}
	if high != nil && high.Direction == "rising" && c.rise != nil {
		if out.Rise == nil {
			out.Rise = spacing(*c.rise, *high)
		}
		c.rise = nil
	}
	if low != nil && low.Direction == "falling" && c.fall != nil {
		if out.Fall == nil {
			out.Fall = spacing(*c.fall, *low)
		}
		c.fall = nil
	}
}

func cloneTimingTransitions(in *TimingTransitions) *TimingTransitions {
	if in == nil {
		return nil
	}
	out := *in
	out.Rise = cloneTimingSpacing(in.Rise)
	out.Fall = cloneTimingSpacing(in.Fall)
	return &out
}
