package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativeSmoke(a *app.Application, s *screen, g *cogent.MeshCanvas, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(a.Context(), timeout)
	defer cancel()
	stage := "construction and first GPU frame"
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %w", stage, err)
		}
	}()
	setStage := func(value string) {
		stage = value
		log.Printf("smoke: %s", stage)
	}
	setStage(stage)
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
	setStage("nativeRCSmoke")
	if err := nativeRCSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeScopeVisibilitySmoke")
	if err := nativeScopeVisibilitySmoke(ctx, s); err != nil {
		return err
	}
	setStage("nativeNPNSmoke")
	if err := nativeNPNSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeNPNPulseSmoke")
	if err := nativeNPNPulseSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeMixedSmoke")
	if err := nativeMixedSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeWorkflowSmoke")
	if err := nativeWorkflowSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeVerificationSmoke")
	if err := nativeVerificationSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeRecoverySmoke")
	if err := nativeRecoverySmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeRecordingSmoke")
	if err := nativeRecordingSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeProjectRecordingSmoke")
	if err := nativeProjectRecordingSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeRecordingComparisonSmoke")
	if err := nativeRecordingComparisonSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativePrimitivesSmoke")
	if err := nativePrimitivesSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeSwitchSmoke")
	if err := nativeSwitchSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeDiodeSmoke")
	if err := nativeDiodeSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativePotentiometerSmoke")
	if err := nativePotentiometerSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeProjectOpenSmoke")
	if err := nativeProjectOpenSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeExportSmoke")
	if err := nativeExportSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeWaveformSmoke")
	if err := nativeWaveformSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeAuditSmoke")
	if err := nativeAuditSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeCapacitySmoke")
	if err := nativeCapacitySmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeTraceFocusSmoke")
	if err := nativeTraceFocusSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativePinnedReadingsSmoke")
	if err := nativePinnedReadingsSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeMeasurementInspectionSmoke")
	if err := nativeMeasurementInspectionSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeDisplaySmoke")
	if err := nativeDisplaySmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeDiagnosticSmoke")
	if err := nativeDiagnosticSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeExamplesSmoke")
	if err := nativeExamplesSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeIntegrationLimitSmoke")
	if err := nativeIntegrationLimitSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeFieldContextSmoke")
	if err := nativeFieldContextSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeDCSweepSmoke")
	if err := nativeDCSweepSmoke(ctx, a, s); err != nil {
		return fmt.Errorf("native DC sweep: %w", err)
	}
	setStage("nativePrintPreviewSmoke")
	if err := nativePrintPreviewSmoke(ctx, a, s); err != nil {
		return fmt.Errorf("native print preview: %w", err)
	}
	setStage("nativeValueDomainsSmoke")
	if err := nativeValueDomainsSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeVerificationContractsSmoke")
	if err := nativeVerificationContractsSmoke(ctx, a, s); err != nil {
		return err
	}
	setStage("nativeScopeContextSmoke")
	if err := nativeScopeContextSmoke(ctx, a, s); err != nil {
		return err
	}
	return nativeLifecycleSmoke(ctx, a, s)
}

func smokeUI(s *screen, id string, event *ui.ChangeEvent) error {
	node, ok := findAction(s.view(), id)
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
	state := s.service.Snapshot()
	if id == "transient-step" || id == "recovery-checkpoint" || id == "recovery-scan" || ((id == "recovery-restore" || id == "discard-continue") && activeProjectWork(state)) || state.Status == "updating" || activeInputWork(state) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		state, err := s.service.Wait(ctx)
		if err != nil {
			return err
		}
		if id == "transient-step" && (state.StepWork == nil || state.StepWork.Status != "completed") {
			return fmt.Errorf("native manual step failed: %+v", state.StepWork)
		}
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
		if err := smokeUI(s, "cursor-time-a", &ui.ChangeEvent{Text: "100 ms"}); err != nil {
			result <- err
			return
		}
		if err := smokeUI(s, "cursor-select-b", nil); err != nil {
			result <- err
			return
		}
		if _, err := s.scopeMesh(*s.scope.View); err != nil {
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

func nativeVerificationSmoke(ctx context.Context, a *app.Application, s *screen) error {
	started := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "verify-duration", &ui.ChangeEvent{Text: "20 ms"}); err != nil {
			started <- err
			return
		}
		if err := smokeUI(s, "verify-start", nil); err != nil {
			started <- err
			return
		}
		started <- nil
	}); err != nil {
		return err
	}
	if err := <-started; err != nil {
		return err
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		state := s.service.Snapshot()
		for _, operation := range state.Operations {
			if operation.ID != state.LastOperation {
				continue
			}
			if operation.Status == "failed" || operation.Status == "canceled" {
				return fmt.Errorf("native verification failed: %+v", operation)
			}
			if operation.Status == "completed" {
				if state.Verification == nil || state.Verification.Engine != workbench.MixedEngineID || state.Verification.SampleCount < 5 || len(state.Verification.Readings) != 6 || state.Transient.Sample.Time < .299 || state.Transient.Sample.Time > .301 {
					return fmt.Errorf("native recorded result or live-state isolation failed")
				}
				return nil
			}
		}
	}
}

