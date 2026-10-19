package conformance

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/dshills/elab/workbench"
)

// Frozen before acceptance; regenerated only by the independent Decimal script.
//
//go:embed testdata/electrical-reference.json
var electricalReference []byte

const electricalCheckCount = 294

type referenceLED struct {
	Resistance string `json:"resistance_ohm"`
	Current    string `json:"current_a"`
	Voltage    string `json:"voltage_v"`
}
type referenceNPN struct {
	Drive     string `json:"drive_v"`
	Base      string `json:"base_v"`
	Collector string `json:"collector_v"`
	IB        string `json:"base_current_a"`
	IC        string `json:"collector_current_a"`
}
type referenceRC struct {
	Time    string `json:"time_s"`
	Source  string `json:"source_v"`
	Voltage string `json:"voltage_v"`
}
type frozenElectrical struct {
	Version      int               `json:"version"`
	Method       string            `json:"method"`
	LEDModel     map[string]string `json:"led_model"`
	NPNModel     map[string]string `json:"npn_model"`
	RCParameters map[string]string `json:"rc_parameters"`
	LED          []referenceLED    `json:"led"`
	NPN          []referenceNPN    `json:"npn"`
	Ideal        []referenceRC     `json:"rc_ideal"`
	Finite       []referenceRC     `json:"rc_finite"`
	PeriodicLow  string            `json:"rc_periodic_low_v"`
	PeriodicHigh string            `json:"rc_periodic_high_v"`
}
type ElectricalCheck struct {
	Name          string  `json:"name"`
	Actual        float64 `json:"actual"`
	Reference     float64 `json:"reference"`
	AbsoluteError float64 `json:"absolute_error"`
	Tolerance     float64 `json:"tolerance"`
	Passed        bool    `json:"passed"`
}
type ElectricalResult struct {
	ReferenceSHA256 string               `json:"reference_sha256"`
	ReferenceMethod string               `json:"reference_method"`
	Checks          []ElectricalCheck    `json:"checks"`
	Refinement      []FlasherMeasurement `json:"flasher_refinement"`
	Limitations     []string             `json:"limitations"`
}
type FlasherMeasurement struct {
	Step           float64            `json:"output_step_s"`
	Samples        int                `json:"samples"`
	RisingBrackets [][2]float64       `json:"rising_edge_brackets_s"`
	Period         float64            `json:"median_period_s"`
	Minimum        map[string]float64 `json:"minimum_si"`
	Maximum        map[string]float64 `json:"maximum_si"`
	VerifyWall     float64            `json:"verify_200ms_wall_s"`
	GraphSHA256    string             `json:"graph_sha256"`
}

