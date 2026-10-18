package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Query the real Cocoa elements and invoke their native action selectors. This
// covers mounted inspector bindings; it does not certify an external reader or
// explain why a separate CUA projection omits some of these controls.
func nativeLEDInspectorAccessibilitySmoke(ctx context.Context, a *app.Application, s *screen) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var controls []cogent.AccessibilityElement
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		inspection, err := cogent.InspectAccessibility(ctx, "main")
		if err != nil {
			return err
		}
		controls = nil
		for _, element := range inspection.Elements {
			if element.DomainID == "value-entry" || element.DomainID == "apply-value" {
				controls = append(controls, element)
			}
		}
		ready := inspection.Reachable && inspection.NativeWindow && len(controls) == 2
		for _, element := range controls {
			ready = ready && element.Enabled && element.Frame[2] > 0 && element.Frame[3] > 0 && element.HitTestID == element.ID
		}
		if ready {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("LED inspector native accessibility not ready: %w; controls=%+v", ctx.Err(), controls)
		case <-tick.C:
		}
	}
	focus, err := waitNativeAccessibilityFocus(ctx, "value-entry", func(ctx context.Context) (cogent.AccessibilityInspection, error) {
		return cogent.InspectAccessibility(ctx, "main")
	}, func(ctx context.Context, id string) (bool, error) {
		return cogent.PerformAccessibilityAction(ctx, "main", id, "focus", "")
	})
	if err != nil {
		return fmt.Errorf("native inspector focus: %w", err)
	}
	if accepted, err := cogent.PerformAccessibilityAction(ctx, "main", "value-entry", "set", "2.2k"); err != nil || !accepted {
		return fmt.Errorf("native inspector set: accepted=%t err=%v", accepted, err)
	}
	if state, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Selected == "R1" && state.Drawing.Value == "2.2k"
	}); err != nil {
		return fmt.Errorf("native inspector set did not complete: %w; selected=%q draft=%q", err, state.Selected, state.Drawing.Value)
	}
	if accepted, err := cogent.PerformAccessibilityAction(ctx, "main", "apply-value", "press", ""); err != nil || !accepted {
		return fmt.Errorf("native inspector Apply: accepted=%t err=%v", accepted, err)
	}
	state, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Circuit.Resistance == 2200 && state.Result != nil && !state.Result.Stale && math.Abs(state.Result.Current-0.001345) < 0.000002
	})
	if err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"event": "elab.led-inspector.accessibility.smoke", "passed": true, "initial_native_controls": controls, "focus_requests": focus.Requests, "focus_admissions": focus.Admissions, "observed_focus_id": focus.FocusedID, "observed_focus_generation": focus.Generation, "resistance_ohm": state.Circuit.Resistance, "current_a": state.Result.Current, "scope": "in-process Cocoa selectors, not external AX or VoiceOver acceptance"})
	log.Print(string(evidence))
	return nil
}

// Exercise controlled Change -> mounted refresh -> asynchronous Apply. This
// helper runs on the smoke worker, never blocks the owner waiting for its FIFO,
// and does not claim physical typing coverage (the CUA audit covers that).
// An empty value leaves the displayed draft untouched before Apply.
func nativeValueApplySmoke(ctx context.Context, a *app.Application, s *screen, id, value string, wantError bool) error {
	prepared := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "selection", &ui.ChangeEvent{SelectedID: id}); err != nil {
			prepared <- err
			return
		}
		if value == "" {
			value = s.service.Snapshot().Drawing.Value
			prepared <- nil
			return
		}
		prepared <- smokeUI(s, "value-entry", &ui.ChangeEvent{Text: value})
	}); err != nil {
		return err
	}
	select {
	case err := <-prepared:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Selected == id && state.Drawing.Value == value
	}); err != nil {
		return err
	}
	before := s.service.Snapshot()
	started := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		node, ok := findAction(s.view(), "apply-value")
		if !ok || node.Disabled {
			started <- fmt.Errorf("mounted Apply unavailable")
			return
		}
		node.Activate()
		started <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-started:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		type sample struct {
			pending bool
			message string
		}
		observed := make(chan sample, 1)
		if err := a.PostWait(ctx, func() { observed <- sample{s.pendingValueApply != nil, s.errorMessage} }); err != nil {
			return err
		}
		select {
		case got := <-observed:
			if !got.pending {
				if (got.message != "") != wantError {
					return fmt.Errorf("mounted Apply error=%q; wantError=%t", got.message, wantError)
				}
				if wantError && !nativeValueRefusalPreservesState(before, s.service.Snapshot()) {
					after := s.service.Snapshot()
					left, right := reflect.ValueOf(before), reflect.ValueOf(after)
					changed := []string{}
					for i := range left.NumField() {
						if !reflect.DeepEqual(left.Field(i).Interface(), right.Field(i).Interface()) {
							changed = append(changed, left.Type().Field(i).Name)
						}
					}
					return fmt.Errorf("refused mounted Apply changed accepted state: %v", changed)
				}
				_, err := s.service.Wait(ctx)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Showing the refusal message can resize the drawable canvas. That native
// observation advances the two sequencing revisions without accepting a value.
// Permit only those dimensions/revisions; retain every circuit, history, draft,
// selection, simulation and view field in the comparison. With no resize, even
// an unexplained sequencing change remains a failure.
func nativeValueRefusalPreservesState(before, after workbench.Snapshot) bool {
	if before.Viewport.Registered && after.Viewport.Registered &&
		(before.Viewport.Width != after.Viewport.Width || before.Viewport.Height != after.Viewport.Height) {
		before.Viewport.Width, before.Viewport.Height = after.Viewport.Width, after.Viewport.Height
		before.CommandRevision, before.Revision = after.CommandRevision, after.Revision
	}
	return reflect.DeepEqual(before, after)
}
