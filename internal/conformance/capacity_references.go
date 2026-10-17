package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

func checkCapacityRows(checks *ElectricalResult, name, label string, rows []workbench.VerificationRow, step float64) error {
	graph, _, err := capacityFixture(name)
	if err != nil {
		return err
	}
	return checkCapacityGraphRows(checks, name, label, rows, step, graph)
}

func checkCapacityGraphRows(checks *ElectricalResult, name, label string, rows []workbench.VerificationRow, step float64, graph workbench.Graph) error {
	count := int(math.Round(.2 / step))
	grid := make([]int, count+1)
	for i, row := range rows {
		k := int(math.Round(row.Time / step))
		if len(row.Values) != 4 || k < 0 || k > count || math.Abs(row.Time-float64(k)*step) > 1e-12 || i > 0 && row.Time < rows[i-1].Time {
			return fmt.Errorf("capacity capture omitted planned time/probe coverage")
		}
		side := grid[k]
		grid[k]++
		var want [4]float64
		if name == "capacity-rc-mosfet" {
			voltage := capacityRCReference(row.Time)
			load, _ := graph.Component("RL")
			want = [4]float64{voltage[0], voltage[1], voltage[3], mosfetLEDLoadReference(row.Values[2]-1.5, load.Value)}
		} else {
			source := 5.
			if k > count/2 && k < count || k == count/2 && side > 0 || k == count && side == 0 {
				source = 0
			}
			if name == "capacity-100-nodes" {
				first, _ := graph.Component("P1")
				current := source / (485000 + first.Value)
				want = [4]float64{source, source - first.Value/2*current, source - (first.Value+235000)*current, current}
			} else {
				first, _ := graph.Component("R1")
				total := 62000 + first.Value
				want = [4]float64{source, source * 32000 / total, source * 1000 / total, source / total}
			}
		}
		for channel, actual := range row.Values {
			tolerance := 1e-8
			if channel == 3 {
				tolerance = 2e-9
			} else if name == "capacity-rc-mosfet" {
				tolerance = math.Max(.001, .001*math.Abs(want[channel]))
			}
			if err := checks.check(fmt.Sprintf("%s-%s-row-%03d-channel-%d", name, label, i, channel), actual, want[channel], tolerance); err != nil {
				return err
			}
		}
	}
	for k, actual := range grid {
		want := 1
		if k == count/2 || k == count {
			want = 3
		}
		if actual != want {
			return fmt.Errorf("capacity sample/event coverage differs at %g s: %d/%d", float64(k)*step, actual, want)
		}
	}
	return nil
}

func checkCapacityRefinement(checks *ElectricalResult, coarse, fine []workbench.VerificationRow) error {
	for i, row := range coarse {
		found := false
		for _, reference := range fine {
			if math.Abs(row.Time-reference.Time) > 1e-12 {
				continue
			}
			found = true
			for channel, actual := range row.Values {
				tolerance := .001
				if channel == 3 {
					gate := reference.Values[2]
					tolerance = 4e-9 + nmosDriverLEDReference(gate+.001) - nmosDriverLEDReference(gate-.001)
				}
				if err := checks.check(fmt.Sprintf("capacity-refinement-row-%03d-channel-%d", i, channel), actual, reference.Values[channel], tolerance); err != nil {
					return err
				}
			}
			break
		}
		if !found {
			return fmt.Errorf("capacity half-step capture omitted a coarse observation")
		}
	}
	return nil
}

func checkCapacityObservation(name string, r *workbench.MixedResult) error {
	graph, _, err := capacityFixture(name)
	if err != nil {
		return err
	}
	return checkCapacityGraphObservation(name, r, graph)
}