func refNumber(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid frozen number %q", s)
	}
	return v, nil
}
func referenceModelMatches(expected map[string]string, id string, values map[string]float64) error {
	if expected["id"] != id || len(expected) != len(values)+1 {
		return fmt.Errorf("frozen model identity/parameter coverage differs")
	}
	for key, actual := range values {
		want, err := refNumber(expected[key])
		if err != nil {
			return err
		}
		if want != actual {
			return fmt.Errorf("frozen model %s differs: %g/%g", key, want, actual)
		}
	}
	return nil
}
func (out *ElectricalResult) check(name string, actual, reference, tolerance float64) error {
	delta := math.Abs(actual - reference)
	passed := !math.IsNaN(actual) && !math.IsInf(actual, 0) && delta <= tolerance
	out.Checks = append(out.Checks, ElectricalCheck{name, actual, reference, delta, tolerance, passed})
	if !passed {
		return fmt.Errorf("%s: got %.12g, reference %.12g, error %.3g exceeds %.3g", name, actual, reference, delta, tolerance)
	}
	return nil
}
func (out *ElectricalResult) frozenCheck(name string, actual float64, reference string, absolute, relative float64) error {
	want, err := refNumber(reference)
	if err != nil {
		return err
	}
	return out.check(name, actual, want, math.Max(absolute, relative*math.Abs(want)))
}
func electricalSession(ctx context.Context, factory Factory, dir string, run func(*session) error) (err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	client, err := factory(ctx, filepath.Join(dir, "process"))
	if err != nil {
		return err
	}
	s := &session{client: client, ctx: ctx, directory: dir}
	defer func() {
		// Socket/native processes have no stdin EOF. Close our owned domain
		// session explicitly, including failed runs, before waiting for exit.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s.ctx = cleanup
		_, inspectErr := s.state("project.inspect", nil, false)
		if inspectErr == nil {
			inspectErr = s.closeElectrical()
		}
		cancel()
		err = errors.Join(err, inspectErr, client.Close())
	}()
	state, err := s.state("project.inspect", nil, false)
	if err != nil {
		return err
	}
	if len(state.Components) != 0 || len(state.Wires) != 0 {
		return fmt.Errorf("electrical factory must create an empty owned session")
	}
	return run(s)
}
func buildElectrical(s *session, c circuit) error {
	if err := s.ask("simulation.pause", nil, true, nil); err != nil {
		return err
	}
	for _, p := range c.parts {
		if err := s.ask("component.place", workbench.PlaceParams{ID: p.id, Kind: p.kind, Value: p.value, Position: workbench.Point{X: p.x, Y: p.y}}, true, nil); err != nil {
			return err
		}
	}
	for _, w := range c.wires {
		if err := s.ask("wire.connect", map[string]string{"from": w[0], "to": w[1]}, true, nil); err != nil {
			return err
		}
	}
	_, err := restart(s)
	return err
}

