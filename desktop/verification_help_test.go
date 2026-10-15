package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func awaitNativeFailedVerification(t *testing.T, service *workbench.Service, id string) workbench.OperationView {
	t.Helper()
	raw, _ := json.Marshal(workbench.IDParams{ID: id})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response := service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "failure-inspect", Command: "operation.inspect", Params: raw}, "reader")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		view := response.Result.(workbench.OperationView)
		if view.Operation.Status == "failed" {
			return view
		}
		if view.Operation.Status == "completed" || view.Operation.Status == "canceled" {
			t.Fatal("expected numerical failure", view.Operation)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("verification failure did not become terminal")
	return workbench.OperationView{}
}

func TestNativeVerificationFailureGuidanceSharesDisclosureAndCapturedIdentity(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rectifier", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "simulation.wait", struct{}{})
	uiAction(t, s, "verify-duration", &ui.ChangeEvent{Text: "1 ms"})
	uiAction(t, s, "verify-start", nil)
	kept := completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	before := s.service.Snapshot()
	text := strconv.FormatFloat(math.Nextafter(.001, math.Inf(1)), 'g', -1, 64) + " s"
	uiAction(t, s, "verify-duration", &ui.ChangeEvent{Text: text})
	uiAction(t, s, "verify-start", nil)
	failed := awaitNativeFailedVerification(t, s.service, s.service.Snapshot().LastOperation)
	if failed.Operation.FailureHelp == nil {
		t.Fatal("failed native operation has no shared guidance")
	}
	s.app.Do(func() {
		root := s.view()
		summary, ok := findNode(root, "verify-failure-summary")
		if !ok || summary.Text != failed.Operation.FailureHelp.Summary {
			t.Fatal("native summary differs from agent operation guidance")
		}
		details, ok := findNode(root, "verify-failure-details")
		if !ok || details.Kind != ui.Disclosure || details.Checked {
			t.Fatal("failure technical reason is not behind the ordinary disclosure")
		}
	})
	uiAction(t, s, "verify-failure-details", &ui.ChangeEvent{Checked: true})
	after := s.service.Snapshot()
	if !after.DiagnosticPresentation.DetailsVisible || before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("failure/disclosure replaced live state, kept membership or history")
	}
	agentAction(t, s.service, "diagnostics.details.set", workbench.DiagnosticDetailsParams{Visible: false})
	s.app.Do(func() {
		details, _ := findNode(s.view(), "verify-failure-details")
		if details.Checked {
			t.Fatal("agent disclosure state not reflected in verification UI")
		}
	})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "200 ohm"})
	s.app.Do(func() {
		summary, _ := findNode(s.view(), "verify-failure-summary")
		if !strings.Contains(summary.Text, "earlier captured") {
			t.Fatal("old verification error is presented as current work")
		}
	})
	uiAction(t, s, "verify-operation", &ui.ChangeEvent{SelectedID: kept.Operation.ID})
	s.app.Do(func() {
		if _, ok := findNode(s.view(), "verify-failure-summary"); ok {
			t.Fatal("selecting earlier kept result retained another operation's failure")
		}
	})
}

func TestNativeVerificationFailureSmokeWaitsForOwnerCallbacks(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- nativeVerificationFailureSmoke(ctx, s.app, s) }()
	for {
		s.app.Drain()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-ctx.Done():
			t.Fatal("owned verification failure smoke did not finish", ctx.Err())
		default:
		}
		time.Sleep(time.Millisecond)
	}
}
