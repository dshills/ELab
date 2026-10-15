package main

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeValueDomainsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if response := s.invoke("example.rc", workbench.ReplaceParams{DiscardChanges: true}); response.Error != nil {
			result <- fmt.Errorf("value smoke RC: %s", response.Error.Message)
			return
		}
		result <- smokeUI(s, "simulation.pause", nil)
	}); err != nil {
		return err
	}
	if err := <-result; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"selection", &ui.ChangeEvent{SelectedID: "V1"}}, {"value-entry", &ui.ChangeEvent{Text: "500 mV"}}, {"apply-value", nil}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				result <- err
				return
			}
		}
		state := s.service.Snapshot()
		part, _ := (workbench.Graph{Components: state.Components}).Component("V1")
		label, ok := findNode(s.view(), "value-domain-range")
		if part.Value != .5 || !ok || label.Text != "0–12 V" {
			result <- fmt.Errorf("native source value/range not shared")
			return
		}
		if err := smokeUI(s, "value-entry", &ui.ChangeEvent{Text: "1e-999 V"}); err != nil {
			result <- err
			return
		}
		before := s.service.Snapshot()
		apply, _ := findAction(s.view(), "apply-value")
		apply.Activate()
		if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
			result <- fmt.Errorf("native underflow silently committed")
			return
		}
		if err := smokeUI(s, "value-entry", &ui.ChangeEvent{Text: "750 mV"}); err != nil {
			result <- err
			return
		}
		if err := smokeUI(s, "apply-value", nil); err != nil {
			result <- err
			return
		}
		part, _ = (workbench.Graph{Components: s.service.Snapshot().Components}).Component("V1")
		if part.Value != .75 {
			result <- fmt.Errorf("native value correction failed")
			return
		}
		result <- nil
	}); err != nil {
		return err
	}
	return <-result
}
