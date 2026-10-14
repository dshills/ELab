package workbench

import (
	"context"
	"fmt"
	"math"
)

const NPNModelID = "elab.generic-npn.v1"
const NPNEngineID = "elab.npn-switch-dc.v1"
const NPNAnalysis = "NPN DC operating point"

type NPNModel struct {
	ID                string  `json:"id"`
	SaturationCurrent float64 `json:"saturation_current_a"`
	ForwardBeta       float64 `json:"forward_beta"`
	ReverseBeta       float64 `json:"reverse_beta"`
	Temperature       float64 `json:"temperature_c"`
	Limitations       string  `json:"limitations"`
}

func TransistorModel() NPNModel {
	return NPNModel{NPNModelID, 1e-14, 100, 1, 27, "Educational Ebers–Moll DC model; no named physical device, breakdown, Early effect, heating, junction capacitance or switching delay. LED reverse current is neglected. Supports a grounded-emitter switch only."}
}

type NPNResult struct {
	RunID              uint64             `json:"run_id"`
	ElectricalRevision uint64             `json:"electrical_revision"`
	Engine             string             `json:"engine"`
	Model              NPNModel           `json:"model"`
	TransistorID       string             `json:"transistor_id"`
	SupplyID           string             `json:"supply_id"`
	InputID            string             `json:"input_id"`
	BaseResistorID     string             `json:"base_resistor_id"`
	LoadResistorID     string             `json:"load_resistor_id"`
	LEDID              string             `json:"led_id,omitempty"`
	BaseCurrent        float64            `json:"base_current_a"`
	CollectorCurrent   float64            `json:"collector_current_a"`
	EmitterCurrent     float64            `json:"emitter_current_a"`
	VBE                float64            `json:"vbe_v"`
	VCE                float64            `json:"vce_v"`
	LEDVoltage         float64            `json:"led_voltage_v"`
	LEDPower           float64            `json:"led_power_w"`
	TransistorPower    float64            `json:"transistor_power_w"`
	BaseResistorPower  float64            `json:"base_resistor_power_w"`
	LoadResistorPower  float64            `json:"load_resistor_power_w"`
	KCLResidual        float64            `json:"kcl_residual_a"`
	KVLResidual        float64            `json:"kvl_residual_v"`
	State              string             `json:"state"`
	Explanation        string             `json:"explanation"`
	Nodes              map[string]float64 `json:"nodes_v"`
	Stale              bool               `json:"stale"`
}

func cloneNPN(r *NPNResult) *NPNResult {
	if r == nil {
		return nil
	}
	out := *r
	out.Nodes = make(map[string]float64, len(r.Nodes))
	for k, v := range r.Nodes {
		out.Nodes[k] = v
	}
	return &out
}