func nativeRecoverySmoke(ctx context.Context, a *app.Application, s *screen) error {
	completed := make(chan error, 1)
	// Configure on the owner, then wait off it before choosing a manual snapshot.
	if recovery := s.service.Snapshot().Recovery; recovery.Enabled {
		if err := a.PostWait(ctx, func() {
			completed <- smokeUI(s, "recovery-toggle", nil)
		}); err != nil {
			return err
		}
		if err := <-completed; err != nil {
			return err
		}
		if _, err := s.service.Wait(ctx); err != nil {
			return err
		}
	}

	if err := a.PostWait(ctx, func() {
		before := s.service.Snapshot()
		if err := smokeUI(s, "recovery-checkpoint", nil); err != nil {
			completed <- err
			return
		}
		if len(s.service.Snapshot().Recovery.Candidates) == 0 {
			completed <- fmt.Errorf("native recovery snapshot missing")
			return
		}
		if err := smokeUI(s, "recovery-choice", &ui.ChangeEvent{SelectedID: s.service.Snapshot().Recovery.Candidates[0].ID}); err != nil {
			completed <- err
			return
		}
		if err := smokeUI(s, "recovery-restore", nil); err != nil {
			completed <- err
			return
		}
		if s.pendingCommand != "recovery.restore" {
			completed <- fmt.Errorf("recovery bypassed unsaved decision")
			return
		}
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			completed <- err
			return
		}
		after := s.service.Snapshot()
		if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !after.Document.Dirty || after.Document.Path != "" || after.Document.RecoveredFrom == "" {
			completed <- fmt.Errorf("native recovery copy changed schematic")
			return
		}
		completed <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-completed:
		if err != nil {
			return err
		}
		return nativeRecoveryDismissSmoke(ctx, a, s)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func nativeRecordingSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		if !s.service.Snapshot().Paused {
			if err := smokeUI(s, "simulation.pause", nil); err != nil {
				ready <- err
				return
			}
		}
		before := s.service.Snapshot()
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"recording-show", nil}, {"recorded-index", &ui.ChangeEvent{Number: 1}}, {"recorded-cursor-a", &ui.ChangeEvent{Text: "1 ms"}}, {"recorded-cursor-b", &ui.ChangeEvent{Text: "11 ms"}}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				ready <- err
				return
			}
		}
		after := s.service.Snapshot()
		if after.Recording == nil || !after.Recording.Viewing || after.Recording.Index != 1 || after.Transient.Sample != before.Transient.Sample || after.Document != before.Document || after.Recording.Scope.DeltaSeconds == nil {
			ready <- fmt.Errorf("native recorded controls changed live state or lacked recorded measurements")
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		rendered := make(chan error, 1)
		if err := a.PostWait(ctx, func() {
			if _, ok := findNode(s.view(), "recorded-scope-title"); !ok {
				rendered <- fmt.Errorf("recorded plot is not in the native view")
				return
			}
			pixels, err := s.scope.Snapshot()
			if err != nil {
				rendered <- err
				return
			}
			colored := 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 100 && p.R < 130 && p.G > 65 && p.B > 65 {
						colored++
					}
				}
			}
			if colored < 20 {
				rendered <- cogent.ErrFrameNotReady
				return
			}
			rendered <- nil
		}); err != nil {
			return err
		}
		err := <-rendered
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		return err
	}
}

func findNode(n ui.Node, id string) (ui.Node, bool) {
	if n.ID == id {
		return n, true
	}
	for _, child := range n.Children {
		if found, ok := findNode(child, id); ok {
			return found, true
		}
	}
	return ui.Node{}, false
}

