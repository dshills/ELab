package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func TestNativeStepUsesImmediateSharedAdmissionAndReportsCompletion(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "example.rc", nil)
	agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
	for _, service := range []*workbench.Service{s.service, agent} {
		agentAction(t, service, "simulation.pause", struct{}{})
		agentAction(t, service, "simulation.restart", struct{}{})
	}
	var response workbench.Response
	s.app.Do(func() { response = s.invoke("simulation.step.start", workbench.StepParams{Duration: .01}) })
	if response.Error != nil || response.Result.(workbench.Snapshot).StepWork.Status != "queued" {
		t.Fatal("native start waited for numerical completion")
	}
	agentAction(t, agent, "simulation.step.start", workbench.StepParams{Duration: .01})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := s.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if u.StepWork.Status != "completed" || !reflect.DeepEqual(u.Transient, a.Transient) || u.StepWork.Actor != "user" {
		t.Fatal("native/agent background step differs")
	}
	status, ok := findNode(s.view(), "step-work-status")
	if !ok || !strings.Contains(status.Text, "completed") {
		t.Fatal("step status absent")
	}
}
