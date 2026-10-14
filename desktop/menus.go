package main

import (
	"context"
	"fmt"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/ui"
)

type menuChoices struct {
	Options []ui.Option
	Change  func(ui.ChangeEvent)
}

var applicationMenus = []app.MenuGroup{
	{Label: "File", Commands: []string{"project.clear", "file.open", "file.save", "file.save-as"}},
	{Label: "Examples", Commands: []string{"example.reset", "example.rc", "example.npn", "example.npn-pulse", "example.flasher"}},
	{Label: "Appearance", Commands: []string{"appearance.light", "appearance.dark", "appearance.system"}},
}

func (s *screen) registerAppearanceCommands() {
	for _, mode := range []string{"light", "dark", "system"} {
		mode := mode
		if err := s.commands.Register(command.Command{ID: "appearance." + mode, Label: map[string]string{"light": "Light", "dark": "Dark", "system": "System"}[mode], Execute: func(context.Context) error {
			result := s.invoke("appearance.theme.set", map[string]string{"theme": mode})
			if result.Error != nil {
				return fmt.Errorf("%s", result.Error.Message)
			}
			return nil
		}}); err != nil {
			panic(err)
		}
	}
}
