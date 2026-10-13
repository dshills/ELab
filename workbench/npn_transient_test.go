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

func pausedNPNPulse(t *testing.T) *Service {
	t.Helper()
	s := New()
	call(t, s, "example.npn-pulse", ReplaceParams{}, "blink")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	return s
}
func TestNPNPulseEdgesAndDCReferences(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	initial := s.Snapshot()
	if initial.Analysis != NPNTransientAnalysis || initial.Transient.Engine != NPNTransientEngineID || initial.NPN.State != "saturation" || initial.Transient.Sample.Time != 0 {
		t.Fatal("incorrect pulse startup")
	}
	_, on := npnSolved(t, NPNExampleGraph())
	_, off := npnSolved(t, npnValue(NPNExampleGraph(), "VIN", 0))
	for i, tc := range []struct {
		duration  float64
		state     string
		reference NPNResult
	}{{.025, "saturation", on}, {.025, "cutoff", off}, {.025, "cutoff", off}, {.025, "saturation", on}} {
		call(t, s, "simulation.step", StepParams{tc.duration}, string(rune('a'+i)))
		snapshot := s.Snapshot()
		p, r := snapshot.Transient.Sample, snapshot.NPN
		if p.State != tc.state || r.State != tc.state || math.Abs(p.Current-tc.reference.CollectorCurrent) > 1e-14 || p.Voltage != r.VCE || p.BaseCurrent != r.BaseCurrent || r.RunID != snapshot.RunID || r.ElectricalRevision != snapshot.ElectricalRevision {
			t.Fatal("phase result does not match DC reference")
		}
	}
	trace := s.Snapshot().Transient
	for _, edge := range []float64{.05, .1} {
		both := map[float64]bool{}
		for _, p := range trace.Samples {
			if math.Abs(p.Time-edge) < 1e-14 {
				both[p.Input] = true
			}
		}
		if !both[0] || !both[5] {
			t.Fatalf("missing left/right edge samples at %g", edge)
		}
	}
	if len(trace.Events) != 3 {
		t.Fatal("missing pulse events")
	}
	// A coarse grid deliberately misses edges; exact edge insertion still works.
	coarse := pausedNPNPulse(t)
	defer coarse.Close()
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = .007
	cfg.HighSeconds = .031
	cfg.LowSeconds = .047
	call(t, coarse, "simulation.transient.set", cfg, "cfg")
	call(t, coarse, "simulation.restart", struct{}{}, "coarse-start")
	call(t, coarse, "simulation.step", StepParams{.078}, "coarse-period")
	got := coarse.Snapshot().Transient
	if got.Sample.Input != 5 || len(got.Events) != 3 || got.Events[1].Time != .031 || got.Events[2].Time != .078 {
		t.Fatal("coarse sampling moved or missed pulse edges")
	}
	call(t, s, "simulation.step", StepParams{1}, "history")
	trace = s.Snapshot().Transient
	if len(trace.Samples) > 512 || len(trace.Events) > 64 {
		t.Fatal("unbounded trace")
	}
	for i := 1; i < len(trace.Samples); i++ {
		if trace.Samples[i].Time < trace.Samples[i-1].Time {
			t.Fatal("unordered trace")
		}
	}
}
func TestNPNPulseInputAndLifecycle(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.025}, "step")
	before := s.Snapshot()
	call(t, s, "simulation.input.set", InputParams{"low"}, "queue-low")
	queued := s.Snapshot()
	if queued.Transient.PendingMode != "low" || queued.Transient.Sample != before.Transient.Sample || queued.NPN.State != "saturation" {
		t.Fatal("queued input changed paused state")
	}
	call(t, s, "simulation.step", StepParams{.001}, "apply-low")
	low := s.Snapshot()
	if low.NPN.State != "cutoff" || low.RunID != before.RunID || math.Abs(low.Transient.Sample.Time-.026) > 1e-12 || low.Transient.Mode != "low" {
		t.Fatal("manual low restarted or failed")
	}
	call(t, s, "component.move", MoveParams{ID: "Q1", Position: Point{120, 60}}, "move")
	call(t, s, "component.rotate", IDParams{ID: "Q1"}, "rotate")
	pose := s.Snapshot()
	if pose.RunID != low.RunID || !reflect.DeepEqual(pose.NPN, low.NPN) || pose.Transient.Sample != low.Transient.Sample {
		t.Fatal("geometry restarted time")
	}
	call(t, s, "simulation.input.set", InputParams{"high"}, "queue-high")
	call(t, s, "simulation.resume", struct{}{}, "resume")
	time.Sleep(120 * time.Millisecond)
	continued := s.Snapshot()
	if continued.RunID != low.RunID || continued.Transient.Sample.Time <= .026 || continued.NPN.State != "saturation" {
		t.Fatal("resume did not continue queued high")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause-again")
	frozen := s.Snapshot()
	time.Sleep(80 * time.Millisecond)
	if s.Snapshot().Transient.Sample != frozen.Transient.Sample {
		t.Fatal("paused clock advanced")
	}
	call(t, s, "component.set", ValueParams{ID: "VIN", Value: "0.8 V"}, "amplitude")
	if state := s.Snapshot(); !state.Transient.Stale || !state.NPN.Stale {
		t.Fatal("electrical edit did not stale both readings")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume-new")
	updated := s.Snapshot()
	if updated.RunID == frozen.RunID || updated.Transient.Sample.Time != 0 || updated.NPN.State != "active" {
		t.Fatal("new amplitude did not restart")
	}
	copy := cloneSnapshot(updated)
	copy.Transient.Samples[0].State = "bad"
	copy.NPN.Nodes["bad"] = 99
	if s.Snapshot().Transient.Samples[0].State == "bad" || s.Snapshot().NPN.Nodes["bad"] == 99 {
		t.Fatal("trace/result aliases service")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause-broken")
	call(t, s, "object.delete", IDParams{ID: "W5"}, "broken")
	if !s.Snapshot().NPN.Stale || !s.Snapshot().Transient.Stale {
		t.Fatal("broken circuit not stale")
	}
	rev := s.Snapshot().Revision
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "step-broken", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":0.001}`), ExpectedRevision: &rev, Key: "step-broken"}, "test")
	if response.Error == nil || response.Error.Code != "unavailable" {
		t.Fatal("stepped a broken circuit")
	}
}
func TestNPNPulseLiveClockPersistenceAndExampleReset(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	cfg := DefaultTransientSettings()
	cfg.HighSeconds = .005
	cfg.LowSeconds = .005
	cfg.PlaybackSpeed = .1
	call(t, s, "simulation.transient.set", cfg, "fast")
	call(t, s, "simulation.restart", struct{}{}, "new-fast")
	call(t, s, "simulation.resume", struct{}{}, "live")
	before := s.Snapshot()
	seen := map[string]bool{before.NPN.State: true}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		seen[state.NPN.State] = true
		if seen["cutoff"] && seen["saturation"] && state.Transient.Sample.Time > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	live := s.Snapshot()
	if !seen["cutoff"] || !seen["saturation"] || live.CommandRevision != before.CommandRevision {
		t.Fatal("live clock does not blink independently of commands")
	}
	call(t, s, "simulation.pause", struct{}{}, "freeze")
	call(t, s, "simulation.input.set", InputParams{"low"}, "low")
	call(t, s, "simulation.step", StepParams{.001}, "low-step")
	path := filepath.Join(t.TempDir(), "Blink.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := fresh.Snapshot()
	if loaded.Transient.Mode != "auto" || loaded.Transient.Sample.Time != 0 || loaded.NPN.State != "saturation" || loaded.TransientSettings != cfg || loaded.Document.Dirty {
		t.Fatal("file did not restore pulse configuration and fresh Auto run")
	}
	call(t, s, "example.npn-pulse", ReplaceParams{}, "fresh-example")
	example := s.Snapshot()
	if example.Transient.Sample.Time != 0 || example.Transient.Mode != "auto" || example.TransientSettings.HighSeconds != .05 || example.TransientSettings.PlaybackSpeed != .1 {
		t.Fatal("example retained prior clock/config/input")
	}
}
