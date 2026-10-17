package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

type CapacityReopen struct {
	Kind                   string `json:"kind"`
	Status                 string `json:"status"`
	GraphAndProbesRestored bool   `json:"graph_and_probes_restored"`
	RecordingByteExact     bool   `json:"recording_byte_exact"`
	OriginalFileUnchanged  bool   `json:"original_file_unchanged"`
	Calls                  []Call `json:"calls"`
}

type CapacityBoundary struct {
	Kind                       string          `json:"kind"`
	Graph                      workbench.Graph `json:"graph"`
	SimulatedParts             int             `json:"simulated_parts"`
	Nodes                      int             `json:"nodes"`
	Status                     string          `json:"status"`
	CompileStatus              string          `json:"compile_status"`
	DiagnosticCode             string          `json:"diagnostic_code"`
	StepErrorCode              string          `json:"step_error_code"`
	RejectedStepPreservedState bool            `json:"rejected_step_preserved_state"`
	GeometryPreservedPhysics   bool            `json:"geometry_preserved_physics"`
}

type CapacityCase struct {
	Name         string                      `json:"name"`
	Graph        workbench.Graph             `json:"graph"`
	GraphSHA256  string                      `json:"graph_sha256"`
	Settings     workbench.TransientSettings `json:"settings"`
	Engine       string                      `json:"engine"`
	Parts        int                         `json:"simulated_parts"`
	Nodes        int                         `json:"nodes"`
	Probes       []string                    `json:"probe_ids"`
	Rows         []workbench.VerificationRow `json:"rows"`
	RefinedRows  []workbench.VerificationRow `json:"refined_rows,omitempty"`
	Observations []workbench.MixedResult     `json:"observations"`
	Boundary     CapacityBoundary            `json:"boundary"`
	Reopens      []CapacityReopen            `json:"reopens"`
	Calls        []Call                      `json:"calls"`
}

type CapacityResult struct {
	Limits      workbench.CircuitLimits `json:"limits"`
	Cases       []CapacityCase          `json:"cases"`
	Checks      []ElectricalCheck       `json:"checks"`
	Limitations []string                `json:"limitations"`
}

