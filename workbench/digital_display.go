package workbench

import "fmt"

const DisplayModelID = "elab.generic-seven-segment.v1"

type DisplayModel struct {
	ID             string     `json:"id"`
	LED            LEDModel   `json:"segment_led"`
	Logic          LogicModel `json:"decoder_logic"`
	VisibleCurrent float64    `json:"visible_current_a"`
	Limitations    string     `json:"limitations"`
}

func DigitalDisplayModel() DisplayModel {
	return DisplayModel{DisplayModelID, Model(), LogicDeviceModel(), .0001, "Seven independent generic red LED segments with a common cathode; each needs an external current-limiting resistor. Powered combinational BCD decoder, active-high outputs a–g, codes 10–15 blank. No latch, lamp test, decimal point, propagation delay, package pinout, hardware current limit or thermal/optical rating. Intermediate BCD inputs interpolate analog output conductance, not a valid digit. Output paths and supply load use the declared generic logic model."}
}
func IsDisplayKind(kind string) bool { return kind == "bcd-decoder" || kind == "seven-segment" }

var segmentNames = [...]string{"a", "b", "c", "d", "e", "f", "g"}
var digitMasks = [...]uint8{0x3f, 0x06, 0x5b, 0x4f, 0x66, 0x6d, 0x7d, 0x07, 0x7f, 0x6f}

func displayPins(kind string) []string {
	if kind == "bcd-decoder" {
		return []string{"b0", "b1", "b2", "b3", "a", "b", "c", "d", "e", "f", "g", "vcc", "gnd"}
	}
	return []string{"a", "b", "c", "d", "e", "f", "g", "common"}
}
func displayPin(p Component, index int) string {
	if p.Kind == "bcd-decoder" {
		return displayPins(p.Kind)[index]
	}
	if index >= 4 && index <= 10 {
		return segmentNames[index-4]
	}
	if index == 12 {
		return "common"
	}
	return ""
}
func displayModelForGraph(g Graph) *DisplayModel {
	for _, p := range g.Components {
		if IsDisplayKind(p.Kind) {
			m := DigitalDisplayModel()
			return &m
		}
	}
	return nil
}
func validateDisplayModel(g Graph, m *DisplayModel) error {
	if m != nil && *m != DigitalDisplayModel() {
		return fmt.Errorf("unsupported digital display model parameters/version")
	}
	if displayModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("display schematics require supported display_model parameters")
	}
	return nil
}

type displayEvaluation struct {
	Current  [13]float64
	Jacobian [13][13]float64
}

func (e *displayEvaluation) branch(a, b int, g float64, dg [13]float64, v [13]float64) {
	drop := v[a] - v[b]
	i := drop * g
	e.Current[a] += i
	e.Current[b] -= i
	for k, d := range dg {
		d *= drop
		if k == a {
			d += g
		}
		if k == b {
			d -= g
		}
		e.Jacobian[a][k] += d
		e.Jacobian[b][k] -= d
	}
}

// The four-input multilinear truth polynomial stays smooth between digital
// levels, preserving KCL and an analytic Newton stamp during bit transitions.
func decoderCurrents(v [13]float64) displayEvaluation {
	e := displayEvaluation{}
	m := LogicDeviceModel()
	for k := range 4 {
		e.branch(k, 12, 1/m.InputResistance, [13]float64{}, v)
	}
	e.branch(11, 12, 1/m.SupplyResistance, [13]float64{}, v)
	span := v[11] - v[12]
	for k := 4; k <= 10; k++ {
		e.branch(k, 12, 1/m.OffResistance, [13]float64{}, v)
	}
	if span <= 0 {
		return e
	}
	var q, dv, ds [4]float64
	for k := range q {
		q[k], dv[k], ds[k] = logicInput(v[k]-v[12], span)
	}
	soft := m.SupplySoftVoltage * m.SupplySoftVoltage
	den := span*span + soft
	enable := span * span / den
	de := 2 * span * soft / (den * den)
	for segment := range 7 {
		weight := 0.
		var derivative [4]float64
		for code, mask := range digitMasks {
			if mask&(1<<segment) == 0 {
				continue
			}
			term := 1.
			for k := range 4 {
				if code&(1<<k) != 0 {
					term *= q[k]
				} else {
					term *= 1 - q[k]
				}
			}
			weight += term
			for k := range 4 {
				d := 1.
				if code&(1<<k) == 0 {
					d = -1
				}
				for j := range 4 {
					if j == k {
						continue
					}
					if code&(1<<j) != 0 {
						d *= q[j]
					} else {
						d *= 1 - q[j]
					}
				}
				derivative[k] += d
			}
		}
		var dq, up, down [13]float64
		for k := range 4 {
			dq[k] = derivative[k] * dv[k]
			dq[11] += derivative[k] * ds[k]
			dq[12] -= derivative[k] * (dv[k] + ds[k])
		}
		for k := range dq {
			d := 0.
			if k == 11 {
				d = de
			}
			if k == 12 {
				d = -de
			}
			up[k] = ((span/den)*dq[k] + weight*d) / m.OutputResistance
			down[k] = (-(span/den)*dq[k] + (1-weight)*d) / m.OutputResistance
		}
		e.branch(4+segment, 11, enable*weight/m.OutputResistance, up, v)
		e.branch(4+segment, 12, enable*(1-weight)/m.OutputResistance, down, v)
	}
	return e
}
func (n *mixedNetwork) displayVoltages(p Component, x []float64) (v [13]float64) {
	for k := range v {
		if pin := displayPin(p, k); pin != "" {
			v[k] = voltage(x, n.pin(p, pin))
		}
	}
	return
}
func (n *mixedNetwork) displayCurrents(p Component, v [13]float64) displayEvaluation {
	if p.Kind == "bcd-decoder" {
		return decoderCurrents(v)
	}
	e := displayEvaluation{}
	for k := 4; k <= 10; k++ {
		i, g := n.ledCurrent(v[k] - v[12])
		e.Current[k] += i
		e.Current[12] -= i
		e.Jacobian[k][k] += g
		e.Jacobian[12][12] += g
		e.Jacobian[k][12] -= g
		e.Jacobian[12][k] -= g
	}
	return e
}
func (n *mixedNetwork) stampDisplay(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := n.displayCurrents(p, n.displayVoltages(p, x))
	for k := range e.Current {
		pin := displayPin(p, k)
		if pin == "" {
			continue
		}
		row := n.pin(p, pin)
		if row >= 0 {
			f[row] += e.Current[k]
		}
		for j := range e.Current {
			if col := displayPin(p, j); col != "" {
				add(row, n.pin(p, col), e.Jacobian[k][j])
			}
		}
	}
}

