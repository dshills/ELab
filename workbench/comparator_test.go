package workbench

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"
)

func TestComparatorHysteresisThresholdsPowerAndJacobian(t *testing.T) {
	p, _ := MakeComponent("U1", "comparator", Point{}, "200mV")
	for _, ground := range []float64{-2, 0, 3} {
		for _, span := range []float64{2, 3.3, 5, 12} {
			v := [5]float64{ground + 1, ground + 1, ground + .5, ground + span, ground}
			m := ControlMemory{}
			for _, sample := range []struct {
				diff float64
				high bool
			}{{0, false}, {.09, false}, {.1, true}, {.02, true}, {-.09, true}, {-.1, false}, {0, false}} {
				v[0] = v[1] + sample.diff
				next, err := controlDecision(p, v, m, 0, 0)
				if err != nil || next.High != sample.high || !next.Powered {
					t.Fatal("hysteresis", span, ground, sample, next, err)
				}
				m = next
			}
			for _, high := range []bool{false, true} {
				memory := ControlMemory{Powered: true, High: high}
				e := controlCurrents(p, v, memory)
				sum, power := 0., 0.
				for k, i := range e.Current {
					sum += i
					power += v[k] * i
				}
				if math.Abs(sum) > 1e-12 || power < -1e-12 {
					t.Fatal("comparator passivity")
				}
				for col := range v {
					a, b := v, v
					a[col] += 1e-6
					b[col] -= 1e-6
					ea, eb := controlCurrents(p, a, memory), controlCurrents(p, b, memory)
					for row := range v {
						want := (ea.Current[row] - eb.Current[row]) / 2e-6
						if math.Abs(want-e.Jacobian[row][col]) > 1e-9 {
							t.Fatal("control Jacobian")
						}
					}
				}
			}
			v[3] = ground + 1.9
			off, err := controlDecision(p, v, m, 0, 0)
			if err != nil || off != (ControlMemory{}) {
				t.Fatal("unpowered comparator")
			}
			v[3] = ground + 12.1
			if _, err := controlDecision(p, v, m, 0, 0); err == nil {
				t.Fatal("comparator overvoltage")
			}
		}
	}
}
func TestComparatorSineCrossingsAndAnalogClockCounter(t *testing.T) {
	for _, id := range []string{"comparator-threshold", "comparator-counter"} {
		t.Run(id, func(t *testing.T) {
			definition, _ := findExample(id)
			c, _, d := definition.graph.Compile()
			if d != nil {
				t.Fatal(d)
			}
			kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: definition.settings, mode: "auto"})
			if err != nil {
				t.Fatal(err)
			}
			edges := []float64{}
			// Live traces roll; collect newly retained event pairs each step.
			for k := 1; k <= 1250; k++ {
				end := float64(k) * .002
				if err := kernel.advance(t.Context(), end); err != nil {
					t.Fatal(k, err)
				}
				for j := 1; j < len(kernel.trace.Samples); j++ {
					a, b := kernel.trace.Samples[j-1], kernel.trace.Samples[j]
					if b.Time > float64(k-1)*.002 && a.Time == b.Time && (a.frame.values["v:U1.out"] > 3.5) != (b.frame.values["v:U1.out"] > 3.5) {
						edges = append(edges, b.Time)
					}
				}
			}

			phase := math.Asin(.1/.4) / (2 * math.Pi)
			if len(edges) != 5 {
				t.Fatal("missed comparator edges", edges)
			}
			for k, at := range edges {
				want := phase + float64(k)*.5
				if math.Abs(at-want) > 2e-8 {
					t.Fatal("threshold localization", k, at, want)
				}
			}
			if id == "comparator-counter" {
				if kernel.mixed.result.Sequential[0].Memory.Value != 3 {
					t.Fatal("analog clock did not count rising edges")
				}
				for _, d := range kernel.mixed.result.Displays {
					if d.ID == "DS1" && d.Digit != 3 {
						t.Fatal("comparator display")
					}
				}
			}
			if kernel.mixed.result.KCLResidual > 1e-8 || math.Abs(kernel.mixed.result.PowerResidual) > 1e-7 {
				t.Fatal("comparator balance")
			}
		})
	}
}
func TestComparatorSharedCapturePersistenceAndValueDomain(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "comparator-threshold"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "component.set", ValueParams{ID: "U1", Value: "100 mV"}, "hysteresis")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	refuse(t, s, "component.set", ValueParams{ID: "U1", Value: "0 V"}, "invalid-params")
	duration, step := .1, .002
	_, view := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:U1", "i:U1", "v:U1.input-difference"}}, "capture")
	record := s.operations[view.Operation.ID].record
	raw, _ := json.Marshal(VerificationDocument{Metadata: record.metadata, Rows: record.rows})
	if _, err := decodeRecording(raw); err != nil {
		t.Fatal("comparator capture", err)
	}
	path := filepath.Join(t.TempDir(), "comparator.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	a := settled(t, fresh)
	p, _ := fresh.graph.Component("U1")
	if p.Value != .1 || a.ControlModel == nil || a.Document.RecordingCount != 1 {
		t.Fatal("comparator reopen")
	}
	a.Mixed.Controls[0].Memory.High = !a.Mixed.Controls[0].Memory.High
	if err := validateRecordedControls(fresh.mixedState.net.c, fresh.graph, a.Mixed); err == nil {
		t.Fatal("forged comparator memory")
	}
}