func nativeProjectRecordingSmoke(ctx context.Context, a *app.Application, s *screen) error {
	directory, err := os.MkdirTemp("", "elab-native-project-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "experiment.elab")
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		before := s.service.Snapshot()
		if before.Recording == nil || !before.Paused {
			result <- fmt.Errorf("native project smoke lacks paused recorded source")
			return
		}
		if response := s.invoke("project.save", workbench.SaveParams{Path: path}); response.Error != nil {
			result <- fmt.Errorf("native project save: %s", response.Error.Message)
			return
		}
		if response := s.invoke("project.load", workbench.LoadParams{Path: path}); response.Error != nil {
			result <- fmt.Errorf("native project reload: %s", response.Error.Message)
			return
		}
		after, err := s.service.Wait(ctx)
		if err != nil {
			result <- err
			return
		}
		if after.Document.Dirty || after.Document.RecordingCount != 1 || after.Recording == nil || after.Recording.OriginID != before.Recording.OriginID || after.Recording.Total != before.Recording.Total || !after.Recording.Viewing || after.Recording.Playing || !after.Paused || after.Transient == nil || after.Transient.Sample.Time != 0 || after.Transient.Stale {
			result <- fmt.Errorf("native project reload lost captured data or resumed physical history")
			return
		}
		result <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func nativePrimitivesSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"example.reset", nil}, {"selection", &ui.ChangeEvent{SelectedID: "W1"}}, {"wire-insert-junction", nil}, {"part-net-label", nil}, {"place-label", &ui.ChangeEvent{Text: "SUPPLY"}}, {"position-x", &ui.ChangeEvent{Number: 350}}, {"position-y", &ui.ChangeEvent{Number: -180}}, {"place-at-position", nil}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				done <- err
				return
			}
			if action.id == "example.reset" && s.service.Snapshot().Paused {
				if err := smokeUI(s, "simulation.resume", nil); err != nil {
					done <- err
					return
				}
			}
		}
		id := s.service.Snapshot().Selected
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: "V1.positive"}}, {"wire-to", &ui.ChangeEvent{SelectedID: id + ".node"}}, {"connect-terminals", nil}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}); err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	state, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if state.Result == nil || state.Result.Stale || math.Abs(state.Result.Current-.00291679) > 2e-8 || len(state.Components) != 6 {
		return fmt.Errorf("native primitive connectivity did not preserve LED result")
	}
	rendered := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		mesh, err := s.mesh(*s.drawing.View)
		if err == nil && mesh == nil {
			err = fmt.Errorf("native primitive mesh absent")
		}
		rendered <- err
	}); err != nil {
		return err
	}
	return <-rendered
}

func nativeSwitchSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		fail := func(err error) { checked <- err }
		if err := smokeUI(s, "example.reset", nil); err != nil {
			fail(err)
			return
		}
		if s.pendingCommand != "" {
			if err := smokeUI(s, "discard-continue", nil); err != nil {
				fail(err)
				return
			}
		}
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"selection", &ui.ChangeEvent{SelectedID: "W1"}}, {"delete-object", nil},
			{"part-switch", nil}, {"position-x", &ui.ChangeEvent{Number: -100}}, {"position-y", &ui.ChangeEvent{Number: -60}}, {"place-at-position", nil},
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: "V1.positive"}}, {"wire-to", &ui.ChangeEvent{SelectedID: "S1.1"}}, {"connect-terminals", nil},
			{"wire-from", &ui.ChangeEvent{SelectedID: "S1.2"}}, {"wire-to", &ui.ChangeEvent{SelectedID: "R1.1"}}, {"connect-terminals", nil},
			{"switch-actuate", nil},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				fail(err)
				return
			}
		}
		state := s.service.Snapshot()
		if state.Mixed == nil || state.Mixed.LEDs[0].Current < .0028 {
			fail(fmt.Errorf("native closed-switch LED failed"))
			return
		}
		if _, ok := findNode(s.view(), "switch-reading"); !ok {
			fail(fmt.Errorf("native switch reading absent"))
			return
		}
		if _, err := s.mesh(*s.drawing.View); err != nil {
			fail(err)
			return
		}
		if err := smokeUI(s, "switch-actuate", nil); err != nil {
			fail(err)
			return
		}
		state = s.service.Snapshot()
		if state.Mixed == nil || state.Mixed.LEDs[0].Current <= 0 || state.Mixed.LEDs[0].Current > 5e-9 {
			fail(fmt.Errorf("native open-switch leakage failed"))
			return
		}
		checked <- nil
	}); err != nil {
		return err
	}
	return <-checked
}

