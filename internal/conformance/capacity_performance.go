package conformance

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
)

// RunCapacityPerformance measures real edits at the published size limits.
// All samples retain independent-reference-checked captures, outside timing.
func RunCapacityPerformance(ctx context.Context, factory Factory, dir string, repetitions int) (PerformanceResult, error) {
	result, err := runPerformance(ctx, factory, dir, repetitions, capacityCaseNames)
	if err != nil {
		return result, err
	}
	return result, ValidateCapacityPerformance(result)
}

func isCapacityPerformance(name string) bool {
	for _, candidate := range capacityCaseNames {
		if candidate == name {
			return true
		}
	}
	return false
}

func capacityPerformanceVariant(name string, high bool) (workbench.Graph, error) {
	graph, _, err := capacityFixture(name)
	if err != nil {
		return graph, err
	}
	id, value := "R1", 1000.
	switch name {
	case "capacity-100-nodes":
		id, value = "P1", 10000
		if high {
			value = 20000
		}
	case "capacity-64-parts":
		if high {
			value = 2000
		}
	case "capacity-rc-mosfet":
		id = "RL"
		if high {
			value = 2000
		}
	}
	for i, part := range graph.Components {
		if part.ID == id {
			graph.Components[i].Value = value
			return graph, nil
		}
	}
	return graph, fmt.Errorf("capacity timing edit target missing")
}

func capacityPerformanceFresh(s workbench.Snapshot, name string) bool {
	parts, nodes := capacitySize(name)
	if s.RunID == 0 || s.ElectricalRevision == 0 || s.Transient == nil || s.Transient.Stale || s.Transient.Engine != workbench.NetworkEngineID || s.Transient.RunID != s.RunID || s.Transient.ElectricalRevision != s.ElectricalRevision || s.Mixed == nil || s.Mixed.Stale || s.Mixed.Engine != workbench.NetworkEngineID || s.Mixed.RunID != s.RunID || s.Mixed.ElectricalRevision != s.ElectricalRevision || s.SimulatedComponents != parts || len(s.Nets) != nodes || len(s.Mixed.Nodes) != nodes || !reflect.DeepEqual(s.Scope.Selected, capacityProbes(name)) {
		return false
	}
	if name == "capacity-rc-mosfet" {
		return len(s.Mixed.Capacitors) == 4 && len(s.Mixed.MOSFETs) == 1 && s.Mixed.MOSFETs[0].ID == "M1" && s.Mixed.MOSFETs[0].Model == workbench.NMOSDeviceModel() && s.NMOSModel != nil && *s.NMOSModel == workbench.NMOSDeviceModel()
	}
	if name == "capacity-100-nodes" {
		return len(s.Mixed.Potentiometers) == 50 && s.PotentiometerModel != nil && *s.PotentiometerModel == workbench.LinearPotentiometerModel()
	}
	return len(s.Mixed.Resistors) == 63
}
