package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/internal/testutil"
)

func TestClassABSpeakerCompletedOutputsAndWorkCounts(t *testing.T) {
	state, trace := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	budget := newIntegrationBudget()
	stage := &inductorStage{x: state.x, capV: state.capV}
	a, ok := state.net.storageJacobian(stage, trace, 0, budget)
	rate, good := checkedOscillationRate(a)
	if !ok || !good || !state.net.usesModalLCCeiling() || state.net.transientLCCeiling(stage, trace, 0, budget) <= 100*state.net.maximumInductorStep() {
		t.Fatal("class-AB speaker modal analysis unavailable", a, ok, good)
	}
	t.Log("state Jacobian", a, "valid", ok, good, "rate", rate, "old ceiling", state.net.maximumInductorStep(), "new ceiling", state.net.transientLCCeiling(stage, trace, 0, budget))
	peak := time.Duration(0)
	for sample := 1; sample <= 10; sample++ {
		budget = newIntegrationBudget()
		budget.elapsed = func() time.Duration { return 0 } // Deterministic numerical/work-count test; production-clock coverage is separate.
		next, err := state.advanceBudget(trace, float64(sample)*.0001, budget)
		elapsed := time.Since(budget.started)
		peak = max(peak, elapsed)
		if err != nil {
			t.Fatalf("sample %d attempts %d Newton %d time %s: %v", sample, budget.attempts, budget.newtonIterations, elapsed, err)
		}
		state = next
		if state.result.Time != float64(sample)*.0001 || len(state.result.Speakers) != 1 || !finite(state.result.Speakers[0].Voltage) || !finite(state.result.Speakers[0].Current) {
			t.Fatal("missing completed speaker output", sample, state.result.Time)
		}
		if budget.attempts > MaxIntegrationAttempts || budget.modalAnalyses > 2*budget.attempts+1 {
			t.Fatal("modal work bounds", budget.attempts, budget.modalAnalyses)
		}
		if state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-7 {
			t.Fatal("conservation", state.result)
		}
	}
	t.Log("10 completed outputs; elapsed peak (informational, deterministic-clock test)", peak)
}

func TestStorageJacobianAgainstIndependentSeriesRLC(t *testing.T) {
	c := 10e-6
	state, tr := initializeInductorTest(t, inductorTestGraph(t, 0, 100, .1, &c))
	a, ok := state.net.storageJacobian(&inductorStage{x: state.x, capV: state.capV}, tr, 0, newIntegrationBudget())
	if !ok || len(a) != 2 {
		t.Fatal("missing two-storage Jacobian", a)
	}
	want := [][]float64{{0, 1000}, {-1000, -1000}}
	for i := range a {
		for j := range a[i] {
			if math.Abs(a[i][j]-want[i][j]) > 1e-9 {
				t.Fatal("independent energy-scaled series RLC ODE", a, want)
			}
		}
	}
	rate, ok := checkedOscillationRate(a)
	if !ok || rate < math.Sqrt(750000) || rate > math.Sqrt(750000)+1e-5 {
		t.Fatal("independent damped natural frequency", rate, ok)
	}
}

func TestModalRateDistinguishesDecayRotationGrowthAndDefectiveBasis(t *testing.T) {
	for _, v := range []struct {
		a    [][]float64
		want float64
	}{
		{[][]float64{{-1e9, 0, 0}, {0, 0, -628}, {0, 628, 0}}, 628},
		{[][]float64{{-1e9, 0}, {0, -1}}, 0},
		{[][]float64{{100, 0}, {0, -1e9}}, 100},
	} {
		rate, ok := checkedOscillationRate(v.a)
		if !ok || rate < v.want || rate > v.want+.01 {
			t.Fatal("modal rate", rate, ok, v.want)
		}
	}
	for _, a := range [][][]float64{nil, {{math.NaN()}}, {{1, 1}, {0, 1}}, {{1}, {1, 2}}} {
		if rate, ok := checkedOscillationRate(a); ok {
			t.Fatal("unresolved basis accepted", a, rate)
		}
	}
}

func TestModalLegacyFallbackDeadlineAndOldNetworkPreservation(t *testing.T) {
	state, tr := initializeInductorTest(t, speakerTestGraph())
	stage := &inductorStage{x: state.x, capV: state.capV}
	if state.net.usesModalLCCeiling() || state.net.transientLCCeiling(stage, tr, 0, newIntegrationBudget()) != state.net.maximumInductorStep() {
		t.Fatal("existing simple speaker route changed")
	}
	state, tr = initializeInductorTest(t, ClassABSpeakerExampleGraph())
	budget := newIntegrationBudget()
	clockCalls := 0
	budget.elapsed = func() time.Duration {
		clockCalls++
		if clockCalls > 2 {
			return MaxIntegrationWallTime + time.Nanosecond
		}
		return 0
	}
	before := append([]float64{}, state.x...)
	beforeCaps := append([]float64{}, state.capV...)
	beforeStep, beforeCeiling := state.inductorStep, state.inductorCeiling
	beforeResult, marshalErr := json.Marshal(state.result)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if next, err := state.advanceBudget(tr, .0001, budget); next != nil || err == nil {
		t.Fatal("modal probing bypassed deadline", next, err)
	}
	if budget.modalAnalyses < 1 || clockCalls < 3 {
		t.Fatal("deadline did not interrupt modal probing")
	}
	for i, x := range state.x {
		if x != before[i] {
			t.Fatal("deadline changed accepted state")
		}
	}
	afterResult, marshalErr := json.Marshal(state.result)
	if marshalErr != nil || string(beforeResult) != string(afterResult) || !reflect.DeepEqual(beforeCaps, state.capV) || state.inductorStep != beforeStep || state.inductorCeiling != beforeCeiling {
		t.Fatal("deadline changed accepted storage/result/proposals", marshalErr)
	}
}