// RunCapacity constructs full-size circuits with public commands, checks
// independent references and fresh file reopen, then exercises editable,
// savable over-limit drawings. Existing numerical/work limits stay unchanged.
func RunCapacity(ctx context.Context, factory Factory, dir string) (out CapacityResult, err error) {
	out.Limits = workbench.Limits()
	out.Limitations = []string{"three representative boundary fixtures, not every supported device combination", "four-stage RC reference is an independent matrix exponential; MOS load line uses observed gate voltage, separately checked against that reference", "backend work budgets are exercised in ordinary builds; no visible-latency, hardware, arbitrary-layout or distribution claim"}
	checks := ElectricalResult{}
	defer func() { out.Checks = checks.Checks }()
	for _, name := range capacityCaseNames {
		entry := CapacityCase{Name: name, Probes: capacityProbes(name)}
		err = electricalSession(ctx, factory, filepath.Join(dir, name), func(s *session) error {
			initial, err := prepareCapacity(s, name)
			if err != nil {
				return err
			}
			entry.Graph = workbench.Graph{Components: initial.Components, Wires: initial.Wires}
			expected, settings, err := capacityFixture(name)
			if err != nil || !reflect.DeepEqual(entry.Graph, expected) || initial.TransientSettings != settings {
				return fmt.Errorf("capacity public construction differs from its declared fixture: %v", err)
			}
			entry.Settings, entry.Parts, entry.Nodes = settings, initial.SimulatedComponents, len(initial.Nets)
			raw, _ := json.Marshal(entry.Graph)
			entry.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
			observe := func(state workbench.Snapshot) error {
				parts, nodes := capacitySize(name)
				if state.SimulatedComponents != parts || len(state.Nets) != nodes || !reflect.DeepEqual(state.Scope.Selected, entry.Probes) || state.Mixed == nil || state.Transient == nil || state.Transient.Stale || state.Transient.RunID != state.RunID || state.Transient.ElectricalRevision != state.ElectricalRevision || state.Mixed.RunID != state.RunID || state.Mixed.ElectricalRevision != state.ElectricalRevision {
					return fmt.Errorf("capacity result lost its size/probes/fresh identities")
				}
				if err := checkCapacityObservation(name, state.Mixed); err != nil {
					return err
				}
				copy := *state.Mixed
				copy.RunID, copy.ElectricalRevision = 0, 0
				entry.Observations = append(entry.Observations, copy)
				return nil
			}
			if err := observe(initial); err != nil {
				return err
			}
			for range 4 {
				state, err := s.state("simulation.step", workbench.StepParams{Duration: .05}, true)
				if err != nil {
					return err
				}
				if err := observe(state); err != nil {
					return err
				}
			}
			meta, rows, _, err := electricalVerify(s, .2, .001, entry.Probes)
			if err != nil {
				return err
			}
			entry.Engine, entry.Rows = meta.Engine, rows
			if entry.Engine != workbench.NetworkEngineID || !reflect.DeepEqual(meta.Graph, entry.Graph) {
				return fmt.Errorf("capacity recording lost graph or engine identity")
			}
			if err := checkCapacityRows(&checks, name, "coarse", rows, .001); err != nil {
				return err
			}
			if name == "capacity-rc-mosfet" {
				fine, rows, _, err := electricalVerify(s, .2, .0005, entry.Probes)
				if err != nil {
					return err
				}
				entry.RefinedRows = rows
				if err := checkCapacityRows(&checks, name, "fine", rows, .0005); err != nil {
					return err
				}
				if err := checkCapacityRefinement(&checks, entry.Rows, rows); err != nil {
					return err
				}
				if err := s.ask("recording.keep.set", workbench.RecordingKeepParams{ID: fine.OperationID, Keep: false}, true, nil); err != nil {
					return err
				}
			}
			if err := s.ask("operation.select", workbench.IDParams{ID: meta.OperationID}, true, nil); err != nil {
				return err
			}
			recording, project := filepath.Join(s.directory, "recording.json"), filepath.Join(s.directory, "project.elab")
			if err := s.ask("verification.export", workbench.VerificationExportParams{ID: meta.OperationID, Path: recording, Format: "json"}, true, nil); err != nil {
				return err
			}
			if err := s.ask("project.save", workbench.SaveParams{Path: project}, true, nil); err != nil {
				return err
			}
			recordingBytes, err := os.ReadFile(recording)
			if err != nil {
				return err
			}
			reopen, err := reopenCapacity(ctx, factory, filepath.Join(s.directory, "reopen-valid"), project, recordingBytes, entry.Graph, entry.Probes, false, observe)
			if err != nil {
				return err
			}
			entry.Reopens = append(entry.Reopens, reopen)
			entry.Boundary, err = exerciseCapacityBoundary(s, name)
			if err != nil {
				return err
			}
			excess := filepath.Join(s.directory, "excess.elab")
			if err := s.ask("project.save", workbench.SaveParams{Path: excess}, true, nil); err != nil {
				return err
			}
			reopen, err = reopenCapacity(ctx, factory, filepath.Join(s.directory, "reopen-excess"), excess, recordingBytes, entry.Boundary.Graph, entry.Probes, true, nil)
			if err != nil {
				return err
			}
			entry.Reopens = append(entry.Reopens, reopen)
			entry.Calls = append([]Call(nil), s.calls...)
			return nil
		})
		out.Cases = append(out.Cases, entry)
		if err != nil {
			return out, fmt.Errorf("%s: %w", name, err)
		}
	}
	return out, nil
}

func normalizedCapacityCalls(calls []Call, path string) []Call {
	out := append([]Call(nil), calls...)
	encoded, _ := json.Marshal(path)
	for i := range out {
		out[i].Params = bytes.ReplaceAll(out[i].Params, encoded, []byte(`"<source-project>"`))
	}
	return out
}
