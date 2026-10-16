package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

const inductorCheckCount = 22

type InductorObservation struct {
	Name   string                `json:"name"`
	Result workbench.MixedResult `json:"result"`
}

type InductorResult struct {
	Engine       string                `json:"engine"`
	Calls        []Call                `json:"calls"`
	Checks       []ElectricalCheck     `json:"checks"`
	Observations []InductorObservation `json:"observations"`
}

// RunInductors constructs RL from blank and exercises all three coil examples
// plus the coupled RC/NPN/coil regression, using only public commands. The
// analytic RL/RLC checks are distinct from cross-adapter equivalence.
func RunInductors(ctx context.Context, factory Factory, dir string) (out InductorResult, err error) {
	out.Engine = workbench.InductorEngineID
	checks := ElectricalResult{}
	defer func() { out.Checks = checks.Checks }()
	err = electricalSession(ctx, factory, dir, func(s *session) error {
		observe := func(name string, state workbench.Snapshot) error {
			r := state.Mixed
			if r == nil || r.Stale || r.Engine != out.Engine || len(r.Inductors) != 1 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
				return fmt.Errorf("%s lacks fresh conservative coil output: %v", name, state.Diagnostics)
			}
			copy := *r
			copy.RunID, copy.ElectricalRevision = 0, 0
			out.Observations = append(out.Observations, InductorObservation{name, copy})
			return nil
		}
		fixture := circuit{parts: []part{{"V1", "dc-source", "5 V", -200, 0}, {"R1", "resistor", "100 Ω", 0, -100}, {"L1", "inductor", "1 H", 200, 0}, {"GND1", "ground", "", 0, 160}}, wires: [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "L1.1"}, {"L1.2", "GND1.reference"}, {"V1.negative", "GND1.reference"}}}
		if err := buildElectrical(s, fixture); err != nil {
			return err
		}
		if err := s.ask("inductor.initial.set", workbench.ValueParams{ID: "L1", Value: "-20 mA"}, true, nil); err != nil {
			return err
		}
		initial, err := restart(s)
		if err != nil {
			return err
		}
		if err = observe("signed-startup", initial); err != nil {
			return err
		}
		for _, item := range []struct {
			name         string
			actual, want float64
		}{{"startup-current-a", initial.Mixed.Inductors[0].Current, -.02}, {"startup-voltage-v", initial.Mixed.Inductors[0].Voltage, 7}, {"startup-energy-j", initial.Mixed.Inductors[0].Energy, .0002}} {
			if err = checks.check(item.name, item.actual, item.want, 1e-12); err != nil {
				return err
			}
		}
		probes := []string{"i:L1", "v:L1"}
		if err = s.ask("scope.probes.set", workbench.ProbeParams{IDs: probes}, true, nil); err != nil {
			return err
		}
		live, err := s.state("simulation.step", workbench.StepParams{Duration: .005}, true)
		if err != nil {
			return err
		}
		if err = observe("built-rl-5ms", live); err != nil {
			return err
		}
		rlCheck := func(name string, at, current, voltage float64) error {
			want := .05 - .07*math.Exp(-at/.01)
			if err := checks.check(name+"-current-a", current, want, 1e-5); err != nil {
				return err
			}
			return checks.check(name+"-voltage-v", voltage, 5-100*want, .001)
		}
		if err = rlCheck("live", .005, live.Mixed.Inductors[0].Current, live.Mixed.Inductors[0].Voltage); err != nil {
			return err
		}
		metadata, rows, _, err := electricalVerify(s, .005, .001, probes)
		if err != nil {
			return err
		}
		if metadata.Engine != out.Engine || len(rows) != 6 {
			return fmt.Errorf("coil capture coverage/engine differs")
		}
		for i, row := range rows {
			if len(row.Values) != 2 || math.Abs(row.Time-float64(i)*.001) > 1e-12 {
				return fmt.Errorf("coil capture omitted probe values or planned output times")
			}
			if err = rlCheck(fmt.Sprintf("capture-%d", i), row.Time, row.Values[0], row.Values[1]); err != nil {
				return err
			}
		}
		after, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(after.Mixed, live.Mixed) || !reflect.DeepEqual(after.Transient, live.Transient) {
			return fmt.Errorf("coil Verify changed accepted live state")
		}
		path := filepath.Join(dir, "signed-rl.elab")
		if err = s.ask("project.save", workbench.SaveParams{Path: path}, true, nil); err != nil {
			return err
		}
		original, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err = electricalSession(ctx, factory, filepath.Join(dir, "reopened"), func(fresh *session) error {
			if err := fresh.ask("project.load", workbench.LoadParams{Path: path}, true, nil); err != nil {
				return err
			}
			state, err := fresh.state("simulation.wait", nil, false)
			if err != nil {
				return err
			}
			if err = observe("reopened-startup", state); err != nil {
				return err
			}
			if !state.Paused || state.Document.RecordingCount != 1 || !reflect.DeepEqual(state.Scope.Selected, probes) {
				return fmt.Errorf("coil reopen lost paused state, probes or capture")
			}
			if err = checks.check("reopened-current-a", state.Mixed.Inductors[0].Current, -.02, 1e-12); err != nil {
				return err
			}
			return checks.check("reopened-time-s", state.Transient.Sample.Time, 0, 0)
		}); err != nil {
			return err
		}
		unchanged, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(original, unchanged) {
			return fmt.Errorf("coil reopen changed original project bytes")
		}
		for _, id := range []string{"rl", "rlc", "flyback"} {
			if err = s.ask("example.load", workbench.ExampleLoadParams{ID: id, DiscardChanges: true}, true, nil); err != nil {
				return err
			}
			if err = s.ask("simulation.pause", nil, true, nil); err != nil {
				return err
			}
			if _, err = restart(s); err != nil {
				return err
			}
			duration := .05
			if id == "rlc" {
				duration = .005
			}
			state, err := s.state("simulation.step", workbench.StepParams{Duration: duration}, true)
			if err != nil {
				return err
			}
			if err = observe(id, state); err != nil {
				return err
			}
			switch id {
			case "rl":
				err = checks.check("rl-50ms-current-a", state.Mixed.Inductors[0].Current, .05*(1-math.Exp(-5)), 1e-5)
			case "rlc":
				omega, at := math.Sqrt(1e6-250*250), duration
				current := 5 / (.1 * omega) * math.Exp(-250*at) * math.Sin(omega*at)
				voltage := 5 * (1 - math.Exp(-250*at)*(math.Cos(omega*at)+250/omega*math.Sin(omega*at)))
				if err = checks.check("rlc-5ms-current-a", state.Mixed.Inductors[0].Current, current, 1e-5); err == nil {
					err = checks.check("rlc-5ms-capacitor-v", state.Mixed.Capacitors[0].Voltage, voltage, .001)
				}
			case "flyback":
				r := state.Mixed
				if len(r.MOSFETs) != 1 || len(r.Diodes) != 1 || r.MOSFETs[0].State != "cutoff" || r.Diodes[0].Current < .045 || r.Inductors[0].Voltage >= -5 {
					return fmt.Errorf("flyback lacks turn-off return current and clamp")
				}
			}
			if err != nil {
				return err
			}
		}
		if err = buildInductorNPNExperiment(s); err != nil {
			return err
		}
		state, err := s.state("simulation.step", workbench.StepParams{Duration: .12}, true)
		if err != nil {
			return err
		}
		if err = observe("mixed-rc-npn-coil", state); err != nil {
			return err
		}
		if state.Transient.Sample.Time != .12 || len(state.Mixed.Capacitors) != 1 || len(state.Mixed.Transistors) != 1 || state.Mixed.Transistors[0].TransistorID != "QCOIL" || state.Mixed.Transistors[0].State != "saturation" || state.Mixed.Inductors[0].Current < .03 || state.Mixed.Inductors[0].Current > .05 {
			return fmt.Errorf("coupled coil regression did not complete its period")
		}
		out.Calls = append([]Call{}, s.calls...)
		return nil
	})
	return out, err
}