// Build a silicon diode from tray controls and named terminal connections.
// Existing session IDs remain reserved, so inspect the newly selected identity.
func smokeConstructDiode(s *screen) error {
	if err := smokeUI(s, "example.reset", nil); err != nil {
		return err
	}
	if s.pendingCommand != "" {
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			return err
		}
	}
	for _, action := range []struct {
		id    string
		event *ui.ChangeEvent
	}{
		{"selection", &ui.ChangeEvent{SelectedID: "D1"}}, {"delete-object", nil},
		{"part-diode", nil}, {"position-x", &ui.ChangeEvent{Number: 200}}, {"position-y", &ui.ChangeEvent{Number: 0}}, {"place-at-position", nil},
	} {
		if err := smokeUI(s, action.id, action.event); err != nil {
			return err
		}
	}
	id := s.service.Snapshot().Selected
	for _, pair := range [][2]string{{"R1.2", id + ".anode"}, {id + ".cathode", "GND.reference"}} {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: pair[0]}}, {"wire-to", &ui.ChangeEvent{SelectedID: pair[1]}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				return err
			}
		}
	}
	return nil
}
func nativeDiodeSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeConstructDiode(s); err != nil {
			checked <- err
			return
		}
		state := s.service.Snapshot()
		if state.DiodeModel == nil || state.Mixed == nil || len(state.Mixed.Diodes) != 1 || state.Mixed.Diodes[0].Current < .004 {
			checked <- fmt.Errorf("native silicon diode conduction failed")
			return
		}
		if _, ok := findNode(s.view(), "diode-model"); !ok {
			checked <- fmt.Errorf("native diode model absent")
			return
		}
		_, err := s.mesh(*s.drawing.View)
		checked <- err
	}); err != nil {
		return err
	}
	return <-checked
}

func smokeConstructPotentiometer(s *screen) error {
	if err := smokeUI(s, "example.reset", nil); err != nil {
		return err
	}
	if s.pendingCommand != "" {
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			return err
		}
	}
	for _, action := range []struct {
		id    string
		event *ui.ChangeEvent
	}{
		{"selection", &ui.ChangeEvent{SelectedID: "R1"}}, {"delete-object", nil}, {"part-potentiometer", nil},
		{"position-x", &ui.ChangeEvent{Number: 0}}, {"position-y", &ui.ChangeEvent{Number: -110}}, {"place-at-position", nil},
	} {
		if err := smokeUI(s, action.id, action.event); err != nil {
			return err
		}
	}
	id := s.service.Snapshot().Selected
	for _, pair := range [][2]string{{"V1.positive", id + ".1"}, {id + ".2", "GND.reference"}, {id + ".wiper", "D1.anode"}} {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: pair[0]}}, {"wire-to", &ui.ChangeEvent{SelectedID: pair[1]}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				return err
			}
		}
	}
	return nil
}
func nativePotentiometerSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeConstructPotentiometer(s); err != nil {
			checked <- err
			return
		}
		before := s.service.Snapshot()
		if err := smokeUI(s, "pot-wiper", &ui.ChangeEvent{Number: 75}); err != nil {
			checked <- err
			return
		}
		after := s.service.Snapshot()
		if before.Mixed == nil || after.Mixed == nil || len(after.Mixed.Potentiometers) != 1 || after.Mixed.LEDs[0].Current <= before.Mixed.LEDs[0].Current {
			checked <- fmt.Errorf("native pot slider did not increase LED current")
			return
		}
		if _, ok := findNode(s.view(), "pot-reading"); !ok {
			checked <- fmt.Errorf("native pot readout absent")
			return
		}
		if err := smokeConstructPotRC(s); err != nil {
			checked <- err
			return
		}
		before = s.service.Snapshot()
		if err := smokeUI(s, "pot-wiper", &ui.ChangeEvent{Number: 75}); err != nil {
			checked <- err
			return
		}
		after = s.service.Snapshot()
		if after.InputWork == nil || after.InputWork.Status != "completed" || after.Transient.Sample.Time != before.Transient.Sample.Time || math.Abs(after.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-10 || after.RunID != before.RunID {
			checked <- fmt.Errorf("native transient wiper did not commit atomically at preserved time/charge")
			return
		}
		if _, ok := findNode(s.view(), "input-work-status"); !ok {
			checked <- fmt.Errorf("native applying-input status absent")
			return
		}
		_, err := s.mesh(*s.drawing.View)
		checked <- err
	}); err != nil {
		return err
	}
	return <-checked
}

