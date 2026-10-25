package workbench

import (
	"fmt"
	"math"
	"strings"
)

var shiftPins = []string{"data", "clk", "latch", "reset_n", "oe_n", "q0", "q1", "q2", "q3", "q4", "q5", "q6", "q7", "serial", "vcc", "gnd"}

// Internal stages are implementation details; public pins and saved circuits
// retain the functional register identity.
func shiftStage(id string, k int) string           { return fmt.Sprintf("$%s_stage%d", id, k) }
func shiftClear(id string) string                  { return "$" + id + "_clear" }
func stageNet(id string, k int, pin string) string { return shiftStage(id, k) + "_" + pin }
func lowerShift(c *Compiled, p Component) []Component {
	parts := []Component{p}
	clear := Component{ID: shiftClear(p.ID), Kind: "logic-not", Terminals: logicPins("logic-not")}
	parts = append(parts, clear)
	connect := func(id, pin, net string) { c.TerminalNet[id+"."+pin] = net }
	net := func(pin string) string { return c.TerminalNet[p.ID+"."+pin] }
	reset := shiftClear(p.ID) + "_out"
	c.Nets = append(c.Nets, Net{ID: reset})
	connect(clear.ID, "a", net("reset_n"))
	connect(clear.ID, "out", reset)
	connect(clear.ID, "vcc", net("vcc"))
	connect(clear.ID, "gnd", net("gnd"))
	for k := 0; k < 16; k++ {
		id := shiftStage(p.ID, k)
		stage := Component{ID: id, Kind: "d-flipflop", Terminals: []string{"clk", "reset", "vcc", "gnd"}}
		if k == 0 {
			stage.Terminals = append(stage.Terminals, "d")
		}
		parts = append(parts, stage)
		d, clk, r := net("data"), net("clk"), reset
		if k > 0 && k < 8 {
			d = stageNet(p.ID, k-1, "q")
		}
		if k >= 8 {
			d = stageNet(p.ID, k-8, "q")
			clk = net("latch")
			r = net("gnd")
		}
		connect(id, "d", d)
		connect(id, "clk", clk)
		connect(id, "reset", r)
		connect(id, "vcc", net("vcc"))
		connect(id, "gnd", net("gnd"))

	}
	return parts
}

type ShiftResult struct {
	ID          string               `json:"id"`
	Model       string               `json:"model"`
	Shifted     uint8                `json:"shifted_byte"`
	Latched     uint8                `json:"latched_byte"`
	Enabled     bool                 `json:"outputs_enabled"`
	Memories    [16]SequentialMemory `json:"memories"`
	Voltages    [9]float64           `json:"output_voltages_v"`
	PinCurrents [16]float64          `json:"pin_currents_a"`
	Power       float64              `json:"power_w"`
}

