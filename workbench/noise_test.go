package workbench

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func noiseSettings(source, probe string) SweepSettings {
	s := acSettings(source, probe)
	s.Parameter = "noise"
	return s
}
func TestNoiseRCIndependentPSDAndBandIntegral(t *testing.T) {
	cfg := noiseSettings("V1", "v:C1")
	r := sweepForTest(t, ACRCExampleGraph(), cfg)
	if r.Noise == nil || len(r.Noise.Contributors) != 1 || len(r.Points) != 201 {
		t.Fatal("noise inventory", r.Noise)
	}
	for _, p := range r.Points {
		w := 2 * math.Pi * p.Input
		want := 4 * noiseBoltzmann * 300.15 * 1000 / (1 + w*w*.001*.001)
		if math.Abs(p.Values[0]*p.Values[0]/want-1) > 1e-12 {
			t.Fatal("RC Johnson PSD", p, want)
		}
		input := math.Sqrt(4 * noiseBoltzmann * 300.15 * 1000)
		if math.Abs(p.Noise.InputDensity[0]/input-1) > 1e-12 {
			t.Fatal("RC input noise", p)
		}
	}
	// Exact finite-band integral: 4 k T R * [atan(2 pi f R C)]/(2 pi R C).
	integral := 4 * noiseBoltzmann * 300.15 * 1000 / (2 * math.Pi * .001) * (math.Atan(2*math.Pi*r.Stop*.001) - math.Atan(2*math.Pi*r.Start*.001))
	if math.Abs(r.Noise.OutputRMS[0]/math.Sqrt(integral)-1) > .001 {
		t.Fatal("noise band integral", r.Noise.OutputRMS, math.Sqrt(integral))
	}
	// Wide band tends to kT/C. Coarse grid quadrature is explicitly approximate.
	if math.Abs(r.Noise.OutputRMS[0]/math.Sqrt(noiseBoltzmann*300.15/1e-6)-1) > .01 {
		t.Fatal("RC kT/C limit", r.Noise.OutputRMS)
	}
}
func TestNoiseResistorDividerAndOpAmpBudget(t *testing.T) {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -200, 0, 0)
	b.part("R1", "resistor", "1k", 0, 0, 0)
	b.part("R2", "resistor", "1k", 200, 0, 0)
	b.ground("GND", "V1.negative", -200, 200)
	b.wire("V1.positive", "R1.1")
	b.wire("R1.2", "R2.1")
	b.wire("R2.2", "GND.reference")
	cfg := noiseSettings("V1", "v:R2")
	cfg.Points = 3
	cfg.Start = "100"
	cfg.Stop = "1000"
	cfg.Spacing = "linear"
	r := sweepForTest(t, b.graph, cfg)
	want := 4 * noiseBoltzmann * 300.15 * 500
	if len(r.Noise.Contributors) != 2 || math.Abs(r.Points[0].Values[0]*r.Points[0].Values[0]/want-1) > 1e-12 || math.Abs(r.Noise.OutputRMS[0]/math.Sqrt(want*900)-1) > 1e-12 {
		t.Fatal("independent parallel resistance", r.Noise)
	}
	cfg = noiseSettings("VIN", "v:U1")
	cfg.Points = 5
	op := sweepForTest(t, OpAmpInvertingExampleGraph(), cfg)
	found := false
	sum := 0.
	for _, c := range op.Noise.Contributors {
		sum += c.RMS[0] * c.RMS[0]
		if c.ID == "U1:input-voltage-white" {
			found = true
			if c.Density != 1e-8 {
				t.Fatal("op-amp reference density")
			}
			gain := c.RMS[0] / math.Sqrt(op.Stop-op.Start) / 1e-8
			if math.Abs(gain-3) > .002 {
				t.Fatal("op-amp noise gain", gain)
			}
		}
	}
	if !found || math.Abs(math.Sqrt(sum)/op.Noise.OutputRMS[0]-1) > 1e-12 {
		t.Fatal("contribution power sum", op.Noise)
	}
}
func TestNoiseShotLawAndCapturedSourceOwnership(t *testing.T) {
	r := sweepForTest(t, NPNExampleGraph(), noiseSettings("VCC", "v:Q1.collector"))
	if len(r.Noise.Bias.Transistors) != 1 {
		t.Fatal("bias transistor")
	}
	q := r.Noise.Bias.Transistors[0]
	shotFound := false
	for _, c := range r.Noise.Contributors {
		if c.ID == "Q1:collector-shot" {
			shotFound = true
		}
		if c.ID == "Q1:collector-shot" && math.Abs(c.Density*c.Density/(2*noiseCharge*math.Abs(q.CollectorCurrent))-1) > 1e-12 {
			t.Fatal("shot reference", c, q)
		}
	}
	if !shotFound {
		t.Fatal("collector shot contributor missing")
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "noise-rc"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	cfg := noiseSettings("V1", "v:C1")
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "noise")
	awaitSweep(t, s, "completed")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Mixed, after.Mixed) || before.RunID != after.RunID {
		t.Fatal("noise changed live state")
	}
	s.mu.Lock()
	record := cloneSweepRecord(s.sweeps.operations[after.DCSweep.Selected].record)
	s.mu.Unlock()
	raw := marshalNoiseTest(t, record)
	if _, err := validateSweepRecord(raw); err != nil {
		t.Fatal("valid noise capture", err)
	}
	original := cloneSweepRecord(record)
	for _, change := range []func(*SweepRecord){func(r *SweepRecord) { r.Result.Noise.OutputRMS[0]++ }, func(r *SweepRecord) { r.Result.Points[0].Noise.PSD[0][0]++ }, func(r *SweepRecord) { r.Result.Noise.Contributors[0].Density++ }, func(r *SweepRecord) { r.Result.Points[0].Noise.InputDensity[0]++ }} {
		record = cloneSweepRecord(original)
		change(record)
		raw = marshalNoiseTest(t, record)
		if _, err := validateSweepRecord(raw); err == nil {
			t.Fatal("forged noise capture accepted")
		}
	}
	after.DCSweep.Result.Noise.Bias.Resistors[0].Current = 999
	after.DCSweep.Result.Points[0].Noise.PSD[0][0] = 999
	if s.Snapshot().DCSweep.Result.Noise.Bias.Resistors[0].Current == 999 || s.Snapshot().DCSweep.Result.Points[0].Noise.PSD[0][0] == 999 {
		t.Fatal("noise snapshot alias")
	}
	path := filepath.Join(t.TempDir(), "noise.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	settled(t, fresh)
	if !reflect.DeepEqual(fresh.Snapshot().DCSweep.Result, s.Snapshot().DCSweep.Result) {
		t.Fatal("noise project round trip")
	}
	plan, err := newSweepPlan(ACRCExampleGraph(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := computeNoiseSweep(ctx, plan, func(int) {}); err == nil {
		t.Fatal("noise ignores cancellation")
	}
}

func TestNoiseUndefinedInputReferenceAndDeterminism(t *testing.T) {
	cfg := noiseSettings("VCC", "v:VIN")
	cfg.Points = 3
	a := sweepForTest(t, NPNExampleGraph(), cfg)
	b := sweepForTest(t, NPNExampleGraph(), cfg)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("stationary noise is not deterministic")
	}
	if a.Noise.InputDefined[0] || a.Noise.InputRMS[0] != 0 {
		t.Fatal("undefined input reference fabricated")
	}
	for _, p := range a.Points {
		if p.Noise.InputDefined[0] || p.Noise.InputDensity[0] != 0 {
			t.Fatal("zero gain referred noise")
		}
	}
}

func marshalNoiseTest(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestNoiseMalformedShapesRefuseWithoutPanic(t *testing.T) {
	r := sweepForTest(t, ACRCExampleGraph(), noiseSettings("V1", "v:C1"))
	for _, corrupt := range []func(*SweepResult){func(r *SweepResult) { r.Noise.InputDefined = nil }, func(r *SweepResult) { r.Points[0].Noise = nil }, func(r *SweepResult) { r.Points[0].Noise.InputDensity = nil }, func(r *SweepResult) { r.Points[0].Noise.PSD[0] = nil }, func(r *SweepResult) { r.Points[0].Values = append(r.Points[0].Values, 1) }} {
		bad := cloneSweepResult(r)
		corrupt(bad)
		if NoiseSummaryValid(bad) {
			t.Fatal("malformed noise accepted")
		}
		if _, err := encodeSweepExport(&SweepRecord{Result: *bad}, "csv"); err == nil {
			t.Fatal("malformed noise exported")
		}
	}
	if IsFrequencySweep("noise") || !IsSpectralSweep("noise") {
		t.Fatal("noise must not masquerade as AC gain/phase")
	}
}

func TestNoiseMultiOutputCSVContributionColumns(t *testing.T) {
	cfg := noiseSettings("V1", "v:C1")
	cfg.Points = 3
	cfg.ProbeIDs = []string{"v:C1", "v:R1"}
	r := sweepForTest(t, ACRCExampleGraph(), cfg)
	data, err := encodeSweepExport(&SweepRecord{Result: *r}, "csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]int{}
	for k, h := range rows[0] {
		columns[h] = k
	}
	sourceIndex := -1
	for k, c := range r.Noise.Contributors {
		if c.ID == "R1:johnson" {
			sourceIndex = k
		}
	}
	if sourceIndex < 0 {
		t.Fatal("missing resistor noise source")
	}
	for j, p := range r.Probes {
		header := "R1:johnson -> " + p.ID + " [V^2/Hz]"
		col, ok := columns[header]
		if !ok {
			t.Fatal("missing contribution column", header)
		}
		value, err := strconv.ParseFloat(rows[1][col], 64)
		if err != nil || value != r.Points[0].Noise.PSD[sourceIndex][j] {
			t.Fatal("mislabeled contribution", j, value, err)
		}
	}
}
