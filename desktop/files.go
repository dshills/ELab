package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/ui"
)

// AppKit supplies the selected file type's suffix. Supplying that suffix in
// the name as well can produce names such as Untitled.elab.elab. Other native
// pickers still need the complete filename, and command paths are untouched.
func nativeFileOptions(options app.FileOptions, platform string) app.FileOptions {
	if platform != "darwin" || options.Mode != app.SaveFile || len(options.Extensions) != 1 {
		return options
	}
	extension := filepath.Ext(options.Filename)
	if len(extension) > 1 && len(options.Filename) > len(extension) && strings.EqualFold(extension[1:], options.Extensions[0]) {
		options.Filename = strings.TrimSuffix(options.Filename, extension)
	}
	return options
}

func (s *screen) requestReplace(id string, p any) workbench.Response {
	raw, _ := json.Marshal(p)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if fields != nil {
		if _, ok := fields["expected_document_generation"]; !ok {
			fields["expected_document_generation"], _ = json.Marshal(s.service.Snapshot().Document.Generation)
		}
		raw, _ = json.Marshal(fields)
	}
	dispatchID := id
	if id == "recovery.restore" {
		dispatchID = "recovery.restore.start"
	} else if id == "app.close" {
		dispatchID = "app.close.start"
	}
	result := s.invoke(dispatchID, json.RawMessage(raw))
	if result.Error != nil && result.Error.Code == "unsaved-changes" {
		s.setPendingFileAction(id, raw)
		s.errorMessage = ""
	}
	s.app.Invalidate("main")
	return result
}
func (s *screen) beginFile(id string, choose bool, p any, after func()) {
	if s.service.Snapshot().Closed {
		return
	}
	if s.fileBusy {
		return
	}
	window, windowErr := s.windowReference()
	if windowErr != nil {
		s.errorMessage = windowErr.Error()
		return
	}
	s.fileBusy = true
	s.errorMessage = ""
	s.fileMessage = ""
	s.serial++
	serial := s.serial
	state := s.service.Snapshot()
	pendingSerial := s.pendingFileSerial
	var decisionContext *actionContext
	if s.fileDecisionContext != nil {
		copy := *s.fileDecisionContext
		decisionContext = &copy
	}
	raw, _ := json.Marshal(p)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if id == "export.schematic.png" && state.SchematicPage.Tile {
		if _, captured := fields["preview_id"]; !captured {
			if _, explicit := fields["page_index"]; !explicit {
				fields["page_index"] = json.RawMessage("0")
			}
		}
	}
	generation := state.Document.Generation
	// Preserve the original generation through a later unsaved-changes decision.
	if _, present := fields["expected_document_generation"]; !present {
		fields["expected_document_generation"], _ = json.Marshal(generation)
	}
	raw, _ = json.Marshal(fields)
	s.app.Invalidate("main")
	go func() {
		ctx := window.Context()
		var err error
		var restoreFocus func()

		if choose {
			options := app.FileOptions{Mode: app.OpenFile, Title: "Open project", Extensions: []string{"elab"}}
			write := id == "project.save" || strings.HasPrefix(id, "export.") || (id == "verification.export" || id == "sweep.export")
			if write {
				options.Mode = app.SaveFile
				options.Title = "Save project"
				options.Filename = "Untitled.elab"
			}
			switch id {
			case "verification.import", "sweep.import":
				options.Title = "Open recorded run"
				if id == "sweep.import" {
					options.Title = "Open captured frequency or DC sweep"
				}
				options.Extensions = []string{"json"}
			case "export.start":
				var kind string
				_ = json.Unmarshal(fields["kind"], &kind)
				if kind == "audio-wav" {
					options.Title = "Export audio preview WAV"
					options.Extensions = []string{"wav"}
					options.Filename = "Sounder.wav"
				}
			case "export.scope.csv":
				options.Title = "Export scope CSV"
				options.Extensions = []string{"csv"}
				options.Filename = "Waveforms.csv"
			case "export.schematic.svg":
				options.Title = "Export schematic SVG"
				options.Extensions = []string{"svg"}
				options.Filename = "Schematic.svg"
			case "export.schematic.pdf":
				options.Title = "Export printable schematic PDF"
				options.Extensions = []string{"pdf"}
				options.Filename = "Schematic.pdf"
			case "export.schematic.png":
				options.Title = "Export schematic image PNG"
				if state.SchematicPage.Tile {
					options.Title = "Export tiled schematic page PNG"
				}
				options.Extensions = []string{"png"}
				options.Filename = "Schematic.png"
			case "verification.export", "sweep.export":
				var format string
				_ = json.Unmarshal(fields["format"], &format)
				options.Title = "Export recorded verification"
				options.Extensions = []string{format}
				options.Filename = "Verification." + format
				if id == "sweep.export" {
					options.Title = "Export frequency or DC sweep"
					options.Filename = "DCSweep." + format
				}
			}
			if state.Document.Path != "" {
				options.Directory = filepath.Dir(state.Document.Path)
				if id == "project.save" {
					options.Filename = filepath.Base(state.Document.Path)
				}
			}
			options = nativeFileOptions(options, runtime.GOOS)
			var path string
			if s.chooseFile != nil {
				path, err = s.chooseFile(ctx, options)
			} else {
				var services app.Services
				services, err = s.app.Services()
				if err == nil {
					restoreFocus = captureFileDialogFocus()
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
		if err == nil {
			err = ctx.Err()
		}
		var response workbench.Response
		if err == nil {
			revision := s.service.Snapshot().CommandRevision
			request := workbench.Request{Version: 1, ID: fmt.Sprintf("ui-%d", serial), Key: fmt.Sprintf("ui-%d", serial), Command: id, Params: raw, ExpectedCommandRevision: &revision}
			if decisionContext != nil {
				request.ExpectedCommandRevision = &decisionContext.revision
				request.ExpectedDocumentGeneration = &decisionContext.generation
			}
			response = s.service.Dispatch(ctx, request, "user")
		}
		_ = s.app.PostWindow(window, func() {
			if restoreFocus != nil {
				restoreFocus()
			}
			s.fileBusy = false
			if err != nil {
				if !errors.Is(err, app.ErrCanceled) {
					s.errorMessage = err.Error()
				}
			} else if response.Error != nil {
				if (id == "project.load" || id == "project.save" || strings.HasPrefix(id, "export.") || (id == "verification.export" || id == "sweep.export") || id == "verification.import" || id == "sweep.import") && (response.Error.Code == "canceled" || response.Error.Code == "superseded") {
					s.fileMessage = "File operation " + response.Error.Code + "; current project preserved"
				} else if response.Error.Code == "unsaved-changes" {
					s.setPendingFileAction(id, raw)
				} else {
					s.errorMessage = response.Error.Message
				}
			} else {
				state := response.Result.(workbench.Snapshot)
				path := state.Document.Path
				continueAction := true
				saveAssociated := true
				if id == "export.start" && state.ProjectWork != nil {
					s.fileMessage = "Exporting " + filepath.Base(state.ProjectWork.Path)
					continueAction = false
				} else if response.Result.(workbench.Snapshot).LastExport != nil && (strings.HasPrefix(id, "export.") || (id == "verification.export" || id == "sweep.export")) {
					s.fileMessage = "Exported " + filepath.Base(response.Result.(workbench.Snapshot).LastExport.Path)
				} else if id == "verification.import" || id == "sweep.import" {
					var opened string
					_ = json.Unmarshal(fields["path"], &opened)
					s.fileMessage = "Loaded recorded " + filepath.Base(opened)
					if id == "sweep.import" {
						s.fileMessage = "Loaded captured sweep " + filepath.Base(opened)
					}
					if state.ProjectWork != nil && state.ProjectWork.ViewerSelected != nil && !*state.ProjectWork.ViewerSelected {
						s.fileMessage += "; kept without switching to its view"
					}
				} else if id == "project.save" {
					if state.ProjectWork != nil {
						path = state.ProjectWork.Path
						current := s.service.Snapshot()
						sameGeneration := current.Document.Generation == state.ProjectWork.ResultGeneration
						saveAssociated = state.ProjectWork.Associated != nil && *state.ProjectWork.Associated && sameGeneration
						continueAction = saveAssociated && !current.Document.Dirty
						state.Document = current.Document
						if !sameGeneration && after != nil && s.pendingFileSerial == pendingSerial {
							s.setPendingFileAction("", nil)
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
	serial := s.pendingFileSerial
	if id == "" {
		return
	}
	if discard {
		var p map[string]any
		_ = json.Unmarshal(raw, &p)
		p["discard_changes"] = true
		raw, _ = json.Marshal(p)
	}
	clear := func() {
		if s.pendingFileSerial == serial {
			s.setPendingFileAction("", nil)
		}
	}
	if id == "project.load" {
		s.beginFile(id, false, raw, clear)
	} else if result := s.requestReplace(id, raw); result.Error == nil {
		clear()
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
			if s.service.Snapshot().Closed {
				return fmt.Errorf("session is closing")
			}
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
	decision := s.captureFileDecision()
	dismiss := func() {
		s.withFileDecision(decision, false, func() {
			s.setPendingFileAction("", nil)
			s.errorMessage = ""
			s.requestHelp = nil
			s.app.Invalidate("main")
		})
	}
	// Keep the intent while work runs, but expose its progress/cancel controls.
	prompt := ui.Node{ID: "unsaved-prompt", Kind: ui.Dialog, Open: s.pendingCommand != "" && !s.fileBusy, Text: "Unsaved project", Dismiss: dismiss, Children: []ui.Node{heading("unsaved-title", "Keep your changes?"), label("unsaved-description", "This project has unsaved schematic or recorded-result changes. Save it before continuing, or discard the changes.")}}
	if s.errorMessage != "" {
		prompt.Children = append(prompt.Children, label("unsaved-error", s.errorMessage))
	}
	prompt.Children = append(prompt.Children,
		buttonStyle(ui.Node{ID: "save-continue", Kind: ui.Button, Text: "Save and continue", Disabled: s.fileBusy, Activate: func() {
			s.withFileDecision(decision, true, func() {
				s.beginFile("project.save", s.service.Snapshot().Document.Path == "", workbench.SaveParams{ExpectedGeneration: &decision.generation}, func() {
					if s.pendingFileSerial == decision.serial {
						s.continuePending(false)
					}
				})
			})
		}}, true),
		buttonStyle(ui.Node{ID: "discard-continue", Kind: ui.Button, Text: "Discard changes", Disabled: s.fileBusy, Activate: func() { s.withFileDecision(decision, true, func() { s.continuePending(true) }) }}, false),
		buttonStyle(ui.Node{ID: "cancel-file-action", Kind: ui.Button, Text: "Cancel", Disabled: s.fileBusy, Activate: dismiss}, false))
	return prompt
}