func (n *mixedNetwork) shiftMemories(p Component) (m [16]SequentialMemory) {
	for k := range m {
		m[k] = n.sequentialMemory(Component{ID: shiftStage(p.ID, k)})
	}
	return
}
func shiftCurrents(v [16]float64, m [16]SequentialMemory) (i [16]float64, j [16][16]float64, enabled bool) {
	branch := func(a, b int, g float64, dg [16]float64) {
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
	branch(4, 15, 1e-9, [16]float64{})
	span := v[14] - v[15]
	weight, dv, ds := 0., 0., 0.
	if span > 0 {
		weight, dv, ds = logicInput(v[4]-v[15], span)
		dv /= span
		ds /= span
	}
	enabled = span >= 2 && span <= 12 && logicLevel(v[4]-v[15], span) == "low"
	for k := 0; k < 9; k++ {
		mem := m[7]
		if k < 8 {
			mem = m[k+8]
		}
		q := 1 - weight
		if k == 8 {
			q = 1
		}
		rail := 15
		if mem.Value != 0 {
			rail = 14
		}
		g := 1e-9
		d := [16]float64{}
		if mem.Powered {
			g += q / 50
			if k != 8 {
				d[4] = -dv / 50
				d[14] = -ds / 50
				d[15] = (dv + ds) / 50
			}
		}
		branch(k+5, rail, g, d)
	}
	return
}
func (n *mixedNetwork) shiftVoltages(p Component, x []float64) (v [16]float64) {
	for k, pin := range shiftPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}
func (n *mixedNetwork) stampShift(p Component, x []float64, add func(int, int, float64), f []float64) {
	v := n.shiftVoltages(p, x)
	i, j, _ := shiftCurrents(v, n.shiftMemories(p))
	for a, pin := range shiftPins {
		row := n.pin(p, pin)
		if row >= 0 {
			f[row] += i[a]
		}
		for b, other := range shiftPins {
			add(row, n.pin(p, other), j[a][b])
		}
	}
}
func (n *mixedNetwork) shiftResult(p Component, x []float64) ShiftResult {
	v := n.shiftVoltages(p, x)
	mem := n.shiftMemories(p)
	i, _, en := shiftCurrents(v, mem)
	r := ShiftResult{ID: p.ID, Model: ExtendedModelID, Memories: mem, Enabled: en, PinCurrents: i}
	for k, m := range mem {
		if m.Value == 1 {
			if k < 8 {
				r.Shifted |= 1 << k
			} else {
				r.Latched |= 1 << (k - 8)
			}
		}
	}
	// Sum only currents crossing the composite boundary. Internal stage-to-stage
	// currents cancel, and the external output buffers are already included.
	for k := 0; k < 16; k++ {
		stage, _ := n.c.MixedGraph.Component(shiftStage(p.ID, k))
		e := n.sequentialEvaluation(stage, x, mem[k])
		r.PinCurrents[14] += e.Current[13]
		r.PinCurrents[15] += e.Current[14]
		if k < 8 {
			r.PinCurrents[1] += e.Current[4]
			if k == 0 {
				r.PinCurrents[0] += e.Current[0]
			}
		} else {
			r.PinCurrents[2] += e.Current[4]
			r.PinCurrents[15] += e.Current[5]
		}
	}
	clear, _ := n.c.MixedGraph.Component(shiftClear(p.ID))
	e := logicCurrents(clear.Kind, n.logicVoltages(clear, x))
	r.PinCurrents[3] += e.Current[0]
	r.PinCurrents[14] += e.Current[3]
	r.PinCurrents[15] += e.Current[4]
	for k := range r.Voltages {
		r.Voltages[k] = v[k+5] - v[15]
	}
	for k, i := range r.PinCurrents {
		r.Power += (v[k] - v[15]) * i
	}
	return r
}
func (n *mixedNetwork) restoreShift(r *MixedResult, x []float64) error {
	for _, p := range n.authored.Components {
		if p.Kind != "shift-register" {
			continue
		}
		var found *ShiftResult
		for k := range r.Shifts {
			if r.Shifts[k].ID == p.ID {
				found = &r.Shifts[k]
			}
		}
		if found == nil {
			return fmt.Errorf("missing shift register state")
		}
		v := n.shiftVoltages(p, x)
		span := v[14] - v[15]
		enable := 0.
		if span > 0 {
			enable = span * span / (span*span + .01)
		}
		q, _, _ := logicInput(v[3]-v[15], math.Max(span, 1e-20))
		q = 1 - q
		gup, gdown := enable*q/50, 1e-9+enable*(1-q)/50
		x[n.index[shiftClear(p.ID)+"_out"]] = v[15] + span*gup/(gup+gdown+8e-9)
		for k, m := range found.Memories {
			if m.Value < 0 || m.Value > 1 || !m.Powered && m != (SequentialMemory{}) || !m.ClockKnown && m.ClockHigh || m.ResetHigh && m.Value != 0 {
				return fmt.Errorf("invalid register memory")
			}
			for j, s := range n.sequential {
				if s.ID == shiftStage(p.ID, k) {
					n.sequenceMemory[j] = m
				}
			}
		}
		for k, m := range found.Memories {
			stage, _ := n.c.MixedGraph.Component(shiftStage(p.ID, k))
			next, err := sequentialDecision("d-flipflop", n.sequentialVoltages(stage, x), m, false)
			if err != nil || next != m {
				return fmt.Errorf("register guards disagree with terminal voltages")
			}
		}
	}
	return nil
}
func validateRecordedShifts(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, s := range r.Shifts {
		ids = append(ids, s.ID)
	}
	if err := recordedObjectIDs(g, "shift-register", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, s := range r.Shifts {
		p, _ := g.Component(s.ID)
		e := n.shiftResult(p, x)
		if s.Model != e.Model || s.Shifted != e.Shifted || s.Latched != e.Latched || s.Enabled != e.Enabled || s.Memories != e.Memories || !recordingNear(s.Power, e.Power) {
			return fmt.Errorf("invalid shift register readings")
		}
		for k, v := range s.Voltages {
			if !recordingNear(v, e.Voltages[k]) {
				return fmt.Errorf("invalid register output")
			}
		}
		for k, i := range s.PinCurrents {
			if !recordingNear(i, e.PinCurrents[k]) {
				return fmt.Errorf("invalid register current")
			}
		}
	}
	return nil
}

// Diagnostics identify the authored register, never its private solver stages.
func (n *mixedNetwork) sequentialPublicError(p Component, err error) error {
	for _, parent := range n.authored.Components {
		if parent.Kind != "shift-register" {
			continue
		}
		for k := 0; k < 16; k++ {
			if p.ID != shiftStage(parent.ID, k) {
				continue
			}
			msg := err.Error()
			if strings.HasPrefix(msg, "d ") {
				msg = "DATA " + strings.TrimPrefix(msg, "d ")
			}
			return fmt.Errorf("%s: %s", parent.ID, msg)
		}
	}
	return fmt.Errorf("%s: %w", p.ID, err)
}
