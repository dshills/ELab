package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lcSamplingGraph() Graph {
	g := RLCExampleGraph()
	g.Components[2].Value = .01
	g.Components[3].Value = 1e-6
	return g
}

func lcSamplingStepReference(at float64) float64 {
	// Independent series-RLC step: 5 V, 50 ohms, .01 H, 1 uF.
	const alpha, omega = 2500., 10000.
	damped := math.Sqrt(omega*omega - alpha*alpha)
	return 5 * (1 - math.Exp(-alpha*at)*(math.Cos(damped*at)+alpha/damped*math.Sin(damped*at)))
}

func TestSamplingFitRecoversMissedRingingPeak(t *testing.T) {
	s := newGraphService(SolveLED, lcSamplingGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration := .002
	capture := func(key string) (float64, int) {
		t.Helper()
		_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, key)
		page := call(t, s, "operation.samples", VerificationPageParams{ID: view.Operation.ID, Limit: 200}, key+"-rows").Result.(VerificationPage)
		peak := 0.
		for _, row := range page.Rows {
			if math.Abs(row.Values[0]-lcSamplingStepReference(row.Time)) > .001 {
				t.Fatal("captured circuit differs from independent RLC equation", row)
			}
			peak = math.Max(peak, row.Values[0])
		}
		return peak, len(page.Rows)
	}
	coarse, coarseRows := capture("coarse")
	call(t, s, "simulation.step", StepParams{Duration: .0002}, "live-before-fit")
	before := s.Snapshot()
	if before.Mixed.Time == 0 {
		t.Fatal("fixture must preserve an advanced live experiment")
	}
	coarseID := before.LastOperation
	coarsePage := call(t, s, "operation.samples", VerificationPageParams{ID: coarseID, Limit: 200}, "coarse-preserved-before").Result.(VerificationPage)
	if before.LCSampling == nil || !before.LCSampling.Unresolved || !before.LCSampling.CanFit || len(before.SourceWarnings) != 0 {
		t.Fatal("missing source-independent LC advice")
	}
	call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	fitted := s.Snapshot()
	preserved := call(t, s, "operation.samples", VerificationPageParams{ID: coarseID, Limit: 200}, "coarse-preserved-after").Result.(VerificationPage)
	if !reflect.DeepEqual(coarsePage, preserved) {
		t.Fatal("fitting changed samples in an existing recording")
	}
	if !fitted.Paused || fitted.LCSampling.Unresolved || fitted.LCSampling.CanFit || !reflect.DeepEqual(before.Components, fitted.Components) || !reflect.DeepEqual(before.Wires, fitted.Wires) || fitted.Mixed.Time != before.Mixed.Time || fitted.Mixed.Inductors[0].Current != before.Mixed.Inductors[0].Current || fitted.Mixed.Capacitors[0].Voltage != before.Mixed.Capacitors[0].Voltage || !fitted.Mixed.Stale {
		t.Fatal("fit changed parts/accepted physics or omitted paused stale semantics")
	}
	fine, fineRows := capture("fitted")
	peak := lcSamplingStepReference(math.Pi / math.Sqrt(10000.*10000-2500.*2500))
	if coarseRows != 3 || fineRows <= 100 || coarse > peak-1 || math.Abs(fine-peak) > .025 {
		t.Fatal("fit did not recover the otherwise missed peak", coarse, coarseRows, fine, fineRows, peak)
	}
	// An explicit offline override remains independent of the live fit.
	step := .001
	_, overridden := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}}, "explicit-coarse")
	if overridden.Metadata.SampleCount != 3 || overridden.Metadata.Settings.StepSeconds != step {
		t.Fatal("live fit replaced an explicit Verify override")
	}
	t.Logf("analytic peak %.9g V; coarse %.9g V /%d rows; fitted %.9g V /%d rows", peak, coarse, coarseRows, fine, fineRows)
}

