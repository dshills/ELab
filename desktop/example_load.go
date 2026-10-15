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
	if err := s.commands.Register(command.Command{ID: "example.led-fade", Label: "Load capacitor-buffered LED fade", Available: func() error {
		if s.fileBusy || s.pendingCommand != "" || s.service.Snapshot().Closed {
			return fmt.Errorf("finish the current operation before loading an example")
		}
		return nil
	}, Execute: func(context.Context) error {
		s.requestReplace("example.load", workbench.ExampleLoadParams{ID: "led-fade"})
		return nil
	}}); err != nil {
		panic(err)
	}
}
