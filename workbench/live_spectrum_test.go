package workbench

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func liveSpectrumFixture(t *testing.T) *Service {
	t.Helper()
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1", "i:R1"}}, "probes")
	call(t, s, "simulation.step", StepParams{Duration: .1}, "step")
	return s
}
func awaitLiveSpectrum(t *testing.T, s *Service) *LiveSpectrumState {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Spectrum.Live
		if v != nil && v.Result != nil {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("live spectrum not published", s.Snapshot().Spectrum.Live)
	return nil
}
func TestLiveSpectrumReusesRecordedFFTAndOwnsData(t *testing.T) {
	s := liveSpectrumFixture(t)
	before := s.Snapshot()
	call(t, s, "scope.view.set", ScopeViewParams{View: "spectrum"}, "tab")
	view := awaitLiveSpectrum(t, s)
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Transient, after.Transient) || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || before.Document.Dirty != after.Document.Dirty || before.CommandRevision+1 != after.CommandRevision {
		t.Fatal("FFT changed physics, dirty state or command authority")
	}
	s.mu.Lock()
	input, reason := s.liveSpectrumInputLocked()
	s.mu.Unlock()
	if reason != "" {
		t.Fatal(reason)
	}
	expected, err := analyzeSpectrum(input.record, input.params)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(view.Result.Bins, expected.Bins) || view.Result.SampleStep != expected.SampleStep {
		t.Fatal("live and recorded numerical engines differ")
	}
	if len(view.Result.Bins) > MaxLiveSpectrumSamples/2+1 || view.Result.Probe.ID != "v:C1" {
		t.Fatal("unbounded/wrong probe")
	}
	view.Result.Bins[0].Peak = 999
	view.Result.Assumptions[0] = "bad"
	view.Settings.ProbeID = "bad"
	if s.Snapshot().Spectrum.Live.Result.Bins[0].Peak == 999 {
		t.Fatal("live snapshot borrowed worker result")
	}
	p := after.Spectrum.Live.Settings
	p.ProbeID = "i:R1"
	call(t, s, "spectrum.live.set", p, "current")
	next := awaitLiveSpectrum(t, s)
	if next.Result.Probe.Unit != "A" || next.Result.Probe.ID != "i:R1" {
		t.Fatal("current probe not shared", next.Result.Probe)
	}
}
func TestLiveSpectrumHoldFreshnessLifecycleAndRecordedSeparation(t *testing.T) {
	s := liveSpectrumFixture(t)
	duration, step := .05, .001
	_, capture := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}}, "capture")
	call(t, s, "spectrum.analyze", SpectrumParams{ID: capture.Operation.ID, ProbeID: "v:C1"}, "recorded")
	recorded := cloneSpectrumResult(s.Snapshot().Spectrum.Result)
	call(t, s, "scope.view.set", ScopeViewParams{View: "spectrum"}, "tab")
	live := awaitLiveSpectrum(t, s)
	p := live.Settings
	p.Hold = true
	call(t, s, "spectrum.live.set", p, "hold")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step2")
	s.refreshLiveSpectrum()
	held := s.Snapshot().Spectrum.Live
	if held.Status != "held" || !reflect.DeepEqual(live.Result, held.Result) {
		t.Fatal("hold did not freeze derived frame")
	}
	if !reflect.DeepEqual(recorded, s.Snapshot().Spectrum.Result) {
		t.Fatal("live erased recorded result")
	}
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:R1"}}, "change-scope")
	if s.Snapshot().Spectrum.Live.Result != nil {
		t.Fatal("old scope selection stayed valid under hold")
	}
	p.Hold = false
	p.ProbeID = "i:R1"
	call(t, s, "spectrum.live.set", p, "follow")
	awaitLiveSpectrum(t, s)
	call(t, s, "simulation.restart", struct{}{}, "restart2")
	if s.Snapshot().Spectrum.Live.Result != nil {
		t.Fatal("restart borrowed old run")
	}
	call(t, s, "project.clear", map[string]bool{"discard_changes": true}, "clear")
	snap := s.Snapshot()
	if snap.Spectrum.Live != nil || snap.ScopeView != "scope" {
		t.Fatal("document replacement retained spectrum state")
	}
	s.Close()
}
func TestLiveSpectrumRejectsInvalidSettingsAndUnits(t *testing.T) {
	s := liveSpectrumFixture(t)
	for _, raw := range []string{`{"enabled":true,"samples":512}`, `{"enabled":true,"probe_id":"v:V1.positive"}`, `{"enabled":true,"window":"bad"}`, `{"enabled":true,"fundamental_hz":-1}`, `{"hold":true}`} {
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: raw, Command: "spectrum.live.set", Params: json.RawMessage(raw)}, "agent")
		if r.Error == nil || s.Snapshot().Spectrum.Live != nil {
			t.Fatal("invalid setting admitted", raw, r.Error)
		}
	}
	call(t, s, "spectrum.live.set", map[string]any{"enabled": true, "fundamental_hz": "2 kHz"}, "units")
	if s.Snapshot().Spectrum.Live.Settings.Fundamental != 2000 {
		t.Fatal("frequency text units lost")
	}
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad-view", Command: "scope.view.set", Params: json.RawMessage(`{"view":"other"}`)}, "agent")
	if r.Error == nil || s.Snapshot().ScopeView != "scope" {
		t.Fatal("invalid tab admitted")
	}
}

