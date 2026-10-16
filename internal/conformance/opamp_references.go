package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

func checkOpAmpRows(checks *ElectricalResult, entry OpAmpCase) error {
	grid := make([]int, 101)
	mixed := entry.Name == "opamp-rc-npn" || entry.Name == "opamp-rc-mosfet"
	for i, row := range entry.Rows {
		index := int(math.Round(row.Time / .001))
		if len(row.Values) != len(entry.Probes) || index < 0 || index > 100 || math.Abs(row.Time-float64(index)*.001) > 1e-12 || i > 0 && row.Time < entry.Rows[i-1].Time {
			return fmt.Errorf("%s capture omitted planned times/probes", entry.Name)
		}
		grid[index]++
		want := math.Sin(20 * math.Pi * row.Time)
		tolerance := .0001
		if entry.Name == "opamp-inverting" {
			want = -want
		}
		if mixed {
			want = -5 * math.Expm1(-row.Time/.01)
			if row.Time > .05 {
				want = -5 * math.Expm1(-5) * math.Exp(-(row.Time-.05)/.01)
			}
			// Use the existing RC qualification envelope: 1 mV or 0.1%.
			// A 5 V fixture must not inherit the 1 V example's absolute bound.
			tolerance = math.Max(.001, .001*math.Abs(want))
			if err := checks.check(fmt.Sprintf("%s-rc-%03d", entry.Name, i), row.Values[0], want, tolerance); err != nil {
				return err
			}
			m := workbench.OpAmpDeviceModel()
			conductance, offset := 1./10000+1/m.DifferentialResistance, -row.Values[0]/m.DifferentialResistance
			if entry.Name == "opamp-rc-mosfet" {
				conductance += 1 / m.DifferentialResistance
				offset -= row.Values[2] / m.DifferentialResistance
				second := opAmpFollowerReference(row.Values[1], 1./100000+1/m.DifferentialResistance, -row.Values[1]/m.DifferentialResistance)
				if err := checks.check(fmt.Sprintf("%s-second-buffer-%03d", entry.Name, i), row.Values[2], second, 1e-7); err != nil {
					return err
				}
			} else {
				conductance += 1. / 10000
				offset -= row.Values[4] / 10000
			}
			want = opAmpFollowerReference(row.Values[0], conductance, offset)
			tolerance = 1e-7
		} else {
			input := want
			if entry.Name == "opamp-inverting" {
				input *= -.5
			}
			if err := checks.check(fmt.Sprintf("%s-sine-%03d", entry.Name, i), row.Values[0], input, 1e-11); err != nil {
				return err
			}
		}
		if err := checks.check(fmt.Sprintf("%s-buffer-%03d", entry.Name, i), row.Values[1], want, tolerance); err != nil {
			return err
		}
	}
	for index, count := range grid {
		want := 1
		if mixed && (index == 50 || index == 100) {
			want = 3
		}
		if count != want {
			return fmt.Errorf("%s sample/event coverage differs at %d ms: %d/%d", entry.Name, index, count, want)
		}
	}
	return nil
}

// Solve just the follower's scalar output KCL with a linear external load.
// This does not call the application's MNA, device evaluator or integrator.
// For the NPN fixture, the observed base voltage sets the resistor offset.
func opAmpFollowerReference(input, conductance, offset float64) float64 {
	m := workbench.OpAmpDeviceModel()
	enable := 100 / (100 + m.SupplySoftVoltage*m.SupplySoftVoltage)
	lo, hi := -5., 5.
	for range 100 {
		output := (lo + hi) / 2
		target := 5 * math.Tanh(m.OpenLoopGain*(input-output)/5)
		drop := target - output
		delivered := enable*m.ActiveCurrentLimit*drop/math.Hypot(m.OutputResistance*m.ActiveCurrentLimit, drop) - (output+5)/m.OutputBleedResistance
		if delivered > conductance*output+offset {
			lo = output
		} else {
			hi = output
		}
	}
	return (lo + hi) / 2
}
