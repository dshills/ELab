package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpAmpGuidedSineReferencesCaptureAndSavedOrigin(t *testing.T) {
	for _, id := range []string{"opamp-follower", "opamp-inverting"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.Example == nil || initial.Example.Modified || initial.Selected != "U1" || initial.Analysis != WaveformAnalysis || initial.OpAmpModel == nil || initial.WaveformModel == nil || len(initial.Scope.Selected) != 4 {
				t.Fatal("guided amplifier lacks source identity or useful opening state", initial.Diagnostics)
			}
			duration, step := .1, .001
			_, capture := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: initial.Scope.Selected}, "verify")
			if !reflect.DeepEqual(initial.Mixed, s.Snapshot().Mixed) || !reflect.DeepEqual(initial.Transient, s.Snapshot().Transient) || capture.Metadata.OpAmpModel == nil || capture.Metadata.WaveformModel == nil {
				t.Fatal("Verify lost either model or changed live state")
			}
			rows := call(t, s, "operation.samples", VerificationPageParams{ID: capture.Operation.ID, Limit: 200}, "rows").Result.(VerificationPage)
			if len(rows.Rows) != 101 || rows.NextOffset != rows.Total {
				t.Fatal("incomplete one-cycle capture", rows.Total)
			}
			gain, amplitude := 1., 1.
			if id == "opamp-inverting" {
				gain, amplitude = -2, .5
			}
			for _, row := range rows.Rows {
				input := amplitude * math.Sin(20*math.Pi*row.Time)
				if math.Abs(row.Values[0]-input) > 1e-11 || math.Abs(row.Values[1]-gain*input) > .0001 {
					t.Fatal("captured gain/phase differs from independent sine/amplifier envelope", row)
				}
				load := row.Values[1] / 10000
				if id == "opamp-inverting" {
					load += (row.Values[1] + row.Values[3]) / 20000
				}
				if math.Abs(row.Values[2]-load) > 1e-10 {
					t.Fatal("delivered output current lost its load/feedback sign", row, load)
				}
				if math.Abs(row.Time-.025) < 1e-12 {
					want := .999989811376
					if id == "opamp-inverting" {
						want = -.999969336181
					}
					if math.Abs(row.Values[1]-want) > 1e-7 {
						t.Fatal("positive peak differs from frozen independent Decimal amplifier reference", row, want)
					}
				}
			}
			call(t, s, "simulation.step", StepParams{Duration: .025}, "positive-peak")
			live := s.Snapshot()
			if math.Abs(live.Mixed.OpAmps[0].OutputVoltage-gain*amplitude) > .0001 || live.Mixed.KCLResidual > 1e-9 || math.Abs(live.Mixed.PowerResidual) > 1e-8 {
				t.Fatal("guided live peak/conservation", live.Mixed)
			}
			path := filepath.Join(t.TempDir(), "guided.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "load-file")
			reopened := settled(t, fresh)
			if reopened.Example == nil || reopened.Example.Info.ID != id || reopened.Example.Modified || reopened.Document.RecordingCount != 1 || !reflect.DeepEqual(reopened.Scope.Selected, initial.Scope.Selected) {
				t.Fatal("reopen lost learning origin, probes or capture")
			}
		})
	}
}

func TestOpAmpGuidedLoadLimitGainEditAndReset(t *testing.T) {
	for _, id := range []string{"opamp-follower", "opamp-inverting"} {
		s := New()
		call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
		call(t, s, "simulation.pause", struct{}{}, "pause")
		settled(t, s)
		part, value := "RL", "10 Ω"
		if id == "opamp-inverting" {
			part, value = "RF", "10k"
		}
		call(t, s, "component.set", ValueParams{ID: part, Value: value}, "edit")
		if !s.Snapshot().Example.Modified || !s.Snapshot().Mixed.Stale {
			t.Fatal("paused lesson edit lost modified/stale state")
		}
		call(t, s, "simulation.restart", struct{}{}, "restart")
		settled(t, s)
		call(t, s, "simulation.step", StepParams{Duration: .025}, "peak")
		u := s.Snapshot().Mixed.OpAmps[0]
		if id == "opamp-follower" {
			if u.OutputVoltage < .19 || u.OutputVoltage > .201 || u.OutputCurrent < .019 || u.OutputCurrent > .020001 {
				t.Fatal("loaded follower failed to expose declared active current scale", u)
			}
			// Warm continuation must leave overload, sink on the opposite half
			// cycle and recover again without changing solver work bounds.
			for i, point := range []struct{ duration, sign float64 }{{.025, 0}, {.025, -1}, {.025, 0}, {.025, 1}} {
				call(t, s, "simulation.step", StepParams{Duration: point.duration}, fmt.Sprintf("overload-cycle-%d", i))
				r := s.Snapshot().Mixed
				u = r.OpAmps[0]
				if point.sign == 0 && math.Abs(u.OutputVoltage) > .0001 || point.sign != 0 && (point.sign*u.OutputVoltage < .19 || point.sign*u.OutputVoltage > .201 || point.sign*u.OutputCurrent < .019 || point.sign*u.OutputCurrent > .020001) || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
					t.Fatal("warm loaded follower failed signed recovery/conservation", point, u, r.KCLResidual, r.PowerResidual)
				}
			}
		} else if math.Abs(u.OutputVoltage+.5) > .0001 {
			t.Fatal("feedback edit did not halve inverted amplitude", u)
		}
		refuse(t, s, "example.current.reset", ExampleResetParams{}, "unsaved-changes")
		call(t, s, "example.current.reset", ExampleResetParams{ExpectedID: id, DiscardChanges: true}, "reset")
		reset := settled(t, s)
		if reset.Example.Modified || !reset.Paused || reset.Transient.Sample.Time != 0 || reset.Mixed.Stale {
			t.Fatal("reset lost paused fresh template identity", reset.Diagnostics)
		}
		s.Close()
	}
}
