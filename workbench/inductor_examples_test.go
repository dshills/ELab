package workbench

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestInductorGuidedExamplesPulsePhases(t *testing.T) {
	for _, id := range []string{"rl", "rlc", "flyback"} {
		t.Run(id, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: id}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			initial := settled(t, s)
			if initial.Mixed == nil || initial.Mixed.Stale || initial.Analysis != InductorAnalysis || len(initial.Mixed.Inductors) != 1 || initial.Mixed.Inductors[0].Current != 0 {
				t.Fatal("example startup", initial.Diagnostics)
			}
			flybackCurrent := 0.
			for sample := 1; sample <= 120; sample++ {
				call(t, s, "simulation.step", StepParams{Duration: .001}, fmt.Sprint("step-", sample))
				state := s.Snapshot()
				if state.Mixed.Stale || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
					t.Fatal("example conservation", sample, state.Diagnostics, state.Mixed)
				}
				if id == "rl" {
					at := float64(sample) * .001
					want := .05 * -math.Expm1(-at/.01)
					if sample >= 50 && sample < 100 {
						want = .05 * -math.Expm1(-5) * math.Exp(-(at-.05)/.01)
					} else if sample >= 100 {
						start := .05 * -math.Expm1(-5) * math.Exp(-5)
						want = .05 + (start-.05)*math.Exp(-(at-.1)/.01)
					}
					if math.Abs(state.Mixed.Inductors[0].Current-want) > 1e-5 {
						t.Fatal("pulse RL exponential", sample, state.Mixed.Inductors[0], want)
					}
				}
				if id == "flyback" && sample == 50 {
					coil, diode, mos := state.Mixed.Inductors[0], state.Mixed.Diodes[0], state.Mixed.MOSFETs[0]
					if coil.Current < .045 || diode.Current < .045 || mos.State != "cutoff" || mos.VDS < 5.7 || mos.VDS > 6 || math.Abs(coil.Current-diode.Current-mos.DrainCurrent) > 1e-9 {
						t.Fatal("off edge did not recirculate coil current", coil, diode, mos)
					}
				}
				if id == "flyback" && sample <= 55 {
					// Independent scalar ODE, RK4 with 100 fixed substeps per
					// output. The off reference stops before the stiff nA tail.
					flybackCurrent = flybackReferenceStep(flybackCurrent, sample <= 50)
					if math.Abs(state.Mixed.Inductors[0].Current-flybackCurrent) > 1e-5 {
						t.Fatal("independent flyback ODE", sample, state.Mixed.Inductors[0], flybackCurrent)
					}
				}
				if id == "flyback" && sample == 99 {
					if math.Abs(state.Mixed.Inductors[0].Current-1e-9) > 1e-9 || math.Abs(state.Mixed.MOSFETs[0].VDS-(5-1e-7)) > .001 {
						t.Fatal("off-phase leakage equilibrium", state.Mixed)
					}
				}
			}
			// A fresh run restores startup intent instead of the accepted current.
			call(t, s, "simulation.restart", struct{}{}, "fresh")
			restarted := settled(t, s)
			if restarted.Transient.Sample.Time != 0 || !reflect.DeepEqual(restarted.Mixed.Inductors, initial.Mixed.Inductors) {
				t.Fatal("example restart imported live current", restarted.Diagnostics)
			}
		})
	}
}

// Pin the documented generic constants without calling the MNA/device kernel.
// On: invert the triode quadratic with channel modulation (body/flyback
// reverse leakages cancel to <1 nA here). Off: invert the diode Shockley law;
// the MOS drain contributes its 1 nA reverse body-diode leakage.
func flybackReferenceStep(current float64, on bool) float64 {
	derivative := func(i float64) float64 {
		if !on {
			junction := i - 1e-9
			drop := 1.7*1.380649e-23*300.15/1.602176634e-19*math.Log1p(junction/1e-9) + junction
			return -100*i - drop
		}
		u := 5.*100000/100100 - 1.5
		lo, hi := 0., u
		for k := 0; k < 60; k++ {
			v := (lo + hi) / 2
			if .1*(u*v-.5*v*v)*(1+.02*v) > i {
				hi = v
			} else {
				lo = v
			}
		}
		return 5 - 100*i - (lo+hi)/2
	}
	const h = .00001
	for k := 0; k < 100; k++ {
		a := derivative(current)
		b := derivative(current + h*a/2)
		c := derivative(current + h*b/2)
		d := derivative(current + h*c)
		current += h * (a + 2*b + 2*c + d) / 6
	}
	return current
}

func TestInductorMissingFlybackPathRefusesWithoutPublishingPartialTime(t *testing.T) {
	g := FlybackExampleGraph()
	parts := g.Components[:0]
	for _, p := range g.Components {
		if p.ID != "D1" && p.ID != "JD1" && p.ID != "JD2" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	g.Wires = g.Wires[:len(g.Wires)-4]
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "simulation.step", StepParams{Duration: .049}, "before-edge")
	before := s.Snapshot()
	refuseStepWork(t, s, "simulation.step", StepParams{Duration: .002}, "solve-failed")
	after := s.Snapshot()
	if !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("missing return path published partial time or current", after.Diagnostics)
	}
}

func TestInductorExampleLabelMasksLeaveWiresContinuous(t *testing.T) {
	assertLabelMasksLeaveWiresContinuous(t, RLExampleGraph(), []string{"R1", "L1"})
	assertLabelMasksLeaveWiresContinuous(t, RLCExampleGraph(), []string{"R1", "L1", "C1"})
	assertLabelMasksLeaveWiresContinuous(t, FlybackExampleGraph(), []string{"RG", "RPD", "RL", "L1", "D1", "M1"})
}
