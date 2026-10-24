package workbench

import "fmt"

type ExampleInfo struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Question   string `json:"question"`
	Expected   string `json:"expected"`
	Try        string `json:"try"`
	Limitation string `json:"limitation"`
}
type ExampleContext struct {
	Info     ExampleInfo `json:"info"`
	Modified bool        `json:"modified"`
}
type ExampleLoadParams struct {
	ID                 string  `json:"id"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
}
type ExampleResetParams struct {
	ExpectedID         string  `json:"expected_example_id,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
}

// legacyExampleID keeps named loaders and their restart semantics in sync.
func legacyExampleID(command string) string {
	switch command {
	case "example.reset":
		return "led"
	case "example.rc":
		return "rc"
	case "example.npn":
		return "npn"
	case "example.npn-pulse":
		return "blink"
	case "example.flasher":
		return "flasher"
	case "example.switch-npn":
		return "switch-npn"
	case "example.rectifier":
		return "rectifier"
	default:
		return ""
	}
}

func ExampleCatalog() []ExampleInfo {
	return []ExampleInfo{
		{ID: "led", Title: "LED current limiter", Question: "How does resistance change LED current?", Expected: "Increasing the series resistance lowers current and the illustrative LED brightness.", Try: "Change R1 from 1 kΩ to 2 kΩ and compare signed current.", Limitation: "Generic red LED; brightness is illustrative and there is no thermal behavior or physical part rating."},
		{ID: "split-supply", Title: "Split-rail DC supply", Question: "How do two positive batteries create a negative rail?", Expected: "The midpoint reference makes the rails +3 V and −3 V. Each source remains +3 V across its own terminals; R1 sees 6 V and carries 6 mA.", Try: "Inspect V2.negative relative to ground, then V2 across its terminals. Change either source to 5 V or move ground to the bottom rail and compare.", Limitation: "Ideal DC sources have no output resistance, current limit, battery discharge or physical ratings. The network still needs one common reference; ideal voltage-source/capacitor constraint loops refuse."},
		{ID: "rc", Title: "RC charge and discharge", Question: "How quickly does a capacitor charge and discharge?", Expected: "Capacitor voltage changes smoothly; switching from charge to discharge reverses current while preserving voltage at the switching instant.", Try: "Pause, step 10 ms, then choose Discharge. Increase R1 or C1 to slow the response.", Limitation: "Ideal capacitor/resistor with zero ESR and leakage; this example uses an ideal source edge."},
		{ID: "rc-high-pass", Title: "RC high-pass edge detector", Question: "Why does a capacitor pass changes but block a steady voltage?", Expected: "The voltage across R1 jumps positive at a rising input edge and negative at a falling edge, then decays toward zero. C1 voltage stays continuous at each edge.", Try: "Pause and step 50 ms to the falling edge. Compare input, R1 output and C1 voltage; increase R1 or C1 to lengthen the decay (10 ms in the template).", Limitation: "Ideal resistor/capacitor and instantaneous 0–5 V square source. ESR, leakage, component ratings and additional output loading are omitted."},
		{ID: "rl", Title: "Coil current buildup and decay", Question: "Why does coil current keep flowing when the voltage changes?", Expected: "Current approaches 50 mA while input is high. At 50 ms, coil voltage reverses while current stays continuous and decays through R1; the time constant is 10 ms.", Try: "Plot coil current and voltage across the falling edge. Double L1 to 2 H, or set a signed startup current such as -20 mA and Restart.", Limitation: "Ideal 1 H coil and explicit 100 Ω resistance; no winding resistance beyond R1, saturation, parasitic capacitance, heating or ratings. Source low supplies an ideal return path."},
		{ID: "rlc", Title: "RLC ringing and damping", Question: "How do electric and magnetic stored energy produce ringing?", Expected: "Capacitor voltage overshoots 5 V and rings while current reverses; R1 dissipates energy and the response settles.", Try: "Compare 50 Ω underdamping with 200 Ω critical damping and 500 Ω overdamping. Inspect signed L1 current and C1 voltage; startup edits reset the experiment.", Limitation: "Ideal L/C with explicit series resistance. No winding saturation, ESR, parasitic capacitance or ratings. Internal integration and output sampling have separate work/resolution limits."},
		{ID: "flyback", Title: "Coil switching and flyback diode", Question: "Where does coil current go when the MOSFET turns off?", Expected: "After the falling gate edge, D1 provides a return path for the coil current. Drain voltage rises above the 5 V supply while stored magnetic energy drains through RL and D1.", Try: "Pause and step across 50 ms. Compare L1 current, drain voltage and diode current; increase L1 to lengthen decay. A missing diode path can exceed the model's node-voltage envelope.", Limitation: "Generic MOSFET/diode and ideal coil with explicit 100 Ω load resistance. No real coil saturation, switch avalanche, diode charge storage, heating or ratings."},
		{ID: "pnp", Title: "PNP high-side LED switch", Question: "How does a PNP transistor switch the supply side?", Expected: "Low base drive lights D1; high drive turns it off. PNP base and collector currents are negative under the shared into-device convention.", Try: "Choose Low / High under the scope. Plot Q1 VBE and collector current; increase RB and compare.", Limitation: "Generic complementary Ebers–Moll model; no device identity, breakdown, heating, internal capacitance or switching delay. LED brightness is illustrative."},
		{ID: "emitter-follower", Title: "NPN emitter follower", Question: "Why does the emitter follow the base with a voltage drop?", Expected: "The emitter follows the 10 Hz input below the base voltage; RL carries much more current than the base resistor.", Try: "Compare VIN, Q1 base and emitter voltages and base/emitter currents. Change RB or RL to see loading.", Limitation: "Generic transistor with instantaneous response; no high-frequency bandwidth, Early effect, heating or physical ratings."},
		{ID: "bipolar-push-pull", Title: "Complementary transistor buffer", Question: "How can an NPN/PNP pair source and sink load current?", Expected: "Q1 supplies the positive half-cycle, Q2 the negative half-cycle. The unbiased stage has a visible crossover dead band near zero input.", Try: "Plot VIN, the shared emitters and both collector currents. Pause and compare positive, negative and near-zero input samples.", Limitation: "Unbiased educational class-B stage with illustrative complementary parameters; no thermal behavior, internal charge, real matched pair or physical power ratings."},
		{ID: "npn", Title: "Transistor LED switch", Question: "How does a small base current control the LED?", Expected: "Low base drive turns the LED off; higher drive increases collector current until the transistor approaches saturation.", Try: "Choose base 0 V, 0.8 V and 5 V, then change RB and compare base/collector currents.", Limitation: "Educational Ebers–Moll NPN; no physical transistor identity, heating, breakdown or internal charge storage."},
		{ID: "switch-npn", Title: "Manual transistor LED switch", Question: "How can a switch control an LED through a transistor?", Expected: "Closing S1 supplies base current and lights the LED. Opening S1 lets RPD hold the base near ground; the LED turns off.", Try: "Select S1 and choose Close switch / Open switch. Inspect Q1 base and collector currents, then increase RB.", Limitation: "Finite 1 Ω on / 1 GΩ off switch; educational NPN and illustrative LED brightness. No contact bounce, heating or physical ratings."},
		{ID: "pmos", Title: "High-side MOSFET LED switch", Question: "How does a P-channel MOSFET switch the supply side?", Expected: "Gate high at the 5 V source turns M1 off. Pulling the gate low turns the LED on; signed drain current is negative because it leaves the drain.", Try: "Compare M1 gate-source voltage and drain current across a pulse edge. Change RG to 10k and add an external 1uF gate-source capacitor for delayed switching.", Limitation: PMOSDeviceModel().Limitations},
		{ID: "nmos", Title: "MOSFET LED switch", Question: "How does gate voltage switch an LED without base current?", Expected: "A high gate voltage makes M1 conduct in its low-resistance triode region; a low gate turns the LED off. Gate current is zero in this model.", Try: "Pause and step across a pulse edge. Compare M1 gate-source voltage, drain-source voltage and drain current; change RG or add an external gate capacitor.", Limitation: NMOSDeviceModel().Limitations},
		{ID: "cmos", Title: "Complementary MOSFET inverter", Question: "How do N-channel and P-channel MOSFETs invert a signal?", Expected: "High input turns MN on and drives output near ground. Low input turns MP on and drives output near 5 V. PMOS drain current is negative; NMOS drain current is positive.", Try: "Pause and step 200 ms to the falling input edge. Compare input/output and both drain currents. Change VIN high to 2.5 V and Restart to see both channels conduct.", Limitation: "Generic educational MOSFETs, not a physical matched pair or rated logic gate. The explicit 10 kΩ output load defines an otherwise undriven output. Zero intrinsic gate charge, capacitance, delay, heating and breakdown; ideal pulse edges have no modeled transition loss."},
		{ID: "opamp-follower", Title: "Op-amp voltage follower", Question: "How does negative feedback make an output follow its input?", Expected: "The 10 Hz, ±1 V sine and output closely overlap. The explicit ±5 V supplies provide the delivered load current; input current is much smaller.", Try: "Pause and step 25 ms from Restart to inspect the positive peak. Reduce RL from 10 kΩ to 10 Ω and Restart to see the generic output-current limit distort the signal.", Limitation: OpAmpDeviceModel().Limitations},
		{ID: "opamp-inverting", Title: "Inverting op-amp amplifier", Question: "How do two resistors set an amplifier's sign and gain?", Expected: "With RI 10 kΩ and RF 20 kΩ, the ±0.5 V sine produces approximately ∓1 V at the output. The input difference stays small; signed output current reverses with the waveform.", Try: "Pause and step 25 ms from Restart. Change RF from 20 kΩ to 10 kΩ, then Restart: the output amplitude halves and remains inverted.", Limitation: OpAmpDeviceModel().Limitations},
		{ID: "blink", Title: "Pulse-controlled LED", Question: "What happens when transistor base drive alternates?", Expected: "The LED follows the high/low base-drive phases; the scope separates signed base and collector behavior.", Try: "Change pulse high/low durations, then pause and measure matching edges.", Limitation: "Instantaneous junction response; no physical switching delay or stored transistor charge."},
		{ID: "flasher", Title: "Capacitor-timed LED flasher", Question: "How can two RC stages make alternating light?", Expected: "The LEDs alternate as capacitor voltages drive the two transistor bases; larger timing capacitors generally lengthen the period.", Try: "Plot both LED currents and collector voltages; double both capacitors and compare measured periods.", Limitation: "Generic grounded-emitter NPN/LED models, ideal capacitors and an explicit asymmetric startup; hardware tolerances are not modeled."},
		{ID: "led-fade", Title: "Capacitor-buffered LED fade", Question: "How does stored capacitor charge change an LED pulse?", Expected: "C1 charges across D1 while the input is high; after the input falls, stored charge keeps LED current flowing briefly as voltage decays.", Try: "Compare input voltage, C1 voltage and D1 current. Change C1 from 100 µF to 220 µF and compare the falling-edge tail.", Limitation: "Ideal capacitor and generic red LED; source low discharges through R1 as well as D1. Brightness is illustrative; no physical LED rating, ESR or heating."},
		{ID: "rectifier", Title: "Diode rectifier and smoothing", Question: "How does a diode and capacitor turn AC into a rippling DC voltage?", Expected: "D1 charges C1 during positive peaks. Between peaks the diode blocks and RL drains C1; a larger capacitor reduces ripple.", Try: "Compare input and C1 voltage on the scope. Change C1 from 10 µF to 47 µF, then compare settled ripple after 2 simulated seconds.", Limitation: "A 5 V peak, 10 Hz educational source; no mains, transformer or physical power-supply ratings. Ideal capacitor and generic diode omit ESR, breakdown, reverse recovery and heating."},
	}
}

