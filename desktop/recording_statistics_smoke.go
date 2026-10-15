package main

import (
	"context"
	"fmt"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeRecordingStatisticsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	completed := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		before := s.service.Snapshot()
		r := before.Recording
		if r == nil || r.Duration < .02 || len(r.Scope.Available) == 0 {
			completed <- fmt.Errorf("interval measurement smoke requires a transient capture")
			return
		}
		for _, action := range []struct {
			id string
			e  ui.ChangeEvent
		}{{"recorded-statistics-probe", ui.ChangeEvent{SelectedID: r.Scope.Available[0].ID}}, {"recorded-statistics-start", ui.ChangeEvent{Text: "1 ms"}}, {"recorded-statistics-end", ui.ChangeEvent{Text: "20 ms"}}} {
			if err := smokeUI(s, action.id, &action.e); err != nil {
				completed <- err
				return
			}
		}
		after := s.service.Snapshot()
		result := after.Recording.Statistics
		if result == nil || result.Settings.Start != .001 || result.Settings.End != .02 || result.Segments == 0 || result.RMS < 0 || result.ACRMS < 0 || after.Document != before.Document || after.RunID != before.RunID || after.Transient.Sample != before.Transient.Sample {
			completed <- fmt.Errorf("interval measurement readout or live-state preservation")
			return
		}
		if _, ok := findNode(s.view(), "recorded-statistics-values"); !ok {
			completed <- fmt.Errorf("interval measurement labels absent")
			return
		}
		completed <- smokeUI(s, "recorded-statistics-probe", &ui.ChangeEvent{SelectedID: "__none"})
	}); err != nil {
		return err
	}
	select {
	case err := <-completed:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