func buildInductorNPNExperiment(s *session) error {
	for _, call := range []struct {
		command string
		params  any
	}{
		{"object.delete", workbench.IDParams{ID: "M1"}},
		{"component.place", workbench.PlaceParams{ID: "QCOIL", Kind: "npn", Position: workbench.Point{X: 200, Y: 200}}},
		{"wire.connect", map[string]string{"from": "JDR.node", "to": "QCOIL.collector"}},
		{"wire.connect", map[string]string{"from": "JG.node", "to": "QCOIL.base"}},
		{"wire.connect", map[string]string{"from": "QCOIL.emitter", "to": "GND1.reference"}},
		{"component.set", workbench.ValueParams{ID: "RG", Value: "10k Ω"}},
		{"component.place", workbench.PlaceParams{ID: "CB", Kind: "capacitor", Value: "1 µF", Position: workbench.Point{X: 700, Y: 200}}},
		{"wire.connect", map[string]string{"from": "CB.1", "to": "QCOIL.base"}},
		{"wire.connect", map[string]string{"from": "CB.2", "to": "GND1.reference"}},
		{"scope.probes.set", workbench.ProbeParams{IDs: []string{"i:L1", "v:QCOIL.collector", "v:CB", "v:VIN.positive"}}},
	} {
		if err := s.ask(call.command, call.params, true, nil); err != nil {
			return err
		}
	}
	_, err := restart(s)
	return err
}

// CompareInductors rejects missing coverage as well as differing observations.
// Only runtime run/revision provenance is normalized at observation capture.
func CompareInductors(reference, actual InductorResult) error {
	if reference.Engine != workbench.InductorEngineID || reference.Engine != actual.Engine || len(reference.Checks) != inductorCheckCount || len(actual.Checks) != inductorCheckCount || len(reference.Observations) != 7 || len(actual.Observations) != 7 || len(reference.Calls) == 0 || !reflect.DeepEqual(collapseCalls(reference.Calls), collapseCalls(actual.Calls)) {
		return fmt.Errorf("coil engine/command/reference coverage differs")
	}
	for k, name := range []string{"signed-startup", "built-rl-5ms", "reopened-startup", "rl", "rlc", "flyback", "mixed-rc-npn-coil"} {
		if reference.Observations[k].Name != name || actual.Observations[k].Name != name {
			return fmt.Errorf("coil observation %d omitted %s", k, name)
		}
	}
	for k, check := range reference.Checks {
		other := actual.Checks[k]
		if !check.Passed || !other.Passed || check.Name == "" || check.Name != other.Name || check.Reference != other.Reference || check.Tolerance != other.Tolerance || !finiteConformance(check.Actual) || !finiteConformance(other.Actual) || math.Abs(check.Actual-check.Reference) > check.Tolerance || math.Abs(other.Actual-other.Reference) > other.Tolerance || math.Abs(check.Actual-other.Actual) > 1e-10 {
			return fmt.Errorf("coil reference check %d differs", k)
		}
	}
	left, err := json.Marshal(reference.Observations)
	if err != nil {
		return err
	}
	right, err := json.Marshal(actual.Observations)
	if err != nil {
		return err
	}
	if !bytes.Equal(left, right) {
		return fmt.Errorf("coil numerical observations differ")
	}
	return nil
}

func finiteConformance(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