func TestClassABSpeakerOutputRefinementAndBothPolarities(t *testing.T) {
	if testing.Short() {
		t.Skip("full 200 ms, two-polarity refinement")
	}
	advance := func(state *mixedState, tr *Transient, at float64) (*mixedState, error) {
		b := newIntegrationBudget()
		b.elapsed = func() time.Duration { return 0 }
		return state.advanceBudget(tr, at, b)
	}
	coarse, tr := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	fine, refTrace := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	positive, negative := false, false
	for sample := 1; sample <= 2000; sample++ {
		at := float64(sample) * .0001
		var err error
		coarse, err = advance(coarse, tr, at)
		if err != nil {
			t.Fatal("coarse", sample, err)
		}
		fine, err = advance(fine, refTrace, at-.00005)
		if err != nil {
			t.Fatal("refined middle", sample, err)
		}
		fine, err = advance(fine, refTrace, at)
		if err != nil {
			t.Fatal("refined end", sample, err)
		}
		a, b := coarse.result.Speakers[0], fine.result.Speakers[0]
		if math.Abs(a.Voltage-b.Voltage) > 5e-5 || math.Abs(a.Current-b.Current) > 2e-6 {
			t.Fatal("output refinement", sample, a, b)
		}
		if coarse.result.KCLResidual > 1e-9 || math.Abs(coarse.result.PowerResidual) > 1e-8 || a.Energy < 0 || a.CoilHeat < 0 || a.MotionalHeat < 0 {
			t.Fatal("speaker conservation", sample, a)
		}
		if at > .05 {
			positive = positive || a.Voltage > .05
			negative = negative || a.Voltage < -.05
		}
	}
	if !positive || !negative {
		t.Fatal("missing complete bipolar waveform", positive, negative)
	}
}

func TestClassABSpeakerSharedCaptureAndSavedExample(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "class-ab-speaker"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	if before.Document.Dirty || before.Example.Modified || len(before.Scope.Selected) != 6 || before.Selected != "SP1" || before.TransientSettings.StepSeconds != .0001 {
		t.Fatal("example defaults", before.Diagnostics)
	}
	duration, step := .01, .0001
	_, record := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:SP1", "i:SP1", "v:C1"}}, "capture")
	if record.Operation.Status != "completed" || len(record.Metadata.Mixed.Transistors) != 8 || len(record.Metadata.Mixed.Speakers) != 1 {
		t.Fatal("missing model-bearing capture", record)
	}
	if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) {
		t.Fatal("recording advanced live state")
	}
	path := filepath.Join(t.TempDir(), "class-ab-speaker.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	other := New()
	defer other.Close()
	call(t, other, "project.load", LoadParams{Path: path}, "open")
	after := settled(t, other)
	if after.Document.Dirty || !after.Paused || !reflect.DeepEqual(before.Components, after.Components) || after.Document.RecordingCount != 1 {
		t.Fatal("saved model/example/capture lost", after.Document)
	}
}

func TestModalStartupMatchesOriginalFineLCCeiling(t *testing.T) {
	modern, tr := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	legacy, oldTrace := initializeInductorTest(t, ClassABSpeakerExampleGraph())
	// Test-only original selector: the compiled physical network, equations,
	// storage states and tolerances stay exact. Eligibility alone is disabled.
	net := *legacy.net
	net.authored = Graph{}
	legacy.net = &net
	var err error
	modern, err = modern.advance(tr, 1e-7)
	if err != nil {
		t.Fatal("modal", err)
	}
	legacy, err = legacy.advance(oldTrace, 1e-7)
	if err != nil {
		t.Fatal("original fine ceiling", err)
	}
	for i := range modern.net.nodes {
		a, b := modern.x[i], legacy.x[i]
		if math.Abs(a-b) > 5e-8+5e-7*math.Max(math.Abs(a), math.Abs(b)) {
			t.Fatal("startup differs from independently fine proposals", i, a, b)
		}
	}
	copied := cloneMixedState(modern)
	if copied.inductorStep != modern.inductorStep || copied.inductorCeiling != modern.inductorCeiling || copied.inductorStep <= 0 {
		t.Fatal("manual-step clone lost proposal")
	}
	originalX, originalCap := modern.x[0], modern.capV[0]
	copied.x[0] += 1
	copied.capV[0] += 1
	if modern.x[0] != originalX || modern.capV[0] != originalCap {
		t.Fatal("manual-step storage clone aliases")
	}
}
