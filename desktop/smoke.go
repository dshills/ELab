package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativeSmoke(a *app.Application, s *screen, g *cogent.MeshCanvas) error {
	ctx, cancel := context.WithTimeout(a.Context(), 20*time.Second)
	defer cancel()
	constructed := make(chan error, 1)
	if err := a.PostWait(ctx, func() { constructed <- smokeConstruct(s) }); err != nil {
		return err
	}
	select {
	case err := <-constructed:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	state, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if state.Result == nil || math.Abs(state.Result.Current-0.0029168) > 0.000002 {
		return fmt.Errorf("initial electrical result is incorrect")
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		type probe struct {
			ready bool
			err   error
		}
		completed := make(chan probe, 1)
		err := a.PostWait(ctx, func() {
			image, err := g.Snapshot()
			if errors.Is(err, cogent.ErrFrameNotReady) {
				completed <- probe{}
				return
			}
			if err != nil {
				completed <- probe{err: err}
				return
			}
			colored := 0
			for y := 0; y < image.Bounds().Dy(); y++ {
				for x := 0; x < image.Bounds().Dx(); x++ {
					p := image.NRGBAAt(x, y)
					if p.A > 100 && p.R < 120 && p.G > 60 && p.B > 60 {
						colored++
					}
				}
			}
			if colored < 500 {
				completed <- probe{err: fmt.Errorf("schematic drawing absent: %d colored pixels", colored)}
				return
			}
			if err := smokeUI(s, "selection", &ui.ChangeEvent{SelectedID: "R1"}); err != nil {
				completed <- probe{err: err}
				return
			}
			if err := smokeUI(s, "value-entry", &ui.ChangeEvent{Text: "2k"}); err != nil {
				completed <- probe{err: err}
				return
			}
			if err := smokeUI(s, "apply-value", nil); err != nil {
				completed <- probe{err: err}
				return
			}
			completed <- probe{ready: true}
		})
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case probe := <-completed:
			if probe.err != nil {
				return probe.err
			}
			if probe.ready {
				goto rendered
			}
		}
	}
rendered:
	state, err = s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if state.Circuit.Resistance != 2000 || state.Result == nil || math.Abs(state.Result.Current-0.0014767) > 0.000002 {
		return fmt.Errorf("UI edit did not update live result")
	}
	if err := nativeRCSmoke(ctx, a, s); err != nil {
		return err
	}
	if err := nativeNPNSmoke(ctx, a, s); err != nil {
		return err
	}
	if err := nativeNPNPulseSmoke(ctx, a, s); err != nil {
		return err
	}
	if err := nativeMixedSmoke(ctx, a, s); err != nil {
		return err
	}
	return nativeWorkflowSmoke(ctx, a, s)
}

func smokeUI(s *screen, id string, event *ui.ChangeEvent) error {
	var find func(ui.Node) (ui.Node, bool)
	find = func(n ui.Node) (ui.Node, bool) {
		if n.ID == id {
			return n, true
		}
		for _, child := range n.Children {
			if target, ok := find(child); ok {
				return target, true
			}
		}
		return ui.Node{}, false
	}
	node, ok := find(s.view())
	if !ok || node.Disabled {
		return fmt.Errorf("UI control %s is missing or disabled", id)
	}
	if event == nil {
		node.Activate()
	} else {
		node.Change(*event)
	}
	if s.errorMessage != "" {
		return fmt.Errorf("%s: %s", id, s.errorMessage)
	}
	return nil
}
func smokeConstruct(s *screen) error {
	for _, p := range []struct {
		kind     string
		position workbench.Point
	}{{"dc-source", workbench.Point{X: -200}}, {"resistor", workbench.Point{Y: -110}}, {"led", workbench.Point{X: 200}}, {"ground", workbench.Point{Y: 110}}} {
		if err := smokeUI(s, "part-"+p.kind, nil); err != nil {
			return err
		}
		if err := smokeUI(s, "position-x", &ui.ChangeEvent{Number: float32(p.position.X)}); err != nil {
			return err
		}
		if err := smokeUI(s, "position-y", &ui.ChangeEvent{Number: float32(p.position.Y)}); err != nil {
			return err
		}
		if err := smokeUI(s, "place-at-position", nil); err != nil {
			return err
		}
	}
	if err := smokeUI(s, "tool-wire", nil); err != nil {
		return err
	}
	for _, pair := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "D1.anode"}, {"D1.cathode", "GND1.reference"}, {"GND1.reference", "V1.negative"}} {
		if err := smokeUI(s, "wire-from", &ui.ChangeEvent{SelectedID: pair[0]}); err != nil {
			return err
		}
		if err := smokeUI(s, "wire-to", &ui.ChangeEvent{SelectedID: pair[1]}); err != nil {
			return err
		}
		if err := smokeUI(s, "connect-terminals", nil); err != nil {
			return err
		}
	}
	return nil
}

func nativeRCSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "example.rc", nil); err != nil {
			checked <- err
			return
		}
		if s.pendingCommand != "" {
			if err := smokeUI(s, "discard-continue", nil); err != nil {
				checked <- err
				return
			}
		}
		for _, id := range []string{"simulation.pause", "simulation.restart"} {
			if err := smokeUI(s, id, nil); err != nil {
				checked <- err
				return
			}
		}
		for i := 0; i < 10; i++ {
			if err := smokeUI(s, "transient-step", nil); err != nil {
				checked <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.Transient == nil || math.Abs(state.Transient.Sample.Voltage-3.1606027941427883) > 1e-12 {
			checked <- fmt.Errorf("native RC reference failed")
			return
		}
		checked <- nil
	}); err != nil {
		return err
	}
	if err := <-checked; err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		result := make(chan error, 1)
		if err := a.PostWait(ctx, func() {
			image, err := s.scope.Snapshot()
			if err != nil {
				result <- err
				return
			}
			colored := 0
			for y := 0; y < image.Bounds().Dy(); y++ {
				for x := 0; x < image.Bounds().Dx(); x++ {
					p := image.NRGBAAt(x, y)
					if p.A > 100 && p.R < 80 && p.G > 70 && p.B > 70 {
						colored++
					}
				}
			}
			if colored < 100 {
				result <- cogent.ErrFrameNotReady
				return
			}
			result <- nil
		}); err != nil {
			return err
		}
		err := <-result
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		return err
	}
}

func nativeNPNSmoke(ctx context.Context, a *app.Application, s *screen) error {
	apply := func(ids ...string) error {
		result := make(chan error, 1)
		if err := a.PostWait(ctx, func() {
			for _, id := range ids {
				if err := smokeUI(s, id, nil); err != nil {
					result <- err
					return
				}
			}
			result <- nil
		}); err != nil {
			return err
		}
		return <-result
	}
	if err := apply("example.npn", "discard-continue"); err != nil {
		return err
	}
	on, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if on.NPN == nil || on.NPN.State != "saturation" {
		return fmt.Errorf("native NPN example failed")
	}
	if err := apply("npn-off"); err != nil {
		return err
	}
	off, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if off.NPN == nil || off.NPN.State != "cutoff" {
		return fmt.Errorf("native NPN off control failed")
	}
	if err := apply("npn-on"); err != nil {
		return err
	}
	on, err = s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if on.NPN == nil || math.Abs(on.NPN.CollectorCurrent-.002860330492) > 1e-10 {
		return fmt.Errorf("native NPN live result failed")
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		ready := make(chan bool, 1)
		if err := a.PostWait(ctx, func() {
			img, err := s.drawing.Snapshot()
			if err != nil {
				ready <- false
				return
			}
			red := 0
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					p := img.NRGBAAt(x, y)
					if p.R > 220 && p.G < 180 && p.B < 120 {
						red++
					}
				}
			}
			ready <- red > 50
		}); err != nil {
			return err
		}
		if <-ready {
			return nil
		}
	}
}

func nativeNPNPulseSmoke(ctx context.Context, a *app.Application, s *screen) error {
	applied := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, id := range []string{"example.npn-pulse", "discard-continue", "simulation.pause", "simulation.restart"} {
			if err := smokeUI(s, id, nil); err != nil {
				applied <- err
				return
			}
		}
		for i := 0; i < 50; i++ {
			if err := smokeUI(s, "transient-step", nil); err != nil {
				applied <- err
				return
			}
		}
		if state := s.service.Snapshot(); state.NPN == nil || state.NPN.State != "cutoff" || state.Transient.Sample.Input != 0 {
			applied <- fmt.Errorf("native pulse falling edge failed")
			return
		}
		if err := smokeUI(s, "input-high", nil); err != nil {
			applied <- err
			return
		}
		if err := smokeUI(s, "transient-step", nil); err != nil {
			applied <- err
			return
		}
		if state := s.service.Snapshot(); state.NPN.State != "saturation" || state.Transient.Sample.Input != 5 {
			applied <- fmt.Errorf("native pulse high override failed")
			return
		}
		applied <- nil
	}); err != nil {
		return err
	}
	if err := <-applied; err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		result := make(chan bool, 1)
		if err := a.PostWait(ctx, func() {
			img, err := s.scope.Snapshot()
			if err != nil {
				result <- false
				return
			}
			coral, teal := 0, 0
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					p := img.NRGBAAt(x, y)
					if p.R > 180 && p.G < 130 && p.B < 130 {
						coral++
					}
					if p.R < 80 && p.G > 70 && p.B > 70 {
						teal++
					}
				}
			}
			result <- coral > 50 && teal > 50
		}); err != nil {
			return err
		}
		if <-result {
			return nil
		}
	}
}

