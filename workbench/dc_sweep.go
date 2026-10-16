package workbench

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const DCSweepEngineID = "elab.network-mna-dc.v1"
const MaxDCSweepPoints = 201

type SweepSettings struct {
	ComponentID string   `json:"component_id"`
	Parameter   string   `json:"parameter"`
	Spacing     string   `json:"spacing,omitempty"`
	Start       string   `json:"start"`
	Stop        string   `json:"stop"`
	Points      int      `json:"points"`
	ProbeIDs    []string `json:"probe_ids,omitempty"`
}
type SweepTarget struct {
	ComponentID string  `json:"component_id"`
	Parameter   string  `json:"parameter"`
	Label       string  `json:"label"`
	Unit        string  `json:"unit"`
	Minimum     float64 `json:"minimum"`
	Maximum     float64 `json:"maximum"`
}
type SweepPoint struct {
	Input         float64   `json:"input"`
	Values        []float64 `json:"values"`
	KCLResidual   float64   `json:"kcl_residual_a"`
	PowerResidual float64   `json:"power_residual_w"`
}
type SweepSource struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	DCLevel float64 `json:"dc_level_v"`
	Origin  string  `json:"origin"`
}
type SweepStatistics struct {
	ProbeID string  `json:"probe_id"`
	Minimum float64 `json:"minimum"`
	Maximum float64 `json:"maximum"`
}
type SweepResult struct {
	Target     SweepTarget       `json:"target"`
	Spacing    string            `json:"spacing,omitempty"`
	Start      float64           `json:"start_si"`
	Stop       float64           `json:"stop_si"`
	Probes     []Probe           `json:"probes"`
	Points     []SweepPoint      `json:"points"`
	Statistics []SweepStatistics `json:"statistics"`
}
type sweepPlan struct {
	graph       Graph
	settings    SweepSettings
	target      SweepTarget
	start, stop float64
	probes      []Probe
	sources     []SweepSource
}

// Shared by target inventory and on-demand agent discovery.
type sweepDomain struct {
	Parameter string  `json:"parameter"`
	Unit      string  `json:"unit"`
	Minimum   float64 `json:"minimum"`
	Maximum   float64 `json:"maximum"`
}

func sweepDomains() []sweepDomain {
	return []sweepDomain{{"voltage", "V", -12, 12}, {"resistance", "Ω", 10, 100000}, {"wiper", "ratio", 0, 1}}
}

