package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func completedVerification(t *testing.T, s *workbench.Service, id string) workbench.OperationView {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := json.Marshal(workbench.IDParams{ID: id})
		response := s.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "inspect", Command: "operation.inspect", Params: raw}, "test")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		view := response.Result.(workbench.OperationView)
		switch view.Operation.Status {
		case "completed":
			return view
		case "failed", "canceled":
			t.Fatalf("verification terminated %+v", view.Operation)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("verification timed out")
	return workbench.OperationView{}
}
func TestNativeVerifyControlsMatchAgentScenarios(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "example.rc", nil)
	agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	uiAction(t, user, "verify-duration", &ui.ChangeEvent{Text: "20 ms"})
	settings := agent.Snapshot().VerificationSettings
	settings.Duration = .02
	agentAction(t, agent, "verification.settings.set", settings)
	uiAction(t, user, "verify-mode", &ui.ChangeEvent{SelectedID: "discharge"})
	settings.InputMode = "discharge"
	agentAction(t, agent, "verification.settings.set", settings)
	uiAction(t, user, "verify-cap-C1", &ui.ChangeEvent{Number: 2})
	settings.Capacitors = []workbench.CapacitorInitial{{ID: "C1", Voltage: 2}}
	agentAction(t, agent, "verification.settings.set", settings)
	before := user.service.Snapshot()
	uiAction(t, user, "verify-start", nil)
	duration := .02
	agentAction(t, agent, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	u := completedVerification(t, user.service, user.service.Snapshot().LastOperation)
	a := completedVerification(t, agent, agent.Snapshot().LastOperation)
	if !reflect.DeepEqual(u.Metadata.Graph, a.Metadata.Graph) || u.Metadata.Settings != a.Metadata.Settings || !reflect.DeepEqual(u.Metadata.Statistics, a.Metadata.Statistics) || u.Metadata.Engine != a.Metadata.Engine {
		t.Fatal("native/agent verification inputs or values differ")
	}
	after := user.service.Snapshot()
	if after.Transient.Sample != before.Transient.Sample || after.ElectricalRevision != before.ElectricalRevision || schematicDocumentState(after.Document) != schematicDocumentState(before.Document) {
		t.Fatal("native verify touched live state")
	}
	if after.Verification == nil || math.Abs(after.Verification.Readings[0].Value-2*math.Exp(-2)) > 1e-12 {
		t.Fatal("recorded native reading missing")
	}
	root := user.view()
	if err := ui.Validate(root); err != nil {
		t.Fatal(err)
	}
	if _, ok := findNode(root, "verify-recorded"); !ok {
		t.Fatal("record status not visible")
	}
	uiAction(t, user, "verify-project-startup", nil)
	settings.InitialVoltage = nil
	settings.Capacitors = []workbench.CapacitorInitial{}
	settings.InputMode = "auto"
	settings.Step = 0
	agentAction(t, agent, "verification.settings.set", settings)
	if !reflect.DeepEqual(user.service.Snapshot().VerificationSettings, agent.Snapshot().VerificationSettings) {
		t.Fatal("startup reset differs")
	}
	uiAction(t, user, "verify-operation", &ui.ChangeEvent{SelectedID: "__none"})
	agentAction(t, agent, "operation.select", workbench.IDParams{ID: ""})
	if user.service.Snapshot().Verification != nil || agent.Snapshot().Verification != nil {
		t.Fatal("clear recorded selection did not hide preview")
	}
}
