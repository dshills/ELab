package workbench

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDeviceExportRetainsActiveSavedParameters(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		device *DeviceParameters
		want   []string
		absent string
	}{
		{"pwm-controller", &DeviceParameters{PWM: &PWMConfig{Frequency: 123, Duty: .37}}, []string{"PWM 123 Hz", "duty 37%"}, ""},
		{"opamp", &DeviceParameters{Dynamics: &OpAmpDynamics{GBW: 3456, Slew: 7890}}, []string{"GBW 3456 Hz", "slew 7890 V/s"}, ""},
		{"piezo", &DeviceParameters{Sound: &SoundConfig{Capacitance: 47e-9, Frequency: 2300}}, []string{"piezo C 4.7e-08 F"}, "2300 Hz"},
		{"active-buzzer", &DeviceParameters{Sound: &SoundConfig{Capacitance: 47e-9, Frequency: 2300}}, []string{"tone 2300 Hz"}, "4.7e-08 F"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			p, err := MakeComponent("U1", tc.kind, Point{}, "")
			if err != nil {
				t.Fatal(err)
			}
			p.Device = tc.device
			g := Graph{Components: []Component{p}}
			bounds, lines := exportedParts(t, schematicSVG(g, 1, 1))
			var text []string
			for _, line := range lines {
				text = append(text, line.text)
				if line.x+12*float64(utf8.RuneCountInString(line.text)) > bounds[0]+bounds[2]-15 || line.y+4 > bounds[1]+bounds[3] {
					t.Fatal("configured parts text escapes the narrow page", line, bounds)
				}
			}
			joined := strings.Join(text, " ")
			for _, want := range tc.want {
				if !strings.Contains(joined, want) {
					t.Errorf("printed %s loses active saved parameter %q: %s", tc.kind, want, joined)
				}
			}
			if tc.absent != "" && strings.Contains(joined, tc.absent) {
				t.Fatal("unused configuration presented as an active model parameter", joined)
			}
		})
	}
}
