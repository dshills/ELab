package conformance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/dshills/elab/workbench"
)

func measureFlasher(s *session, step float64) (out FlasherMeasurement, err error) {
	out = FlasherMeasurement{Step: step, Minimum: map[string]float64{}, Maximum: map[string]float64{}}
	if err = s.ask("example.flasher", nil, true, nil); err != nil {
		return out, err
	}
	if err = s.ask("simulation.pause", nil, true, nil); err != nil {
		return out, err
	}
	cfg := workbench.DefaultTransientSettings()
	cfg.StepSeconds = step
	if err = s.ask("simulation.transient.set", cfg, true, nil); err != nil {
		return out, err
	}
	state, err := restart(s)
	if err != nil {
		return out, err
	}
	probes := []string{"i:D1", "i:D2", "v:C1", "v:C2"}
	if err = s.ask("scope.probes.set", workbench.ProbeParams{IDs: probes}, true, nil); err != nil {
		return out, err
	}
	graph, err := json.Marshal(workbench.Graph{Components: state.Components, Wires: state.Wires})
	if err != nil {
		return out, err
	}
	out.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(graph))
	_, _, out.VerifyWall, err = electricalVerify(s, .2, step, probes)
	if err != nil {
		return out, err
	}
	series := map[string][]workbench.ScopePoint{}
	for _, id := range probes {
		out.Minimum[id] = math.Inf(1)
		out.Maximum[id] = math.Inf(-1)
	}
	last := -1.
	for batch := 0; batch < 20; batch++ {
		state, err = s.state("simulation.step", workbench.StepParams{Duration: .1}, true)
		if err != nil {
			return out, err
		}
		if state.Transient == nil || state.Transient.Stale || !state.Paused || state.Mixed == nil || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			return out, fmt.Errorf("flasher lost fresh paused/conserved state")
		}
		count := -1
		for _, id := range probes {
			found := false
			n := 0
			for _, trace := range state.Scope.Traces {
				if trace.Probe.ID != id {
					continue
				}
				found = true
				for _, p := range trace.Points {
					if p.Time <= last+1e-12 {
						continue
					}
					series[id] = append(series[id], p)
					n++
					out.Minimum[id] = math.Min(out.Minimum[id], p.Value)
					out.Maximum[id] = math.Max(out.Maximum[id], p.Value)
				}
			}
			if !found || n == 0 || count >= 0 && n != count {
				return out, fmt.Errorf("flasher missing aligned %s samples", id)
			}
			count = n
		}
		last = state.Transient.Sample.Time
	}
	points := series[probes[0]]
	out.Samples = len(points)
	if math.Abs(last-2) > 1e-10 || out.Samples < int(math.Round(2/step))+1 {
		return out, fmt.Errorf("flasher sampling horizon incomplete: %g/%d", last, out.Samples)
	}
	for i := 1; i < len(points); i++ {
		if math.Abs(points[i].Time-points[i-1].Time-step) > 1e-10 {
			return out, fmt.Errorf("flasher rolling history lost a sample")
		}
		if points[i-1].Value <= .001 && points[i].Value > .001 && points[i-1].Time >= .4 {
			out.RisingBrackets = append(out.RisingBrackets, [2]float64{points[i-1].Time, points[i].Time})
		}
	}
	if len(out.RisingBrackets) < 3 {
		return out, fmt.Errorf("flasher needs two settled periods, got %d rising brackets", len(out.RisingBrackets))
	}
	periods := []float64{}
	for i := 1; i < len(out.RisingBrackets); i++ {
		a, b := out.RisingBrackets[i-1], out.RisingBrackets[i]
		periods = append(periods, (b[0]+b[1]-a[0]-a[1])/2)
	}
	sort.Float64s(periods)
	out.Period = periods[len(periods)/2]
	if len(periods)%2 == 0 {
		out.Period = (periods[len(periods)/2-1] + out.Period) / 2
	}
	return out, nil
}
