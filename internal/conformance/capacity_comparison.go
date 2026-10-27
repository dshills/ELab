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

const capacityReferenceCount = 4 * (3*205 + 405 + 205)

func CompareCapacity(reference, actual CapacityResult) error {
	for _, result := range []CapacityResult{reference, actual} {
		if !capacityLimitsCompatible(result.Limits) || len(result.Cases) != len(capacityCaseNames) || len(result.Checks) != capacityReferenceCount {
			return fmt.Errorf("capacity limits/case/reference coverage differs")
		}
		checks := ElectricalResult{}
		for i, entry := range result.Cases {
			name := capacityCaseNames[i]
			graph, settings, err := capacityFixture(name)
			parts, nodes := capacitySize(name)
			raw, _ := json.Marshal(entry.Graph)
			if err != nil || entry.Name != name || !reflect.DeepEqual(entry.Graph, graph) || entry.GraphSHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) || entry.Settings != settings || entry.Engine != workbench.NetworkEngineID || entry.Parts != parts || entry.Nodes != nodes || !reflect.DeepEqual(entry.Probes, capacityProbes(name)) || len(entry.Calls) == 0 || len(entry.Observations) != 6 || len(entry.Reopens) != 2 {
				return fmt.Errorf("capacity fixture identity/coverage differs")
			}
			if err := checkCapacityRows(&checks, name, "coarse", entry.Rows, .001); err != nil {
				return err
			}
			if name == "capacity-rc-mosfet" {
				if err := checkCapacityRows(&checks, name, "fine", entry.RefinedRows, .0005); err != nil {
					return err
				}
				if err := checkCapacityRefinement(&checks, entry.Rows, entry.RefinedRows); err != nil {
					return err
				}
			} else if len(entry.RefinedRows) != 0 {
				return fmt.Errorf("linear capacity case has unrelated refinement")
			}
			for k, observation := range entry.Observations {
				at := float64(k) * .05
				if k == 5 {
					at = 0
				}
				if math.Abs(observation.Time-at) > 1e-12 || observation.RunID != 0 || observation.ElectricalRevision != 0 {
					return fmt.Errorf("capacity observation time/provenance differs")
				}
				if err := checkCapacityObservation(name, &observation); err != nil {
					return err
				}
			}
			boundary := entry.Boundary
			expected := graph.Clone()
			kind := "65th-simulated-part"
			if name == "capacity-100-nodes" {
				kind = "101st-electrical-node"
				if expected.Wires[50].ID != "W51" || expected.Wires[50].From != "P50.wiper" {
					return fmt.Errorf("capacity boundary fixture changed its explicit tie")
				}
				expected.Wires = append(expected.Wires[:50], expected.Wires[51:]...)
				for j := range expected.Components {
					if expected.Components[j].ID == "P1" {
						expected.Components[j].Position = workbench.Point{X: 2200, Y: 1400}
					}
				}
			} else {
				p, err := workbench.MakeComponent("Overflow", "resistor", workbench.Point{X: 2200, Y: 1400}, "1k")
				if err != nil {
					return err
				}
				expected.Components = append(expected.Components, p)
			}
			compiled, status, diagnostic := expected.Compile()
			if !reflect.DeepEqual(boundary.Graph, expected) || boundary.Kind != kind || boundary.SimulatedParts != workbench.SimulatedComponentCount(expected) || boundary.Nodes != len(compiled.Nets) || status != "unsupported" || diagnostic == nil || diagnostic.Code != "mixed-limit" || boundary.Status != "paused" || boundary.CompileStatus != status || boundary.DiagnosticCode != diagnostic.Code || boundary.StepErrorCode != "unavailable" || !boundary.RejectedStepPreservedState || !boundary.GeometryPreservedPhysics {
				return fmt.Errorf("capacity over-limit identity/atomicity differs")
			}
			for k, reopened := range entry.Reopens {
				kind, status := "valid", "paused"
				if k == 1 {
					kind = "excess"
				}
				if reopened.Kind != kind || reopened.Status != status || !reopened.GraphAndProbesRestored || !reopened.RecordingByteExact || !reopened.OriginalFileUnchanged || len(reopened.Calls) == 0 {
					return fmt.Errorf("capacity file reopen coverage/outcome differs")
				}
			}
		}
		if !reflect.DeepEqual(checks.Checks, result.Checks) {
			return fmt.Errorf("capacity checks differ from recomputed references")
		}
	}
	normalize := func(result CapacityResult) CapacityResult {
		result.Limitations = nil
		result.Cases = append([]CapacityCase(nil), result.Cases...)
		for i := range result.Cases {
			entry := &result.Cases[i]
			entry.Calls = collapseCalls(entry.Calls)
			entry.Reopens = append([]CapacityReopen(nil), entry.Reopens...)
			for j := range entry.Reopens {
				entry.Reopens[j].Calls = collapseCalls(entry.Reopens[j].Calls)
			}
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
		return fmt.Errorf("capacity public command/numerical evidence differs")
	}
	return nil
}

// The frozen capacity fixtures contain passive/RC/MOSFET circuits, not BJTs.
// Accept their exact historical limit envelope as well as the current one;
// added bipolar discovery fields do not retrospectively qualify PNP capacity.
// All fixture graphs, numerical bounds, samples and other limits stay exact.
func capacityLimitsCompatible(actual workbench.CircuitLimits) bool {
	// These frozen passive/RC/MOS reports predate optional JFET discovery fields.
	// Accept only the exact paired absence, never arbitrary mixed capacities.
	for _, jfetDiscovery := range []bool{true, false} {
		current := workbench.Limits()
		if !jfetDiscovery {
			current.JFETs, current.CombinedFETs = 0, 0
		}
		if actual == current {
			return true
		}
		current.Transistors, current.PNP, current.CombinedBipolar = 2, 2, 2
		if actual == current {
			return true
		}
		current.Timers555 = 0
		if actual == current {
			return true
		}
		current.PNP, current.CombinedBipolar = 0, 0
		if actual == current {
			return true
		}
	}
	return false
}