func checkCapacityGraphObservation(name string, r *workbench.MixedResult, graph workbench.Graph) error {
	if r == nil || r.Stale || r.Engine != workbench.NetworkEngineID || r.NumericalDamping != 0 || !finiteConformance(r.KCLResidual) || !finiteConformance(r.PowerResidual) || r.KCLResidual < 0 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
		return fmt.Errorf("capacity state lacks fresh finite conservative output")
	}
	compiled, _, diagnostic := graph.Compile()
	if diagnostic != nil || len(r.Nodes) != len(compiled.Nets) || len(r.Transistors)+len(r.OpAmps)+len(r.Inductors)+len(r.Diodes)+len(r.Switches) != 0 {
		return fmt.Errorf("capacity observation contains missing or unrelated device/node results")
	}
	for _, node := range compiled.Nets {
		value, ok := r.Nodes[node.ID]
		if !ok || !finiteConformance(value) || math.Abs(value) > 30 || node.ID == compiled.Ground && value != 0 {
			return fmt.Errorf("capacity observation has incomplete, nonfinite or unreferenced nodes")
		}
	}
	source := 5.
	if r.Time >= .1-1e-12 && r.Time < .2-1e-12 {
		source = 0
	}
	if name == "capacity-100-nodes" {
		if len(r.Potentiometers) != 50 || len(r.Nodes) != 100 || len(r.Capacitors)+len(r.Resistors)+len(r.MOSFETs) != 0 {
			return fmt.Errorf("100-node capacity identity differs")
		}
		first, _ := graph.Component("P1")
		current := source / (485000 + first.Value)
		prefix := 0.
		for i, p := range r.Potentiometers {
			number := i + 1
			value := 10000.
			if i == 0 {
				value = first.Value
			}
			want := source - current*(prefix+value/2)
			terminal, wiper := current, 0.
			if number == 50 {
				want, terminal, wiper = source-current*prefix, 0, current
			}
			if p.ID != fmt.Sprintf("P%d", number) || !finiteConformance(p.WiperVoltage) || !finiteConformance(p.Terminal1Current) || !finiteConformance(p.WiperCurrent) || math.Abs(p.WiperVoltage-want) > 1e-7 || math.Abs(p.Terminal1Current-terminal) > 1e-10 || math.Abs(p.WiperCurrent-wiper) > 1e-10 {
				return fmt.Errorf("capacity potentiometer violates independent series/divider reference")
			}
			prefix += value
		}
	} else if name == "capacity-64-parts" {
		if len(r.Resistors) != 63 || len(r.Nodes) != 64 || len(r.Capacitors)+len(r.Potentiometers)+len(r.MOSFETs) != 0 {
			return fmt.Errorf("64-part linear capacity identity differs")
		}
		for i, p := range r.Resistors {
			first, _ := graph.Component("R1")
			current, resistance := source/(62000+first.Value), 1000.
			if i == 0 {
				resistance = first.Value
			}
			if p.ID != fmt.Sprintf("R%d", i+1) || !finiteConformance(p.Current) || !finiteConformance(p.Voltage) || math.Abs(p.Current-current) > 1e-10 || math.Abs(p.Voltage-current*resistance) > 1e-7 {
				return fmt.Errorf("capacity resistor violates independent series reference")
			}
		}
	} else {
		if len(r.Resistors) != 56 || len(r.Capacitors) != 4 || len(r.MOSFETs) != 1 || len(r.LEDs) != 1 || len(r.Nodes) != 60 || r.MOSFETs[0].ID != "M1" || r.LEDs[0].ID != "D1" || r.MOSFETs[0].Model != workbench.NMOSDeviceModel() || r.MOSFETs[0].GateCurrent != 0 {
			return fmt.Errorf("64-part RC/MOSFET capacity identity differs")
		}
		want := capacityRCReference(r.Time)
		for i, p := range r.Capacitors {
			if p.ID != fmt.Sprintf("C%d", i+1) || !finiteConformance(p.Voltage) || math.Abs(p.Voltage-want[i]) > math.Max(.001, .001*math.Abs(want[i])) {
				return fmt.Errorf("capacity storage violates independent matrix-exponential reference")
			}
		}
		load, _ := graph.Component("RL")
		if !finiteConformance(r.LEDs[0].Current) || math.Abs(r.LEDs[0].Current-mosfetLEDLoadReference(r.Capacitors[3].Voltage-1.5, load.Value)) > 2e-9 {
			return fmt.Errorf("capacity LED violates independent gate/load reference")
		}
	}
	return nil
}