// RunElectrical uses only the same public commands available to users/agents.
// It adds independently frozen references and sample-step refinement; adapter
// equivalence alone is deliberately not used as a numerical-accuracy test.
func RunElectrical(ctx context.Context, factory Factory, dir string) (out ElectricalResult, err error) {
	var frozen frozenElectrical
	if err = json.Unmarshal(electricalReference, &frozen); err != nil {
		return out, err
	}
	if frozen.Version != 1 || len(frozen.LED) != 2 || len(frozen.NPN) != 3 || len(frozen.Ideal) != 22 || len(frozen.Finite) != 15 {
		return out, fmt.Errorf("missing frozen electrical coverage")
	}
	out.ReferenceSHA256 = fmt.Sprintf("%x", sha256.Sum256(electricalReference))
	out.ReferenceMethod = frozen.Method
	out.Limitations = []string{"independent high-precision implementations of pinned educational equations; no physical-device or independently configured SPICE qualification", "flasher refinement compares the same adaptive kernel at three output steps; it is not an independent flasher reference", "timings are backend Verify completion on this run, not p95 input-to-visible or edit-to-result qualification", "existing model, 1000-step, rolling-history and integration work limits remain enforced"}
	defer func() {
		data, e := json.MarshalIndent(out, "", "  ")
		if e == nil {
			e = os.WriteFile(filepath.Join(dir, "electrical.json"), append(data, '\n'), 0600)
		}
		err = errors.Join(err, e)
	}()
	if err = os.MkdirAll(dir, 0700); err != nil {
		return out, err
	}
	err = electricalSession(ctx, factory, filepath.Join(dir, "led"), func(s *session) error {
		if err := buildElectrical(s, circuits[0]); err != nil {
			return err
		}
		for _, ref := range frozen.LED {
			if err := s.ask("component.set", workbench.ValueParams{ID: "Limit", Value: ref.Resistance + " ohm"}, true, nil); err != nil {
				return err
			}
			state, err := restart(s)
			if err != nil {
				return err
			}
			r := state.Result
			if r == nil || r.Stale {
				return fmt.Errorf("LED result unavailable")
			}
			if err = referenceModelMatches(frozen.LEDModel, r.Model.ID, map[string]float64{"is_a": r.Model.SaturationCurrent, "n": r.Model.EmissionCoefficient, "rs_ohm": r.Model.SeriesResistance, "temperature_c": r.Model.Temperature}); err != nil {
				return err
			}
			if err = out.frozenCheck("LED current R="+ref.Resistance, r.Current, ref.Current, 0, .01); err != nil {
				return err
			}
			if err = out.frozenCheck("LED voltage R="+ref.Resistance, r.LEDVoltage, ref.Voltage, .001, .01); err != nil {
				return err
			}
			if err = out.check("LED independent KVL R="+ref.Resistance, 5-r.Current*state.Circuit.Resistance-r.LEDVoltage, 0, 1e-10); err != nil {
				return err
			}
			currents := map[string]float64{}
			for _, id := range []string{"i:Supply", "i:Limit", "i:Light"} {
				var inspection workbench.MeasurementInspection
				if err = s.ask("measurement.inspect", workbench.MeasurementQueryParams{Target: &workbench.MeasurementSelectParams{Kind: "probe", ID: id}}, false, &inspection); err != nil {
					return err
				}
				if !inspection.Available || len(inspection.Readings) != 1 || inspection.Readings[0].Value == nil {
					return fmt.Errorf("LED KCL probe unavailable: %s", id)
				}
				currents[id] = *inspection.Readings[0].Value
			}
			if err = out.check("LED supply KCL R="+ref.Resistance, currents["i:Supply"]-currents["i:Limit"], 0, 1e-10); err != nil {
				return err
			}
			if err = out.check("LED junction KCL R="+ref.Resistance, currents["i:Limit"]-currents["i:Light"], 0, 1e-10); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	err = electricalSession(ctx, factory, filepath.Join(dir, "npn"), func(s *session) error {
		if err := buildElectrical(s, circuits[2]); err != nil {
			return err
		}
		for _, ref := range frozen.NPN {
			if err := s.ask("component.set", workbench.ValueParams{ID: "Drive", Value: ref.Drive + " V"}, true, nil); err != nil {
				return err
			}
			state, err := restart(s)
			if err != nil {
				return err
			}
			r := state.NPN
			if r == nil || r.Stale {
				return fmt.Errorf("NPN result unavailable")
			}
			if err = referenceModelMatches(frozen.NPNModel, r.Model.ID, map[string]float64{"saturation_current_a": r.Model.SaturationCurrent, "forward_beta": r.Model.ForwardBeta, "reverse_beta": r.Model.ReverseBeta, "temperature_c": r.Model.Temperature}); err != nil {
				return err
			}
			for _, p := range []struct {
				name string
				got  float64
				want string
				abs  float64
			}{{"VCE", r.VCE, ref.Collector, .001}, {"VBE", r.VBE, ref.Base, .001}, {"IB", r.BaseCurrent, ref.IB, 1e-9}, {"IC", r.CollectorCurrent, ref.IC, 1e-9}} {
				if err = out.frozenCheck("NPN "+p.name+" drive="+ref.Drive, p.got, p.want, p.abs, .01); err != nil {
					return err
				}
			}
			if err = out.check("NPN KCL drive="+ref.Drive, r.KCLResidual, 0, 1e-10); err != nil {
				return err
			}
			if err = out.check("NPN independent load KVL drive="+ref.Drive, 5-r.VCE-1000*r.CollectorCurrent, 0, 1e-10); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	for _, finite := range []bool{false, true} {
		name, refs := "rc-ideal", frozen.Ideal
		if finite {
			name, refs = "rc-finite", frozen.Finite
		}
		err = electricalSession(ctx, factory, filepath.Join(dir, name), func(s *session) error {
			c := circuits[1]
			c.parts = append([]part(nil), c.parts...)
			if finite {
				c.parts[0].kind = "finite-pulse-source"
			}
			if err := buildElectrical(s, c); err != nil {
				return err
			}
			for _, step := range []float64{.001, .0005} {
				metadata, rows, _, err := electricalVerify(s, .4, step, []string{"v:Storage.1", "i:Limit", "v:Supply.positive"})
				if err != nil {
					return err
				}
				engine := workbench.RCEngineID
				if finite {
					engine = workbench.WaveformEngineID
				}
				if metadata.Engine != engine {
					return fmt.Errorf("RC engine changed: %s", metadata.Engine)
				}
				parameters := map[string]float64{}
				for _, component := range metadata.Graph.Components {
					switch component.ID {
					case "Limit":
						parameters["resistance_ohm"] = component.Value
					case "Storage":
						parameters["capacitance_f"] = component.Value
					case "Supply":
						parameters["high_v"], parameters["low_v"] = component.Value, 0
						parameters["high_s"], parameters["low_s"] = metadata.Settings.HighSeconds, metadata.Settings.LowSeconds
						if finite {
							if component.Pulse == nil {
								return fmt.Errorf("finite pulse metadata missing")
							}
							p := component.Pulse
							parameters["high_v"], parameters["low_v"], parameters["high_s"], parameters["low_s"], parameters["rise_s"], parameters["fall_s"] = p.HighVoltage, p.LowVoltage, p.HighSeconds, p.LowSeconds, p.RiseSeconds, p.FallSeconds
						}
					}
				}
				if len(parameters) != 6 && !finite || len(parameters) != 8 && finite {
					return fmt.Errorf("RC parameter coverage incomplete")
				}
				for key, value := range parameters {
					want, e := refNumber(frozen.RCParameters[key])
					if e != nil {
						return e
					}
					if value != want {
						return fmt.Errorf("RC stimulus/parameter %s differs", key)
					}
				}
				for _, ref := range refs {
					at, e := refNumber(ref.Time)
					if e != nil {
						return e
					}
					source, e := refNumber(ref.Source)
					if e != nil {
						return e
					}
					found := false
					for _, row := range rows {
						if math.Abs(row.Time-at) > 1e-12 || len(row.Values) != 3 || math.Abs(row.Values[2]-source) > 1e-9 {
							continue
						}
						found = true
						label := fmt.Sprintf("%s step=%g t=%g input=%g", name, step, at, source)
						if e = out.frozenCheck(label+" voltage", row.Values[0], ref.Voltage, .001, .001); e != nil {
							return e
						}
						want, e := refNumber(ref.Voltage)
						if e != nil {
							return e
						}
						if e = out.check(label+" current", row.Values[1], (source-want)/10000, math.Max(.001, .001*math.Abs(want))/10000); e != nil {
							return e
						}
						break
					}
					if !found {
						near := []workbench.VerificationRow{}
						for _, row := range rows {
							if math.Abs(row.Time-at) < 1e-8 {
								near = append(near, row)
							}
						}
						return fmt.Errorf("missing %s reference sample step=%g t=%g source=%g; nearby rows %+v", name, step, at, source, near)
					}
				}
				if !finite {
					minimum, maximum := math.Inf(1), math.Inf(-1)
					for i, row := range rows {
						if row.Time >= .3-1e-12 {
							minimum = math.Min(minimum, row.Values[0])
							maximum = math.Max(maximum, row.Values[0])
						}
						if i > 0 && row.Time == rows[i-1].Time {
							if e := out.check("RC pulse capacitor continuity", row.Values[0], rows[i-1].Values[0], 1e-10); e != nil {
								return e
							}
						}
					}
					if e := out.frozenCheck("RC settled periodic minimum", minimum, frozen.PeriodicLow, .001, .001); e != nil {
						return e
					}
					if e := out.frozenCheck("RC settled periodic maximum", maximum, frozen.PeriodicHigh, .001, .001); e != nil {
						return e
					}
				}
			}
			return nil
		})
		if err != nil {
			return out, err
		}
	}
	err = electricalSession(ctx, factory, filepath.Join(dir, "npn-pulse"), func(s *session) error { return checkNPNPulse(s, frozen, &out) })
	if err != nil {
		return out, err
	}
	for _, step := range []float64{.001, .0005, .00025} {
		var measurement FlasherMeasurement
		err = electricalSession(ctx, factory, filepath.Join(dir, fmt.Sprintf("flasher-%g", step)), func(s *session) error { var e error; measurement, e = measureFlasher(s, step); return e })
		if err != nil {
			return out, err
		}
		out.Refinement = append(out.Refinement, measurement)
	}
	for i := 0; i < 2; i++ {
		a, b := out.Refinement[i], out.Refinement[i+1]
		if err = out.check(fmt.Sprintf("flasher period refinement %g/%g", a.Step, b.Step), a.Period, b.Period, math.Max(2*a.Step, .01*b.Period)); err != nil {
			return out, err
		}
		ids := make([]string, 0, len(b.Maximum))
		for id := range b.Maximum {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			want := b.Maximum[id]
			abs := .001
			if id[0] == 'i' {
				abs = 1e-6
			}
			if err = out.check("flasher max "+id, a.Maximum[id], want, math.Max(abs, .01*math.Abs(want))); err != nil {
				return out, err
			}
			if err = out.check("flasher min "+id, a.Minimum[id], b.Minimum[id], math.Max(abs, .01*math.Max(math.Abs(want), math.Abs(b.Minimum[id])))); err != nil {
				return out, err
			}
		}
	}
	if len(out.Checks) != electricalCheckCount {
		return out, fmt.Errorf("electrical coverage incomplete: %d/%d", len(out.Checks), electricalCheckCount)
	}
	return out, nil
}
func electricalVerify(s *session, duration, step float64, probes []string) (*workbench.VerificationMetadata, []workbench.VerificationRow, float64, error) {
	before, err := s.state("project.inspect", nil, false)
	if err != nil {
		return nil, nil, 0, err
	}
	started := time.Now()
	state, err := s.state("simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, true)
	if err != nil {
		return nil, nil, 0, err
	}
	var view workbench.OperationView
	for {
		if err = s.ask("operation.inspect", workbench.IDParams{ID: state.LastOperation}, false, &view); err != nil {
			return nil, nil, 0, err
		}
		if view.Operation.Status == "completed" {
			break
		}
		if view.Operation.Status == "failed" || view.Operation.Status == "canceled" {
			return nil, nil, 0, fmt.Errorf("electrical Verify: %+v", view.Operation.Error)
		}
		select {
		case <-s.ctx.Done():
			return nil, nil, 0, s.ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
	elapsed := time.Since(started).Seconds()
	if view.Metadata == nil || !view.Operation.SourceUnchanged || view.Metadata.Duration != duration {
		return nil, nil, 0, fmt.Errorf("verification lost source identity")
	}
	var rows []workbench.VerificationRow
	for offset := 0; ; {
		var page workbench.VerificationPage
		if err = s.ask("operation.samples", workbench.VerificationPageParams{ID: state.LastOperation, Offset: offset, Limit: 200}, false, &page); err != nil {
			return nil, nil, 0, err
		}
		if page.Offset != offset || len(page.Probes) != len(probes) || len(page.Rows) == 0 {
			return nil, nil, 0, fmt.Errorf("verification paging missing evidence")
		}
		for i, p := range page.Probes {
			if p.ID != probes[i] {
				return nil, nil, 0, fmt.Errorf("verification probe order differs")
			}
		}
		rows = append(rows, page.Rows...)
		if len(rows) == page.Total {
			break
		}
		if page.NextOffset <= offset {
			return nil, nil, 0, fmt.Errorf("verification paging did not advance")
		}
		offset = page.NextOffset
	}
	after, err := s.state("project.inspect", nil, false)
	if err != nil {
		return nil, nil, 0, err
	}
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) {
		return nil, nil, 0, fmt.Errorf("qualification Verify changed live physics")
	}
	return view.Metadata, rows, elapsed, nil
}
