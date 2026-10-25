package workbench

import (
	"context"
	"fmt"
	"math"
	"strconv"
)

// Pin currents enter the device in sequentialPinOrder; unused entries are zero.
type SequentialResult struct {
	ID                string           `json:"id"`
	Kind              string           `json:"kind"`
	Model             SequentialModel  `json:"model"`
	Memory            SequentialMemory `json:"memory"`
	State             string           `json:"state"`
	Clock             string           `json:"clock"`
	Reset             string           `json:"reset"`
	SupplyVoltage     float64          `json:"supply_voltage_v"`
	OutputVoltage     float64          `json:"output_voltage_v"`
	ComplementVoltage float64          `json:"complement_voltage_v"`
	BitVoltages       [4]float64       `json:"bit_voltages_v"`
	PinCurrents       [15]float64      `json:"pin_currents_a"`
	Power             float64          `json:"power_w"`
}

func (n *mixedNetwork) sequentialVoltages(p Component, x []float64) (v [15]float64) {
	for k, pin := range sequentialPinOrder {
		for _, present := range p.Terminals {
			if pin == present {
				v[k] = voltage(x, n.pin(p, pin))
				break
			}
		}
	}
	if source, ok := n.stageInputs[p.ID]; ok {
		m := n.sequentialMemory(Component{ID: source})
		v[0] = v[14]
		if m.Powered && m.Value == 1 {
			v[0] = v[13]
		}
	}
	return
}
func (n *mixedNetwork) sequentialMemory(p Component) SequentialMemory {
	for k, u := range n.sequential {
		if u.ID == p.ID {
			return n.sequenceMemory[k]
		}
	}
	return SequentialMemory{}
}
func (n *mixedNetwork) stampSequential(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := n.sequentialEvaluation(p, x, n.sequentialMemory(p))
	for row, pin := range sequentialPinOrder {
		if !containsPin(p, pin) {
			continue
		}
		node := n.pin(p, pin)
		if node >= 0 {
			f[node] += e.Current[row]
		}
		for col, other := range sequentialPinOrder {
			if containsPin(p, other) {
				add(node, n.pin(p, other), e.Jacobian[row][col])
			}
		}
	}
}
func containsPin(p Component, pin string) bool {
	for _, v := range p.Terminals {
		if v == pin {
			return true
		}
	}
	return false
}
func (n *mixedNetwork) sequentialResult(p Component, x []float64) SequentialResult {
	v := n.sequentialVoltages(p, x)
	memory := n.sequentialMemory(p)
	e := n.sequentialEvaluation(p, x, memory)
	span := v[13] - v[14]
	r := SequentialResult{ID: p.ID, Kind: p.Kind, Model: SequentialDeviceModel(), Memory: memory, State: strconv.Itoa(memory.Value), Clock: logicLevel(v[4]-v[14], span), Reset: logicLevel(v[5]-v[14], span), SupplyVoltage: span, PinCurrents: e.Current}
	if !memory.Powered {
		r.State = "unpowered"
	}
	if IsCounterKind(p.Kind) {
		for k := range r.BitVoltages {
			r.BitVoltages[k] = v[9+k] - v[14]
		}
		r.OutputVoltage = r.BitVoltages[0]
	} else {
		r.OutputVoltage = v[7] - v[14]
		r.ComplementVoltage = v[8] - v[14]
	}
	for k, i := range e.Current {
		r.Power += (v[k] - v[14]) * i
	}
	return r
}
func (s *mixedState) sequentialChanges(sample bool) ([]SequentialMemory, bool, error) {
	return s.sequentialChangesWithBaseline(sample, nil)
}
func (s *mixedState) sequentialChangesWithBaseline(sample bool, baseline []bool) ([]SequentialMemory, bool, error) {
	next := append([]SequentialMemory{}, s.net.sequenceMemory...)
	changed := false
	for k, p := range s.net.sequential {
		m, err := sequentialDecision(p.Kind, s.net.sequentialVoltages(p, s.x), next[k], sample && (len(baseline) == 0 || !baseline[k]))
		if err != nil {
			return nil, false, s.net.sequentialPublicError(p, err)
		}
		changed = changed || m != next[k]
		next[k] = m
	}
	return next, changed, nil
}
func (s *mixedState) hybridAnalogGuardChanged() (bool, error) {
	_, timer, err := s.timer555Changes()
	if err != nil {
		return false, err
	}
	_, sequence, err := s.sequentialChanges(false)
	if err != nil {
		return false, err
	}
	for k, p := range s.net.controls {
		if IsButtonKind(p.Kind) {
			continue
		}
		next, err := controlDecision(p, s.net.controlVoltages(p, s.x), s.net.controlMemory[k], s.result.Time, s.net.relayCurrent(p, s.x))
		if err != nil {
			return false, err
		}
		if next != s.net.controlMemory[k] {
			return true, nil
		}
	}
	return timer || sequence, nil
}
func (s *mixedState) hybridGuardChanged() (bool, error) {
	changed, err := s.hybridAnalogGuardChanged()
	if err != nil {
		return false, err
	}
	_, control, err := s.controlChanges()
	return changed || control, err
}

