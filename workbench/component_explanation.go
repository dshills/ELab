package workbench

import "fmt"

// ComponentExplanation separates structural facts and accepted readings from
// educational model advice. It never guesses a failure's physical cause.
type ComponentExplanation struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	ReadingState string               `json:"reading_state"`
	Summary      string               `json:"summary"`
	Facts        []string             `json:"facts"`
	Readings     []MeasurementReading `json:"readings"`
	Guidance     []DiagnosticHelp     `json:"guidance"`
	Model        string               `json:"model"`
	Limitations  string               `json:"limitations"`
}

func cloneComponentExplanation(in *ComponentExplanation) *ComponentExplanation {
	if in == nil {
		return nil
	}
	out := *in
	out.Facts = append([]string{}, in.Facts...)
	out.Readings = cloneMeasurementReadings(in.Readings)
	out.Guidance = append([]DiagnosticHelp{}, in.Guidance...)
	for i := range out.Guidance {
		out.Guidance[i] = *cloneDiagnosticHelp(&in.Guidance[i])
	}
	return &out
}

func (s *Service) componentExplanationLocked(c Compiled, id string, inspection *MeasurementInspection) (*ComponentExplanation, error) {
	part, ok := s.graph.Component(id)
	if !ok {
		return nil, &actionError{"not-found", "Choose a component ID from project.inspect"}
	}
	if inspection == nil {
		value := s.measurementInspectionLocked(c, &MeasurementTarget{Kind: "object", ID: id})
		inspection = &value
	}
	out := &ComponentExplanation{ID: id, Name: part.Name, ReadingState: "unavailable", Facts: []string{}, Readings: cloneMeasurementReadings(inspection.Readings), Guidance: []DiagnosticHelp{}, Model: part.Model, Limitations: partLimitations(part.Kind)}
	for _, reading := range out.Readings {
		if reading.State == "stale" {
			out.ReadingState = "stale"
			break
		}
		if reading.State == "live" {
			out.ReadingState = "live"
		}
	}
	switch out.ReadingState {
	case "live":
		out.Summary = "Current accepted calculation for " + id + "."
		if s.paused {
			out.Summary = "Paused at the current accepted calculation for " + id + "."
		}
	case "stale":
		out.Summary = "Previous results for " + id + "; they do not describe the edited circuit."
	default:
		out.Summary = "No accepted electrical reading for " + id + "."
	}
	if part.Unit != "" {
		out.Facts = append(out.Facts, "Configured value: "+FormatValue(part.Value, part.Unit)+".")
	}
	for _, pin := range part.Terminals {
		ref := id + "." + pin
		if c.TerminalNet[ref] == "" {
			continue // A failed compile is not proof that a terminal is open.
		}
		peers := 0
		for _, net := range c.Nets {
			if net.ID == c.TerminalNet[ref] {
				peers = len(net.Terminals) - 1
				break
			}
		}
		if peers == 0 {
			out.Facts = append(out.Facts, ref+" has no other terminal on its net.")
		}
	}
	for _, help := range s.diagnosticPresentationLocked(c).Items {
		related := false
		for _, object := range help.Objects {
			related = related || object == id
		}
		for _, ref := range help.Terminals {
			owner, _, exists := s.graph.Terminal(ref)
			related = related || exists && owner.ID == id
		}
		if related {
			out.Guidance = append(out.Guidance, help)
		}
	}
	switch part.Kind {
	case "current-source", "electronic-load":
		out.Model, out.Limitations = CurrentDeviceModelID, currentDeviceDescription
		out.Facts = append(out.Facts, currentDeviceName(part))
	case "resistor":
		out.Facts = append(out.Facts, "In this ideal model, voltage across the resistor equals current times resistance (V = I × R).")
	case "capacitor":
		if hasPassiveLoss(part) {
			out.Model, out.Limitations = PassiveLossModelID, passiveLossDescription
			out.Facts = append(out.Facts, "Charging current changes internal stored voltage (Icharge = C × dVstored/dt). ESR adds Icharge × ESR to terminal voltage; external leakage adds to terminal current.")
		} else {
			out.Facts = append(out.Facts, "Positive current into pin 1 increases V(1) − V(2); capacitance sets how quickly that voltage changes (I = C × dV/dt).")
		}
	case "inductor":
		if hasPassiveLoss(part) {
			out.Model, out.Limitations = PassiveLossModelID, passiveLossDescription
			out.Facts = append(out.Facts, "Terminal voltage includes winding resistance: V = L × dI/dt + Rw × I. Winding heat is I² × Rw; magnetic energy remains L × I² / 2.")
		}
	case "led":
		out.Facts = append(out.Facts, "Forward current flows from anode to cathode. The brightness cue is illustrative, not a physical light-output prediction.")
		for _, reading := range out.Readings {
			if reading.Probe.ID == "i:"+id && reading.State == "live" && reading.Value != nil {
				if *reading.Value <= 0 {
					out.Facts = append(out.Facts, "The accepted current is not forward; the LED model is not conducting light-producing forward current at this sample.")
				} else {
					out.Facts = append(out.Facts, fmt.Sprintf("The model is conducting %.5g mA from anode to cathode at the accepted sample.", *reading.Value*1000))
				}
			}
		}
	case "npn", "pnp":
		if out.ReadingState == "live" {
			state := ""
			if s.npn != nil && !s.npn.Stale && s.npn.TransistorID == id {
				state = s.npn.State
			}
			if s.mixed != nil && !s.mixed.Stale {
				for _, transistor := range s.mixed.Transistors {
					if transistor.TransistorID == id {
						state = transistor.State
					}
				}
			}
			if state != "" {
				out.Facts = append(out.Facts, fmt.Sprintf("Accepted transistor operating region: %s.", state))
			}
		}
	}
	return out, nil
}
