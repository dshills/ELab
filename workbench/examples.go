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

func ExampleCatalog() []ExampleInfo {
	return []ExampleInfo{
		{ID: "led", Title: "LED current limiter", Question: "How does resistance change LED current?", Expected: "Increasing the series resistance lowers current and the illustrative LED brightness.", Try: "Change R1 from 1 kΩ to 2 kΩ and compare signed current.", Limitation: "Generic red LED; brightness is illustrative and there is no thermal behavior or physical part rating."},
		{ID: "rc", Title: "RC charge and discharge", Question: "How quickly does a capacitor charge and discharge?", Expected: "Capacitor voltage changes smoothly; switching from charge to discharge reverses current while preserving voltage at the switching instant.", Try: "Pause, step 10 ms, then choose Discharge. Increase R1 or C1 to slow the response.", Limitation: "Ideal capacitor/resistor with zero ESR and leakage; this example uses an ideal source edge."},
		{ID: "npn", Title: "Transistor LED switch", Question: "How does a small base current control the LED?", Expected: "Low base drive turns the LED off; higher drive increases collector current until the transistor approaches saturation.", Try: "Choose base 0 V, 0.8 V and 5 V, then change RB and compare base/collector currents.", Limitation: "Educational Ebers–Moll NPN; no physical transistor identity, heating, breakdown or internal charge storage."},
		{ID: "switch-npn", Title: "Manual transistor LED switch", Question: "How can a switch control an LED through a transistor?", Expected: "Closing S1 supplies base current and lights the LED. Opening S1 lets RPD hold the base near ground; the LED turns off.", Try: "Select S1 and choose Close switch / Open switch. Inspect Q1 base and collector currents, then increase RB.", Limitation: "Finite 1 Ω on / 1 GΩ off switch; educational NPN and illustrative LED brightness. No contact bounce, heating or physical ratings."},
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
	case "led":
		d.graph = ExampleGraph()
	case "rc":
		d.graph = RCExampleGraph()
		d.view = View{Center: Point{X: -130, Y: 20}, Zoom: .7}
	case "npn":
		d.graph = NPNExampleGraph()
	case "switch-npn":
		d.graph = SwitchedNPNExampleGraph()
		d.view = View{Center: Point{X: -40, Y: 20}, Zoom: .7}
		d.selected = "S1"
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
