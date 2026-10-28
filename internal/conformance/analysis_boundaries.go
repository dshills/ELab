package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
)

// AnalysisBoundaryResult records semantic evidence, excluding presentation bytes.
type AnalysisBoundaryResult struct {
	Resistors        int      `json:"resistors"`
	Parts            int      `json:"parts"`
	SweepPoints      []int    `json:"sweep_points"`
	CapturesReopened int      `json:"captures_reopened"`
	NoiseRMS         float64  `json:"noise_rms_v"`
	Edited           bool     `json:"edited"`
	Commands         []string `json:"commands"`
}

// RunAnalysisBoundaries owns its sessions and exercises maximum-width analyses,
// persistence and further edits at small, intermediate and maximum part capacity.
func RunAnalysisBoundaries(ctx context.Context, factory Factory, dir string) ([]AnalysisBoundaryResult, error) {
	out := []AnalysisBoundaryResult{}
	for _, count := range []int{1, 16, 62} {
		result, err := runAnalysisBoundary(ctx, factory, filepath.Join(dir, fmt.Sprint(count)), count)
		if err != nil {
			return out, fmt.Errorf("%d resistors: %w", count, err)
		}
		out = append(out, result)
	}
	return out, nil
}
func runAnalysisBoundary(ctx context.Context, factory Factory, dir string, count int) (out AnalysisBoundaryResult, err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}
	client, err := factory(ctx, filepath.Join(dir, "session"))
	if err != nil {
		return out, err
	}
	s := &session{client: client, ctx: ctx, directory: dir}
	defer func() {
		_, closeErr := s.state("app.close", map[string]bool{"discard_changes": true}, true)
		err = errors.Join(err, closeErr, client.Close())
	}()
	out.Resistors = count
	if _, err = s.state("example.load", workbench.ExampleLoadParams{ID: "noise-rc"}, true); err != nil {
		return
	}
	if _, err = s.state("simulation.pause", nil, true); err != nil {
		return
	}
	for n := 2; n <= count; n++ {
		id := fmt.Sprintf("R%d", n)
		if _, err = s.state("component.place", workbench.PlaceParams{Kind: "resistor", ID: id, Position: workbench.Point{X: 500 + float64(n%8)*180, Y: 400 + float64(n/8)*160}}, true); err != nil {
			return
		}
		for _, pair := range [][2]string{{"V1.positive", id + ".1"}, {id + ".2", "C1.1"}} {
			if _, err = s.state("wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]}, true); err != nil {
				return
			}
		}
	}
	state, err := s.state("simulation.wait", nil, false)
	if err != nil {
		return out, err
	}
	if len(state.Diagnostics) != 0 || state.SimulatedComponents != count+2 {
		return out, fmt.Errorf("unexpected simulation capacity/diagnostics")
	}
	out.Parts = state.SimulatedComponents
	probes := []string{"v:C1", "v:R1", "v:V1"}
	if count > 1 {
		probes[2] = "v:R2"
	}
	expectedResults := []workbench.SweepResult{}
	for _, parameter := range []string{"voltage", "frequency", "noise"} {
		cfg := workbench.SweepSettings{ComponentID: "V1", Parameter: parameter, Start: "0", Stop: "5", Points: 201, ProbeIDs: probes}
		if parameter != "voltage" {
			cfg.Start = "1 Hz"
			cfg.Stop = "100 kHz"
			cfg.Spacing = "logarithmic"
		}
		started, startErr := s.state("sweep.start", workbench.SweepStartParams{Settings: &cfg}, true)
		err = startErr
		if err != nil {
			return
		}
		if started.DCSweep == nil || started.DCSweep.Selected == "" {
			return out, fmt.Errorf("sweep admission missing identity")
		}
		identity := started.DCSweep.Selected
		deadline := time.Now().Add(35 * time.Second)
		for {
			state, err = s.state("project.inspect", nil, false)
			if err != nil {
				return
			}
			if state.DCSweep != nil && len(state.DCSweep.Operations) > 0 {
				op := state.DCSweep.Operations[len(state.DCSweep.Operations)-1]
				if op.ID == identity && op.Status == "completed" {
					break
				}
				if op.ID == identity && (op.Status == "failed" || op.Status == "canceled") {
					return out, fmt.Errorf("sweep %s: %v", parameter, op.Error)
				}
			}
			if time.Now().After(deadline) {
				return out, fmt.Errorf("sweep deadline")
			}
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(20 * time.Millisecond):
			}
		}
		result := state.DCSweep.Result
		width := 3
		if parameter == "frequency" {
			width = 6
		}
		if result == nil || len(result.Points) != 201 || len(result.Probes) != width {
			return out, fmt.Errorf("sweep %s shape", parameter)
		}
		for _, point := range result.Points {
			if len(point.Values) != width {
				return out, fmt.Errorf("sweep row width")
			}
			want := point.Input
			actual := point.Values[0]
			switch parameter {
			case "frequency":
				omega := 2 * math.Pi * point.Input * 1000 / float64(count) * 1e-6
				want = 1 / math.Sqrt(1+omega*omega)
				if len(point.AC) != 3 {
					return out, fmt.Errorf("AC response width")
				}
				actual = math.Hypot(point.AC[0].Real, point.AC[0].Imaginary)
			case "noise":
				omega := 2 * math.Pi * point.Input * 1000 / float64(count) * 1e-6
				want = math.Sqrt(4 * 1.380649e-23 * 300.15 * 1000 / float64(count) / (1 + omega*omega))
			}
			if math.Abs(actual-want) > 1e-9*math.Max(math.Abs(want), 1e-20) {
				return out, fmt.Errorf("independent %s reference: got %g want %g", parameter, actual, want)
			}
		}
		if parameter == "noise" {
			if result.Noise == nil || len(result.Noise.OutputRMS) != 3 {
				return out, fmt.Errorf("noise RMS shape")
			}
			out.NoiseRMS = result.Noise.OutputRMS[0]
		}
		expectedResults = append(expectedResults, *result)
		out.SweepPoints = append(out.SweepPoints, len(result.Points))
	}
	saved := filepath.Join(dir, "analyses.elab")
	if _, err = s.state("project.save", workbench.SaveParams{Path: saved}, true); err != nil {
		return
	}
	before := state
	if _, err = s.state("project.load", workbench.LoadParams{Path: saved, DiscardChanges: true}, true); err != nil {
		return
	}
	state, err = s.state("simulation.wait", nil, false)
	if err != nil {
		return out, err
	}
	data, e := os.ReadFile(saved)
	if e != nil {
		return out, e
	}
	var document workbench.Document
	if e = json.Unmarshal(data, &document); e != nil {
		return out, e
	}
	if !state.Paused || state.Document.Dirty || !reflect.DeepEqual(before.Components, state.Components) || document.Sweeps == nil || len(document.Sweeps.Entries) != 3 {
		return out, fmt.Errorf("reopen graph/captures")
	}
	for i, record := range document.Sweeps.Entries {
		if !reflect.DeepEqual(record.Result, expectedResults[i]) {
			return out, fmt.Errorf("saved analysis %d changed", i)
		}
	}
	out.CapturesReopened = len(document.Sweeps.Entries)
	if _, err = s.state("component.set", map[string]string{"id": "R1", "value": "2k"}, true); err != nil {
		return
	}
	state, err = s.state("simulation.wait", nil, false)
	if err != nil {
		return out, err
	}
	if state.DCSweep == nil || len(state.DCSweep.Operations) != 3 {
		return out, fmt.Errorf("captures missing after edit")
	}
	for _, op := range state.DCSweep.Operations {
		if op.SourceUnchanged {
			return out, fmt.Errorf("edit did not retire capture source association")
		}
	}
	out.Edited = true
	if _, err = s.state("project.save", workbench.SaveParams{Path: filepath.Join(dir, "edited.elab")}, true); err != nil {
		return
	}
	if _, err = s.state("project.load", workbench.LoadParams{Path: filepath.Join(dir, "edited.elab"), DiscardChanges: true}, true); err != nil {
		return
	}
	state, err = s.state("simulation.wait", nil, false)
	if err != nil {
		return out, err
	}
	part, _ := (workbench.Graph{Components: state.Components}).Component("R1")
	if part.Value != 2000 || state.Document.Dirty {
		return out, fmt.Errorf("post-capture edit lost")
	}
	out.Commands = append([]string{}, s.commands...)
	return
}