func smokeConstructPotRC(s *screen) error {
	if err := smokeUI(s, "example.rc", nil); err != nil {
		return err
	}
	if s.pendingCommand != "" {
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			return err
		}
	}
	for _, action := range []struct {
		id    string
		event *ui.ChangeEvent
	}{
		{"simulation.pause", nil}, {"selection", &ui.ChangeEvent{SelectedID: "R1"}},
		{"delete-object", nil}, {"part-potentiometer", nil},
		{"position-x", &ui.ChangeEvent{Number: 0}}, {"position-y", &ui.ChangeEvent{Number: -110}}, {"place-at-position", nil},
	} {
		if err := smokeUI(s, action.id, action.event); err != nil {
			return err
		}
	}
	id := s.service.Snapshot().Selected
	for _, pair := range [][2]string{{"V1.positive", id + ".1"}, {id + ".2", "GND.reference"}, {id + ".wiper", "C1.1"}} {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: pair[0]}}, {"wire-to", &ui.ChangeEvent{SelectedID: pair[1]}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				return err
			}
		}
	}
	if err := smokeUI(s, "simulation.restart", nil); err != nil {
		return err
	}
	return smokeUI(s, "transient-step", nil)
}

func smokeConstructWaveform(s *screen, kind string) error {
	if err := smokeUI(s, "example.rc", nil); err != nil {
		return err
	}
	if s.pendingCommand != "" {
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			return err
		}
	}
	for _, action := range []struct {
		id    string
		event *ui.ChangeEvent
	}{
		{"selection", &ui.ChangeEvent{SelectedID: "V1"}}, {"delete-object", nil}, {"part-" + kind, nil},
		{"position-x", &ui.ChangeEvent{Number: -200}}, {"position-y", &ui.ChangeEvent{Number: 0}}, {"place-at-position", nil},
	} {
		if err := smokeUI(s, action.id, action.event); err != nil {
			return err
		}
	}
	id := s.service.Snapshot().Selected
	for _, pair := range [][2]string{{id + ".positive", "R1.1"}, {id + ".negative", "GND.reference"}} {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: pair[0]}}, {"wire-to", &ui.ChangeEvent{SelectedID: pair[1]}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				return err
			}
		}
	}
	return nil
}
func nativeWaveformSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
			checked <- err
			return
		}
		for _, id := range []string{"fit-waveform-sampling", "simulation.pause", "simulation.restart", "transient-step"} {
			if err := smokeUI(s, id, nil); err != nil {
				checked <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.Transient == nil || state.Transient.Engine != workbench.WaveformEngineID || state.Transient.Sample.Input <= 0 || state.Mixed.Capacitors[0].Voltage <= 0 {
			checked <- fmt.Errorf("native finite ramp/RC failed")
			return
		}
		if err := smokeConstructWaveform(s, "sine-source"); err != nil {
			checked <- err
			return
		}
		if err := smokeUI(s, "sine-frequency", &ui.ChangeEvent{Number: 20}); err != nil {
			checked <- err
			return
		}
		for _, id := range []string{"simulation.pause", "simulation.restart", "transient-step"} {
			if err := smokeUI(s, id, nil); err != nil {
				checked <- err
				return
			}
		}
		state = s.service.Snapshot()
		if state.Transient == nil || state.Transient.Engine != workbench.WaveformEngineID || state.Transient.Sample.Input <= 0 {
			checked <- fmt.Errorf("native sine/RC failed")
			return
		}
		_, err := s.mesh(*s.drawing.View)
		checked <- err
	}); err != nil {
		return err
	}
	return <-checked
}

func nativeAuditSmoke(ctx context.Context, a *app.Application, s *screen) error {
	checked := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "part-search", &ui.ChangeEvent{Text: "battery"}); err != nil {
			checked <- err
			return
		}
		if _, ok := findNode(s.view(), "part-dc-source"); !ok {
			checked <- fmt.Errorf("native battery alias absent")
			return
		}
		if err := smokeUI(s, "part-search", &ui.ChangeEvent{Text: ""}); err != nil {
			checked <- err
			return
		}
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
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{
			{"simulation.pause", nil}, {"selection", &ui.ChangeEvent{SelectedID: "C1"}},
		} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				checked <- err
				return
			}
		}
		if _, ok := findNode(s.view(), "part-assumptions"); !ok {
			checked <- fmt.Errorf("native model assumptions absent")
			return
		}
		if err := smokeUI(s, "delete-object", nil); err != nil {
			checked <- err
			return
		}
		state := s.service.Snapshot()
		if len(state.Scope.Unavailable) != 1 {
			checked <- fmt.Errorf("native removed default source not marked unavailable")
			return
		}
		missing := state.Scope.Unavailable[0].ID
		if err := smokeUI(s, "remove-probe-"+missing, nil); err != nil {
			checked <- err
			return
		}
		if len(s.service.Snapshot().Scope.Unavailable) != 0 {
			checked <- fmt.Errorf("native unavailable-probe removal failed")
			return
		}
		checked <- nil
	}); err != nil {
		return err
	}
	return <-checked
}