func sweepTargets(g Graph) []SweepTarget {
	out := []SweepTarget{}
	domains := sweepDomains()
	for _, p := range g.Components {
		for _, d := range domains {
			label := ""
			switch d.Parameter {
			case "voltage":
				if IsVoltageSource(p.Kind) {
					label = "DC bias"
				}
			case "resistance":
				if p.Kind == "resistor" || p.Kind == "potentiometer" {
					label = "Resistance"
				}
			case "wiper":
				if p.Kind == "potentiometer" {
					label = "Wiper position"
				}
			}
			if label != "" {
				out = append(out, SweepTarget{p.ID, d.Parameter, p.ID + " · " + label, d.Unit, d.Minimum, d.Maximum})
			}
		}
	}
	return out
}
func dcSourceLevels(g Graph) []SweepSource {
	out := []SweepSource{}
	for _, p := range g.Components {
		if !IsVoltageSource(p.Kind) {
			continue
		}
		level, origin := p.Value, "configured DC value"
		switch p.Kind {
		case "pulse-source":
			level, origin = 0, "ideal pulse low level"
		case "sine-source":
			level, origin = p.Sine.Offset, "sine offset; phase/frequency ignored"
		case "finite-pulse-source":
			level, origin = p.Pulse.LowVoltage, "finite pulse low level; timing ignored"
		}
		out = append(out, SweepSource{p.ID, p.Kind, level, origin})
	}
	return out
}
func newSweepPlan(g Graph, settings SweepSettings) (sweepPlan, error) {
	return buildSweepPlan(g, settings, true)
}
func buildSweepPlan(g Graph, settings SweepSettings, requireComplete bool) (sweepPlan, error) {
	plan := sweepPlan{graph: g.Clone(), settings: settings}
	plan.settings.ProbeIDs = append([]string{}, settings.ProbeIDs...)
	found := false
	for _, target := range sweepTargets(g) {
		if target.ComponentID == settings.ComponentID && target.Parameter == settings.Parameter {
			plan.target = target
			found = true
			break
		}
	}
	if !found {
		return plan, fmt.Errorf("choose an existing DC bias, resistance or wiper target")
	}
	if settings.Points < 2 || settings.Points > MaxDCSweepPoints {
		return plan, fmt.Errorf("DC sweep requires 2–201 points")
	}
	parse := func(text string) (float64, error) {
		if len(text) == 0 || len(text) > 128 {
			return 0, fmt.Errorf("sweep endpoints require 1–128 bytes")
		}
		unit := plan.target.Unit
		if unit == "ratio" {
			unit = ""
		}
		value, err := ParseValue(text, unit)
		if plan.target.Parameter == "wiper" {
			value, err = strconv.ParseFloat(strings.TrimSpace(text), 64)
		}
		if err != nil {
			return 0, err
		}
		if !finite(value) || value < plan.target.Minimum || value > plan.target.Maximum {
			return 0, fmt.Errorf("sweep endpoint must be %.9g–%.9g %s", plan.target.Minimum, plan.target.Maximum, plan.target.Unit)
		}
		return value, nil
	}
	var err error
	if plan.start, err = parse(settings.Start); err != nil {
		return plan, err
	}
	if plan.stop, err = parse(settings.Stop); err != nil {
		return plan, err
	}
	if plan.start == plan.stop {
		return plan, fmt.Errorf("sweep endpoints must differ")
	}
	if settings.Spacing != "" && settings.Spacing != "linear" && settings.Spacing != "logarithmic" {
		return plan, fmt.Errorf("sweep spacing must be linear or logarithmic")
	}
	if settings.Spacing == "logarithmic" && (plan.start <= 0 || plan.stop <= 0) {
		return plan, fmt.Errorf("logarithmic sweep endpoints must both be positive; use linear spacing for zero or negative values")
	}
	previous := plan.start
	for i := 1; i < settings.Points; i++ {
		value := sweepAxisPoint(plan.start, plan.stop, settings.Points, i, settings.Spacing)
		if !finite(value) || value == previous || (plan.stop > plan.start && value < previous) || (plan.stop < plan.start && value > previous) {
			return plan, fmt.Errorf("range is too small to represent distinct sweep points; expand it or reduce points")
		}
		previous = value
	}

	if _, _, diagnostic := g.Compile(); requireComplete && diagnostic != nil {
		return plan, &actionError{"unavailable", "Complete a supported circuit before sweeping: " + diagnostic.Message}
	}
	available := probeInventory(g)
	ids := settings.ProbeIDs
	if ids == nil {
		for _, wanted := range []string{"i:" + settings.ComponentID, "v:" + settings.ComponentID} {
			for _, p := range available {
				if p.ID == wanted {
					ids = append(ids, p.ID)
					break
				}
			}
		}
		if len(ids) == 0 && len(available) > 0 {
			ids = []string{available[0].ID}
		}
	}

	if len(ids) == 0 || len(ids) > MaxScopeProbes {
		return plan, fmt.Errorf("choose 1–6 sweep probes")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		found := false
		for _, probe := range available {
			if probe.ID == id && !seen[id] {
				plan.probes = append(plan.probes, probe)
				found = true
				break
			}
		}
		if !found {
			return plan, fmt.Errorf("unknown or duplicate sweep probe %q", id)
		}
		seen[id] = true
	}
	plan.settings.ProbeIDs = append([]string{}, ids...)
	plan.sources = dcSourceLevels(g)
	return plan, nil
}

