package conformance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
)

func capacityTimingHash(graph workbench.Graph) string {
	raw, _ := json.Marshal(graph)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// ValidateCapacityPerformance checks the retained evidence independently of
// aggregate success flags. It can also review a decoded external report.
func ValidateCapacityPerformance(result PerformanceResult) error {
	if result.Repetitions < 20 || result.Repetitions > 100 || !reflect.DeepEqual(result.Families, capacityCaseNames) || len(result.Cases) != 6 || len(result.Circuits) != 6 {
		return fmt.Errorf("capacity timing family/repetition/variant coverage differs")
	}
	targets := true
	for ci, entry := range result.Cases {
		name, condition := capacityCaseNames[ci/2], []string{"fresh-session", "warm-session"}[ci%2]
		if entry.Name != name || entry.Condition != condition || len(entry.Samples) != result.Repetitions {
			return fmt.Errorf("capacity timing case/sample coverage differs")
		}
		ack, fresh, verify := []float64{}, []float64{}, []float64{}
		var previousRun, previousRevision uint64
		for i, sample := range entry.Samples {
			if sample.Index != i || sample.Condition != condition || condition == "warm-session" && i > 0 && (sample.RunID <= previousRun || sample.ElectricalRevision <= previousRevision) {
				return fmt.Errorf("capacity timing sample/run order differs")
			}
			previousRun, previousRevision = sample.RunID, sample.ElectricalRevision
			expected, err := capacityPerformanceVariant(name, condition == "fresh-session" || i%2 == 1)
			if err != nil || sample.GraphSHA256 != capacityTimingHash(expected) || !reflect.DeepEqual(result.Circuits[sample.GraphSHA256], expected) {
				return fmt.Errorf("capacity timing resistor/track alternation differs")
			}
			if err := validateCapacityPerformanceSample(name, sample); err != nil {
				return err
			}
			ack = append(ack, sample.AcknowledgmentMS)
			fresh = append(fresh, sample.FreshResultMS)
			verify = append(verify, sample.VerifyMS)
		}
		a, e1 := summarizePerformance(ack)
		f, e2 := summarizePerformance(fresh)
		v, e3 := summarizePerformance(verify)
		if e1 != nil || e2 != nil || e3 != nil || a != entry.Acknowledgment || f != entry.FreshResult || v != entry.Verify || entry.FreshTargetPassed != (f.P95 <= 500) || entry.VerifyTargetPassed != (v.Maximum <= 2000) {
			return fmt.Errorf("capacity timing distributions/targets differ")
		}
		targets = targets && entry.FreshTargetPassed && entry.VerifyTargetPassed
	}
	if result.TargetsPassed != targets {
		return fmt.Errorf("capacity timing aggregate target differs")
	}
	return nil
}

func validateCapacityPerformanceSample(name string, sample PerformanceSample) error {
	capture := sample.CapacityVerification
	_, settings, err := capacityFixture(name)
	if err != nil || capture == nil || sample.RunID == 0 || sample.ElectricalRevision == 0 || sample.Engine != workbench.NetworkEngineID || !reflect.DeepEqual(sample.ProbeIDs, capacityProbes(name)) || sample.Settings != settings || sample.VerifyDuration != .2 || sample.VerifySampleStep != .001 {
		return fmt.Errorf("capacity timing lacks exact live/captured identities")
	}
	for _, value := range []float64{sample.AcknowledgmentMS, sample.FreshResultMS, sample.VerifyMS} {
		if !finiteConformance(value) || value < 0 {
			return fmt.Errorf("capacity timing has nonfinite/negative duration")
		}
	}
	if sample.FreshResultMS < sample.AcknowledgmentMS {
		return fmt.Errorf("capacity fresh result precedes admission")
	}
	meta := capture.Metadata
	low, _ := capacityPerformanceVariant(name, false)
	high, _ := capacityPerformanceVariant(name, true)
	if (!reflect.DeepEqual(meta.Graph, low) && !reflect.DeepEqual(meta.Graph, high)) || capacityTimingHash(meta.Graph) != sample.GraphSHA256 || meta.Format != "elab.verification" || meta.Version != 1 || !workbench.ValidID(meta.OperationID) || meta.Engine != sample.Engine || meta.Analysis != workbench.NetworkTransientAnalysis || meta.InputMode != "auto" || meta.ElectricalRevision != sample.ElectricalRevision || meta.Settings != settings || meta.Duration != .2 || meta.SampleCount != 205 || len(meta.Probes) != 4 || meta.Mixed == nil || meta.Mixed.RunID == 0 || meta.Mixed.ElectricalRevision != sample.ElectricalRevision || !finiteConformance(meta.Mixed.Time) || math.Abs(meta.Mixed.Time-.2) > 1e-12 || meta.LED != nil || meta.NPN != nil {
		return fmt.Errorf("capacity timing captured input/result association differs")
	}
	if _, err := time.Parse(time.RFC3339Nano, meta.CreatedAt); err != nil {
		return fmt.Errorf("capacity timing timestamp is invalid")
	}
	models := map[string]string{}
	for _, part := range meta.Graph.Components {
		models[part.ID] = part.Model
	}
	if !reflect.DeepEqual(models, sample.Models) || meta.LEDModel != workbench.Model() || meta.NPNModel != nil || meta.PMOSModel != nil || meta.OpAmpModel != nil || meta.DiodeModel != nil || meta.SwitchModel != nil || meta.WaveformModel != nil {
		return fmt.Errorf("capacity timing model dependencies differ")
	}
	if name == "capacity-rc-mosfet" {
		if meta.NMOSModel == nil || *meta.NMOSModel != workbench.NMOSDeviceModel() {
			return fmt.Errorf("capacity timing MOSFET model missing/changed")
		}
	} else if meta.NMOSModel != nil {
		return fmt.Errorf("capacity timing has an unrelated MOSFET model")
	}
	if name == "capacity-100-nodes" {
		if meta.PotentiometerModel == nil || *meta.PotentiometerModel != workbench.LinearPotentiometerModel() {
			return fmt.Errorf("capacity timing track model missing/changed")
		}
	} else if meta.PotentiometerModel != nil {
		return fmt.Errorf("capacity timing has an unrelated track model")
	}
	for i, probe := range meta.Probes {
		unit := "V"
		if i == 3 {
			unit = "A"
		}
		if probe.ID != sample.ProbeIDs[i] || probe.Unit != unit {
			return fmt.Errorf("capacity timing captured probe differs")
		}
	}
	if err := checkCapacityGraphObservation(name, meta.Mixed, meta.Graph); err != nil {
		return err
	}
	checks := ElectricalResult{}
	if err := checkCapacityGraphRows(&checks, name, "timed", capture.Rows, .001, meta.Graph); err != nil {
		return err
	}
	return checkCapacityCaptureStatistics(*capture)
}