type DisplayResult struct {
	ID              string       `json:"id"`
	Kind            string       `json:"kind"`
	Model           DisplayModel `json:"model"`
	Digit           int          `json:"digit"`
	State           string       `json:"state"`
	SegmentVoltages [7]float64   `json:"segment_voltages_v"`
	SegmentCurrents [7]float64   `json:"segment_currents_a"`
	Lit             [7]bool      `json:"lit"`
	PinCurrents     [13]float64  `json:"pin_currents_a"`
	Power           float64      `json:"power_w"`
}

func (n *mixedNetwork) displayResult(p Component, x []float64) DisplayResult {
	v := n.displayVoltages(p, x)
	e := n.displayCurrents(p, v)
	r := DisplayResult{ID: p.ID, Kind: p.Kind, Model: DigitalDisplayModel(), Digit: -1, State: "blank", PinCurrents: e.Current}
	mask := uint8(0)
	for k := range 7 {
		r.SegmentVoltages[k] = v[4+k] - v[12]
		r.SegmentCurrents[k] = e.Current[4+k]
		if p.Kind == "bcd-decoder" {
			r.SegmentCurrents[k] *= -1
		}
		r.Lit[k] = r.SegmentCurrents[k] >= r.Model.VisibleCurrent
		if r.Lit[k] {
			mask |= 1 << k
		}
	}
	if p.Kind == "bcd-decoder" {
		span := v[11] - v[12]
		r.Digit = 0
		for k := range 4 {
			l := logicLevel(v[k]-v[12], span)
			if l == "high" {
				r.Digit |= 1 << k
			} else if l != "low" {
				r.Digit = -1
				break
			}
		}
		switch {
		case span <= 0:
			r.State = "unpowered"
			r.Digit = -1
		case r.Digit < 0:
			r.State = "intermediate"
		case r.Digit >= 10:
			r.State = "invalid BCD · blank"
			r.Digit = -1
		default:
			r.State = fmt.Sprint(r.Digit)
		}
	} else {
		for digit, bits := range digitMasks {
			if bits == mask {
				r.Digit = digit
				r.State = fmt.Sprint(digit)
				break
			}
		}
		if mask != 0 && r.Digit < 0 {
			r.State = "partial"
		}
	}
	for k, i := range e.Current {
		r.Power += (v[k] - v[12]) * i
	}
	return r
}
func validateRecordedDisplays(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, d := range r.Displays {
		p, ok := g.Component(d.ID)
		if !ok || !IsDisplayKind(p.Kind) || p.Kind != d.Kind {
			return fmt.Errorf("invalid recorded display identity")
		}
		ids[d.Kind] = append(ids[d.Kind], d.ID)
	}
	for _, kind := range []string{"bcd-decoder", "seven-segment"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.Displays) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, actual := range r.Displays {
		p, _ := g.Component(actual.ID)
		expected := n.displayResult(p, x)
		if actual.Model != expected.Model || actual.Digit != expected.Digit || actual.State != expected.State || actual.Lit != expected.Lit {
			return fmt.Errorf("recorded display model/segments disagree with terminal voltages")
		}
		pairs := [][2]float64{{actual.Power, expected.Power}}
		for k := range actual.SegmentVoltages {
			pairs = append(pairs, [2]float64{actual.SegmentVoltages[k], expected.SegmentVoltages[k]}, [2]float64{actual.SegmentCurrents[k], expected.SegmentCurrents[k]})
		}
		for k := range actual.PinCurrents {
			pairs = append(pairs, [2]float64{actual.PinCurrents[k], expected.PinCurrents[k]})
		}
		for _, pair := range pairs {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded display readings disagree with terminal voltages")
			}
		}
	}
	return nil
}
