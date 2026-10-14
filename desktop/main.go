package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"time"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func main() {
	endpoint := flag.String("agent-endpoint", "", "enable local agent access and write a private descriptor to this new file")
	recoveryDirectory := flag.String("recovery-directory", "", "private autosave directory (default: user configuration directory)")
	noRecovery := flag.Bool("no-recovery", false, "skip automatic recovery setup")
	smoke := flag.Bool("smoke", false, "verify native drawing and UI command results, then close")
	flag.Parse()
	service := workbench.New()
	defer service.Close()
	if !*noRecovery {
		directory := *recoveryDirectory
		if *smoke && directory == "" {
			var err error
			directory, err = os.MkdirTemp("", "elab-smoke-recovery-")
			if err != nil {
				log.Fatal(err)
			}
			defer func() { service.Close(); _ = os.RemoveAll(directory) }()
		}
		if err := setupRecovery(service, directory); err != nil {
			log.Print("Recovery unavailable: ", err)
		}
	}
	a, err := app.New(app.Config{})
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	screen := &screen{app: a, service: service, commands: command.New(), displayStarted: time.Now()}
	view := canvas.Viewport{Width: 800, Height: 500, Zoom: 1}
	renderer := &cogent.MeshCanvas{Owner: a, Window: "main", View: &view, Prepare: func(v canvas.Viewport) (*canvas.Mesh, error) { return screen.mesh(v) }, OnError: func(err error) { log.Print("schematic: ", err) }}
	screen.drawing = &commandCanvas{MeshCanvas: renderer, screen: screen}
	scopeView := canvas.Viewport{Width: 800, Height: 120, Zoom: 1}
	screen.scope = &scopeCanvas{MeshCanvas: &cogent.MeshCanvas{Owner: a, Window: "main", View: &scopeView, Prepare: screen.scopeMesh, OnError: func(err error) { log.Print("scope: ", err) }}, screen: screen}
	screen.drawing.pointer = newPointerBridge(a, screen.drawing.routePointer)
	defer screen.drawing.pointer.close()
	screen.registerCommands()
	err = a.Open(app.Window{ID: "main", Title: "ELab · Live electronics", Width: 1260, Height: 900, View: screen.view, CloseRequest: func() bool { screen.requestReplace("app.close", workbench.ReplaceParams{}); return false }})
	if err != nil {
		log.Fatal(err)
	}
	if *endpoint != "" {
		server, err := transport.Listen(service, *endpoint)
		if err != nil {
			log.Fatal(err)
		}
		defer server.Close()
		log.Printf("Local agent endpoint descriptor: %s", *endpoint)
	}
	go func() {
		ticker := time.NewTicker(33 * time.Millisecond)
		defer ticker.Stop()
		var last, lastCommand uint64
		var lastFrame, lastMotion time.Time
		for {
			select {
			case <-a.Context().Done():
				return
			case <-ticker.C:
				state := service.Snapshot()
				if state.Closed {
					a.Close()
					return
				}
				if state.Display.Animate && time.Since(lastMotion) >= 125*time.Millisecond {
					lastMotion = time.Now()
					_ = a.Post(func() {
						if !service.Snapshot().Display.Animate || screen.drawing.widget == nil {
							return
						}
						screen.renderRevision++
						screen.drawing.Revision = screen.renderRevision
						// Sync requests a canvas repaint without rebuilding the form
						// tree, preserving text focus while illustrative dots move.
						if err := screen.drawing.Sync(screen.drawing.widget); err != nil && !errors.Is(err, cogent.ErrFrameNotReady) {
							log.Printf("current animation: %v", err)
						}
					})
				}
				// Solver samples remain independent of presentation. Keep commands immediate
				// while pacing autonomous trace repaints to four frames per second.
				if state.Revision != last && (!workbench.IsTransientAnalysis(state.Analysis) || state.CommandRevision != lastCommand || time.Since(lastFrame) >= 250*time.Millisecond) {
					lastFrame = time.Now()
					lastCommand = state.CommandRevision
					last = state.Revision
					a.Invalidate("main")
				}
			}
		}
	}()
	smokeResult := make(chan error, 1)
	if *smoke {
		go func() { smokeResult <- nativeSmoke(a, screen, renderer); a.Close() }()
	}
	if err := a.Run(context.Background(), cogent.Driver{Commands: screen.commands, Theme: ui.Theme{Default: ui.Style{Typography: ui.Some(ui.Typography{Size: 13}), Foreground: ui.Some(ink)}, Kinds: map[ui.Kind]ui.Style{ui.Extension: {Background: ui.Some(color.NRGBA{R: 252, G: 253, B: 254, A: 255})}}}}); err != nil {
		log.Fatal(err)
	}
	if *smoke {
		if err := <-smokeResult; err != nil {
			log.Fatal(err)
		}
		fmt.Println(`{"event":"elab.native.smoke","passed":true}`)
	}
}

type screen struct {
	requestHelp                      *workbench.DiagnosticHelp
	requestHelpRevision              uint64
	displayStarted                   time.Time
	app                              *app.Application
	service                          *workbench.Service
	commands                         *command.Registry
	drawing                          *commandCanvas
	scope                            *scopeCanvas
	fileBusy                         bool
	fileMessage, pendingCommand      string
	pendingParams                    json.RawMessage
	chooseFile                       func(context.Context, app.FileOptions) (string, error)
	serial                           uint64
	errorMessage                     string
	renderRevision, lastDrawRevision uint64
	lastDrawAnalysis                 string
}

