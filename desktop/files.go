package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/ui"
)

func (s *screen) requestReplace(id string, p any) {
	raw, _ := json.Marshal(p)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if fields != nil {
		if _, ok := fields["expected_document_generation"]; !ok {
			fields["expected_document_generation"], _ = json.Marshal(s.service.Snapshot().Document.Generation)
		}
		raw, _ = json.Marshal(fields)
	}
	result := s.invoke(id, json.RawMessage(raw))
	if result.Error != nil && result.Error.Code == "unsaved-changes" {
		s.pendingCommand = id
		s.pendingParams = append(json.RawMessage{}, raw...)
		s.errorMessage = ""
	}
	s.app.Invalidate("main")
}
func (s *screen) beginFile(id string, choose bool, p any, after func()) {
	if s.fileBusy {
		return
	}
	s.fileBusy = true
	s.errorMessage = ""
	s.fileMessage = ""
	s.serial++
	serial := s.serial
	state := s.service.Snapshot()
	raw, _ := json.Marshal(p)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	generation := state.Document.Generation
	// Preserve the original generation through a later unsaved-changes decision.
	if _, present := fields["expected_document_generation"]; !present {
		fields["expected_document_generation"], _ = json.Marshal(generation)
	}
	raw, _ = json.Marshal(fields)
	s.app.Invalidate("main")
	go func() {
		ctx := s.app.Context()
		var err error

		if choose {
			options := app.FileOptions{Mode: app.OpenFile, Title: "Open project", Extensions: []string{"elab"}}
			write := id == "project.save" || strings.HasPrefix(id, "export.") || id == "verification.export"
			if write {
				options.Mode = app.SaveFile
				options.Title = "Save project"
				options.Filename = "Untitled.elab"
			}
			switch id {
			case "verification.import":
				options.Title = "Open recorded run"
				options.Extensions = []string{"json"}
			case "export.scope.csv":
				options.Title = "Export scope CSV"
				options.Extensions = []string{"csv"}
				options.Filename = "Waveforms.csv"
			case "export.schematic.svg":
				options.Title = "Export schematic SVG"
				options.Extensions = []string{"svg"}
				options.Filename = "Schematic.svg"
			case "verification.export":
				var format string
				_ = json.Unmarshal(fields["format"], &format)
				options.Title = "Export recorded verification"
				options.Extensions = []string{format}
				options.Filename = "Verification." + format
			}
			if state.Document.Path != "" {
				options.Directory = filepath.Dir(state.Document.Path)
				if id == "project.save" {
					options.Filename = filepath.Base(state.Document.Path)
				}
			}
			var path string
			if s.chooseFile != nil {
				path, err = s.chooseFile(ctx, options)
			} else {
				var services app.Services
				services, err = s.app.Services()
				if err == nil {
					path, err = services.ChooseFile(ctx, options)
				}
			}
			if err == nil {
				// Preserve command-specific parameters and the original document generation.
				fields["path"], _ = json.Marshal(path)
				if write {
					fields["overwrite"] = json.RawMessage("true")
				}
				raw, err = json.Marshal(fields)
			}
		}
		var response workbench.Response
		if err == nil {
			revision := s.service.Snapshot().CommandRevision
			request := workbench.Request{Version: 1, ID: fmt.Sprintf("ui-%d", serial), Key: fmt.Sprintf("ui-%d", serial), Command: id, Params: raw, ExpectedCommandRevision: &revision}
			response = s.service.Dispatch(ctx, request, "user")
		}
		_ = s.app.PostWindow("main", func() {
			s.fileBusy = false
			if err != nil {
				if !errors.Is(err, app.ErrCanceled) {
					s.errorMessage = err.Error()
				}
			} else if response.Error != nil {
				if (id == "project.load" || id == "project.save") && (response.Error.Code == "canceled" || response.Error.Code == "superseded") {
					s.fileMessage = "Project " + response.Error.Code + "; current project preserved"
				} else if response.Error.Code == "unsaved-changes" {
					s.pendingCommand = id
					s.pendingParams = append(json.RawMessage{}, raw...)
				} else {
					s.errorMessage = response.Error.Message
				}
			} else {
				state := response.Result.(workbench.Snapshot)
				path := state.Document.Path
				continueAction := true
				saveAssociated := true
				if response.Result.(workbench.Snapshot).LastExport != nil && (strings.HasPrefix(id, "export.") || id == "verification.export") {
					s.fileMessage = "Exported " + filepath.Base(response.Result.(workbench.Snapshot).LastExport.Path)
				} else if id == "verification.import" {
					var opened string
					_ = json.Unmarshal(fields["path"], &opened)
					s.fileMessage = "Loaded recorded " + filepath.Base(opened)
				} else if id == "project.save" {
					if state.ProjectWork != nil {
						path = state.ProjectWork.Path
						current := s.service.Snapshot()
						sameGeneration := current.Document.Generation == state.ProjectWork.ResultGeneration
						saveAssociated = state.ProjectWork.Associated != nil && *state.ProjectWork.Associated && sameGeneration
						continueAction = saveAssociated && !current.Document.Dirty
						state.Document = current.Document
						if !sameGeneration && after != nil {
							s.pendingCommand = ""
							s.pendingParams = nil
						}
					}
					s.fileMessage = "Saved " + filepath.Base(path)
					if !saveAssociated {
						s.fileMessage += "; current project preserved"
					} else if state.Document.Dirty {
						s.fileMessage += "; later edits remain unsaved"
					}
				} else {
					s.fileMessage = "Opened " + filepath.Base(path)
				}
				if after != nil && continueAction {
					after()
				}
			}
			s.app.Invalidate("main")
		})
	}()
}
func (s *screen) continuePending(discard bool) {
	id, raw := s.pendingCommand, append(json.RawMessage{}, s.pendingParams...)
	s.pendingCommand = ""
	s.pendingParams = nil
	if discard {
		var p map[string]any
		_ = json.Unmarshal(raw, &p)
		p["discard_changes"] = true
		raw, _ = json.Marshal(p)
	}
	if id == "project.load" {
		s.beginFile(id, false, raw, nil)
	} else {
		s.requestReplace(id, raw)
	}
}
func (s *screen) registerFileCommands() {
	for _, entry := range []struct {
		id, text, shortcut string
		action             func()
	}{
		{"file.open", "Open…", "Meta+O", func() { s.beginFile("project.load", true, workbench.LoadParams{}, nil) }},
		{"file.save", "Save", "Meta+S", func() {
			s.beginFile("project.save", s.service.Snapshot().Document.Path == "", workbench.SaveParams{}, nil)
		}},
		{"file.save-as", "Save As…", "Meta+Shift+S", func() { s.beginFile("project.save", true, workbench.SaveParams{}, nil) }},
	} {
		entry := entry
		if err := s.commands.Register(command.Command{ID: entry.id, Label: entry.text, Shortcut: entry.shortcut, ShortcutScope: command.AfterFocusShortcuts, Available: func() error {
			if s.fileBusy || s.pendingCommand != "" {
				return fmt.Errorf("finish the current file operation first")
			}
			return nil
		}, Execute: func(context.Context) error { entry.action(); return nil }}); err != nil {
			panic(err)
		}
	}
}
func (s *screen) filePrompt() ui.Node {
	prompt := ui.Node{ID: "unsaved-prompt", Kind: ui.Dialog, Open: s.pendingCommand != "", Text: "Unsaved project", Dismiss: func() {
		if s.fileBusy {
			return
		}
		s.pendingCommand = ""
		s.pendingParams = nil
		s.app.Invalidate("main")
	}, Children: []ui.Node{heading("unsaved-title", "Keep your changes?"), label("unsaved-description", "This project has unsaved schematic or recorded-result changes. Save it before continuing, or discard the changes.")}}
	if s.errorMessage != "" {
		prompt.Children = append(prompt.Children, label("unsaved-error", s.errorMessage))
	}
	prompt.Children = append(prompt.Children,
		buttonStyle(ui.Node{ID: "save-continue", Kind: ui.Button, Text: "Save and continue", Disabled: s.fileBusy, Activate: func() {
			s.beginFile("project.save", s.service.Snapshot().Document.Path == "", workbench.SaveParams{}, func() { s.continuePending(false) })
		}}, true),
		buttonStyle(ui.Node{ID: "discard-continue", Kind: ui.Button, Text: "Discard changes", Disabled: s.fileBusy, Activate: func() { s.continuePending(true) }}, false),
		buttonStyle(ui.Node{ID: "cancel-file-action", Kind: ui.Button, Text: "Cancel", Disabled: s.fileBusy, Activate: prompt.Dismiss}, false))
	return prompt
}
