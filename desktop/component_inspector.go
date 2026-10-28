package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Put the selected part's main edits before explanatory and whole-network
// content so they remain reachable in a short inspector. All controls retain
// their ordinary shared commands and the assembled view's revision binding.
func (s *screen) componentInspector(state workbench.Snapshot, graph workbench.Graph, part workbench.Component) []ui.Node {
	if part.Block != nil {
		return s.blockInspector(state, part)
	}
	inspector := []ui.Node{}
	terminals := "Terminals: " + strings.Join(part.Terminals, ", ")
	if part.Kind == "timer555" {
		terminals = "2 TRIG · 6 THR · 7 DISCH · 3 OUT\n4 RESET · 5 CTRL · 8 VCC · 1 GND"
	}
	if part.Kind == "opamp" {
		terminals = "Inputs + / - · out · power V+ / V-"
	}
	inspector = append(inspector, heading("inspector", part.ID+" · "+part.Name), quiet("part-terminals", terminals))
	if part.Kind == "opamp" {
		if r := state.Mixed; r != nil {
			for _, u := range r.OpAmps {
				if u.ID == part.ID {
					status := "Live"
					if r.Stale {
						status = "Stale"
					}
					inspector = append(inspector, quiet("opamp-selected-reading", fmt.Sprintf("%s · %s\nOut %.5g V · delivered %.4g mA\nΔin %.4g µV · supply %.4g V\nInto V+ %.4g / V- %.4g mA\nPower %.4g mW", status, u.State, u.OutputVoltage, u.OutputCurrent*1000, u.InputDifference*1e6, u.SupplyVoltage, u.PositiveSupplyCurrent*1000, u.NegativeSupplyCurrent*1000, u.Power*1000)))
				}
			}
		}
	} else {
		model := part.Model
		if part.Device != nil && part.Device.Passive != nil {
			model = workbench.PassiveLossModelID
		}
		inspector = append(inspector, quiet("part-model-id", "Model: "+model))
	}
	if r := state.Mixed; r != nil {
		for _, q := range r.Transistors {
			if q.TransistorID == part.ID {
				status := "Live"
				if r.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("bipolar-selected-reading", fmt.Sprintf("%s · %s\nVBE %.6g V · VCE %.6g V\nIB %.6g µA · IC %.6g mA\nIE %.6g mA · power %.6g mW", status, q.State, q.VBE, q.VCE, q.BaseCurrent*1e6, q.CollectorCurrent*1000, q.EmitterCurrent*1000, q.TransistorPower*1000)), quiet("bipolar-current-signs", "IB / IC enter base / collector; IE leaves emitter. PNP readings normally have negative signs."))
			}
		}
	}
	if workbench.IsLogicKind(part.Kind) && state.Mixed != nil {
		for _, gate := range state.Mixed.LogicGates {
			if gate.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("logic-selected-reading", fmt.Sprintf("%s · output %s\nA %s · B %s\nOut %.5g V · delivered %.5g mA\nSupply %.5g V · power %.5g mW", status, gate.State, gate.InputA, gate.InputB, gate.OutputVoltage, -gate.PinCurrents[2]*1000, gate.SupplyVoltage, gate.Power*1000)))
			}
		}
	}
	if workbench.IsDisplayKind(part.Kind) && state.Mixed != nil {
		for _, d := range state.Mixed.Displays {
			if d.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("display-selected-reading", fmt.Sprintf("%s · %s\nSegments a–g: %v\nPower %.5g mW", status, d.State, d.Lit, d.Power*1000)))
			}
		}
	}
	if workbench.IsSequentialKind(part.Kind) && state.Mixed != nil {
		for _, u := range state.Mixed.Sequential {
			if u.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("sequential-selected-reading", fmt.Sprintf("%s · state %s\nClock %s · reset %s\nPrimary output %.5g V\nSupply %.5g V · power %.5g mW", status, u.State, u.Clock, u.Reset, u.OutputVoltage, u.SupplyVoltage, u.Power*1000)))
			}
		}
	}
	if part.Kind == "timer555" && state.Mixed != nil {
		for _, timer := range state.Mixed.Timers555 {
			if timer.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("timer555-selected-reading", fmt.Sprintf("%s · output %s\nOut %.5g V · delivered %.5g mA\nTrigger level %.5g V · threshold %.5g V\nSupply %.5g V · power %.5g mW", status, timer.State, timer.OutputVoltage, -timer.PinCurrents[2]*1000, timer.TriggerLevel, timer.ThresholdLevel, timer.SupplyVoltage, timer.Power*1000)))
			}
		}
	}
	if workbench.IsJFETKind(part.Kind) {
		inspector = append(inspector, s.jfetControls(state, part)...)
	}
	if workbench.IsCurrentDevice(part.Kind) {
		inspector = append(inspector, s.currentControls(state, part)...)
	}
	if workbench.IsPassiveStorageKind(part.Kind) {
		inspector = append(inspector, s.passiveControls(state, part)...)
	}
	if part.Kind == "speaker" || workbench.IsBipolarKind(part.Kind) || part.Kind == "pwm-controller" || part.Kind == "opamp" || part.Kind == "shift-register" || part.Kind == "piezo" || part.Kind == "active-buzzer" {
		inspector = append(inspector, s.extendedControls(state, part)...)
	}
	if part.Kind == "hbridge" {
		inspector = append(inspector, s.hbridgeControls(state, part)...)
	}
	if part.Kind == "transformer" {
		inspector = append(inspector, s.transformerControls(state, part)...)
	}
	if part.Kind == "optocoupler" {
		inspector = append(inspector, s.optocouplerControls(state, part)...)
	}
	if part.Kind == "dc-motor" {
		inspector = append(inspector, s.motorControls(state, part)...)
	}
	if part.Kind == "battery" {
		inspector = append(inspector, s.batteryControls(state, part)...)
	}
	if workbench.IsSensorKind(part.Kind) {
		inspector = append(inspector, s.sensorControls(state, part)...)
	}
	if part.Kind == "potentiometer" {
		inspector = append(inspector, s.potentiometerControls(state, part)...)
	}
	if workbench.IsRegulatorKind(part.Kind) && state.Mixed != nil {
		for _, r := range state.Mixed.Regulators {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("regulator-selected-reading", fmt.Sprintf("%s · %s\nInput %.5g V · output %.5g V\nDelivered %.5g mA · loss %.5g mW", status, r.State, r.InputVoltage, r.OutputVoltage, r.Current*1000, r.Power*1000)))
			}
		}
	}
	if part.Kind == "relay" && state.Mixed != nil {
		for _, c := range state.Mixed.Controls {
			if c.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("relay-selected-reading", fmt.Sprintf("%s · %s\nCoil %.5g mA · %.5g V\nStored energy %.5g µJ\nDissipation %.5g mW", status, c.State, c.Current*1000, c.Voltage, c.Energy*1e6, c.Power*1000)))
			}
		}
	}
	if part.Kind == "comparator" && state.Mixed != nil {
		for _, c := range state.Mixed.Controls {
			if c.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				inspector = append(inspector, quiet("comparator-selected-reading", fmt.Sprintf("%s · %s\nOutput %.5g V · delivered %.5g mA\nHysteresis %.5g mV · ±H/2", status, c.State, c.Voltage, c.Current*1000, part.Value*1000)))
			}
		}
	}
	if workbench.IsButtonKind(part.Kind) {
		inspector = append(inspector, s.buttonControls(state, part)...)
	}
	if part.Kind == "switch" {
		inspector = append(inspector, s.switchControls(state, part)...)
	}

	if part.Kind == "net-label" {
		inspector = append(inspector, section("label-title", "Net label · exact matching"), ui.Node{ID: "value-entry", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "Net label name"}, Change: s.changeValueDraft}, s.valueApplyAction(state, part.ID, "Apply label"), quiet("label-connectivity-help", "Names are case-sensitive. GND is a name; connect it to a ground symbol to establish the reference."))
	}
	if workbench.IsConnectivityPrimitive(part.Kind) {
		for _, net := range state.Nets {
			for _, ref := range net.Terminals {
				if ref == part.ID+".node" {
					inspector = append(inspector, quiet("primitive-net", net.ID+"\n"+strings.Join(net.Terminals, " · ")))
				}
			}
		}
	}
	if part.Unit != "" {
		valueTitle := "Value · " + part.Unit
		if part.Kind == "battery" {
			valueTitle = "Full-charge voltage · V"
		}
		if part.Kind == "thermistor" {
			valueTitle = "Reference resistance at 25 °C · Ω"
		}
		if part.Kind == "ldr" {
			valueTitle = "Reference resistance at 10 lux · Ω"
		}
		if workbench.IsRegulatorKind(part.Kind) {
			valueTitle = "Output setpoint · V"
		}
		if part.Kind == "relay" {
			valueTitle = "Coil inductance · H"
		}
		if part.Kind == "comparator" {
			valueTitle = "Hysteresis width · V"
		}
		if part.Kind == "zener" {
			valueTitle = "Breakdown voltage · V"
		}
		inspector = append(inspector, section("value-title", valueTitle), ui.Node{ID: "value-entry", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "Component value", Help: "Use the shown part bounds and units, such as 2.2k Ω, 500 mV, or 1 µF"}, Change: s.changeValueDraft}, buttonStyle(s.valueApplyAction(state, part.ID, "Apply value"), true))
		if domain, ok := workbench.ValueDomainForKind(part.Kind); ok {
			inspector = append(inspector, quiet("value-domain-range", domain.Range), quiet("value-domain-syntax", domain.Syntax))
			if domain.Relation != "" {
				inspector = append(inspector, quiet("value-domain-relation", domain.Relation))
			}
		}

		if part.Kind == "resistor" {
			inspector = append(inspector, ui.Node{ID: "presets", Kind: ui.Row, Children: []ui.Node{s.valuePresetAction(state, "r-1k", "1 kΩ", workbench.ValueParams{ID: part.ID, Value: "1k"}), s.valuePresetAction(state, "r-2k", "2 kΩ", workbench.ValueParams{ID: part.ID, Value: "2k"})}})
		}
		if part.Kind == "inductor" {
			inspector = append(inspector, s.inductorControls(state, part)...)
		}
		if part.Kind == "capacitor" {
			initial := state.TransientSettings.InitialVoltage
			if c, _, d := graph.Compile(); d == nil && c.Analysis == "RC transient" && c.TerminalNet[part.ID+".1"] != c.Output {
				initial = -initial
			}
			if part.InitialVoltage != nil {
				initial = *part.InitialVoltage
			}
			inspector = append(inspector, quiet("cap-initial-label", "Initial voltage · pin 1 minus pin 2"), ui.Node{ID: "cap-initial", Kind: ui.NumberField, Number: float32(initial), Numeric: ui.Numeric{Min: -12, Max: 12, Step: .01}, Accessibility: ui.Accessibility{Name: "Capacitor initial voltage"}, Change: func(e ui.ChangeEvent) {
				s.changeComponentStartup(state, part, fmt.Sprintf("%.9g V", e.Number))
			}}, s.componentStartupDefault(state, part))
			inspector = append(inspector, ui.Node{ID: "cap-presets", Kind: ui.Row, Children: []ui.Node{s.valuePresetAction(state, "c-1u", "1 µF", workbench.ValueParams{ID: part.ID, Value: "1uF"}), s.valuePresetAction(state, "c-100u", "100 µF", workbench.ValueParams{ID: part.ID, Value: "100uF"})}})
		}

	} else if workbench.IsBipolarKind(part.Kind) {
		inspector = append(inspector, quiet("npn-model", fmt.Sprintf("Generic %s · βF 100 / βR 1\nIS 10 fA · 27°C\nInstantaneous junction response.\nNo heating or switching delay.", strings.ToUpper(part.Kind))))
	} else if part.Kind == "schottky" {
		inspector = append(inspector, quiet("diode-model", "Generic Schottky diode · 27°C\nIS 0.2 µA · n 1.05 · Rs 2 Ω\nLower forward drop; reverse leakage included."))
	} else if part.Kind == "diode" {
		inspector = append(inspector, quiet("diode-model", "Generic silicon diode · 27°C\nIS 1 nA · n 1.7 · Rs 1 Ω\nReverse leakage included. No breakdown, charge storage or ratings."))
	} else if part.Kind == "led" {
		inspector = append(inspector, quiet("led-model", "Generic red LED\nBrightness is illustrative."))
	}
	inspector = append(inspector, s.waveformControls(state, part)...)
	inspector = append(inspector, s.partPlotControls(state, part)...)
	if part.Kind == "opamp" {
		inspector = append(inspector, s.partGeometryControls(state, part)...)
		inspector = append(inspector, quiet("part-model-id", "Model: "+part.Model), quiet("opamp-model", "Gain 100,000 · input 10 MΩ\nOutput 50 Ω scale · active limit 20 mA\nBandwidth and slew are optional below."))
	}
	for _, entry := range workbench.Library() {
		if entry.Kind == part.Kind {
			inspector = append(inspector, quiet("part-assumptions", entry.Limitations))
			break
		}
	}

	if r := state.Mixed; r != nil {
		for _, branches := range [][]workbench.BranchResult{r.LEDs, r.Diodes, r.Resistors, r.Sources} {
			for _, b := range branches {
				if b.ID == part.ID {
					status := "Live"
					if r.Stale {
						status = "Stale"
					}
					inspector = append(inspector, quiet("mixed-selected-reading", fmt.Sprintf("%s · %.6g mA · %.6g V\nPower %.6g mW", status, b.Current*1000, b.Voltage, b.Power*1000)))
				}
			}
		}
	}
	if part.Kind != "opamp" {
		inspector = append(inspector, s.partGeometryControls(state, part)...)
	}
	if explanation := state.Explanation; explanation != nil && explanation.ID == part.ID {
		inspector = append(inspector, section("component-explanation-title", "Understand this component"), quiet("component-explanation-summary", explanation.Summary))
		for i, fact := range explanation.Facts {
			inspector = append(inspector, quiet(fmt.Sprintf("component-fact-%d", i), fact))
		}
		for i, help := range explanation.Guidance {
			inspector = append(inspector, quiet(fmt.Sprintf("component-guidance-%d", i), help.Summary+" "+help.Action))
		}
	}
	for i := range inspector {
		scopeComponentControlState(&inspector[i], part.ID)
		if inspector[i].ID == "cap-initial" || inspector[i].ID == "inductor-initial" {
			inspector[i].ControlStateKey = s.startupControlStateKey(part.ID)
		}
	}
	return inspector
}

func (s *screen) partGeometryControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	rotate := s.action("rotate-part", "Rotate 90°", "component.rotate", workbench.IDParams{ID: part.ID})
	duplicate := s.action("duplicate-part", "Duplicate", "component.duplicate", workbench.DuplicateParams{ID: part.ID})
	if part.Unit != "" || part.Kind == "net-label" {
		owner := workbench.ValueParams{ID: part.ID}
		rotate = s.valueRelatedAction(state, "rotate-part", "Rotate 90°", owner, true, "component.rotate", func(value workbench.ValueParams) any { return workbench.IDParams{ID: value.ID} })
		duplicate = s.valueRelatedAction(state, "duplicate-part", "Duplicate", owner, true, "component.duplicate", func(value workbench.ValueParams) any { return workbench.DuplicateParams{ID: value.ID} })
	}
	return []ui.Node{quiet("part-position", fmt.Sprintf("Position %.0f, %.0f · rotation %d°", part.Position.X, part.Position.Y, part.Rotation)), rotate, duplicate}
}
