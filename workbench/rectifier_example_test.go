package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func pausedRectifier(t *testing.T) *Service {
	t.Helper()
	s := New()
	call(t, s, "example.rectifier", ReplaceParams{}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "startup")
	return s
}

// This scalar ODE uses the inverse current-domain junction equation and RK4,
// independently of ELab's nodal Newton/adaptive backward-Euler integration.
func rectifierReferenceStep(at, v, dt, capacitance float64) float64 {
	slope := func(t, output float64) float64 {
		input := 5 * math.Sin(20*math.Pi*t)
		return (referenceSiliconCurrent(input-output, 100) - output/10000) / capacitance
	}
	a := slope(at, v)
	b := slope(at+dt/2, v+dt*a/2)
	c := slope(at+dt/2, v+dt*b/2)
	d := slope(at+dt, v+dt*c)
	return v + dt*(a+2*b+2*c+d)/6
}

func TestRectifierExampleAgainstIndependentTransientReference(t *testing.T) {
	s := pausedRectifier(t)
	defer s.Close()
	initial := s.Snapshot()
	if initial.Example.Info.ID != "rectifier" || initial.Selected != "C1" || initial.Analysis != WaveformAnalysis || initial.Transient.Sample.Time != 0 || initial.Mixed.Capacitors[0].Voltage != 0 {
		t.Fatal("rectifier startup/configuration", initial.Diagnostics)
	}
	for _, id := range []string{"i:D1", "v:C1", "v:V1.positive"} {
		found := false
		for _, p := range initial.Scope.Traces {
			found = found || p.Probe.ID == id
		}
		if !found {
			t.Fatal("rectifier default scope omits", id)
		}
	}
	ref, maximumError := 0., 0.
	for ms := 1; ms <= 200; ms++ {
		for j := 0; j < 200; j++ {
			at := float64((ms-1)*200+j) * 5e-6
			ref = rectifierReferenceStep(at, ref, 5e-6, 10e-6)
		}
		call(t, s, "simulation.step", StepParams{Duration: .001}, fmt.Sprintf("sample-%d", ms))
		state := s.Snapshot()
		got := state.Mixed.Capacitors[0].Voltage
		maximumError = math.Max(maximumError, math.Abs(got-ref))
		if math.Abs(got-ref) > .003 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatalf("rectifier at %g: output %g, RK4 %g, residuals %g/%g", state.Transient.Sample.Time, got, ref, state.Mixed.KCLResidual, state.Mixed.PowerResidual)
		}
		if ms == 75 && (got < 1 || math.Abs(state.Mixed.Diodes[0].Current+1e-9) > 1e-12) {
			t.Fatal("negative input fails reverse isolation/discharge")
		}
	}
	t.Logf("maximum independent reference error: %.9g V", maximumError)
	before := s.Snapshot()
	duration := .2
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "v:V1.positive", "i:D1"}}, "verify")
	after := s.Snapshot()
	if record.Metadata.Engine != WaveformEngineID || record.Metadata.DiodeModel == nil || record.Metadata.WaveformModel == nil || math.Abs(record.Metadata.Mixed.Capacitors[0].Voltage-before.Mixed.Capacitors[0].Voltage) > 1e-9 || !reflect.DeepEqual(before.Transient, after.Transient) || schematicDocumentState(before.Document) != schematicDocumentState(after.Document) || after.Document.RecordingCount != before.Document.RecordingCount+1 {
		t.Fatal("rectifier live/Verify mismatch or live state changed")
	}
	path := filepath.Join(t.TempDir(), "rectifier.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	reopened := settled(t, fresh)
	if reopened.Example.Info.ID != "rectifier" || reopened.Example.Modified || !reflect.DeepEqual(reopened.Components, initial.Components) || reopened.DiodeModel == nil || reopened.WaveformModel == nil {
		t.Fatal("rectifier source/template/models changed on reopening")
	}
}

func TestRectifierLargerCapacitorReducesSettledRipple(t *testing.T) {
	ripple := func(capacitance string) float64 {
		s := pausedRectifier(t)
		defer s.Close()
		if capacitance != "10u" {
			call(t, s, "component.set", ValueParams{ID: "C1", Value: capacitance}, "capacitance")
			call(t, s, "simulation.restart", struct{}{}, "restart-edited")
		}
		for i := 0; i < 20; i++ {
			call(t, s, "simulation.step", StepParams{Duration: .1}, fmt.Sprintf("settle-%d", i))
			if state := s.Snapshot(); state.StepWork.Status != "completed" {
				t.Fatalf("rectifier %s step %d: %+v, diagnostics %+v", capacitance, i, state.StepWork, state.Diagnostics)
			}
		}
		state := s.Snapshot()
		low, high := math.Inf(1), math.Inf(-1)
		for _, sample := range state.Transient.Samples {
			if sample.Time >= 1.9-1e-9 {
				low, high = math.Min(low, sample.CapacitorVoltage), math.Max(high, sample.CapacitorVoltage)
			}
		}
		if low < 1 || high > 5 || !finite(high-low) {
			t.Fatal("implausible settled rectifier output", low, high)
		}
		return high - low
	}
	base, larger := ripple("10u"), ripple("47u")
	if base < .5 || larger > base*.5 || larger <= 0 {
		t.Fatal("larger smoothing capacitor fails ripple experiment", base, larger)
	}
	t.Logf("settled ripple: 10 µF %.9g V; 47 µF %.9g V", base, larger)
}
