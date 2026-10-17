package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

// Exact linear RLC response to an ideal .2 V buffer, Rseries=Rload=1k,
// L=.1 H, C=1uF, Vc(0)=.05 V and I(0)=-50uA. The declared tolerance also
// covers the educational buffer's finite-gain departure from the ideal drive.
func bufferedRLCReference(at float64) (voltage, current float64) {
	const steady = .1
	root := math.Sqrt(5500*5500 - 2e7)
	a, b := -5500+root, -5500-root
	coefficient := (-100 - b*(.05-steady)) / (a - b)
	other := .05 - steady - coefficient
	voltage = steady + coefficient*math.Exp(a*at) + other*math.Exp(b*at)
	derivative := a*coefficient*math.Exp(a*at) + b*other*math.Exp(b*at)
	return voltage, 1e-6*derivative + voltage/1000
}

// Independent scalar load line for this 5 V high-side PMOS/1k/red-LED fixture.
// It uses the fixed v1 square-law/LED equations, without the network solver.
// Replacing reverse body leakage by 1 nA costs less than the 2 nA check bound.
func pmosDriverLEDReference(gate float64) float64 {
	u := 5 - gate - 1.5
	if u <= 0 {
		return 1e-9
	}
	lo, hi := 0., .005
	for range 100 {
		current := (lo + hi) / 2
		led := 2*1.380649e-23*300.15/1.602176634e-19*math.Log1p(current/1e-20) + current
		vds := 5 - 1000*current - led
		channel := 0.
		if vds > 0 {
			if vds < u {
				channel = .1 * (u*vds - .5*vds*vds) * (1 + .02*vds)
			} else {
				channel = .05 * u * u * (1 + .02*vds)
			}
		}
		if channel+1e-9 > current {
			lo = current
		} else {
			hi = current
		}
	}
	return (lo + hi) / 2
}

func checkOpAmpStorageRows(checks *ElectricalResult, entry OpAmpCase) error {
	grid := make([]int, 101)
	step, final := .001, 100
	if entry.Name == "opamp-buffered-rlc" {
		step, final = .0005, 20
		grid = make([]int, 21)
	}
	for i, row := range entry.Rows {
		k := int(math.Round(row.Time / step))
		if len(row.Values) != 4 || k < 0 || k > final || math.Abs(row.Time-float64(k)*step) > 1e-12 || i > 0 && row.Time < entry.Rows[i-1].Time {
			return fmt.Errorf("%s missing planned time/probe coverage", entry.Name)
		}
		grid[k]++
		if entry.Name == "opamp-buffered-rlc" {
			if err := checkBufferedRLCRow(checks, fmt.Sprintf("%s-%03d", entry.Name, i), row); err != nil {
				return err
			}
		} else {
			voltage := -5 * math.Expm1(-row.Time/.01)
			if row.Time > .05 {
				voltage = -5 * math.Expm1(-5) * math.Exp(-(row.Time-.05)/.01)
			}
			m := workbench.OpAmpDeviceModel()
			for _, item := range []struct {
				name                 string
				got, want, tolerance float64
			}{{"capacitor-v", row.Values[0], voltage, math.Max(.001, .001*math.Abs(voltage))}, {"buffer-v", row.Values[1], opAmpFollowerReference(row.Values[0], 1./10000+1./100000+1/m.DifferentialResistance, -row.Values[0]/m.DifferentialResistance), 1e-7}, {"led-a", row.Values[2], pmosDriverLEDReference(row.Values[1]), 2e-9}, {"signed-drain-a", row.Values[3], -row.Values[2], 1e-9}} {
				if err := checks.check(fmt.Sprintf("%s-%s-%03d", entry.Name, item.name, i), item.got, item.want, item.tolerance); err != nil {
					return err
				}
			}
		}
	}
	for k, count := range grid {
		want := 1
		if entry.Name == "opamp-rc-pmos" && (k == 50 || k == 100) {
			want = 3
		}
		if count != want {
			return fmt.Errorf("%s grid/event coverage %d: %d/%d", entry.Name, k, count, want)
		}
	}
	return nil
}

func checkBufferedRLCRow(checks *ElectricalResult, label string, row workbench.VerificationRow) error {
	voltage, current := bufferedRLCReference(row.Time)
	for _, item := range []struct {
		name                 string
		got, want, tolerance float64
	}{{"capacitor-v", row.Values[0], voltage, 1e-4}, {"coil-a", row.Values[1], current, 1e-7}, {"buffer-v", row.Values[2], opAmpFollowerReference(.2, 1./10000+1./1e7, row.Values[1]-.2/1e7), 1e-7}, {"delivered-a", row.Values[3], row.Values[2]/10000 + row.Values[1] + (row.Values[2]-.2)/1e7, 1e-10}} {
		if err := checks.check(label+"-"+item.name, item.got, item.want, item.tolerance); err != nil {
			return err
		}
	}
	return nil
}

func checkRLCRefinement(checks *ElectricalResult, coarse, fine []workbench.VerificationRow) error {
	if len(coarse) != 21 || len(fine) != 41 {
		return fmt.Errorf("RLC refinement coverage differs")
	}
	for i, row := range fine {
		if len(row.Values) != 4 || math.Abs(row.Time-float64(i)*.00025) > 1e-12 {
			return fmt.Errorf("RLC refinement omitted planned times/probes")
		}
		if err := checkBufferedRLCRow(checks, fmt.Sprintf("opamp-buffered-rlc-fine-%03d", i), row); err != nil {
			return err
		}
	}
	for i, row := range coarse {
		for p, value := range row.Values {
			tolerance := 1e-7
			if p == 1 || p == 3 {
				tolerance = 1e-9
			}
			if err := checks.check(fmt.Sprintf("rlc-refinement-%03d-%d", i, p), value, fine[2*i].Values[p], tolerance); err != nil {
				return err
			}
		}
	}
	return nil
}
