package workbench

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLogSweepLoadedDividerAndDisplayAxis(t *testing.T) {
	for _, endpoints := range [][2]string{{"10 ohm", "100k ohm"}, {"100k ohm", "10 ohm"}} {
		cfg := SweepSettings{ComponentID: "R1", Parameter: "resistance", Spacing: "logarithmic", Start: endpoints[0], Stop: endpoints[1], Points: 9, ProbeIDs: []string{"v:P1.wiper"}}
		result := sweepForTest(t, potGraph(t, "resistor"), cfg)
		for i, p := range result.Points {
			exponent := 1 + float64(i)/2
			fraction := float64(i) / 8
			if endpoints[0] == "100k ohm" {
				exponent = 5 - float64(i)/2
				fraction = 1 - fraction
			}
			wantInput := math.Pow(10, exponent)
			if math.Abs(p.Input/wantInput-1) > 3e-15 {
				t.Fatalf("axis %d: %g != %g", i, p.Input, wantInput)
			}
			parallel := 1 / (1./5000 + 1/p.Input)
			want := 5 * parallel / (5000 + parallel)
			if math.Abs(p.Values[0]-want) > 1e-10 {
				t.Fatal("loaded divider", p.Values[0], want)
			}
			if math.Abs(result.AxisFraction(p.Input)-fraction) > 1e-14 || math.Abs(result.AxisValue(fraction)/p.Input-1) > 3e-15 {
				t.Fatal("display and stored axis disagree")
			}
		}
		if math.Abs(result.AxisValue(.5)-1000) > 1e-10 {
			t.Fatal("log axis midpoint")
		}
	}
}