func TestLCSamplingMinimumIsExplainedWithoutFalseResolution(t *testing.T) {
	g := lcSamplingGraph()
	g.Components[2].Value, g.Components[3].Value = 1e-6, 1e-9
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	state := s.Snapshot()
	a := state.LCSampling
	if a == nil || !a.MinimumLimited || !a.Unresolved || a.CanFit || a.TargetStep >= 1e-6 || a.RecommendedStep != 1e-6 || state.TransientSettings.StepSeconds != 1e-6 || state.TransientSettings.PlaybackSpeed != .001 {
		t.Fatal("minimum clipping falsely resolved the LC warning", a, state.TransientSettings)
	}
	found := false
	for _, help := range state.DiagnosticPresentation.Items {
		if help.Code == "lc-sampling" {
			found = true
			if help.Severity != "warning" || !strings.Contains(help.Action, "cannot resolve") || !reflect.DeepEqual(help.Objects, []string{"C1", "L1"}) {
				t.Fatal("minimum warning lost its limitation or contributing parts", help)
			}
		}
	}
	if !found {
		t.Fatal("missing actionable minimum-step guidance")
	}
	copy := cloneSnapshot(state)
	copy.LCSampling.InductorID = "borrowed"
	if state.LCSampling.InductorID != "L1" {
		t.Fatal("cloned snapshot aliases the original advice")
	}
	state.LCSampling.CapacitorID = "borrowed"
	if s.Snapshot().LCSampling.InductorID != "L1" || s.Snapshot().LCSampling.CapacitorID != "C1" {
		t.Fatal("advice pointer aliases service state")
	}
	before := s.Snapshot()
	call(t, s, "simulation.sampling.fit", struct{}{}, "already-finest")
	after := s.Snapshot()
	if after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || after.UndoName != before.UndoName || after.Document.Dirty != before.Document.Dirty {
		t.Fatal("unimprovable fitting made an electrical edit")
	}
}

func TestLCSamplingUsesSmallestValuesAndExistingSourceLimits(t *testing.T) {
	g := RLCExampleGraph()
	source, _ := MakeComponent("V1", "sine-source", g.Components[0].Position, "5 V")
	source.Sine.Frequency = 1000
	g.Components[0] = source
	cfg := DefaultTransientSettings()
	step, advice := samplingRecommendation(g, cfg)
	if step != 1./32000 || advice == nil || advice.TargetStep <= step {
		t.Fatal("source's finer recommendation was lost", step, advice)
	}
	coil, _ := MakeComponent("LSMALL", "inductor", Point{X: 1000}, "1 mH")
	cap, _ := MakeComponent("CSMALL", "capacitor", Point{X: 1200}, "100 nF")
	g.Components = append(g.Components, coil, cap)
	step, advice = samplingRecommendation(g, cfg)
	want := 2 * math.Pi * math.Sqrt(1e-3*100e-9) / 32
	if advice.InductorID != "LSMALL" || advice.CapacitorID != "CSMALL" || math.Abs(step-want) > 1e-20 || !strings.Contains(advice.Note, "not a measured period") {
		t.Fatal("LC reference used the wrong values or implied actual circuit timing", step, advice)
	}
	cfg.StepSeconds, cfg.PlaybackSpeed = 1e-6, .001
	if step, advice = samplingRecommendation(g, cfg); step != cfg.StepSeconds || advice.CanFit || advice.Unresolved {
		t.Fatal("fit coarsened a finer user step", step, advice)
	}
	for _, graph := range []Graph{EmptyGraph(), RLExampleGraph()} {
		if _, advice := samplingRecommendation(graph, cfg); advice != nil {
			t.Fatal("invented LC timing without both storage types")
		}
	}
}

