package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

type OpAmpCase struct {
	Name         string                      `json:"name"`
	Graph        workbench.Graph             `json:"graph"`
	GraphSHA256  string                      `json:"graph_sha256"`
	Probes       []string                    `json:"probe_ids"`
	Rows         []workbench.VerificationRow `json:"rows"`
	Observations []workbench.MixedResult     `json:"observations"`
	Calls        []Call                      `json:"calls"`
}

type OpAmpSuiteResult struct {
	Model       workbench.OpAmpModel `json:"opamp_model"`
	Cases       []OpAmpCase          `json:"cases"`
	Checks      []ElectricalCheck    `json:"checks"`
	Limitations []string             `json:"limitations"`
}

var opAmpCaseNames = []string{"opamp-follower", "opamp-inverting", "opamp-rc-npn", "opamp-rc-mosfet"}

// Two independent peaks, two sine checks per ordinary output, and five RC /
// buffer checks across the two mixed captures (including four event rows).
const opAmpReferenceCount = 2 + 4*101 + 5*105

// RunOpAmps uses owned blank sessions and public commands only. The mixed
// fixtures add real wiring/storage/downstream loads to the guided follower.
// Their RC envelope is analytic; cross-adapter equality is a separate check.
func RunOpAmps(ctx context.Context, factory Factory, dir string) (out OpAmpSuiteResult, err error) {
	out.Model = workbench.OpAmpDeviceModel()
	out.Limitations = []string{"generic educational model; no physical amplifier bandwidth, stability or named-device qualification", "analytic sine/RC envelopes and frozen Decimal peak roots qualify only these fixtures", "mixed follower references use independent scalar bisection of the declared law and resistor loads; NPN base voltage is a captured boundary condition, not an independent NPN oracle", "adapter equality does not independently validate the numerical kernel", "owned native socket sessions exercise semantic commands, not physical keyboard/pointer/accessibility or visible latency", "no repeated performance or maximum-capacity claim"}
	checks := ElectricalResult{}
	defer func() { out.Checks = checks.Checks }()
	for _, name := range opAmpCaseNames {
		entry := OpAmpCase{Name: name}
		err = electricalSession(ctx, factory, filepath.Join(dir, name), func(s *session) error {
			if err := prepareOpAmpCase(s, name); err != nil {
				return err
			}
			initial, err := restart(s)
			if err != nil {
				return err
			}
			if initial.OpAmpModel == nil || *initial.OpAmpModel != out.Model {
				return fmt.Errorf("%s lost its exact model dependency", name)
			}
			entry.Graph = workbench.Graph{Components: initial.Components, Wires: initial.Wires}
			data, err := json.Marshal(entry.Graph)
			if err != nil {
				return err
			}
			entry.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
			entry.Probes = opAmpCaseProbes(name)
			if err = s.ask("scope.probes.set", workbench.ProbeParams{IDs: entry.Probes}, true, nil); err != nil {
				return err
			}
			observe := func(state workbench.Snapshot) error {
				r := state.Mixed
				count := 1
				if name == "opamp-rc-mosfet" {
					count = 2
				}
				if r == nil || r.Stale || len(r.OpAmps) != count || r.RunID != state.RunID || r.ElectricalRevision != state.ElectricalRevision || !finiteConformance(r.KCLResidual) || !finiteConformance(r.PowerResidual) || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
					return fmt.Errorf("%s lacks fresh conservative powered output: %v", name, state.Diagnostics)
				}
				for _, u := range r.OpAmps {
					if u.Model != out.Model || u.State != "powered" || math.Abs(u.NoninvertingCurrent+u.InvertingCurrent-u.OutputCurrent+u.PositiveSupplyCurrent+u.NegativeSupplyCurrent) > 1e-14 {
						return fmt.Errorf("%s lost signed pin current/model identity", name)
					}
				}
				copy := *r
				copy.RunID, copy.ElectricalRevision = 0, 0
				entry.Observations = append(entry.Observations, copy)
				return nil
			}
			if err = observe(initial); err != nil {
				return err
			}
			mixed := name == "opamp-rc-npn" || name == "opamp-rc-mosfet"
			duration := .025
			if mixed {
				duration = .05
			}
			live, err := s.state("simulation.step", workbench.StepParams{Duration: duration}, true)
			if err != nil {
				return err
			}
			if err = observe(live); err != nil {
				return err
			}
			if mixed {
				if len(live.Mixed.LEDs) != 1 || live.Mixed.LEDs[0].Current < .002 || live.Mixed.LEDs[0].Current > .005 {
					return fmt.Errorf("%s RC driver did not light its load", name)
				}
				live, err = s.state("simulation.step", workbench.StepParams{Duration: .05}, true)
				if err != nil {
					return err
				}
				if err = observe(live); err != nil {
					return err
				}
				if live.Mixed.LEDs[0].Current > 1e-6 {
					return fmt.Errorf("%s RC driver did not turn its load off", name)
				}
			} else {
				// Frozen 90-digit Decimal scalar roots from opamp-reference.py.
				want := .999989811376
				if name == "opamp-inverting" {
					want = -.999969336181
				}
				if err = checks.check(name+"-peak", live.Mixed.OpAmps[0].OutputVoltage, want, 1e-7); err != nil {
					return err
				}
			}
			metadata, rows, _, err := electricalVerify(s, .1, .001, entry.Probes)
			if err != nil {
				return err
			}
			count := 101
			if mixed {
				count = 105 // Grid samples plus pre/post observations at both edges.
			}
			if metadata.OpAmpModel == nil || *metadata.OpAmpModel != out.Model || len(rows) != count || metadata.Settings.StepSeconds != .001 || mixed && metadata.Mixed == nil {
				return fmt.Errorf("%s capture lost model, final state or sample coverage (rows %d, step %g, model %v, mixed %v)", name, len(rows), metadata.Settings.StepSeconds, metadata.OpAmpModel != nil, metadata.Mixed != nil)
			}
			if mixed && (metadata.NPNModel == nil && name == "opamp-rc-npn" || metadata.NMOSModel == nil && name == "opamp-rc-mosfet") {
				return fmt.Errorf("%s capture lost downstream device dependency", name)
			}
			entry.Rows = rows
			if err = checkOpAmpRows(&checks, entry); err != nil {
				return err
			}
			path := filepath.Join(s.directory, "captured.elab")
			if err = s.ask("project.save", workbench.SaveParams{Path: path}, true, nil); err != nil {
				return err
			}
			if err = s.ask("project.load", workbench.LoadParams{Path: path}, true, nil); err != nil {
				return err
			}
			reopened, err := s.state("simulation.wait", nil, false)
			if err != nil {
				return err
			}
			if !reopened.Paused || reopened.Document.RecordingCount != 1 || !reflect.DeepEqual(reopened.Scope.Selected, entry.Probes) || reopened.OpAmpModel == nil || *reopened.OpAmpModel != out.Model || !reflect.DeepEqual(reopened.Components, initial.Components) || !reflect.DeepEqual(reopened.Wires, initial.Wires) {
				return fmt.Errorf("%s reopen lost graph/model/probes/capture", name)
			}
			if err = observe(reopened); err != nil {
				return err
			}
			entry.Calls = append([]Call(nil), s.calls...)
			return nil
		})
		out.Cases = append(out.Cases, entry)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
