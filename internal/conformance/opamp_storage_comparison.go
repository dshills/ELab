package conformance

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/dshills/elab/workbench"
)

const opAmpStorageReferenceCount = 4*(21+41) + 21*4 + 105*4

func checkStorageObservation(name string, r *workbench.MixedResult) error {
	if r == nil || r.Stale || len(r.OpAmps) != 1 || r.OpAmps[0].Model != workbench.OpAmpDeviceModel() || r.OpAmps[0].State != "powered" || !finiteConformance(r.KCLResidual) || !finiteConformance(r.PowerResidual) || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
		return fmt.Errorf("%s lacks conservative powered state", name)
	}
	if name == "opamp-buffered-rlc" {
		if r.Engine != workbench.InductorEngineID || len(r.Inductors) != 1 || len(r.Capacitors) != 1 || len(r.MOSFETs) != 0 {
			return fmt.Errorf("buffered RLC lost its coupled storage engine")
		}
	} else if len(r.MOSFETs) != 1 || r.MOSFETs[0].Model != workbench.PMOSDeviceModel() || r.MOSFETs[0].DrainCurrent > 0 || len(r.Capacitors) != 1 || len(r.LEDs) != 1 || len(r.Inductors) != 0 {
		return fmt.Errorf("buffered PMOS lost signed high-side state")
	}
	return nil
}

// CompareOpAmpStorage recomputes references, refinement and complete file
// refusal coverage before comparing the adapters' normalized public evidence.
func CompareOpAmpStorage(reference, actual OpAmpStorageResult) error {
	for _, report := range []OpAmpStorageResult{reference, actual} {
		if report.Model != workbench.OpAmpDeviceModel() || len(report.Cases) != 2 || len(report.Checks) != opAmpStorageReferenceCount {
			return fmt.Errorf("op-amp storage model/case/reference coverage differs")
		}
		checks := ElectricalResult{}
		for i, entry := range report.Cases {
			name := opAmpStorageCaseNames[i]
			rows := 105
			if i == 0 {
				rows = 21
			}
			raw, err := json.Marshal(entry.Graph)
			if err != nil || entry.Name != name || entry.GraphSHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) || len(entry.Rows) != rows || len(entry.Observations) != 4 || !reflect.DeepEqual(entry.Probes, opAmpStorageProbes(name)) || len(entry.Calls) == 0 || len(entry.ReopenCalls) == 0 {
				return fmt.Errorf("op-amp storage fixture coverage/identity differs")
			}
			if err = checkOpAmpStorageRows(&checks, entry.OpAmpCase); err != nil {
				return err
			}
			if i == 0 {
				if err = checkRLCRefinement(&checks, entry.Rows, entry.RefinedRows); err != nil {
					return err
				}
			} else if len(entry.RefinedRows) != 0 {
				return fmt.Errorf("PMOS case has unrelated refinement")
			}
			times := []float64{0, .05, .1, 0}
			if i == 0 {
				times = []float64{0, .005, .01, 0}
			}
			for k, observation := range entry.Observations {
				if observation.Time != times[k] || observation.RunID != 0 || observation.ElectricalRevision != 0 {
					return fmt.Errorf("storage observation time/provenance differs")
				}
				if err = checkStorageObservation(name, &observation); err != nil {
					return err
				}
			}
			mutations := storageCorruptionNames(name)
			if len(entry.Refusals) != 2*len(mutations) {
				return fmt.Errorf("storage refusal coverage incomplete")
			}
			for k, mutation := range mutations {
				for j, command := range []string{"verification.import", "project.load"} {
					code := []string{"invalid-recording", "invalid-document"}[j]
					if entry.Refusals[2*k+j] != (StorageRefusal{mutation, command, code, true}) {
						return fmt.Errorf("storage refusal identity/outcome differs")
					}
				}
			}
		}
		if !reflect.DeepEqual(checks.Checks, report.Checks) {
			return fmt.Errorf("storage checks differ from recorded evidence")
		}
	}
	normalize := func(result OpAmpStorageResult) OpAmpStorageResult {
		result.Cases = append([]OpAmpStorageCase(nil), result.Cases...)
		result.Limitations = nil
		for i := range result.Cases {
			result.Cases[i].Calls = collapseCalls(result.Cases[i].Calls)
			result.Cases[i].ReopenCalls = collapseCalls(result.Cases[i].ReopenCalls)
		}
		return result
	}
	left, err := json.Marshal(normalize(reference))
	if err != nil {
		return err
	}
	right, err := json.Marshal(normalize(actual))
	if err != nil {
		return err
	}
	if !bytes.Equal(left, right) {
		return fmt.Errorf("op-amp storage public-command/numerical evidence differs")
	}
	return nil
}
