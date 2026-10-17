package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

type StorageRefusal struct {
	Mutation         string `json:"mutation"`
	Command          string `json:"command"`
	ErrorCode        string `json:"error_code"`
	ProjectUnchanged bool   `json:"project_unchanged"`
}
type OpAmpStorageCase struct {
	OpAmpCase
	RefinedRows []workbench.VerificationRow `json:"refined_rows,omitempty"`
	Refusals    []StorageRefusal            `json:"refusals"`
	ReopenCalls []Call                      `json:"reopen_calls"`
}
type OpAmpStorageResult struct {
	Model       workbench.OpAmpModel `json:"opamp_model"`
	Cases       []OpAmpStorageCase   `json:"cases"`
	Checks      []ElectricalCheck    `json:"checks"`
	Limitations []string             `json:"limitations"`
}

// RunOpAmpStorage qualifies public RLC/PMOS composition and file boundaries.
// All circuits, captures, reloads and refusals use the ordinary command API.
func RunOpAmpStorage(ctx context.Context, factory Factory, dir string) (out OpAmpStorageResult, err error) {
	out.Model = workbench.OpAmpDeviceModel()
	out.Limitations = []string{"two curated public fixtures, not every supported device/storage combination", "RLC analytic reference assumes an ideal .2 V buffer; tolerances include its small finite-gain error", "buffer scalar references use observed load current/input as boundary conditions, separately checked against analytic RLC/RC envelopes", "PMOS load-line reference neglects at most 1 nA of reverse-diode leakage within its 2 nA bound", "same-kernel half-step and cross-adapter comparisons supplement independent references; no physical-device, stability, visible-latency or full mounted-input claim"}
	checks := ElectricalResult{}
	defer func() { out.Checks = checks.Checks }()
	for _, name := range opAmpStorageCaseNames {
		entry := OpAmpStorageCase{OpAmpCase: OpAmpCase{Name: name, Probes: opAmpStorageProbes(name)}}
		err = electricalSession(ctx, factory, filepath.Join(dir, name), func(s *session) error {
			if err := prepareOpAmpStorage(s, name); err != nil {
				return err
			}
			initial, err := restart(s)
			if err != nil {
				return err
			}
			entry.Graph = workbench.Graph{Components: initial.Components, Wires: initial.Wires}
			raw, err := json.Marshal(entry.Graph)
			if err != nil {
				return err
			}
			entry.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
			observe := func(state workbench.Snapshot) error {
				r := state.Mixed
				if r == nil || r.RunID != state.RunID || r.ElectricalRevision != state.ElectricalRevision || state.OpAmpModel == nil || *state.OpAmpModel != out.Model {
					return fmt.Errorf("%s lacks fresh powered model identity", name)
				}
				if err := checkStorageObservation(name, r); err != nil {
					return err
				}
				if name == "opamp-rc-pmos" && (state.PMOSModel == nil || *state.PMOSModel != workbench.PMOSDeviceModel()) {
					return fmt.Errorf("buffered PMOS lacks its fixed model metadata")
				}
				copy := *r
				copy.RunID, copy.ElectricalRevision = 0, 0
				entry.Observations = append(entry.Observations, copy)
				return nil
			}
			if err = observe(initial); err != nil {
				return err
			}
			duration, step, advance := .1, .001, .05
			if name == "opamp-buffered-rlc" {
				duration, step, advance = .01, .0005, .005
				if math.Abs(initial.Mixed.Capacitors[0].Voltage-.05) > 1e-12 || math.Abs(initial.Mixed.Inductors[0].Current+50e-6) > 1e-12 {
					return fmt.Errorf("RLC signed startup state was not applied")
				}
			}
			for i := 0; i < 2; i++ {
				live, e := s.state("simulation.step", workbench.StepParams{Duration: advance}, true)
				if e != nil {
					return e
				}
				if e = observe(live); e != nil {
					return e
				}
				if name == "opamp-rc-pmos" && (i == 0 && live.Mixed.LEDs[0].Current > 1e-6 || i == 1 && live.Mixed.LEDs[0].Current < .002) {
					return fmt.Errorf("RC PMOS did not reverse LED control sense")
				}
			}
			meta, rows, _, err := electricalVerify(s, duration, step, entry.Probes)
			if err != nil {
				return err
			}
			if meta.OpAmpModel == nil || *meta.OpAmpModel != out.Model || meta.Mixed == nil {
				return fmt.Errorf("storage capture omitted its powered model")
			}
			entry.Rows = rows
			if err = checkOpAmpStorageRows(&checks, entry.OpAmpCase); err != nil {
				return err
			}
			if name == "opamp-buffered-rlc" {
				fine, rows, _, e := electricalVerify(s, duration, step/2, entry.Probes)
				if e != nil {
					return e
				}
				if len(rows) != 41 {
					return fmt.Errorf("RLC half-step omitted its 41 outputs")
				}
				entry.RefinedRows = rows
				if err = checkRLCRefinement(&checks, entry.Rows, rows); err != nil {
					return err
				}
				if err = s.ask("recording.keep.set", workbench.RecordingKeepParams{ID: fine.OperationID, Keep: false}, true, nil); err != nil {
					return err
				}
			}
			if err = s.ask("operation.select", workbench.IDParams{ID: meta.OperationID}, true, nil); err != nil {
				return err
			}
			recording := filepath.Join(s.directory, "recording.json")
			project := filepath.Join(s.directory, "project.elab")
			if err = s.ask("verification.export", workbench.VerificationExportParams{ID: meta.OperationID, Path: recording, Format: "json"}, true, nil); err != nil {
				return err
			}
			if err = s.ask("project.save", workbench.SaveParams{Path: project}, true, nil); err != nil {
				return err
			}
			recordingBytes, e := os.ReadFile(recording)
			if e != nil {
				return e
			}
			projectBytes, e := os.ReadFile(project)
			if e != nil {
				return e
			}
			entry.Refusals, err = refuseOpAmpStorageCorruptions(s, name, recordingBytes, projectBytes)
			if err != nil {
				return err
			}
			if err = electricalSession(ctx, factory, filepath.Join(s.directory, "reopened"), func(fresh *session) error {
				if err := fresh.ask("project.load", workbench.LoadParams{Path: project}, true, nil); err != nil {
					return err
				}
				reopened, err := fresh.state("simulation.wait", nil, false)
				if err != nil {
					return err
				}
				if !reopened.Paused || reopened.Transient == nil || reopened.Transient.Sample.Time != 0 || reopened.Document.RecordingCount != 1 || !reflect.DeepEqual(reopened.Components, initial.Components) || !reflect.DeepEqual(reopened.Wires, initial.Wires) || !reflect.DeepEqual(reopened.Scope.Selected, entry.Probes) {
					return fmt.Errorf("storage reopen did not restore graph/probes/capture with fresh startup")
				}
				if err = observe(reopened); err != nil {
					return err
				}
				if !reflect.DeepEqual(reopened.Mixed.OpAmps, initial.Mixed.OpAmps) || !reflect.DeepEqual(reopened.Mixed.Inductors, initial.Mixed.Inductors) || !reflect.DeepEqual(reopened.Mixed.Capacitors, initial.Mixed.Capacitors) {
					return fmt.Errorf("storage reopen lost exact startup state")
				}
				entry.ReopenCalls = append([]Call(nil), fresh.calls...)
				encoded, _ := json.Marshal(project)
				for i := range entry.ReopenCalls {
					entry.ReopenCalls[i].Params = bytes.ReplaceAll(entry.ReopenCalls[i].Params, encoded, []byte(`"<source-project>"`))
				}
				return nil
			}); err != nil {
				return err
			}
			untouched, e := os.ReadFile(project)
			if e != nil {
				return e
			}
			if !bytes.Equal(untouched, projectBytes) {
				return fmt.Errorf("reopen/refusal changed original project bytes")
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
