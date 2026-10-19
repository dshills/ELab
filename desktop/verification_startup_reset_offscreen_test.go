//go:build offscreen

package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestMountedVerificationProjectStartupRestoresCoilCurrent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareInductorVerification(t, s.service, "rlc")
	before := s.service.Snapshot()
	m := mountedVerificationStart(t, s)
	typeMountedCursor(t, m, "verify-duration", "8 ms")
	clickMountedControl(t, m, "verify-project-startup")
	reset := s.service.Snapshot()
	expected := workbench.VerificationSettings{Duration: .008, InputMode: "auto", Capacitors: []workbench.CapacitorInitial{}}
	if s.errorMessage != "" || !reflect.DeepEqual(reset.VerificationSettings, expected) {
		t.Fatal("one reset click did not settle duration and clear every startup override", s.errorMessage, reset.VerificationSettings)
	}
	clickMountedControl(t, m, "verify-start")
	captured := completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	page := verificationSamplePage(t, s.service, captured.Operation.ID)
	current := -1
	for i, probe := range page.Probes {
		if probe.ID == "i:L1" {
			current = i
		}
	}
	if current < 0 || math.Abs(page.Rows[0].Values[current]-(-.02)) > 1e-12 {
		t.Fatal("mounted reset did not capture saved signed coil startup")
	}
	reference := workbench.New()
	t.Cleanup(reference.Close)
	prepareInductorVerification(t, reference, "rlc")
	agentAction(t, reference, "verification.settings.set", expected)
	duration := .008
	agentAction(t, reference, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	want := completedVerification(t, reference, reference.Snapshot().LastOperation)
	other := verificationSamplePage(t, reference, want.Operation.ID)
	if !reflect.DeepEqual(captured.Metadata.Graph, want.Metadata.Graph) || !reflect.DeepEqual(captured.Metadata.Statistics, want.Metadata.Statistics) || !reflect.DeepEqual(page.Probes, other.Probes) || !reflect.DeepEqual(page.Rows, other.Rows) {
		t.Fatal("mounted reset capture differs from independent public agent")
	}
	after := s.service.Snapshot()
	if before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("mounted startup reset changed live experiment or saved startup")
	}
}