func TestLiveSpectrumPublicationRefusesInterveningAgentChanges(t *testing.T) {
	for _, change := range []string{"hold", "settings", "selection", "restart", "document", "close"} {
		t.Run(change, func(t *testing.T) {
			s := liveSpectrumFixture(t)
			call(t, s, "scope.view.set", ScopeViewParams{View: "spectrum"}, "tab")
			awaitLiveSpectrum(t, s)
			s.mu.Lock()
			input, reason := s.liveSpectrumInputLocked()
			s.mu.Unlock()
			if reason != "" {
				t.Fatal(reason)
			}
			result, err := analyzeSpectrum(input.record, input.params)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "hold":
				p := s.Snapshot().Spectrum.Live.Settings
				p.Hold = true
				call(t, s, "spectrum.live.set", p, "hold")
			case "settings":
				p := s.Snapshot().Spectrum.Live.Settings
				p.Window = "rectangular"
				call(t, s, "spectrum.live.set", p, "setting")
			case "selection":
				call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:R1"}}, "selection")
			case "restart":
				call(t, s, "simulation.restart", struct{}{}, "after-restart")
			case "document":
				call(t, s, "project.clear", map[string]bool{"discard_changes": true}, "clear")
			case "close":
				s.Close()
			}
			s.mu.Lock()
			revision := s.revision
			before := cloneLiveSpectrumState(s.liveSpectrum)
			s.publishLiveSpectrumLocked(input, result, nil)
			if revision != s.revision || !reflect.DeepEqual(before, s.liveSpectrum) {
				t.Error("retired analysis published after", change)
			}
			s.mu.Unlock()
		})
	}
}

func TestLiveSpectrumRemovedProbeCanBeHeldAndDisabled(t *testing.T) {
	s := liveSpectrumFixture(t)
	p := DefaultLiveSpectrumParams()
	p.ProbeID = "i:R1"
	call(t, s, "spectrum.live.set", p, "configure")
	awaitLiveSpectrum(t, s)
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{}}, "hide-probes")
	p.Hold = true
	call(t, s, "spectrum.live.set", p, "hold-missing")
	if s.Snapshot().Spectrum.Live.Result != nil {
		t.Fatal("held missing probe remained fresh")
	}
	p.Enabled = false
	call(t, s, "spectrum.live.set", p, "recorded")
	if s.Snapshot().Spectrum.Live.Settings.Enabled {
		t.Fatal("unavailable probe prevented return to recorded")
	}
}

func TestLiveSpectrumDiscoveryStaysCompactAndOutputNumeric(t *testing.T) {
	data, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 64*1024 {
		t.Fatal("ordinary discovery exceeded existing budget", len(data))
	}
	t.Logf("ordinary command catalog: %d bytes; %d commands", len(data), len(Catalog()))
	output := structSchema(reflect.TypeOf(LiveSpectrumParams{}))
	if inputProperty(output, "fundamental_hz")["type"] != "number" {
		t.Fatal("input unit text widened numeric output settings")
	}
	var input map[string]any
	if err = json.Unmarshal(SchemaRegistry()["urn:elab:live-spectrum-input:1"], &input); err != nil {
		t.Fatal(err)
	}
	if inputProperty(input, "fundamental_hz")["anyOf"] == nil {
		t.Fatal("frequency text missing from discoverable input")
	}
}

func TestLiveSpectrumRevalidatesIdenticalDataAfterOtherProbeRemoved(t *testing.T) {
	s := liveSpectrumFixture(t)
	call(t, s, "scope.view.set", ScopeViewParams{View: "spectrum"}, "tab")
	first := awaitLiveSpectrum(t, s)
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1"}}, "remove-other")
	fresh := awaitLiveSpectrum(t, s)
	if !reflect.DeepEqual(first.Result.Bins, fresh.Result.Bins) || first.Result.ID != fresh.Result.ID {
		t.Fatal("same accepted data changed numerically")
	}
	if s.Snapshot().Spectrum.Live.Result == nil {
		t.Fatal("same-data deduplication prevented source-key refresh")
	}
}