func nativeMixedSmoke(ctx context.Context, a *app.Application, s *screen) error {
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, id := range []string{"example.flasher", "discard-continue", "simulation.pause", "simulation.restart"} {
			if err := smokeUI(s, id, nil); err != nil {
				result <- err
				return
			}
		}
		for i := 0; i < 60; i++ {
			if err := smokeUI(s, "transient-step", nil); err != nil {
				result <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.Mixed == nil || state.Mixed.Time < .299 || state.Mixed.KCLResidual > 1e-9 || len(state.Mixed.Capacitors) != 2 || len(state.Mixed.Transistors) != 2 {
			result <- fmt.Errorf("native mixed transient failed")
			return
		}
		if err := smokeUI(s, "probe-5", &ui.ChangeEvent{SelectedID: "i:C2"}); err != nil {
			result <- err
			return
		}
		if err := smokeUI(s, "cursor-time-a", &ui.ChangeEvent{Number: 100}); err != nil {
			result <- err
			return
		}
		if err := smokeUI(s, "cursor-select-b", nil); err != nil {
			result <- err
			return
		}
		s.scope.placeCursor(58 + (s.scope.View.Width-78-58)*.75)
		sc := s.service.Snapshot().Scope
		if sc.A == nil || sc.B == nil || !sc.A.Valid || !sc.B.Valid || sc.DeltaSeconds == nil || len(sc.Readings) != 6 {
			result <- fmt.Errorf("native probe/cursor controls failed")
			return
		}

		result <- nil
	}); err != nil {
		return err
	}
	if err := <-result; err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		ready := make(chan bool, 1)
		if err := a.PostWait(ctx, func() {
			img, err := s.scope.Snapshot()
			if err != nil {
				ready <- false
				return
			}
			purple, coral := 0, 0
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					p := img.NRGBAAt(x, y)
					if p.R > 90 && p.R < 170 && p.G < 130 && p.B > 140 {
						purple++
					}
					if p.R > 180 && p.G < 130 && p.B < 130 {
						coral++
					}
				}
			}
			state := s.service.Snapshot()
			t := state.Transient
			cursorLines := 0
			for _, cursor := range []*workbench.ScopeCursor{state.Scope.A, state.Scope.B} {
				if cursor == nil || !cursor.Valid {
					continue
				}
				px := 58 + (s.scope.View.Width-78-58)*(cursor.Time-t.Samples[0].Time)/math.Max(t.Settings.StepSeconds, t.Sample.Time-t.Samples[0].Time)
				sx := float64(img.Bounds().Dx()) / s.scope.View.Width
				center := int(px * sx)
				pixels := 0
				for y := int(float64(img.Bounds().Dy()) * .15); y < int(float64(img.Bounds().Dy())*.72); y++ {
					for x := max(0, center-2); x <= min(img.Bounds().Dx()-1, center+2); x++ {
						p := img.NRGBAAt(x, y)
						if p.R < 60 && p.G < 70 && p.B < 85 {
							pixels++
							break
						}
					}
				}
				if pixels > int(float64(img.Bounds().Dy())*.3) {
					cursorLines++
				}
			}
			ready <- purple > 50 && coral > 50 && cursorLines == 2
		}); err != nil {
			return err
		}
		if <-ready {
			return nil
		}
	}
}

func nativeWorkflowSmoke(ctx context.Context, a *app.Application, s *screen) error {
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "part-search", &ui.ChangeEvent{Text: "capacitor"}); err != nil {
			result <- err
			return
		}
		if len(s.service.Snapshot().MatchingParts) != 1 {
			result <- fmt.Errorf("native component search failed")
			return
		}
		if err := smokeUI(s, "part-search", &ui.ChangeEvent{Text: ""}); err != nil {
			result <- err
			return
		}
		for _, id := range []string{"cap-pair-20uF", "edit.undo", "simulation.restart"} {
			if err := smokeUI(s, id, nil); err != nil {
				result <- err
				return
			}
		}
		for i := 0; i < 60; i++ {
			if err := smokeUI(s, "transient-step", nil); err != nil {
				result <- err
				return
			}
		}
		if err := smokeUI(s, "selection", &ui.ChangeEvent{SelectedID: "C2"}); err != nil {
			result <- err
			return
		}
		for _, id := range []string{"plot-v:C2", "scope-latest-window"} {
			if err := smokeUI(s, id, nil); err != nil {
				result <- err
				return
			}
		}
		state := s.service.Snapshot()
		if !state.Scope.WindowValid || state.Scope.Window.Mode != "fixed" || len(state.Scope.Statistics) != 6 || state.Scope.Statistics[0].Count > 42 {
			result <- fmt.Errorf("native interval plotting/statistics failed")
			return
		}
		result <- nil
	}); err != nil {
		return err
	}
	return <-result
}