// Capacitors are open in a DC operating point. Removing their startup constraints
// from the private MNA plan leaves live transient initialization unchanged. Source
// waveforms contribute an explicit DC bias, never an arbitrary time-zero sample.
func solveDCOperatingPoint(ctx context.Context, g Graph, levels []SweepSource) (*MixedResult, TraceSample, error) {
	compiled, _, diagnostic := g.Compile()
	if diagnostic != nil {
		return nil, TraceSample{}, fmt.Errorf("%s", diagnostic.Message)
	}
	compiled.Analysis = NetworkDCAnalysis
	compiled.MixedGraph = g.Clone()
	for _, p := range g.Components {
		if p.Kind == "ground" {
			compiled.Ground = compiled.TerminalNet[p.ID+".reference"]
			break
		}
	}
	network := newMixedNetwork(compiled)
	network.caps = nil
	network.dc = true
	for i, p := range network.sources {
		for _, source := range levels {
			if source.ID == p.ID {
				p.Kind = "dc-source"
				p.Sine, p.Pulse = nil, nil
				p.Value = source.DCLevel
				network.sources[i] = p
				break
			}
		}
	}
	state, err := network.solveContext(ctx, nil, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
	if err != nil {
		return nil, TraceSample{}, err
	}
	// A zero residual alone does not establish a determined operating point:
	// an open capacitor island can satisfy KCL for arbitrary node voltages.
	// Check the solved local system even when Newton needed no iteration.
	matrix, _ := network.equations(state.x, nil, 0, &Transient{Mode: "auto", Settings: DefaultTransientSettings()}, 0)
	if _, err := linearSolve(matrix, make([]float64, len(matrix))); err != nil {
		return nil, TraceSample{}, fmt.Errorf("DC operating point is not locally determined: %w", err)
	}
	result := state.result
	result.Engine = DCSweepEngineID
	for _, p := range g.Components {
		if p.Kind != "capacitor" {
			continue
		}
		v := result.Nodes[compiled.TerminalNet[p.ID+".1"]] - result.Nodes[compiled.TerminalNet[p.ID+".2"]]
		result.Capacitors = append(result.Capacitors, CapacitorResult{ID: p.ID, Voltage: v, Current: 0, Energy: .5 * p.Value * v * v})
	}
	if !finite(result.KCLResidual) || !finite(result.PowerResidual) {
		return nil, TraceSample{}, fmt.Errorf("nonfinite DC conservation residual")
	}
	return result, mixedSample(compiled, result, 0), nil
}
func computeDCSweep(ctx context.Context, plan sweepPlan, progress func(int)) (*SweepResult, error) {
	out := &SweepResult{Target: plan.target, Spacing: plan.settings.Spacing, Start: plan.start, Stop: plan.stop, Probes: append([]Probe{}, plan.probes...), Points: []SweepPoint{}, Statistics: []SweepStatistics{}}
	for index := 0; index < plan.settings.Points; index++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value := sweepAxisPoint(plan.start, plan.stop, plan.settings.Points, index, plan.settings.Spacing)
		graph := plan.graph.Clone()
		levels := append([]SweepSource{}, plan.sources...)
		if plan.target.Parameter == "voltage" {
			for i := range levels {
				if levels[i].ID == plan.target.ComponentID {
					levels[i].DCLevel = value
				}
			}
		} else {
			for i := range graph.Components {
				if graph.Components[i].ID != plan.target.ComponentID {
					continue
				}
				if plan.target.Parameter == "wiper" {
					graph.Components[i].WiperPosition = &value
				} else {
					graph.Components[i].Value = value
				}
			}
		}
		result, sample, err := solveDCOperatingPoint(ctx, graph, levels)
		if err != nil {
			return nil, fmt.Errorf("DC sweep point %d at %.17g %s: %w", index+1, value, plan.target.Unit, err)
		}
		point := SweepPoint{Input: value, Values: []float64{}, KCLResidual: result.KCLResidual, PowerResidual: result.PowerResidual}
		for _, probe := range plan.probes {
			reading, ok := frameValue(sample, probe)
			if !ok || !finite(reading) {
				return nil, fmt.Errorf("DC point did not produce probe %s", probe.ID)
			}
			point.Values = append(point.Values, reading)
		}
		out.Points = append(out.Points, point)
		progress(index + 1)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for i, p := range out.Probes {
		min, max := math.Inf(1), math.Inf(-1)
		for _, point := range out.Points {
			min = math.Min(min, point.Values[i])
			max = math.Max(max, point.Values[i])
		}
		out.Statistics = append(out.Statistics, SweepStatistics{p.ID, min, max})
	}
	return out, nil
}
func cloneSweepResult(in *SweepResult) *SweepResult {
	if in == nil {
		return nil
	}
	out := *in
	out.Probes = append([]Probe{}, in.Probes...)
	out.Statistics = append([]SweepStatistics{}, in.Statistics...)
	out.Points = make([]SweepPoint, len(in.Points))
	for i, p := range in.Points {
		out.Points[i] = p
		out.Points[i].Values = append([]float64{}, p.Values...)
	}
	return &out
}
