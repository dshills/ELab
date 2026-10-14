package main

import (
	"context"
	"fmt"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/ui"
)

type menuChoices struct {
	Kind    ui.Kind
	Options []ui.Option
	Change  func(ui.ChangeEvent)
}

var applicationMenus = []app.MenuGroup{
	{Label: "File", Commands: []string{"project.clear", "file.open", "file.save", "file.save-as", "app.close"}},
	{Label: "Examples", Commands: []string{"example.reset", "example.rc", "example.npn", "example.switch-npn", "example.npn-pulse", "example.flasher"}},
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

// Refuse a stale native popup if an agent or another UI command changed its
// captured state while AppKit was tracking the menu. Live samples do not change
// CommandRevision and can continue normally. Runs on the application owner.
func (s *screen) acceptMenuChoice(node ui.Node, choices menuChoices, index int, serial, revision uint64) {
	if index < 0 || index >= len(choices.Options) || node.Disabled || choices.Change == nil || s.menuSerial != serial || s.app.Context().Err() != nil || s.fileBusy || s.pendingCommand != "" || s.service.Snapshot().CommandRevision != revision {
		return
	}
	option := choices.Options[index]
	if !option.Disabled && !option.Separator {
		previous := s.menuCommandRevision
		s.menuCommandRevision = &revision
		defer func() { s.menuCommandRevision = previous }()
		kind := choices.Kind
		if kind == "" {
			kind = ui.Selector
		}
		choices.Change(ui.ChangeEvent{ID: node.ID, Kind: kind, SelectedID: option.ID, Checked: option.Checkable && !option.Checked})
	}
}
