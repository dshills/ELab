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