func TestLCSamplingFitHistoryRetryRefusalAndPersistence(t *testing.T) {
	s := newGraphService(SolveLED, lcSamplingGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	revision := before.Revision
	request := Request{Version: 1, ID: "fit", Key: "fit", Command: "simulation.sampling.fit", Params: json.RawMessage(`{}`), ExpectedRevision: &revision}
	response := s.Dispatch(t.Context(), request, "test")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	fitted := s.Snapshot()
	retry := s.Dispatch(t.Context(), request, "test")
	if retry.Error != nil || retry.Revision != response.Revision || !reflect.DeepEqual(fitted, s.Snapshot()) {
		t.Fatal("fit retry executed another edit")
	}
	refuse(t, s, "simulation.sampling.fit", map[string]any{"step_s": .001}, "invalid-params")
	if !reflect.DeepEqual(fitted, s.Snapshot()) {
		t.Fatal("refused fit changed state")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().TransientSettings != before.TransientSettings || !s.Snapshot().LCSampling.Unresolved {
		t.Fatal("undo did not restore the coarse experiment")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	if s.Snapshot().TransientSettings != fitted.TransientSettings {
		t.Fatal("redo did not restore the fitted experiment")
	}
	path := filepath.Join(t.TempDir(), "fitted.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	opened := settled(t, fresh)
	if opened.TransientSettings != fitted.TransientSettings || !reflect.DeepEqual(opened.LCSampling, fitted.LCSampling) || !opened.Paused || opened.Mixed.Time != 0 {
		t.Fatal("reopen lost settings/derived advice or resumed prior physics")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo-before-batch")
	call(t, s, "edit.batch", BatchParams{Name: "Fit LC sampling", Edits: []BatchEdit{{Command: "simulation.sampling.fit", Params: json.RawMessage(`{}`)}}}, "batch")
	if s.Snapshot().TransientSettings != fitted.TransientSettings || s.Snapshot().UndoName != "Fit LC sampling" {
		t.Fatal("shared fit was unavailable to atomic batches")
	}
}

func TestLCSamplingDiscoveryIsPureAndNumeric(t *testing.T) {
	s := newGraphService(SolveLED, lcSamplingGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	schema := sweepDetailSchema(t, "simulation.sampling.fit")
	policy := schema["x-sampling-policy"].(map[string]any)
	contract := schema["x-command-contract"].(map[string]any)
	if policy["minimum_step_s"] != 1e-6 || policy["lc_samples_per_reference_period"] != float64(32) || contract["scope"] != "output-sampling-fit" || contract["completion"] == nil || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("fit contract missing bounds or changed the experiment")
	}
	var snapshot map[string]any
	if err := json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &snapshot); err != nil {
		t.Fatal(err)
	}
	if inputProperty(snapshot, "lc_sampling")["anyOf"] == nil {
		t.Fatal("derived sampling advice absent from shared snapshot schema")
	}
	withoutLC := NewExample()
	defer withoutLC.Close()
	state := settled(t, withoutLC)
	raw, _ := json.Marshal(state)
	if state.LCSampling != nil || strings.Contains(string(raw), `"lc_sampling"`) {
		t.Fatal("new derived field changed a no-LC snapshot's optional shape")
	}
}

func TestLCSamplingBatchUsesStagedValuesAndRollsBack(t *testing.T) {
	s := newGraphService(SolveLED, lcSamplingGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	edits := []BatchEdit{edit("component.set", ValueParams{ID: "L1", Value: "2.5 mH"}), edit("simulation.sampling.fit", struct{}{})}
	bad := append(append([]BatchEdit{}, edits...), edit("component.set", ValueParams{ID: "missing", Value: "1k"}))
	refuse(t, s, "edit.batch", BatchParams{Name: "Rejected tune", Edits: bad}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("failed batch leaked fitted settings or staged part values")
	}
	call(t, s, "edit.batch", BatchParams{Name: "Tune and fit", Edits: edits}, "tune")
	after := s.Snapshot()
	want := 2 * math.Pi * math.Sqrt(.0025*1e-6) / 32
	if after.TransientSettings.StepSeconds != want || after.ElectricalRevision != before.ElectricalRevision+1 || after.UndoName != "Tune and fit" || after.Components[2].Value != .0025 {
		t.Fatal("fit did not use the preceding staged value as one edit", after.TransientSettings)
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo-tune")
	if restored := s.Snapshot(); restored.TransientSettings != before.TransientSettings || !reflect.DeepEqual(restored.Components, before.Components) {
		t.Fatal("batch undo failed to restore part and sampling together")
	}
}

func TestSamplingFitIncludesDCPoweredLCTiming(t *testing.T) {
	s := newGraphService(SolveLED, lcSamplingGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	call(t, s, "simulation.sampling.fit", struct{}{}, "fit")
	after := s.Snapshot()
	want := 2 * math.Pi * math.Sqrt(.01*1e-6) / 32
	if after.TransientSettings.StepSeconds > want || after.TransientSettings.StepSeconds >= before.TransientSettings.StepSeconds {
		t.Fatalf("DC-powered ringing still has coarse output: got %.9g s, want at most %.9g s", after.TransientSettings.StepSeconds, want)
	}
}
