package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

func inspectOperation(t *testing.T, s *Service, id string) OperationView {
	t.Helper()
	raw, _ := json.Marshal(IDParams{ID: id})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "inspect", Command: "operation.inspect", Params: raw}, "test")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	return response.Result.(OperationView)
}
func awaitOperation(t *testing.T, s *Service, id string) OperationView {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		view := inspectOperation(t, s, id)
		if !operationActive(view.Operation.Status) {
			return view
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("verification did not finish")
	return OperationView{}
}
func verify(t *testing.T, s *Service, p VerifyParams, key string) (Response, OperationView) {
	t.Helper()
	response := call(t, s, "simulation.verify", p, key)
	id := response.Result.(Snapshot).LastOperation
	if id == "" {
		t.Fatal("missing accepted operation ID")
	}
	view := awaitOperation(t, s, id)
	if view.Operation.Status != "completed" {
		t.Fatalf("verify failed %+v", view.Operation)
	}
	return response, view
}
func TestVerificationRCReferenceIndependentStateAndMetadata(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.03}, "live-time")
	before := s.Snapshot()
	duration := .01
	response, view := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	if view.Metadata == nil || view.Metadata.Engine != RCEngineID || view.Metadata.Duration != .01 || view.Metadata.SampleCount != 11 || !view.Operation.SourceUnchanged {
		t.Fatal("missing verified input identity")
	}
	want := 5 * (1 - math.Exp(-1))
	last := s.operations[view.Operation.ID].record.rows[10]
	if math.Abs(last.Values[0]-want) > 1e-12 {
		t.Fatalf("verify RC reference got %.15g", last.Values[0])
	}
	after := s.Snapshot()
	if after.Transient.Sample != before.Transient.Sample || after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID || after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || after.CanUndo != before.CanUndo {
		t.Fatal("verify changed live/source state")
	}
	retry := call(t, s, "simulation.verify", VerifyParams{Duration: &duration}, "verify")
	if !retry.Duplicate || retry.Result.(Snapshot).LastOperation != response.Result.(Snapshot).LastOperation || len(s.operationOrder) != 1 {
		t.Fatal("retry duplicated computation")
	}
	view.Metadata.Graph.Components[0].Value = 999
	view.Metadata.Probes[0].Label = "bad"
	fresh := inspectOperation(t, s, view.Operation.ID)
	if fresh.Metadata.Graph.Components[0].Value == 999 || fresh.Metadata.Probes[0].Label == "bad" {
		t.Fatal("record metadata aliases client")
	}
	raw, _ := json.Marshal(VerificationPageParams{ID: view.Operation.ID, Limit: 5})
	pageResponse := s.Dispatch(context.Background(), Request{Version: 1, ID: "page", Command: "operation.samples", Params: raw}, "test")
	page := pageResponse.Result.(VerificationPage)
	if len(page.Rows) != 5 || page.NextOffset != 5 || page.Total != 11 {
		t.Fatal("sample page lost bounds")
	}
	page.Rows[0].Values[0] = 999
	pageResponse = s.Dispatch(context.Background(), Request{Version: 1, ID: "page2", Command: "operation.samples", Params: raw}, "test")
	if pageResponse.Result.(VerificationPage).Rows[0].Values[0] == 999 {
		t.Fatal("page values alias record")
	}
}
func TestVerificationSourceCapturedThroughEditsAndExplicitStartup(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	initial := 2.
	response := call(t, s, "simulation.verify", VerifyParams{Duration: &duration, InitialVoltage: &initial, InputMode: "discharge", ProbeIDs: []string{"v:C1", "i:C1"}}, "capture")
	id := response.Result.(Snapshot).LastOperation
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "intervene")
	view := awaitOperation(t, s, id)
	if view.Operation.Status != "completed" || view.Operation.SourceUnchanged || view.Metadata.Settings.InitialVoltage != 2 {
		t.Fatal("captured scenario changed with editor")
	}
	rows := s.operations[id].record.rows
	want := 2 * math.Exp(-2)
	if rows[0].Values[0] != 2 || math.Abs(rows[len(rows)-1].Values[0]-want) > 1e-12 || rows[len(rows)-1].Values[1] >= 0 {
		t.Fatal("startup/discharge override not applied")
	}
	component, _ := s.graph.Component("C1")
	if component.InitialVoltage != nil || s.transientSettings.InitialVoltage != 0 {
		t.Fatal("verify override modified source")
	}
}
func TestVerificationNPNDCAndPulseEdges(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.npn", ReplaceParams{}, "npn")
	before := settled(t, s)
	_, view := verify(t, s, VerifyParams{}, "dc")
	if view.Metadata.NPN == nil || view.Metadata.SampleCount != 1 || view.Metadata.Duration != 0 || math.Abs(view.Metadata.NPN.CollectorCurrent-before.NPN.CollectorCurrent) > 1e-12 {
		t.Fatal("NPN DC verify differs from live")
	}
	call(t, s, "example.npn-pulse", ReplaceParams{DiscardChanges: true}, "pulse")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	duration := .1
	_, view = verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive", "i:Q1.collector"}}, "pulse-verify")
	rows := s.operations[view.Operation.ID].record.rows
	left, right := false, false
	for _, row := range rows {
		if math.Abs(row.Time-.05) < 1e-12 {
			if row.Values[0] == 5 && row.Values[1] > .002 {
				left = true
			}
			if row.Values[0] == 0 && row.Values[1] < 1e-8 {
				right = true
			}
		}
	}
	if !left || !right || view.Metadata.NPN.State != "saturation" {
		t.Fatal("pulse edges/final phase were lost")
	}
}
func TestVerificationFlasherStartupAndNumericalParity(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	duration := .1
	_, view := verify(t, s, VerifyParams{Duration: &duration, Capacitors: []CapacitorInitial{{ID: "C1", Voltage: .1}}, ProbeIDs: []string{"i:D1", "i:D2", "v:C1", "v:C2"}}, "flasher")
	if view.Metadata.Mixed == nil || len(view.Metadata.Mixed.Capacitors) != 2 || view.Metadata.Mixed.KCLResidual > 1e-9 {
		t.Fatal("mixed verification missing conserved state")
	}
	call(t, s, "simulation.step", StepParams{.1}, "live-step")
	live := s.Snapshot().Mixed
	for i, c := range live.Capacitors {
		if math.Abs(c.Voltage-view.Metadata.Mixed.Capacitors[i].Voltage) > 1e-8 {
			t.Fatal("offline/live coupled kernels differ")
		}
	}
}
func TestVerificationCancellationFailureAndAdmissionBounds(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .1
	s.mu.Lock()
	operation := &verificationOperation{summary: OperationSummary{ID: "queued-test", Status: "queued"}}
	operation.ctx, operation.cancel = context.WithCancel(context.Background())
	s.operations = map[string]*verificationOperation{"queued-test": operation}
	s.operationOrder = []string{"queued-test"}
	s.mu.Unlock()
	call(t, s, "operation.cancel", IDParams{ID: "queued-test"}, "cancel")
	input := verificationInput{}
	s.runVerification(operation, input)
	if inspectOperation(t, s, "queued-test").Operation.Status != "canceled" || operation.record != nil {
		t.Fatal("queued cancellation did not terminate without data")
	}
	excessive := 10.
	refuse(t, s, "simulation.verify", VerifyParams{Duration: &excessive}, "invalid-params")
	before := s.Snapshot()
	refuse(t, s, "simulation.verify", VerifyParams{Duration: &duration, Capacitors: []CapacitorInitial{{ID: "C1", Voltage: 13}}}, "invalid-params")
	if !reflect.DeepEqual(s.Snapshot(), before) {
		t.Fatal("invalid verification mutated state")
	}
	s2 := pausedFlasher(t)
	defer s2.Close()
	tiny := 1e-12
	response := call(t, s2, "simulation.verify", VerifyParams{Duration: &tiny}, "failed-job")
	view := awaitOperation(t, s2, response.Result.(Snapshot).LastOperation)
	if view.Operation.Status != "failed" || view.Operation.Error == nil || view.Metadata == nil || view.Metadata.SampleCount != 0 || view.Metadata.Mixed != nil {
		t.Fatal("numerical failure presented a valid record")
	}
}

func TestCancelRunningVerificationAndClose(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	duration := 5.
	response := call(t, s, "simulation.verify", VerifyParams{Duration: &duration}, "long-run")
	id := response.Result.(Snapshot).LastOperation
	call(t, s, "operation.cancel", IDParams{ID: id}, "cancel-real")
	view := awaitOperation(t, s, id)
	if view.Operation.Status != "canceled" || s.operations[id].record != nil {
		t.Fatal("real verification did not cancel without publishing partial data")
	}
	response = call(t, s, "simulation.verify", VerifyParams{Duration: &duration}, "close-run")
	id = response.Result.(Snapshot).LastOperation
	s.Close()
	if inspectOperation(t, s, id).Operation.Status != "canceled" {
		t.Fatal("close left an active operation")
	}
}
