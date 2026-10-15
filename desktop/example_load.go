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
		{"example.led-fade", "Load capacitor-buffered LED fade", "led-fade"},
		{"example.nmos", "Load MOSFET LED switch", "nmos"},
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