// Sample every device from the pre-update analog solution, then publish every
// new memory together. A common-clock pipeline therefore reads the old Qs.
func (s *mixedState) resolveSequential(ctx context.Context, t *Transient) (*mixedState, error) {
	cur := s
	baseline := make([]bool, len(s.net.sequential))
	for range 32 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tm, tc, err := cur.timer555Changes()
		if err != nil {
			return nil, err
		}
		// Newly powered devices establish their clock throughout the entire
		// zero-time settle, including clocks derived from a timer or another Q.
		for k, m := range cur.net.sequenceMemory {
			if !m.Powered {
				baseline[k] = true
			}
		}
		sm, sc, err := cur.sequentialChangesWithBaseline(true, baseline)
		if err != nil {
			return nil, err
		}
		cm, cc, err := cur.controlChanges()
		if err != nil {
			return nil, err
		}
		if !tc && !sc && !cc {
			return cur, nil
		}
		net := *cur.net
		net.timerMemory = tm
		net.sequenceMemory = sm
		net.controlMemory = cm
		cur, err = net.solveAnalogContext(ctx, cur.x, cur.capV, 0, t, cur.result.Time)
		if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("sequential/timer inputs create an instantaneous switching loop; correct feedback or add timing storage")
}
func (n *mixedNetwork) inheritSequential(old *mixedNetwork) {
	for i, p := range n.sequential {
		for j, q := range old.sequential {
			if p.ID == q.ID && p.Kind == q.Kind {
				n.sequenceMemory[i] = old.sequenceMemory[j]
			}
		}
	}
}
func (s *mixedState) advanceSequential(t *Transient, end float64, budget *integrationBudget) (*mixedState, error) {
	cur := s
	for cur.result.Time < end {
		at := cur.result.Time
		scan := cur.net.maximumWaveformStep(t, at)
		if len(cur.net.caps) == 0 && len(cur.net.inductors) == 0 {
			scan = math.Min(scan, .001)
		}
		stop := math.Min(math.Min(end, at+scan), cur.net.nextControlEvent(at))
		if sameSampleTime(stop, end) {
			stop = end
		}
		trial, err := cur.advanceAnalogBudget(t, stop, budget)
		if err != nil {
			return nil, err
		}
		changed, err := trial.hybridGuardChanged()
		if err != nil {
			return nil, err
		}
		stop = trial.result.Time
		if len(cur.net.caps) == 0 && len(cur.net.inductors) == 0 {
			mid, err := cur.advanceAnalogBudget(t, at+(stop-at)/2, budget)
			if err != nil {
				return nil, err
			}
			mc, err := mid.hybridGuardChanged()
			if err != nil {
				return nil, err
			}
			if mc {
				trial, stop, changed = mid, mid.result.Time, true
			}
		}
		if !changed {
			cur = trial
			continue
		}
		lo, hi := at, stop
		lower := cur
		// A scheduled contact change already has an exact timestamp. Localize
		// only if an analog guard changed before its frozen-contact left side.
		exactTimed := controlTimeReached(stop, cur.net.nextControlEvent(at))
		if exactTimed {
			analog, err := trial.hybridAnalogGuardChanged()
			if err != nil {
				return nil, err
			}
			exactTimed = !analog
		}
		for !exactTimed && hi-lo > math.Max(1e-9, 8*math.Abs(math.Nextafter(hi, math.Inf(1))-hi)) {
			when := lo + (hi-lo)/2
			from := cur
			if len(cur.net.controls) > 0 {
				from = lower
			}
			candidate, err := from.advanceAnalogBudget(t, when, budget)
			if err != nil {
				return nil, err
			}
			different, err := candidate.hybridGuardChanged()
			if err != nil {
				return nil, err
			}
			if different {
				hi, trial = candidate.result.Time, candidate
			} else {
				lo = candidate.result.Time
				lower = candidate
			}
		}
		appendNetworkTrace(t, mixedSample(cur.net.c, trial.result, mixedSourceVoltage(cur.net.c, trial.result)))
		right, err := trial.resolveSequential(context.Background(), t)
		if err != nil {
			return nil, err
		}
		if err := budget.checkTime(); err != nil {
			return nil, err
		}
		for k, p := range cur.net.sequential {
			if !privatePart(p.ID) && cur.net.sequenceMemory[k] != right.net.sequenceMemory[k] {
				appendInputEvent(t, InputEvent{Time: right.result.Time, Mode: "sequential:" + p.ID + ":" + right.net.sequentialResult(p, right.x).State})
			}
		}
		for _, p := range cur.net.authored.Components {
			if p.Kind != "shift-register" {
				continue
			}
			before, after := cur.net.shiftMemories(p), right.net.shiftMemories(p)
			if before != after {
				u := right.net.shiftResult(p, right.x)
				appendInputEvent(t, InputEvent{Time: right.result.Time, Mode: fmt.Sprintf("shift:%s:%02X:%02X", p.ID, u.Shifted, u.Latched)})
			}
		}
		for k, p := range cur.net.timers {
			if cur.net.timerMemory[k] != right.net.timerMemory[k] {
				appendInputEvent(t, InputEvent{Time: right.result.Time, Mode: "timer555:" + p.ID + ":" + right.result.Timers555[k].State})
			}
		}
		appendNetworkTrace(t, mixedSample(cur.net.c, right.result, mixedSourceVoltage(cur.net.c, right.result)))
		cur = right
	}
	return cur, nil
}

// Private shift stages exchange binary memories without unnecessary analog
// output nodes. Their declared input loading still draws real supply current.
func (n *mixedNetwork) sequentialEvaluation(p Component, x []float64, m SequentialMemory) sequentialEvaluation {
	v := n.sequentialVoltages(p, x)
	e := sequentialCurrentsForPins(p.Kind, v, m, p.Terminals)
	if source, ok := n.stageInputs[p.ID]; ok {
		input := n.sequentialMemory(Component{ID: source})
		if input.Powered && input.Value == 1 {
			g := 1 / SequentialDeviceModel().InputResistance
			i := (v[13] - v[14]) * g
			e.Current[13] += i
			e.Current[14] -= i
			e.Jacobian[13][13] += g
			e.Jacobian[14][14] += g
			e.Jacobian[13][14] -= g
			e.Jacobian[14][13] -= g
		}
	}
	return e
}