type exampleDefinition struct {
	info     ExampleInfo
	graph    Graph
	settings TransientSettings
	view     View
	selected string
	probes   []string
}

func findExample(id string) (exampleDefinition, bool) {
	d := exampleDefinition{settings: DefaultTransientSettings(), view: View{Zoom: 1}}
	found := false
	for _, info := range ExampleCatalog() {
		if info.ID == id {
			d.info = info
			found = true
			break
		}
	}
	if !found {
		return d, false
	}
	switch id {
	case "opamp-follower", "opamp-inverting":
		d.graph = OpAmpFollowerExampleGraph()
		if id == "opamp-inverting" {
			d.graph = OpAmpInvertingExampleGraph()
		}
		d.selected = "U1"
		d.settings.PlaybackSpeed = .05
		d.probes = []string{"v:VIN.positive", "v:U1", "i:U1", "v:U1.input-difference"}
		d.view = View{Center: Point{X: 20, Y: -40}, Zoom: .7}
	case "led":
		d.graph = ExampleGraph()
	case "split-supply":
		d.graph = SplitSupplyExampleGraph()
		d.selected = "R1"
		d.view = View{Zoom: .9}
	case "rc":
		d.graph = RCExampleGraph()
		d.view = View{Center: Point{X: -130, Y: 20}, Zoom: .7}
	case "rc-high-pass":
		d.graph = RCHighPassExampleGraph()
		d.selected = "R1"
		d.probes = []string{"v:V1.positive", "v:R1", "v:C1"}
		d.view = View{Center: Point{X: -70}, Zoom: .7}
	case "pnp":
		d.graph = PNPExampleGraph()
		d.selected = "Q1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:VIN.positive", "i:D1", "v:Q1.base-emitter", "i:Q1.collector"}
	case "emitter-follower", "bipolar-push-pull":
		d.graph = EmitterFollowerExampleGraph()
		d.selected = "Q1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:VIN.positive", "v:Q1.base", "v:Q1.emitter", "i:Q1.base", "i:Q1.emitter"}
		if id == "bipolar-push-pull" {
			d.graph = ComplementaryBipolarExampleGraph()
			d.probes = []string{"v:VIN.positive", "v:Q1.emitter", "i:Q1.collector", "i:Q2.collector"}
		}
	case "npn":
		d.graph = NPNExampleGraph()
	case "rl":
		d.graph = RLExampleGraph()
		d.selected = "L1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"i:L1", "v:L1", "v:V1.positive"}
	case "rlc":
		d.graph = RLCExampleGraph()
		d.selected = "L1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .0001, .01
		d.probes = []string{"v:C1", "i:L1", "v:L1", "v:V1.positive"}
	case "flyback":
		d.graph = FlybackExampleGraph()
		d.selected = "L1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"i:L1", "v:M1.drain", "i:D1", "v:VIN.positive"}
	case "switch-npn":
		d.graph = SwitchedNPNExampleGraph()
		d.view = View{Center: Point{X: -40, Y: 20}, Zoom: .7}
		d.selected = "S1"
	case "pmos":
		d.graph = PMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: 20, Y: 20}, Zoom: .5}
		d.selected = "M1"
	case "nmos":
		d.graph = NMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: 20}, Zoom: .65}
		d.selected = "M1"
	case "cmos":
		d.graph = CMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: 30}, Zoom: .5}
		d.selected = "MN"
		d.probes = []string{"v:VIN.positive", "v:MN", "i:MP.drain", "i:MN.drain"}
	case "blink":
		d.graph = NPNPulseExampleGraph()
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: -30}, Zoom: .5}
	case "flasher":
		d.graph = FlasherExampleGraph()
		d.settings.StepSeconds = .005
		d.settings.PlaybackSpeed = .5
		d.view = View{Center: Point{Y: -10}, Zoom: .55}
	case "led-fade":
		d.graph = LEDFadeExampleGraph()
		d.settings.HighSeconds = .2
		d.settings.LowSeconds = .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: 40}, Zoom: .65}
		d.selected = "C1"
	case "rectifier":
		d.graph = RectifierExampleGraph()
		d.settings.StepSeconds = .001
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: -180, Y: 30}, Zoom: .55}
		d.selected = "C1"
	}
	return d, true
}
func (s *Service) exampleContextLocked() *ExampleContext {
	d, ok := findExample(s.exampleID)
	if !ok {
		return nil
	}
	return &ExampleContext{Info: d.info, Modified: contentHash(s.graph, s.transientSettings) != contentHash(d.graph, d.settings)}
}
func cloneExampleContext(in *ExampleContext) *ExampleContext {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
func (s *Service) loadExampleLocked(id string, discard bool) error {
	d, ok := findExample(id)
	if !ok {
		return &actionError{"not-found", "Choose an example ID from examples.list"}
	}
	if err := s.requireCleanLocked(discard); err != nil {
		return err
	}
	s.detachDocumentLocked()
	s.exampleID = id
	s.graph = d.graph
	s.transientSettings = d.settings
	s.view = d.view
	s.paused = false
	// Model state belongs to the old document. Leave no old transient clock
	// interpretable as this template's startup while asynchronous work begins.
	s.transient = nil
	s.npn = nil
	s.result = nil
	s.mixed = nil
	s.mixedState = nil
	for _, c := range s.graph.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range s.graph.Wires {
		s.usedIDs[w.ID] = true
	}
	s.drawing = DefaultDrawing()
	s.selected = d.selected
	s.viewport.Following = true
	s.fitViewportLocked()
	// A shipped template is the clean starting point for this editable copy.
	// It has no file association, but loading it is not an unsaved user edit.
	s.savedGraph = contentHash(s.graph, s.transientSettings)
	s.savedPreferences = preferencesHash(s.pins, s.displayPreferences, s.exampleID)
	return nil
}
func (s *Service) resetCurrentExampleLocked(p ExampleResetParams) error {
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if p.ExpectedID != "" && p.ExpectedID != s.exampleID {
		return &actionError{"conflict", "Current example changed while reset was pending; choose Reset again"}
	}
	if s.exampleID == "" {
		return &actionError{"unavailable", "Load an example before resetting it"}
	}
	if _, ok := findExample(s.exampleID); !ok {
		return fmt.Errorf("current example is not recognized")
	}
	paused := s.paused
	if err := s.loadExampleLocked(s.exampleID, p.DiscardChanges); err != nil {
		return err
	}
	s.paused = paused
	s.restorePaused = paused
	return nil
}
