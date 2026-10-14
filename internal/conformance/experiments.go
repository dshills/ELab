package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

func restart(s *session) (workbench.Snapshot, error) {
	if err := s.ask("simulation.restart", nil, true, nil); err != nil {
		return workbench.Snapshot{}, err
	}
	return s.state("simulation.wait", nil, false)
}
func fresh(s workbench.Snapshot, name string) bool {
	switch name {
	case "led":
		return s.Result != nil && !s.Result.Stale
	case "rc":
		return s.Transient != nil && !s.Transient.Stale
	case "npn":
		return s.NPN != nil && !s.NPN.Stale
	}
	return false
}
func experiment(s *session, name string, out *Result) error {
	switch name {
	case "led":
		start, err := s.state("project.inspect", nil, false)
		if err != nil {
			return err
		}
		if start.Result == nil || math.Abs(start.Result.Current-.0029167966) > .0000292 {
			return fmt.Errorf("LED reference current differs")
		}
		out.Measurements["current_1k_a"] = start.Result.Current
		// Resume makes this an automatic update caused by a value edit.
		if err = s.ask("simulation.resume", nil, true, nil); err != nil {
			return err
		}
		if err = s.ask("component.set", workbench.ValueParams{ID: "Limit", Value: "2k"}, true, nil); err != nil {
			return err
		}
		changed, err := s.state("simulation.wait", nil, false)
		if err != nil {
			return err
		}
		if changed.Result == nil || changed.Result.Stale || changed.Result.Current >= start.Result.Current {
			return fmt.Errorf("LED edit does not lower accepted live current")
		}
		out.Measurements["current_2k_a"] = changed.Result.Current
		return s.ask("simulation.pause", nil, true, nil)
	case "rc":
		stepped, err := s.state("simulation.step", workbench.StepParams{Duration: .01}, true)
		if err != nil {
			return err
		}
		if stepped.Transient == nil || math.Abs(stepped.Transient.Sample.Voltage-3.160602794142788) > .003161 {
			return fmt.Errorf("RC 10 ms reference differs")
		}
		out.Measurements["voltage_10ms_v"] = stepped.Transient.Sample.Voltage
		if err = s.ask("component.set", workbench.ValueParams{ID: "Limit", Value: "20k"}, true, nil); err != nil {
			return err
		}
		if _, err = restart(s); err != nil {
			return err
		}
		changed, err := s.state("simulation.step", workbench.StepParams{Duration: .01}, true)
		if err != nil {
			return err
		}
		if changed.Transient.Sample.Voltage >= stepped.Transient.Sample.Voltage {
			return fmt.Errorf("larger RC resistance does not slow charge")
		}
		out.Measurements["voltage_20k_10ms_v"] = changed.Transient.Sample.Voltage
		return s.ask("component.set", workbench.ValueParams{ID: "Limit", Value: "10k"}, true, nil)
	case "npn":
		for _, drive := range []struct{ value, state string }{{"0 V", "cutoff"}, {"0.8 V", "active"}, {"5 V", "saturation"}} {
			if err := s.ask("component.set", workbench.ValueParams{ID: "Drive", Value: drive.value}, true, nil); err != nil {
				return err
			}
			state, err := restart(s)
			if err != nil {
				return err
			}
			if state.NPN == nil || state.NPN.Stale || state.NPN.State != drive.state || state.NPN.KCLResidual > 1e-10 || math.Abs(state.NPN.KVLResidual) > 1e-10 {
				return fmt.Errorf("NPN %s state/conservation differs", drive.state)
			}
			if drive.state == "saturation" && math.Abs(state.NPN.VCE-.0704) > .001 {
				return fmt.Errorf("NPN resistor-load on-state reference differs: %g V", state.NPN.VCE)
			}
			out.Measurements[drive.state+"_collector_a"] = state.NPN.CollectorCurrent
			out.Measurements[drive.state+"_vce_v"] = state.NPN.VCE
		}
	}
	return nil
}