func (s *screen) invoke(id string, p any) workbench.Response {
	return s.invokeAtCommandRevision(id, p, s.service.Snapshot().CommandRevision)
}
func (s *screen) invokeAtCommandRevision(id string, p any, rev uint64) workbench.Response {
	s.fileMessage = ""
	s.serial++
	raw, _ := json.Marshal(p)
	if s.requestHelp != nil && rev != s.requestHelpRevision {
		s.requestHelp = nil
		s.errorMessage = ""
	}
	request := workbench.Request{Version: 1, ID: fmt.Sprintf("ui-%d", s.serial), Command: id, Params: raw, ExpectedCommandRevision: &rev, Key: fmt.Sprintf("ui-%d", s.serial)}
	result := s.service.Dispatch(context.Background(), request, "user")
	if result.Error != nil {
		s.errorMessage = result.Error.Message
		s.requestHelp = result.Help
		s.requestHelpRevision = rev
	} else if id == "diagnostics.details.set" && s.requestHelp != nil {
		if state, ok := result.Result.(workbench.Snapshot); ok {
			s.requestHelpRevision = state.CommandRevision
		}
	} else if id != "diagnostics.details.set" {
		s.errorMessage = ""
		s.requestHelp = nil
	}
	s.app.Invalidate("main")
	return result
}
func (s *screen) registerCommands() {
	s.registerFileCommands()
	for _, entry := range []struct{ id, label, shortcut string }{
		{"edit.undo", "Undo", "Meta+Z"}, {"edit.redo", "Redo", "Meta+Shift+Z"},
		{"project.clear", "New blank", ""}, {"drawing.cancel", "Cancel drawing", "Escape"},
		{"example.current.reset", "Reset current example", ""}, {"example.reset", "Load LED example", ""}, {"example.rc", "Load RC example", ""}, {"example.npn", "Load NPN LED example", ""}, {"example.npn-pulse", "Load blinking NPN LED", ""}, {"example.flasher", "Load capacitor-timed LED flasher", ""},
		{"simulation.pause", "Pause", ""}, {"simulation.resume", "Resume", ""}, {"simulation.restart", "Restart", ""}, {"view.fit", "Fit circuit", ""},
	} {
		entry := entry
		err := s.commands.Register(command.Command{ID: entry.id, Label: entry.label, Shortcut: entry.shortcut, ShortcutScope: command.AfterFocusShortcuts,
			Available: func() error {
				if s.fileBusy || s.pendingCommand != "" {
					return fmt.Errorf("finish the current file operation first")
				}
				state := s.service.Snapshot()
				if entry.id == "example.current.reset" && state.Example == nil {
					return fmt.Errorf("load an example before resetting it")
				}
				if entry.id == "edit.undo" && !state.CanUndo {
					return fmt.Errorf("nothing to undo")
				}
				if entry.id == "edit.redo" && !state.CanRedo {
					return fmt.Errorf("nothing to redo")
				}
				return nil
			}, Execute: func(context.Context) error {
				if entry.id == "example.current.reset" {
					state := s.service.Snapshot()
					if state.Example == nil {
						return fmt.Errorf("load an example before resetting it")
					}
					generation := state.Document.Generation
					s.requestReplace(entry.id, workbench.ExampleResetParams{ExpectedID: state.Example.Info.ID, ExpectedGeneration: &generation})
					return nil
				}
				if entry.id == "project.clear" || entry.id == "example.reset" || entry.id == "example.rc" || entry.id == "example.npn" || entry.id == "example.npn-pulse" || entry.id == "example.flasher" {
					s.requestReplace(entry.id, workbench.ReplaceParams{})
					return nil
				}
				var p any = struct{}{}
				if entry.id == "edit.undo" || entry.id == "edit.redo" {
					state := s.service.Snapshot()
					tip := state.UndoEntry
					if entry.id == "edit.redo" {
						tip = state.RedoEntry
					}
					if tip != nil {
						revision := tip.CommandRevision
						p = workbench.HistoryParams{ExpectedEntryRevision: &revision, AllowOtherActor: tip.Actor != "user"}
					}
				}
				if entry.id == "view.fit" {
					p = workbench.FitParams{Width: s.drawing.View.Width, Height: s.drawing.View.Height}
				}
				r := s.invoke(entry.id, p)
				if r.Error != nil {
					return fmt.Errorf("%s", r.Error.Message)
				}
				return nil
			},
		})
		if err != nil {
			panic(err)
		}
	}
}
func (s *screen) button(id, text string) ui.Node {
	help := ""
	state := s.service.Snapshot()
	if id == "edit.undo" {
		help = state.UndoName
	}
	if id == "edit.redo" {
		help = state.RedoName
	}
	tip := state.UndoEntry
	if id == "edit.redo" {
		tip = state.RedoEntry
	}
	if (id == "edit.undo" || id == "edit.redo") && tip != nil {
		help = tip.Name + " · by " + tip.Actor
		if tip.Actor != "user" && tip.Actor != "" {
			text += " " + tip.Actor
		}
	}

	activate := func() { s.commands.Invoke(context.Background(), id) }
	if (id == "edit.undo" || id == "edit.redo") && tip != nil {
		revision := tip.CommandRevision
		target := workbench.HistoryParams{ExpectedEntryRevision: &revision, AllowOtherActor: tip.Actor != "user"}
		activate = func() {
			if err := s.commands.Enabled(id); err != nil {
				s.errorMessage = err.Error()
				return
			}
			s.invoke(id, target)
		}
	}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Disabled: s.commands.Enabled(id) != nil, Accessibility: ui.Accessibility{Name: text, Help: help}, Activate: activate}, false)
}