func TestLogSweepEndpointRulesAndRepresentability(t *testing.T) {
	cfg := SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0.1 V", Stop: "10 V", Points: 5, Spacing: "logarithmic"}
	for _, pair := range [][2]string{{"0", "5"}, {"-1", "5"}, {"-5", "-1"}, {"1", "1"}, {"1", "1.0000000000000002"}} {
		bad := cfg
		bad.Start, bad.Stop = pair[0], pair[1]
		if _, err := newSweepPlan(ExampleGraph(), bad); err == nil {
			t.Fatal("accepted invalid log axis", pair)
		}
	}
	cfg.Spacing = "log"
	if _, err := newSweepPlan(ExampleGraph(), cfg); err == nil {
		t.Fatal("unknown spacing accepted")
	}
	// A full positive voltage span can have an overflowing ratio. Geometry still
	// retains finite monotone points; no inference about solver precision is made.
	cfg.Spacing = "logarithmic"
	cfg.Start = "1e-300"
	cfg.Stop = "12"
	cfg.Points = 201
	plan, err := newSweepPlan(ExampleGraph(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	previous := 0.
	for i := 0; i < 201; i++ {
		v := sweepAxisPoint(plan.start, plan.stop, 201, i, cfg.Spacing)
		if !finite(v) || v <= previous {
			t.Fatal("overflowing ratio axis", i, v)
		}
		previous = v
	}
	// Nearby large endpoints may have identical separately rounded logarithms.
	a, b := 10000., math.Nextafter(10000, math.Inf(1))
	r := SweepResult{Start: a, Stop: b, Spacing: "logarithmic"}
	if r.AxisFraction(a) != 0 || r.AxisFraction(b) != 1 {
		t.Fatal("nearby display endpoints collapsed")
	}
	// Historic linear point arithmetic must stay exact, including non-binary steps.
	for i := 0; i < 19; i++ {
		want := .1 + (5.3-.1)*float64(i)/18
		if i == 18 {
			want = 5.3
		}
		if sweepAxisPoint(.1, 5.3, 19, i, "") != want {
			t.Fatal("legacy linear axis changed")
		}
	}
}

func TestLogSweepSharedPersistenceImportAndInvalidReadPurity(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := SweepSettings{ComponentID: "R1", Parameter: "resistance", Start: "100 ohm", Stop: "100k ohm", Points: 7, Spacing: "logarithmic", ProbeIDs: []string{"i:D1"}}
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	state := awaitSweep(t, s, "completed")
	id := state.DCSweep.Selected
	dir := t.TempDir()
	path := filepath.Join(dir, "curve.json")
	call(t, s, "sweep.export", VerificationExportParams{ID: id, Format: "json", Path: path}, "export")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := validateSweepRecord(data)
	if err != nil || record.Result.Spacing != "logarithmic" {
		t.Fatal("log record invalid", err)
	}
	// A saved interior point can differ by a math-library rounding step.
	original := record.Result.Points[1].Input
	record.Result.Points[1].Input = math.Nextafter(original, record.Result.Stop)
	rounded, _ := json.Marshal(record)
	if _, err := validateSweepRecord(rounded); err != nil {
		t.Fatal("roundoff in stored log axis refused", err)
	}
	record.Result.Points[1].Input = original
	record.Result.Spacing = "linear"
	bad, _ := json.Marshal(record)
	if _, err := validateSweepRecord(bad); err == nil {
		t.Fatal("mismatching result spacing accepted")
	}
	record.Result.Spacing = "logarithmic"
	record.Result.Points[1].Input = 101
	bad, _ = json.Marshal(record)
	if _, err := validateSweepRecord(bad); err == nil {
		t.Fatal("wrong geometric axis accepted")
	}
	project := filepath.Join(dir, "log.elab")
	call(t, s, "project.save", SaveParams{Path: project}, "save")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: project}, "load")
	if !reflect.DeepEqual(loaded.Snapshot().DCSweep.Result, state.DCSweep.Result) {
		t.Fatal("project changed log curve")
	}
	imported := New()
	defer imported.Close()
	call(t, imported, "sweep.import", RecordingImportParams{Path: path}, "import")
	if !reflect.DeepEqual(imported.Snapshot().DCSweep.Result, state.DCSweep.Result) {
		t.Fatal("import changed log curve")
	}
	before := settled(t, s)
	badcfg := cfg
	badcfg.Start = "0"
	refuse(t, s, "sweep.settings.set", badcfg, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("invalid log setting mutated state")
	}
	schema := sweepDetailSchema(t, "sweep.settings.set")
	enums := inputProperty(schema, "spacing")["enum"].([]any)
	if !reflect.DeepEqual(enums, []any{"", "linear", "logarithmic"}) {
		t.Fatal("spacing undiscoverable", enums)
	}
}

func TestLogBiasSweepMatchesIndependentLEDAndNPNKernels(t *testing.T) {
	for _, graph := range []Graph{ExampleGraph(), NPNExampleGraph()} {
		compiled, _, diagnostic := graph.Compile()
		if diagnostic != nil {
			t.Fatal(diagnostic)
		}
		target, probe := "V1", "i:D1"
		if compiled.Analysis == NPNAnalysis {
			target = compiled.Input.ID
			probe = "i:" + compiled.Transistor.ID + ".collector"
		}
		result := sweepForTest(t, graph, SweepSettings{ComponentID: target, Parameter: "voltage", Spacing: "logarithmic", Start: "0.05 V", Stop: "5 V", Points: 11, ProbeIDs: []string{probe}})
		for _, point := range result.Points {
			input := graph.Clone()
			for i := range input.Components {
				if input.Components[i].ID == target {
					input.Components[i].Value = point.Input
				}
			}
			c, _, d := input.Compile()
			if d != nil {
				t.Fatal(d)
			}
			var want float64
			if c.Analysis == NPNAnalysis {
				r, err := SolveNPN(context.Background(), c)
				if err != nil {
					t.Fatal(err)
				}
				want = r.CollectorCurrent
			} else {
				r, err := SolveLED(context.Background(), c.Circuit)
				if err != nil {
					t.Fatal(err)
				}
				want = r.Current
			}
			if math.Abs(point.Values[0]-want) > 2e-9 {
				t.Fatalf("%s bias %g: %g != %g", probe, point.Input, point.Values[0], want)
			}
		}
	}
}
