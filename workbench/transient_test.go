package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func pausedRC(t *testing.T) *Service {
	t.Helper()
	s := New()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart-paused")
	return s
}
func TestRCAnalyticalReferenceChargeDischargeAndPulseEdges(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	if state := s.Snapshot(); state.Transient.Sample.Time != 0 || state.Transient.Sample.Voltage != 0 || state.Transient.TimeConstant != .01 || state.Result != nil {
		t.Fatal("RC startup is not a zero-voltage transient")
	}
	call(t, s, "simulation.step", StepParams{.01}, "10ms")
	state := s.Snapshot()
	if math.Abs(state.Transient.Sample.Voltage-3.1606027941427883) > 1e-12 {
		t.Fatalf("10 ms analytical reference: %.12g", state.Transient.Sample.Voltage)
	}
	call(t, s, "simulation.step", StepParams{.04}, "50ms")
	state = s.Snapshot()
	if math.Abs(state.Transient.Sample.Voltage-5*(1-math.Exp(-5))) > 1e-12 || state.Transient.Sample.Input != 0 {
		t.Fatal("pulse falling edge missed")
	}
	call(t, s, "simulation.step", StepParams{.05}, "100ms")
	state = s.Snapshot()
	if math.Abs(state.Transient.Sample.Voltage-5*(1-math.Exp(-5))*math.Exp(-5)) > 1e-12 || state.Transient.Sample.Input != 5 {
		t.Fatal("pulse discharge or rising edge missed")
	}
	sample := state.Transient.Sample
	if sample.Current != (sample.Input-sample.Voltage)/10000 || sample.Energy != .5*1e-6*sample.Voltage*sample.Voltage {
		t.Fatal("current/energy readings violate RC model")
	}
	// Refinement must not change an exact first-order response. The falling edge
	// deliberately falls between sample boundaries in this second configuration.
	other := pausedRC(t)
	defer other.Close()
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = .0007
	call(t, other, "simulation.transient.set", cfg, "refine")
	call(t, other, "simulation.restart", struct{}{}, "refine-start")
	call(t, other, "simulation.step", StepParams{.1}, "refined-100ms")
	if math.Abs(other.Snapshot().Transient.Sample.Voltage-sample.Voltage) > 1e-12 {
		t.Fatal("sample refinement changed physical response")
	}
	call(t, s, "simulation.step", StepParams{.5}, "cycles")
	state = s.Snapshot()
	expected := 5 / (1 + math.Exp(5))
	if math.Abs(state.Transient.Sample.Voltage-expected) > 1e-12 {
		t.Fatal("periodic 33.464 mV minimum reference mismatch")
	}
	if len(state.Transient.Samples) > MaxTraceSamples {
		t.Fatal("live history unbounded")
	}
}
func TestRCInputPauseResumePresentationAndRestartRules(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.02}, "charge")
	original := s.Snapshot()
	run := original.RunID
	voltage := original.Transient.Sample.Voltage
	call(t, s, "simulation.input.set", InputParams{"discharge"}, "switch-paused")
	if state := s.Snapshot(); state.Transient.Sample.Voltage != voltage || state.Transient.PendingMode != "discharge" || state.Transient.Sample.Time != .02 {
		t.Fatal("paused event advanced or lost capacitor state")
	}
	call(t, s, "simulation.step", StepParams{.01}, "discharge-step")
	if state := s.Snapshot(); math.Abs(state.Transient.Sample.Voltage-voltage*math.Exp(-1)) > 1e-12 || state.RunID != run || len(state.Transient.Events) != 2 {
		t.Fatal("input actuation restarted or failed to discharge")
	}
	call(t, s, "component.move", MoveParams{ID: "C1", Position: Point{240, 0}}, "move")
	call(t, s, "component.rotate", IDParams{"C1"}, "rotate")
	if s.Snapshot().RunID != run || s.Snapshot().Transient.Stale {
		t.Fatal("geometry restarted RC")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume")
	time.Sleep(120 * time.Millisecond)
	advanced := s.Snapshot()
	if advanced.Transient.Sample.Time <= .03 || advanced.RunID != run || advanced.Transient.Sample.Voltage >= voltage {
		t.Fatal("resume failed to continue time and stored energy")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause-again")
	frozen := s.Snapshot()
	time.Sleep(80 * time.Millisecond)
	if s.Snapshot().Transient.Sample != frozen.Transient.Sample {
		t.Fatal("paused clock advanced")
	}
	call(t, s, "component.set", ValueParams{"C1", "2uF"}, "cap-edit")
	if !s.Snapshot().Transient.Stale || !s.Snapshot().Paused {
		t.Fatal("paused electrical edit did not mark stale")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume-new")
	restarted := s.Snapshot()
	if restarted.Transient.Sample.Time != 0 || restarted.Transient.Sample.Voltage != 0 || restarted.Transient.TimeConstant != .02 || restarted.RunID == run {
		t.Fatal("electrical edit did not restart from configured initial condition")
	}
	copy := cloneSnapshot(restarted)
	copy.Transient.Samples[0].Voltage = 99
	copy.Transient.Events[0].Mode = "bad"
	if s.Snapshot().Transient.Samples[0].Voltage == 99 || s.Snapshot().Transient.Events[0].Mode == "bad" {
		t.Fatal("transient snapshots share mutable trace arrays")
	}
}
func TestRCDrawingUnitsValidationPersistenceAndSettingsHistory(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []PlaceParams{{ID: "Supply", Kind: "dc-source", Position: Point{-200, 0}}, {ID: "Limit", Kind: "resistor", Position: Point{0, -110}, Value: "10k"}, {ID: "Storage", Kind: "capacitor", Position: Point{200, 0}, Value: "1 µF"}, {ID: "Return", Kind: "ground", Position: Point{0, 110}}} {
		call(t, s, "component.place", p, p.ID)
	}
	for i, p := range []ConnectParams{{From: "Supply.positive", To: "Limit.2"}, {From: "Limit.1", To: "Storage.2"}, {From: "Storage.1", To: "Return.reference"}, {From: "Return.reference", To: "Supply.negative"}} {
		call(t, s, "wire.connect", p, string(rune('a'+i)))
	}
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "zero")
	call(t, s, "simulation.step", StepParams{.01}, "reference")
	if math.Abs(s.Snapshot().Transient.Sample.Voltage-3.1606027941427883) > 1e-12 {
		t.Fatal("arbitrary IDs/reversed resistor or capacitor pins failed")
	}
	for _, value := range []string{"0", "NaN", "2F", "-1uF"} {
		refuse(t, s, "component.set", ValueParams{"Storage", value}, "invalid-params")
	}
	for _, value := range []string{"1000nF", "1e-6 F", "1μF", ".001mF"} {
		n, err := ParseValue(value, "F")
		if err != nil || math.Abs(n-1e-6) > 1e-20 {
			t.Fatalf("capacitance unit parse %s: %g %v", value, n, err)
		}
	}
	cfg := DefaultTransientSettings()
	cfg.InitialVoltage = 2
	cfg.HighSeconds = .08
	call(t, s, "simulation.transient.set", cfg, "settings")
	call(t, s, "edit.undo", struct{}{}, "undo-settings")
	if s.Snapshot().TransientSettings != DefaultTransientSettings() {
		t.Fatal("settings not undoable")
	}
	call(t, s, "edit.redo", struct{}{}, "redo-settings")
	path := filepath.Join(t.TempDir(), "RC.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	state := fresh.Snapshot()
	if state.Document.Dirty || state.TransientSettings != cfg || state.Transient.Sample.Time != 0 || state.Transient.Sample.Voltage != 2 || !reflect.DeepEqual(state.Components, s.Snapshot().Components) {
		t.Fatal("RC saved configuration did not reopen at its initial condition")
	}
	cfg.StepSeconds = 1e-9
	refuse(t, s, "simulation.transient.set", cfg, "invalid-params")
	call(t, s, "object.delete", IDParams{"Storage"}, "delete-cap")
	if s.Snapshot().Transient == nil || !s.Snapshot().Transient.Stale || s.Snapshot().Status != "paused" {
		t.Fatal("broken RC trace looked current")
	}
}
func TestLegacySchematicVersionOneMigratesTransientDefaults(t *testing.T) {
	d := Document{Format: DocumentFormat, Version: 1, Graph: ExampleGraph(), View: View{Zoom: 1}, LEDModel: Model()}
	data, _ := json.Marshal(d)
	decoded, err := decodeDocument(data)
	if err != nil || *decoded.TransientSettings != DefaultTransientSettings() {
		t.Fatalf("legacy migration: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := New()
	defer s.Close()
	r := s.Dispatch(ctx, Request{Version: 1, ID: "cancel", Command: "project.inspect", Params: json.RawMessage(`{}`)}, "test")
	if r.Error == nil {
		t.Fatal("canceled query accepted")
	}
}

func TestRCInvalidNetworkStopsAndVersionTwoRequiresCompleteSettings(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.01}, "reference")
	old := s.Snapshot().Transient.Sample
	call(t, s, "component.place", PlaceParams{Kind: "led", Position: Point{300, 100}}, "mixed")
	if state := s.Snapshot(); !state.Transient.Stale || state.Transient.Sample != old || len(state.Diagnostics) == 0 || state.Diagnostics[0].Code != "unsupported-topology" {
		t.Fatal("mixed network produced valid transient output")
	}
	refuse(t, s, "simulation.step", StepParams{.001}, "unavailable")
	call(t, s, "edit.undo", struct{}{}, "undo-mixed")
	call(t, s, "simulation.restart", struct{}{}, "fresh")
	call(t, s, "wire.connect", ConnectParams{From: "C1.1", To: "GND.reference"}, "short")
	if state := s.Snapshot(); !state.Transient.Stale || state.Diagnostics[0].Code != "shorted-component" {
		t.Fatal("capacitor short was simulated")
	}
	cfg := DefaultTransientSettings()
	d := Document{Format: DocumentFormat, Version: 2, Graph: RCExampleGraph(), View: View{Zoom: 1}, LEDModel: Model(), TransientSettings: &cfg}
	data, _ := json.Marshal(d)
	var fields map[string]json.RawMessage
	json.Unmarshal(data, &fields)
	var settings map[string]any
	json.Unmarshal(fields["transient_settings"], &settings)
	delete(settings, "initial_voltage_v")
	fields["transient_settings"], _ = json.Marshal(settings)
	bad, _ := json.Marshal(fields)
	if _, err := decodeDocument(bad); err == nil {
		t.Fatal("missing initial condition silently became zero")
	}
}

func TestRCExampleStartsFreshWhenDefinitionIsUnchanged(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.01}, "old-time")
	call(t, s, "simulation.input.set", InputParams{"discharge"}, "old-mode")
	run := s.Snapshot().RunID
	state := call(t, s, "example.rc", ReplaceParams{DiscardChanges: true}, "new-copy").Result.(Snapshot)
	if state.Paused || state.Transient.Sample.Time != 0 || state.Transient.Sample.Voltage != 0 || state.Transient.Mode != "auto" || state.Transient.PendingMode != "" || state.RunID == run {
		t.Fatal("same-definition RC example retained old time/input instead of starting fresh")
	}
}
