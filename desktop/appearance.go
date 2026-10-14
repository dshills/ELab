package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func setupAppearance(service *workbench.Service, path string) error {
	if path == "" {
		var err error
		path, err = workbench.DefaultAppearanceFile()
		if err != nil {
			return err
		}
	}
	raw, _ := json.Marshal(workbench.AppearanceConfigureParams{Path: path})
	revision := service.Snapshot().CommandRevision
	response := service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "startup-appearance", Command: "appearance.configure", Params: raw, ExpectedCommandRevision: &revision, Key: "startup-appearance"}, "user")
	if response.Error != nil {
		return fmt.Errorf("%s", response.Error.Message)
	}
	return nil
}
func (s *screen) appearanceControls(state workbench.Snapshot) []ui.Node {
	a := state.Appearance
	if a.Status == "disabled" {
		path := a.Path
		if path == "" {
			path, _ = workbench.DefaultAppearanceFile()
		}
		nodes := []ui.Node{section("appearance-title", "Appearance"), quiet("appearance-status", "Theme applies to this session only")}
		if path != "" {
			nodes = append(nodes, s.action("appearance-enable", "Use saved theme on this device", "appearance.configure", workbench.AppearanceConfigureParams{Path: path}))
		}
		return nodes
	}
	text := "Theme saved on this device"
	switch a.Status {
	case "loading":
		text = "Loading theme preference…"
	case "saving":
		text = "Saving theme preference…"
	case "error":
		text = "Theme preference could not be saved · " + a.Error
	}
	nodes := []ui.Node{section("appearance-title", "Appearance"), quiet("appearance-status", text)}
	if a.Status == "error" {
		nodes = append(nodes, s.action("appearance-retry", "Reload saved theme", "appearance.configure", workbench.AppearanceConfigureParams{Path: a.Path}))
	}
	nodes = append(nodes, s.action("appearance-session-only", "Use theme for this session only", "appearance.configure", workbench.AppearanceConfigureParams{}))
	return nodes
}
