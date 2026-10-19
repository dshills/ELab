package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func prepareInductorVerification(t *testing.T, service *workbench.Service, example string) {
	t.Helper()
	agentAction(t, service, "example.load", workbench.ExampleLoadParams{ID: example})
	agentAction(t, service, "simulation.pause", struct{}{})
	agentAction(t, service, "inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "-20 mA"})
	agentAction(t, service, "simulation.restart", struct{}{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	agentAction(t, service, "simulation.step", workbench.StepParams{Duration: .002})
	if _, err := service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	cfg := workbench.VerificationSettings{Duration: .005, Step: .0005, InputMode: "high", Inductors: []workbench.InductorInitial{{ID: "L1", Current: .03}}, Capacitors: []workbench.CapacitorInitial{}}
	initial := 1.
	cfg.InitialVoltage = &initial
	if example == "rlc" {
		cfg.Capacitors = []workbench.CapacitorInitial{{ID: "C1", Voltage: 2}}
	}
	agentAction(t, service, "verification.settings.set", cfg)
}

func verificationSamplePage(t *testing.T, service *workbench.Service, id string) workbench.VerificationPage {
	t.Helper()
	raw, _ := json.Marshal(workbench.VerificationPageParams{ID: id, Limit: 200})
	r := service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "samples", Command: "operation.samples", Params: raw}, "test")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	page := r.Result.(workbench.VerificationPage)
	if len(page.Rows) != page.Total || page.NextOffset < page.Total {
		t.Fatal("fixture must inspect the entire capture")
	}
	return page
}

func TestNativeProjectStartupClearsInductorScenarios(t *testing.T) {
	for _, example := range []string{"rl", "rlc"} {
		t.Run(example, func(t *testing.T) {
			user, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			reference := workbench.New()
			t.Cleanup(reference.Close)
			for _, service := range []*workbench.Service{user.service, reference} {
				prepareInductorVerification(t, service, example)
			}
			before := user.service.Snapshot()
			uiAction(t, user, "verify-project-startup", nil)
			reset := user.service.Snapshot()
			uiAction(t, user, "verify-start", nil)
			got := completedVerification(t, user.service, user.service.Snapshot().LastOperation)
			page := verificationSamplePage(t, user.service, got.Operation.ID)
			current := -1
			for i, p := range page.Probes {
				if p.ID == "i:L1" {
					current = i
				}
			}
			if current < 0 || len(page.Rows) == 0 {
				t.Fatal("coil current probe missing")
			}
			if len(reset.VerificationSettings.Inductors) != 0 || math.Abs(page.Rows[0].Values[current]-(-.02)) > 1e-12 {
				t.Fatalf("Use project startup retained scenario current: overrides=%+v captured_current=%g A; saved project requires -0.02 A", reset.VerificationSettings.Inductors, page.Rows[0].Values[current])
			}
			expected := workbench.VerificationSettings{Duration: .005, InputMode: "auto", Capacitors: []workbench.CapacitorInitial{}}
			if !reflect.DeepEqual(reset.VerificationSettings, expected) {
				t.Fatal("startup reset did not preserve only duration", reset.VerificationSettings)
			}
			agentAction(t, reference, "verification.settings.set", expected)
			duration := .005
			agentAction(t, reference, "simulation.verify", workbench.VerifyParams{Duration: &duration})
			want := completedVerification(t, reference, reference.Snapshot().LastOperation)
			other := verificationSamplePage(t, reference, want.Operation.ID)
			if !reflect.DeepEqual(got.Metadata.Graph, want.Metadata.Graph) || got.Metadata.Settings != want.Metadata.Settings || !reflect.DeepEqual(got.Metadata.Statistics, want.Metadata.Statistics) || !reflect.DeepEqual(page.Probes, other.Probes) || !reflect.DeepEqual(page.Rows, other.Rows) {
				t.Fatal("native startup reset capture differs from public agent")
			}
			after := user.service.Snapshot()
			if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || schematicDocumentState(before.Document) != schematicDocumentState(after.Document) {
				t.Fatal("startup reset/capture changed live circuit, saved startup or history")
			}
			if example == "rl" {
				for _, row := range page.Rows {
					want := .05 + (-.02-.05)*math.Exp(-row.Time/.01)
					if math.Abs(row.Values[current]-want) > 1e-5 {
						t.Fatal("reset RL current differs from independent equation", row, want)
					}
				}
			}
		})
	}
}

func TestVerificationStartupResetPreservesAgentDuringSettlement(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareInductorVerification(t, s.service, "rlc")
	var finish func(error)
	first := true
	settle := func(id string, done func(error)) error {
		if first {
			first = false
			if id != "verify-duration" {
				t.Fatal("unexpected first field", id)
			}
			finish = done
		} else {
			done(nil)
		}
		return nil
	}
	s.settleField, s.settleNumber = settle, settle
	button := verificationControl(t, s, "verify-project-startup")
	s.app.Do(button.Activate)
	if finish == nil {
		t.Fatal("reset did not wait for field settlement")
	}
	cfg := s.service.Snapshot().VerificationSettings
	cfg.Duration = .006
	cfg.Inductors = []workbench.InductorInitial{{ID: "L1", Current: .04}}
	agentAction(t, s.service, "verification.settings.set", cfg)
	before := s.service.Snapshot()
	s.app.Do(func() { finish(nil); finish(nil) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("pending reset overwrote the agent's newer scenario", s.errorMessage)
	}
	s.settleField, s.settleNumber = nil, nil
	uiAction(t, s, "verify-project-startup", nil)
	expected := workbench.VerificationSettings{Duration: .006, InputMode: "auto", Capacitors: []workbench.CapacitorInitial{}}
	if !reflect.DeepEqual(s.service.Snapshot().VerificationSettings, expected) {
		t.Fatal("fresh reset did not clear scenario overrides and preserve duration")
	}
}
