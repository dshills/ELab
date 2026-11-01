package workbench

import (
	"context"
	"fmt"
	"math"
)

type Timer555Result struct {
	ID             string        `json:"id"`
	Model          Timer555Model `json:"model"`
	High           bool          `json:"high"`
	Powered        bool          `json:"powered"`
	State          string        `json:"state"`
	SupplyVoltage  float64       `json:"supply_voltage_v"`
	OutputVoltage  float64       `json:"output_voltage_v"`
	TriggerLevel   float64       `json:"trigger_level_v"`
	ThresholdLevel float64       `json:"threshold_level_v"`
	PinCurrents    [8]float64    `json:"pin_currents_a"`
	Power          float64       `json:"power_w"`
}

func (n *mixedNetwork) timer555Voltages(p Component, x []float64) (v [8]float64) {
	for k, pin := range timer555Pins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}
func (n *mixedNetwork) timer555Memory(p Component) timer555Memory {
	for i, t := range n.timers {
		if t.ID == p.ID {
			return n.timerMemory[i]
		}
	}
	return timer555Memory{}
}
func (n *mixedNetwork) stampTimer555(p Component, x []float64, add func(int, int, float64), residual []float64) {
	e := timer555Currents(n.timer555Voltages(p, x), n.timer555Memory(p))
	for row, pin := range timer555Pins {
		node := n.pin(p, pin)
		if node >= 0 {
			residual[node] += e.Current[row]
		}
		for col, other := range timer555Pins {
			add(node, n.pin(p, other), e.Jacobian[row][col])
		}
	}
}
func (n *mixedNetwork) timer555Result(p Component, x []float64) Timer555Result {
	v := n.timer555Voltages(p, x)
	memory := n.timer555Memory(p)
	e := timer555Currents(v, memory)
	r := Timer555Result{ID: p.ID, Model: Timer555DeviceModel(), High: memory.High, Powered: memory.Powered, State: "low", SupplyVoltage: v[7] - v[0], OutputVoltage: v[2] - v[0], TriggerLevel: (v[4] - v[0]) / 2, ThresholdLevel: v[4] - v[0], PinCurrents: e.Current}
	if !memory.Powered {
		r.State = "unpowered"
	} else if memory.High {
		r.State = "high"
	}
	for pin, i := range e.Current {
		r.Power += (v[pin] - v[0]) * i
	}
	return r
}
func (state *mixedState) timer555Changes() ([]timer555Memory, bool, error) {
	next := append([]timer555Memory{}, state.net.timerMemory...)
	changed := false
	for k, p := range state.net.timers {
		m, err := timer555Decision(state.net.timer555Voltages(p, state.x), next[k])
		if err != nil {
			return nil, false, err
		}
		changed = changed || m != next[k]
		next[k] = m
	}
	return next, changed, nil
}

// Network plans and latch memories are immutable. Trials and canceled work
// therefore cannot leak latch changes into the accepted live circuit.
func (state *mixedState) resolveTimers555(ctx context.Context, t *Transient) (*mixedState, error) {
	cur := state
	for range 16 {
		memory, changed, err := cur.timer555Changes()
		if err != nil {
			return nil, err
		}
		if !changed {
			return cur, nil
		}
		net := *cur.net
		net.timerMemory = memory
		cur, err = net.solveAnalogContext(ctx, cur.x, cur.capV, 0, t, cur.result.Time)
		if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("555 inputs create an instantaneous switching loop; add timing storage or correct feedback")
}
func (n *mixedNetwork) inheritTimers555(previous *mixedNetwork) {
	for i, p := range n.timers {
		for j, q := range previous.timers {
			if p.ID == q.ID {
				n.timerMemory[i] = previous.timerMemory[j]
			}
		}
	}
}

func (state *mixedState) advanceTimers555(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	cur := state
	for cur.result.Time < end {
		at := cur.result.Time
		// Known waveform frequencies constrain guard sampling even without storage.
		// A short scan also detects changes in memoryless external input networks.
		scan := cur.net.maximumWaveformStep(t, at)
		if len(cur.net.caps) == 0 && len(cur.net.inductors) == 0 {
			scan = math.Min(scan, .001)
		}
		stop := math.Min(end, at+scan)
		if sameSampleTime(stop, end) {
			stop = end
		}
		trial, err := cur.advanceAnalogBudget(t, stop, budget)
		if err != nil {
			return nil, err
		}
		_, changed, err := trial.timer555Changes()
		if err != nil {
			return nil, err
		}
		stop = trial.result.Time
		// Storage integrations already inspect accepted half/full substeps.
		// Memoryless input networks also sample the middle of each scan.
		if len(cur.net.caps) == 0 && len(cur.net.inductors) == 0 {
			midTime := at + (stop-at)/2
			mid, err := cur.advanceAnalogBudget(t, midTime, budget)
			if err != nil {
				return nil, err
			}
			_, midChanged, err := mid.timer555Changes()
			if err != nil {
				return nil, err
			}
			if midChanged {
				trial, stop, changed = mid, mid.result.Time, true
			}
		}
		if !changed {
			cur = trial
			continue
		}
		// Bracket the first change without modifying memory during an analog trial.
		// Each trial retains the ordinary integration estimator and shared budget.
		// Reuse the accepted unchanged lower state instead of reintegrating the
		// entire prefix on every bisection; all state stays private until publication.
		lo, hi := at, stop
		lower := cur
		for hi-lo > math.Max(1e-9, 8*math.Abs(math.Nextafter(hi, math.Inf(1))-hi)) {
			when := lo + (hi-lo)/2
			candidate, err := lower.advanceAnalogBudget(t, when, budget)
			if err != nil {
				return nil, err
			}
			_, different, err := candidate.timer555Changes()
			if err != nil {
				return nil, err
			}
			if different {
				hi, trial = candidate.result.Time, candidate
			} else {
				lo = when
				lower = candidate
			}
		}
		appendNetworkTrace(t, mixedSample(cur.net.c, trial.result, mixedSourceVoltage(cur.net.c, trial.result)))
		right, err := trial.resolveTimers555(budget.solverContext(), t)
		if err != nil {
			return nil, err
		}
		if err := budget.checkTime(); err != nil {
			return nil, err
		}
		for k, p := range cur.net.timers {
			if right.net.timerMemory[k] != cur.net.timerMemory[k] {
				appendInputEvent(t, InputEvent{Time: right.result.Time, Mode: "timer555:" + p.ID + ":" + right.result.Timers555[k].State})
			}
		}
		appendNetworkTrace(t, mixedSample(cur.net.c, right.result, mixedSourceVoltage(cur.net.c, right.result)))
		cur = right
	}
	return cur, nil
}