// CompareAnalysisBoundaries refuses missing cases or numerical/action evidence.
func CompareAnalysisBoundaries(a, b []AnalysisBoundaryResult) error {
	if len(a) != 3 || len(b) != 3 {
		return fmt.Errorf("missing analysis boundary cases")
	}
	for i, count := range []int{1, 16, 62} {
		for _, r := range []AnalysisBoundaryResult{a[i], b[i]} {
			counts := map[string]int{}
			for _, command := range r.Commands {
				counts[command]++
			}
			if counts["sweep.start"] != 3 || counts["project.save"] != 2 || counts["project.load"] != 2 || counts["component.set"] != 1 {
				return fmt.Errorf("missing boundary actions")
			}
			if r.Resistors != count || r.Parts != count+2 || !reflect.DeepEqual(r.SweepPoints, []int{201, 201, 201}) || r.CapturesReopened != 3 || !r.Edited || math.IsNaN(r.NoiseRMS) || math.IsInf(r.NoiseRMS, 0) || r.NoiseRMS <= 0 || len(r.Commands) == 0 {
				return fmt.Errorf("incomplete boundary evidence")
			}
		}
		actions := func(in []string) []string {
			out := []string{}
			for _, c := range in {
				if c != "project.inspect" {
					out = append(out, c)
				}
			}
			return out
		}
		if !reflect.DeepEqual(actions(a[i].Commands), actions(b[i].Commands)) || math.Abs(a[i].NoiseRMS-b[i].NoiseRMS) > 1e-10*math.Max(a[i].NoiseRMS, b[i].NoiseRMS) {
			return fmt.Errorf("boundary parity differs")
		}
	}
	return nil
}
