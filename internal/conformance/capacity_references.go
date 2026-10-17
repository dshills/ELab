package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

// Collapse the 55 unloaded series resistors into four stages. The insulated
// MOS gate draws zero current, so the RC subsystem is exactly linear and
// independent of the drain/LED load. Evaluate its 4x4 matrix exponential by
// scaling/Taylor/squaring, without ELab's MNA or time-stepping implementation.
func capacityRCReference(at float64) [4]float64 {
	var a [4][4]float64
	r := [4]float64{14000, 14000, 14000, 13000}
	for i := range a {
		a[i][i] = -1 / (r[i] * 1e-7)
		if i > 0 {
			a[i][i-1] = 1 / (r[i] * 1e-7)
		}
		if i < 3 {
			a[i][i] -= 1 / (r[i+1] * 1e-7)
			a[i][i+1] = 1 / (r[i+1] * 1e-7)
		}
	}
	propagate := func(value [4]float64, elapsed, source float64) [4]float64 {
		if elapsed == 0 {
			return value
		}
		var matrix [4][4]float64
		norm := 0.
		for i := range a {
			row := 0.
			for j := range a[i] {
				matrix[i][j] = a[i][j] * elapsed
				row += math.Abs(matrix[i][j])
			}
			norm = math.Max(norm, row)
		}
		squares := max(0, int(math.Ceil(math.Log2(norm/.5))))
		for i := range matrix {
			for j := range matrix[i] {
				matrix[i][j] = math.Ldexp(matrix[i][j], -squares)
			}
		}
		var exponential, term [4][4]float64
		for i := range term {
			term[i][i], exponential[i][i] = 1, 1
		}
		multiply := func(left, right [4][4]float64) (product [4][4]float64) {
			for i := range product {
				for j := range product[i] {
					for k := range left[i] {
						product[i][j] += left[i][k] * right[k][j]
					}
				}
			}
			return product
		}
		for order := 1; order <= 32; order++ {
			term = multiply(term, matrix)
			for i := range term {
				for j := range term[i] {
					term[i][j] /= float64(order)
					exponential[i][j] += term[i][j]
				}
			}
		}
		for range squares {
			exponential = multiply(exponential, exponential)
		}
		var result [4]float64
		for i := range result {
			result[i] = source
			for j := range value {
				result[i] += exponential[i][j] * (value[j] - source)
			}
		}
		return result
	}
	charged := propagate([4]float64{}, math.Min(at, .1), 5)
	return propagate(charged, math.Max(0, at-.1), 0)
}

func checkCapacityRows(checks *ElectricalResult, name, label string, rows []workbench.VerificationRow, step float64) error {
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
			want = [4]float64{voltage[0], voltage[1], voltage[3], nmosDriverLEDReference(row.Values[2])}
		} else {
			source := 5.
			if k > count/2 && k < count || k == count/2 && side > 0 || k == count && side == 0 {
				source = 0
			}
			if name == "capacity-100-nodes" {
				current := source / 495000
				want = [4]float64{source, source - 5000*current, source - 245000*current, current}
			} else {
				want = [4]float64{source, source * 32 / 63, source / 63, source / 63000}
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
	if r == nil || r.Stale || r.Engine != workbench.NetworkEngineID || r.NumericalDamping != 0 || !finiteConformance(r.KCLResidual) || !finiteConformance(r.PowerResidual) || r.KCLResidual < 0 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
		return fmt.Errorf("capacity state lacks fresh finite conservative output")
	}
	graph, _, err := capacityFixture(name)
	if err != nil {
		return err
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
		current := source / 495000
		for i, p := range r.Potentiometers {
			number := i + 1
			want := source - current*(float64(i)*10000+5000)
			terminal, wiper := current, 0.
			if number == 50 {
				want, terminal, wiper = source-current*490000, 0, current
			}
			if p.ID != fmt.Sprintf("P%d", number) || !finiteConformance(p.WiperVoltage) || !finiteConformance(p.Terminal1Current) || !finiteConformance(p.WiperCurrent) || math.Abs(p.WiperVoltage-want) > 1e-7 || math.Abs(p.Terminal1Current-terminal) > 1e-10 || math.Abs(p.WiperCurrent-wiper) > 1e-10 {
				return fmt.Errorf("capacity potentiometer violates independent series/divider reference")
			}
		}
	} else if name == "capacity-64-parts" {
		if len(r.Resistors) != 63 || len(r.Nodes) != 64 || len(r.Capacitors)+len(r.Potentiometers)+len(r.MOSFETs) != 0 {
			return fmt.Errorf("64-part linear capacity identity differs")
		}
		for i, p := range r.Resistors {
			if p.ID != fmt.Sprintf("R%d", i+1) || !finiteConformance(p.Current) || !finiteConformance(p.Voltage) || math.Abs(p.Current-source/63000) > 1e-10 || math.Abs(p.Voltage-source/63) > 1e-7 {
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
		if !finiteConformance(r.LEDs[0].Current) || math.Abs(r.LEDs[0].Current-nmosDriverLEDReference(r.Capacitors[3].Voltage)) > 2e-9 {
			return fmt.Errorf("capacity LED violates independent gate/load reference")
		}
	}
	return nil
}
