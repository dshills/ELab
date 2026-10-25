package main

import (
	"context"
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/command"
)

// Native menu IDs can name templates without adding a public command per
// template. Agents discover the same ID through examples.list / example.load.
func (s *screen) registerExampleLoadCommands() {
	for _, template := range []struct{ command, label, id string }{
		{"example.opamp-follower", "Load op-amp voltage follower", "opamp-follower"},
		{"example.opamp-inverting", "Load inverting op-amp amplifier", "opamp-inverting"},
		{"example.rc-high-pass", "Load RC high-pass edge detector", "rc-high-pass"},
		{"example.rl", "Load coil current buildup and decay", "rl"},
		{"example.rlc", "Load RLC ringing and damping", "rlc"},
		{"example.flyback", "Load coil switching and flyback diode", "flyback"},
		{"example.led-fade", "Load capacitor-buffered LED fade", "led-fade"},
		{"example.nmos", "Load MOSFET LED switch", "nmos"},
		{"example.pmos", "Load high-side MOSFET LED switch", "pmos"},
		{"example.pnp", "Load PNP high-side LED switch", "pnp"},
		{"example.emitter-follower", "Load NPN emitter follower", "emitter-follower"},
		{"example.bipolar-push-pull", "Load complementary transistor buffer", "bipolar-push-pull"},
		{"example.cmos", "Load complementary MOSFET inverter", "cmos"},
		{"example.coupled-windings", "Load transformer ratio and polarity", "coupled-windings"}, {"example.transformer-supply", "Load transformer DC supply", "transformer-supply"},
		{"example.opto-switch", "Load optically isolated switch", "opto-switch"},
		{"example.ac-rc", "Load RC Bode filter", "ac-rc"}, {"example.ac-rlc", "Load RLC Bode filter", "ac-rlc"}, {"example.ac-opamp", "Load op-amp Bode filter", "ac-opamp"},
		{"example.hbridge-motor", "Load reversible H-bridge motor", "hbridge-motor"},
		{"example.motor-dc", "Load DC motor", "motor-dc"}, {"example.motor-pwm", "Load PWM motor", "motor-pwm"}, {"example.battery-led", "Load battery LED", "battery-led"}, {"example.battery-regulator", "Load battery regulator", "battery-regulator"}, {"example.night-light", "Load automatic night-light", "night-light"},
		{"example.thermostat", "Load thermistor relay thermostat", "thermostat"},
		{"example.regulator-supply", "Load regulated supply and load", "regulator-supply"},
		{"example.regulator-dropout", "Load linear versus LDO dropout", "regulator-dropout"},
		{"example.relay-switch", "Load pushbutton transistor relay", "relay-switch"},
		{"example.relay-flasher", "Load relay coil and flyback protection", "relay-flasher"},
		{"example.comparator-threshold", "Load comparator threshold detector", "comparator-threshold"},
		{"example.comparator-counter", "Load analog signal to digit counter", "comparator-counter"},
		{"example.button-counter", "Load pushbutton decimal counter", "button-counter"},
		{"example.button-debounce", "Load contact bounce and RC debounce", "button-debounce"},
		{"example.display-counter", "Load 555-driven digit counter", "display-counter"},
		{"example.d-flipflop", "Load D flip-flop sampler", "d-flipflop"},
		{"example.jk-flipflop", "Load JK flip-flop divider", "jk-flipflop"},
		{"example.t-flipflop", "Load T flip-flop toggle", "t-flipflop"},
		{"example.binary-counter", "Load 4-bit binary counter", "binary-counter"},
		{"example.decade-counter", "Load Decade counter", "decade-counter"},
		{"example.logic-inverter", "Load Logic inverter LED", "logic-inverter"},
		{"example.logic-switches", "Load AND gate with switches", "logic-switches"},
		{"example.logic-half-adder", "Load Logic half adder", "logic-half-adder"},
		{"example.555-flasher", "Load 555 LED flasher", "555-flasher"},
		{"example.555-oscillator", "Load adjustable 555 oscillator", "555-oscillator"},
		{"example.555-one-shot", "Load 555 one-shot timer", "555-one-shot"},
		{"example.zener", "Load Zener voltage regulator", "zener"},
		{"example.schottky", "Load Schottky low-voltage rectifier", "schottky"},
		{"example.split-supply", "Load split-rail DC supply", "split-supply"},
	} {
		if err := s.commands.Register(command.Command{ID: template.command, Label: template.label, Available: func() error {
			if s.fileBusy || s.pendingCommand != "" || s.service.Snapshot().Closed {
				return fmt.Errorf("finish the current operation before loading an example")
			}
			return nil
		}, Execute: func(context.Context) error {
			s.requestReplace("example.load", workbench.ExampleLoadParams{ID: template.id})
			return nil
		}}); err != nil {
			panic(err)
		}
	}
}