// Transport form of Ebers–Moll with reciprocal transport saturation current IS.
// Positive IB/IC enter the transistor; IE leaves its grounded emitter.
func npnCurrents(vb, vc float64) (ib, ic float64) {
	m := TransistorModel()
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	f, r := m.SaturationCurrent*math.Expm1(vb/vt), m.SaturationCurrent*math.Expm1((vb-vc)/vt)
	return f/m.ForwardBeta + r/m.ReverseBeta, f - r*(1+1/m.ReverseBeta)
}
func ledDrop(i float64) float64 {
	m := Model()
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	return m.EmissionCoefficient*vt*math.Log1p(i/m.SaturationCurrent) + i*m.SeriesResistance
}
func npnLoadCurrent(c Compiled, vc float64) float64 {
	voltage := c.Source.Value - vc
	if c.LED.ID == "" {
		return voltage / c.Resistor.Value
	}
	if voltage <= 0 {
		return 0
	}
	lo, hi := 0., voltage/c.Resistor.Value
	for n := 0; n < 90; n++ {
		mid := (lo + hi) / 2
		if mid*c.Resistor.Value+ledDrop(mid) > voltage {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}
func SolveNPN(ctx context.Context, c Compiled) (NPNResult, error) {
	if c.Analysis != NPNAnalysis {
		return NPNResult{}, fmt.Errorf("requires a compiled NPN switch")
	}
	// For fixed collector voltage, base KCL is strictly increasing in VB.
	base := func(vc float64) float64 {
		lo, hi := 0., math.Max(1, c.Input.Value)
		for n := 0; n < 100; n++ {
			vb := (lo + hi) / 2
			ib, _ := npnCurrents(vb, vc)
			if ib > (c.Input.Value-vb)/c.BaseResistor.Value {
				hi = vb
			} else {
				lo = vb
			}
		}
		return (lo + hi) / 2
	}
	// Collector KCL is monotone after solving base KCL; the bracket also permits
	// base drive above the supply without inventing a fixed saturation voltage.
	lo, hi := 0., math.Max(1, math.Max(c.Source.Value, c.Input.Value))
	for n := 0; n < 110; n++ {
		if err := ctx.Err(); err != nil {
			return NPNResult{}, err
		}
		vc := (lo + hi) / 2
		_, ic := npnCurrents(base(vc), vc)
		if ic > npnLoadCurrent(c, vc) {
			hi = vc
		} else {
			lo = vc
		}
	}
	vc := (lo + hi) / 2
	vb := base(vc)
	modelIB, modelIC := npnCurrents(vb, vc)
	ib := (c.Input.Value - vb) / c.BaseResistor.Value
	ic := npnLoadCurrent(c, vc)
	led := 0.
	if c.LED.ID != "" {
		led = ledDrop(ic)
	}
	residual := math.Max(math.Abs(ib-modelIB), math.Abs(ic-modelIC))
	if !finite(residual) || residual > 1e-9 {
		return NPNResult{}, fmt.Errorf("NPN solver did not converge (KCL residual %.3g A)", residual)
	}
	state, explanation := "active", "Base–emitter is forward biased and base–collector is reverse biased; collector current follows the available base drive."
	if math.Abs(ic) < 1e-8 && math.Abs(ib) < 1e-8 {
		state = "cutoff"
		explanation = "Base drive is low; only model leakage flows and the LED is off."
	} else if vb > vc {
		state = "saturation"
		explanation = "Both junctions are forward biased; the collector load limits current and additional base drive has little effect on LED brightness."
	}
	nodes := map[string]float64{c.Ground: 0, c.Supply: c.Source.Value, c.Base: vb, c.Collector: vc, c.InputNet: c.Input.Value}
	if c.LED.ID != "" {
		if c.Anode == c.Supply {
			nodes[c.Cathode] = c.Source.Value - led
		} else {
			nodes[c.Anode] = vc + led
			nodes[c.Cathode] = vc
		}
	}
	return NPNResult{Engine: NPNEngineID, Model: TransistorModel(), TransistorID: c.Transistor.ID, SupplyID: c.Source.ID, InputID: c.Input.ID, BaseResistorID: c.BaseResistor.ID, LoadResistorID: c.Resistor.ID, LEDID: c.LED.ID, BaseCurrent: ib, CollectorCurrent: ic, EmitterCurrent: ib + ic, VBE: vb, VCE: vc, LEDVoltage: led, LEDPower: led * ic, TransistorPower: vb*ib + vc*ic, BaseResistorPower: ib * ib * c.BaseResistor.Value, LoadResistorPower: ic * ic * c.Resistor.Value, KCLResidual: residual, KVLResidual: c.Source.Value - vc - ic*c.Resistor.Value - led, State: state, Explanation: explanation, Nodes: nodes}, nil
}

func compileNPN(out Compiled, g Graph, sources, resistors, leds, transistors, capacitors, grounds []Component, groups map[string][]string, root func(string) string) (Compiled, string, *Diagnostic) {
	out.Analysis = NPNAnalysis
	fail := func(status, code, message, id string) (Compiled, string, *Diagnostic) {
		return out, status, &Diagnostic{code, message, id}
	}
	if len(capacitors) > 0 || len(transistors) != 1 || len(sources) > 2 || len(resistors) > 2 || len(leds) > 1 {
		return fail("unsupported", "unsupported-topology", "NPN simulation supports one grounded-emitter NPN, a DC supply and DC/pulse base drive, two resistors, and an optional forward LED. Mixed capacitor/transistor circuits are not supported here.", "")
	}
	if len(sources) == 0 || len(resistors) < 2 || len(grounds) == 0 {
		return fail("incomplete", "missing-parts", "Place NPN, ground, two resistors, and supply/base-drive DC sources; an LED may be added in the collector load.", "")
	}

	net := func(c Component, pin string) string { return out.TerminalNet[c.ID+"."+pin] }
	out.Transistor = transistors[0]
	out.Ground = net(grounds[0], "reference")
	out.Base = net(out.Transistor, "base")
	out.Collector = net(out.Transistor, "collector")
	for _, c := range append(append(append([]Component{}, sources...), resistors...), append(leds, transistors...)...) {
		for _, t := range c.Terminals {
			ref := c.ID + "." + t
			if len(physicalTerminals(g, groups[root(ref)])) < 2 {
				return fail("incomplete", "open-terminal", "Connect terminal "+ref+" to complete the circuit.", ref)
			}
		}
		for i, a := range c.Terminals {
			for _, b := range c.Terminals[i+1:] {
				if net(c, a) == net(c, b) {
					return fail("error", "shorted-component", "A component has shorted terminals; remove the unintended connection.", c.ID)
				}
			}
		}
	}
	if net(out.Transistor, "emitter") != out.Ground {
		return fail("unsupported", "unsupported-topology", "Connect the NPN emitter directly to ground.", out.Transistor.ID+".emitter")
	}
	for _, s := range sources {
		if net(s, "negative") != out.Ground {
			return fail("incomplete", "reference-disconnected", "Connect each DC source negative terminal to ground.", s.ID+".negative")
		}
	}
	other := func(r Component, n string) string {
		a, b := net(r, "1"), net(r, "2")
		if a == n {
			return b
		}
		if b == n {
			return a
		}
		return ""
	}
	for i, r := range resistors {
		if n := other(r, out.Base); n != "" {
			out.BaseResistor = r
			out.Resistor = resistors[1-i]
			out.InputNet = n
			break
		}
	}
	for _, s := range sources {
		if net(s, "positive") == out.InputNet {
			out.Input = s
		}
	}
	loadSupply := other(out.Resistor, out.Collector)
	if len(leds) == 1 {
		out.LED = leds[0]
		out.Anode = net(out.LED, "anode")
		out.Cathode = net(out.LED, "cathode")
		loadSupply = ""
		if out.Cathode == out.Collector {
			loadSupply = other(out.Resistor, out.Anode)
		} else if other(out.Resistor, out.Collector) == out.Cathode {
			loadSupply = out.Anode
		}
	}
	for _, s := range sources {
		if net(s, "positive") == loadSupply {
			out.Source = s
			out.Supply = loadSupply
		}
	}
	if out.Source.ID == "" || out.Input.ID == "" || out.Source.ID == out.Input.ID && len(sources) != 1 || out.Base == out.Collector || out.Base == out.Ground || out.Collector == out.Ground || out.InputNet == out.Collector {
		return fail("unsupported", "unsupported-topology", "Wire DC input → base resistor → base, emitter → ground, and supply → load resistor / forward LED → collector. Unused sources or extra connections are not supported.", out.Transistor.ID)
	}
	// Every source must be used, and six electrical roles must remain distinct
	// except for a deliberately shared supply and input source.
	roles := []string{out.Ground, out.Base, out.Collector, out.Supply}
	if out.InputNet != out.Supply {
		roles = append(roles, out.InputNet)
	}
	if len(leds) > 0 {
		if out.Anode != out.Supply {
			roles = append(roles, out.Anode)
		} else {
			roles = append(roles, out.Cathode)
		}
	}
	seen := map[string]bool{}
	for _, n := range roles {
		if seen[n] {
			return fail("unsupported", "unsupported-topology", "An extra connection merges distinct switch nodes.", out.Transistor.ID)
		}
		seen[n] = true
	}
	if out.Source.Kind != "dc-source" {
		return fail("unsupported", "unsupported-topology", "Use a DC collector supply; the base drive may be a separate pulse source.", out.Source.ID)
	}
	if out.Input.Kind == "pulse-source" {
		out.Analysis = NPNTransientAnalysis
	}
	out.Circuit = Circuit{out.Source.Value, out.Resistor.Value, true}
	return out, "", nil
}
func NPNExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
	}{
		{"VCC", "dc-source", "5 V", Point{260, -100}}, {"VIN", "dc-source", "5 V", Point{-260, 60}}, {"RB", "resistor", "10k", Point{-100, 60}}, {"RL", "resistor", "1k", Point{100, -180}}, {"D1", "led", "", Point{100, -80}}, {"Q1", "npn", "", Point{100, 60}}, {"GND1", "ground", "", Point{100, 180}},
	} {
		c, _ := MakeComponent(p.id, p.kind, p.position, p.value)
		if p.id == "RL" {
			c.Rotation = 90
		}
		g.Components = append(g.Components, c)
	}
	for i, p := range [][2]string{{"VCC.positive", "RL.1"}, {"RL.2", "D1.anode"}, {"D1.cathode", "Q1.collector"}, {"VIN.positive", "RB.1"}, {"RB.2", "Q1.base"}, {"Q1.emitter", "GND1.reference"}, {"VCC.negative", "GND1.reference"}, {"VIN.negative", "GND1.reference"}} {
		g.Wires = append(g.Wires, Wire{fmt.Sprintf("W%d", i+1), p[0], p[1], true})
	}
	return g
}
func (s *Service) startNPNLocked(c Compiled, allowPaused bool) {
	s.result = nil
	s.transient = nil
	s.status = "updating"
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	revision, run := s.electrical, s.run
	go func() {
		r, err := SolveNPN(ctx, c)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.paused && !allowPaused || s.electrical != revision || s.run != run || ctx.Err() != nil {
			return
		}
		s.cancel = nil
		cancel()
		if err != nil {
			s.status = "error"
			s.diagnostics = []Diagnostic{{"solve-failed", err.Error(), c.Transistor.ID}}
		} else {
			r.RunID = run
			r.ElectricalRevision = revision
			s.npn = &r
			s.dcCompiled = &c
			if s.paused {
				s.status = "paused"
			} else {
				s.status = "running"
			}
		}
		s.revision++
		s.signalLocked()
	}()
}
