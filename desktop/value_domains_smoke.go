package main

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
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
	select {
	case err := <-result:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := nativeValueApplySmoke(ctx, a, s, "V1", "500 mV", false); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		part, _ := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("V1")
		label, ok := findNode(s.view(), "value-domain-range")
		if part.Value != .5 || !ok || label.Text != "0–12 V" {
			result <- fmt.Errorf("native source value/range not shared")
			return
		}
		result <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-result:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := nativeValueApplySmoke(ctx, a, s, "V1", "1e-999 V", true); err != nil {
		return err
	}
	if err := nativeValueApplySmoke(ctx, a, s, "V1", "750 mV", false); err != nil {
		return err
	}
	part, _ := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("V1")
	if part.Value != .75 {
		return fmt.Errorf("native value correction failed")
	}
	const precise = "1.234567890123456 µF"
	if err := nativeValueApplySmoke(ctx, a, s, "C1", precise, false); err != nil {
		return err
	}
	before := s.service.Snapshot()
	if err := nativeValueApplySmoke(ctx, a, s, "C1", "", false); err != nil {
		return err
	}
	after := s.service.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		return fmt.Errorf("untouched native value Apply changed stored circuit")
	}
	return nil
}
