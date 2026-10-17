package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFinitePulseMinimumSamplingDoesNotEraseWarning(t *testing.T) {
	g := waveformRCGraph(t, "finite-pulse-source")
	g.Components[0].Pulse.RiseSeconds = 1e-6
	g.Components[0].Pulse.FallSeconds = 2e-6
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	state := s.Snapshot()
	if state.TransientSettings.StepSeconds != 1e-6 || len(state.SourceWarnings) != 1 {
		t.Fatal("minimum clipping falsely resolved the finite-ramp warning", state.TransientSettings, state.SourceWarnings)
	}
	if !strings.Contains(state.SourceWarnings[0].Message, "minimum") {
		t.Fatal("remaining scope limit is not explained")
	}
}

func TestSourceSamplingRawTargetsSupportedFitsAndConstantSources(t *testing.T) {
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = 1e-6
	for _, shortFall := range []bool{false, true} {
		for _, ramp := range []float64{1e-6, 3e-6, math.Nextafter(4e-6, 0), 4e-6, 8e-6} {
			p, err := MakeComponent("V1", "finite-pulse-source", Point{}, "")
			if err != nil {
				t.Fatal(err)
			}
			p.Pulse.RiseSeconds = ramp
			p.Pulse.FallSeconds = 12e-6
			if shortFall {
				p.Pulse.RiseSeconds, p.Pulse.FallSeconds = p.Pulse.FallSeconds, p.Pulse.RiseSeconds
			}
			advice := sourceSamplingAdvice(Graph{Components: []Component{p}}, cfg)
			if len(advice) != 1 {
				t.Fatal("missing configured source advice")
			}
			a := advice[0]
			if a.TargetStep != ramp/4 || a.MinimumStep != 1e-6 || a.RecommendedStep != cfg.StepSeconds || a.CanFit || a.MinimumLimited != (ramp < 4e-6) || a.Unresolved != (ramp < 4e-6) {
				t.Fatal("raw target was confused with supported output interval", ramp, a)
			}
			if len(waveformSamplingWarnings(Graph{Components: []Component{p}}, cfg)) != map[bool]int{true: 1, false: 0}[ramp < 4e-6] {
				t.Fatal("warning differs from raw target")
			}
			p.Pulse.LowVoltage = p.Pulse.HighVoltage
			if len(sourceSamplingAdvice(Graph{Components: []Component{p}}, cfg)) != 0 {
				t.Fatal("constant pulse has an invented timing requirement")
			}
		}
	}
	p, _ := MakeComponent("V2", "sine-source", Point{}, "")
	p.Sine.Frequency = 1000
	cfg = DefaultTransientSettings()
	a := sourceSamplingAdvice(Graph{Components: []Component{p}}, cfg)[0]
	if a.TargetStep != 1./32000 || a.RecommendedStep != a.TargetStep || !a.CanFit || !a.Unresolved || a.MinimumLimited {
		t.Fatal("sine guidance changed", a)
	}
	p.Sine.Amplitude = 0
	if len(sourceSamplingAdvice(Graph{Components: []Component{p}}, cfg)) != 0 {
		t.Fatal("constant sine has an invented timing requirement")
	}
}

func TestMinimumSamplingAdviceOwnedSavedRecomputedAndNoOp(t *testing.T) {
	g := waveformRCGraph(t, "finite-pulse-source")
	g.Components[0].Pulse.RiseSeconds = 1e-6
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	result := call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	fitted := result.Result.(Snapshot)
	a := fitted.SourceSampling[0]
	if !a.Unresolved || !a.MinimumLimited || a.CanFit || a.TargetStep != .25e-6 || a.RecommendedStep != 1e-6 || a.SourceKind != "finite-pulse-source" {
		t.Fatal("minimum advice lost actual state", a)
	}
	found := false
	for _, help := range fitted.DiagnosticPresentation.Items {
		if help.Code == "source-sampling" {
			found = true
			if help.Severity != "warning" || !strings.Contains(help.Action, "cannot meet") || !reflect.DeepEqual(help.Objects, []string{"V1"}) {
				t.Fatal("minimum source recovery guidance differs", help)
			}
		}
	}
	if !found {
		t.Fatal("minimum guidance unavailable to diagnostics")
	}
	cloned := cloneSnapshot(fitted)
	cloned.SourceSampling[0].ID = "borrowed"
	if fitted.SourceSampling[0].ID != "V1" {
		t.Fatal("cloned advice aliases snapshot")
	}
	fitted.SourceSampling[0].TargetStep = 99
	retry := call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	if !retry.Duplicate || retry.Result.(Snapshot).SourceSampling[0] != a || s.Snapshot().SourceSampling[0] != a {
		t.Fatal("returned advice aliases retry/service state")
	}
	before := s.Snapshot()
	call(t, s, "simulation.sampling.fit", struct{}{}, "no-op")
	after := s.Snapshot()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document != after.Document {
		t.Fatal("unimprovable Fit mutated physical/document/history state")
	}
	path := filepath.Join(t.TempDir(), "fast-ramp.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if _, ok := stored["source_sampling"]; ok {
		t.Fatal("derived sampling advice stored as authoritative project data")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	reopened := fresh.Snapshot()
	if len(reopened.SourceSampling) != 1 || reopened.SourceSampling[0] != a || len(reopened.SourceWarnings) != 1 || !reopened.Paused || reopened.Transient.Sample.Time != 0 {
		t.Fatal("fresh project open lost configured minimum warning")
	}
}

func TestSourceSamplingAdviceIsTypedInDiscovery(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &schema); err != nil {
		t.Fatal(err)
	}
	field := schema["properties"].(map[string]any)["source_sampling"].(map[string]any)
	if field["type"] != "array" {
		t.Fatal("source advice not discoverable as an array")
	}
	properties := field["items"].(map[string]any)["properties"].(map[string]any)
	if properties["target_step_s"].(map[string]any)["type"] != "number" || properties["minimum_limited"].(map[string]any)["type"] != "boolean" {
		t.Fatal("source advice types differ")
	}
	raw, _ := json.Marshal(Catalog())
	if len(raw) > 64*1024 {
		t.Fatal("ordinary catalog exceeded existing transport budget")
	}
}
