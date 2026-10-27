package workbench

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Parts descriptions share the SVG/PNG/PDF path, including printing. These
// describe saved intent rather than a momentary live result.
func schematicPartDescription(p Component) string {
	text := p.ID + " · " + p.Name
	if IsCurrentDevice(p.Kind) {
		text += " · " + currentDeviceName(p) + " · " + CurrentDeviceModelID
	}
	if p.Kind == "net-label" {
		text += " · " + p.Label
	}
	if p.Unit != "" {
		text += " · " + displayPartValue(p)
	}
	if hasPassiveLoss(p) {
		c := p.Device.Passive
		if p.Kind == "capacitor" {
			text += fmt.Sprintf(" · ESR %.6g Ω", c.ESR)
			if c.Leakage == 0 {
				text += " · leakage off"
			} else {
				text += fmt.Sprintf(" · leakage %.6g Ω", c.Leakage)
			}
		} else {
			text += fmt.Sprintf(" · winding %.6g Ω", c.Winding)
		}
		text += " · " + c.Version
	}
	if p.Kind == "speaker" && p.Device != nil && p.Device.Speaker != nil {
		c := p.Device.Speaker
		text += fmt.Sprintf(" · coil R %.6g Ω · L %.6g H · fs %.6g Hz · motional R %.6g Ω · L %.6g H · %s", c.Resistance, c.Inductance, c.Resonance, c.MotionalResistance, c.MotionalInductance, c.Version)
	}
	if hasTransistorProfile(p) {
		c := p.Device.Bipolar
		text += fmt.Sprintf(" · β %.6g/%.6g · IS %.6g A · VA %.6g V · CBE %.6g F · CBC %.6g F · %s", c.ForwardBeta, c.ReverseBeta, c.SaturationCurrent, c.EarlyVoltage, c.CBE, c.CBC, c.Version)
	}
	if p.Kind == "transformer" {
		c := p.Power.Transformer
		polarity := "same dots"
		if c.Reversed {
			polarity = "reversed secondary"
		}
		text += fmt.Sprintf(" · Ns/Np %.6g · Lp %.6g H · k %.6g · R %.6g/%.6g Ω · %s", c.Ratio, c.Inductance, c.Coupling, c.PrimaryResistance, c.SecondaryResistance, polarity)
	}
	if p.Kind == "optocoupler" {
		text += fmt.Sprintf(" · CTR %.6g A/A", p.Power.Optocoupler.CTR)
	}
	if p.Kind == "battery" {
		text += fmt.Sprintf(" · startup %.6g%% · %.6g mAh · ESR %.6g Ω", p.Power.Battery.Charge*100, p.Power.Battery.CapacityAh*1000, p.Power.Battery.Resistance)
	}

	if p.Kind == "dc-motor" {
		m := p.Power.Motor
		rotor := "free"
		if m.Locked {
			rotor = "locked"
		}
		text += fmt.Sprintf(" · R %.6g Ω · L %.6g H · Ke=Kt %.6g (V·s/rad, N·m/A) · J %.6g kg·m² · friction %.6g N·m·s/rad · load %.6g N·m·s/rad · rotor %s", m.Resistance, m.Inductance, m.Constant, m.Inertia, m.Friction, m.Load, rotor)
	}
	if IsSensorKind(p.Kind) {
		unit := "lux"
		if p.Kind == "thermistor" {
			unit = "°C"
		}
		text += fmt.Sprintf(" · %.6g %s", sensorEnvironment(p), unit)
	}
	if p.Device != nil {
		if p.Kind == "pwm-controller" && p.Device.PWM != nil {
			c := p.Device.PWM
			text += fmt.Sprintf(" · PWM %.6g Hz · duty %.6g%%", c.Frequency, c.Duty*100)
		}
		if hasDynamics(p) {
			c := p.Device.Dynamics
			text += fmt.Sprintf(" · GBW %.6g Hz · slew %.6g V/s", c.GBW, c.Slew)
		}
		if c := p.Device.Sound; c != nil {
			// Each kind retains both settings, but only one affects its model.
			if p.Kind == "piezo" {
				text += fmt.Sprintf(" · piezo C %.6g F", c.Capacitance)
			} else if p.Kind == "active-buzzer" {
				text += fmt.Sprintf(" · tone %.6g Hz", c.Frequency)
			}
		}
	}
	if p.InitialVoltage != nil {
		text += fmt.Sprintf(" · initial %.4g V", *p.InitialVoltage)
	}
	if p.InitialCurrent != nil {
		text += fmt.Sprintf(" · initial %.4g A", *p.InitialCurrent)
	}
	return text
}

type schematicPartLine struct {
	x, y float64
	text string
}

func schematicPartsList(parts []Component, width float64) ([]schematicPartLine, float64) {
	columns := 2
	if width < 720 {
		columns = 1
	}
	columnWidth := width / float64(columns)
	// A one-em advance is conservative for the 12-point sans-serif font and
	// keeps unit glyphs and long IDs inside their own column.
	capacity := max(1, int((columnWidth-30)/12))
	baseline := 44.
	var lines []schematicPartLine
	for first := 0; first < len(parts); first += columns {
		tallest := 1
		for column := 0; column < columns && first+column < len(parts); column++ {
			description := wrapSchematicPartDescription(schematicPartDescription(parts[first+column]), capacity)
			tallest = max(tallest, len(description))
			for i, text := range description {
				lines = append(lines, schematicPartLine{15 + float64(column)*columnWidth, baseline + 16*float64(i), text})
			}
		}
		baseline += 16*float64(tallest) + 8
	}
	return lines, baseline + 11
}

func wrapSchematicPartDescription(text string, capacity int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		runes := []rune(word)
		for len(runes) > capacity {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			lines = append(lines, string(runes[:capacity]))
			runes = runes[capacity:]
		}
		word = string(runes)
		if word == "" {
			continue
		}
		if line == "" {
			line = word
			continue
		}
		if utf8.RuneCountInString(line)+1+len(runes) <= capacity {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
