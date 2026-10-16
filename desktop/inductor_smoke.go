package main

import (
	"context"
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeInductorSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(run func() error) error {
		result := make(chan error, 1)
		if err := a.PostWait(ctx, func() { result <- run() }); err != nil {
			return err
		}
		select {
		case err := <-result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for _, example := range []string{"rl", "rlc", "flyback"} {
		if err := owner(func() error {
			if err := smokeUI(s, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example}); err != nil {
				return err
			}
			if s.pendingCommand == "example.load" {
				return smokeUI(s, "discard-continue", nil)
			}
			return nil
		}); err != nil {
			return err
		}
		if _, err := s.service.Wait(ctx); err != nil {
			return err
		}
		if err := owner(func() error {
			if err := smokeUI(s, "simulation.pause", nil); err != nil {
				return err
			}
			if err := smokeUI(s, "inductor-initial", &ui.ChangeEvent{Text: "20 mA"}); err != nil {
				return err
			}
			part, _ := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("L1")
			if part.InitialCurrent == nil || *part.InitialCurrent != .02 {
				return fmt.Errorf("native coil startup text did not commit")
			}
			if err := smokeUI(s, "inductor-initial-default", nil); err != nil {
				return err
			}
			return smokeUI(s, "simulation.restart", nil)
		}); err != nil {
			return err
		}
		if _, err := s.service.Wait(ctx); err != nil {
			return err
		}
		for sample := 0; sample < 50; sample++ {
			if err := owner(func() error { return smokeUI(s, "transient-step", nil) }); err != nil {
				return err
			}
		}
		if err := owner(func() error {
			state := s.service.Snapshot()
			if state.Example == nil || state.Example.Info.ID != example || state.Transient == nil || state.Mixed == nil || state.Mixed.Stale || len(state.Mixed.Inductors) != 1 || state.Mixed.Engine != workbench.InductorEngineID || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
				return fmt.Errorf("native %s coil result missing: %v", example, state.Diagnostics)
			}
			if example == "flyback" && (state.Mixed.MOSFETs[0].State != "cutoff" || state.Mixed.Diodes[0].Current < .045) {
				return fmt.Errorf("native flyback turn-off clamp missing")
			}
			root := s.view()
			if err := ui.Validate(root); err != nil {
				return err
			}
			for _, id := range []string{"inductor-initial", "inductor-reading", "plot-v:L1", "plot-i:L1"} {
				if _, ok := findNode(root, id); !ok {
					return fmt.Errorf("native coil control %s missing", id)
				}
			}
			if _, err := s.mesh(*s.drawing.View); err != nil {
				return err
			}
			_, err := s.scopeMesh(*s.scope.View)
			return err
		}); err != nil {
			return err
		}
	}
	return nativeLCSamplingSmoke(s, owner)
}

func nativeLCSamplingSmoke(s *screen, owner func(func() error) error) error {
	return owner(func() error {
		for _, action := range []struct {
			command string
			params  any
		}{
			{"example.load", workbench.ExampleLoadParams{ID: "rlc", DiscardChanges: true}},
			{"simulation.pause", struct{}{}},
			{"component.set", workbench.ValueParams{ID: "L1", Value: "10 mH"}},
			{"component.set", workbench.ValueParams{ID: "C1", Value: "1 µF"}},
			{"component.select", workbench.IDParams{}},
		} {
			if response := s.invoke(action.command, action.params); response.Error != nil {
				return fmt.Errorf("LC sampling fixture: %s", response.Error.Message)
			}
		}
		if err := smokeUI(s, "fit-circuit-sampling", nil); err != nil {
			return err
		}
		state := s.service.Snapshot()
		want := 2 * math.Pi * math.Sqrt(.01*1e-6) / 32
		if state.LCSampling == nil || state.LCSampling.Unresolved || state.TransientSettings.StepSeconds != want || !state.Paused || s.keyboardFocus.ID != "setting-step" {
			return fmt.Errorf("native LC fit/paused focus continuation differs from shared policy")
		}
		for _, value := range []workbench.ValueParams{{ID: "L1", Value: "1 µH"}, {ID: "C1", Value: "1 nF"}} {
			if response := s.invoke("component.set", value); response.Error != nil {
				return fmt.Errorf("minimum LC sampling fixture: %s", response.Error.Message)
			}
		}
		if err := smokeUI(s, "fit-circuit-sampling", nil); err != nil {
			return err
		}
		state = s.service.Snapshot()
		if state.TransientSettings.StepSeconds != 1e-6 || !state.LCSampling.MinimumLimited || !state.LCSampling.Unresolved || state.LCSampling.CanFit {
			return fmt.Errorf("native LC sample floor was presented as sufficient resolution")
		}
		root := s.view()
		if err := ui.Validate(root); err != nil {
			return err
		}
		if _, found := findNode(root, "fit-circuit-sampling"); found {
			return fmt.Errorf("native sample-floor notice retains an ineffective action")
		}
		return nil
	})
}
