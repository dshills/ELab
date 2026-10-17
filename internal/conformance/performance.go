package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"time"

	"github.com/dshills/elab/workbench"
)

type PerformanceSample struct {
	CapacityVerification *workbench.VerificationDocument `json:"capacity_verification,omitempty"`
	Index                int                             `json:"index"`
	Condition            string                          `json:"condition"`
	GraphSHA256          string                          `json:"graph_sha256"`
	Engine               string                          `json:"engine"`
	ElectricalRevision   uint64                          `json:"electrical_revision"`
	RunID                uint64                          `json:"run_id"`
	ProbeIDs             []string                        `json:"probe_ids"`
	Models               map[string]string               `json:"model_ids"`
	Settings             workbench.TransientSettings     `json:"transient_settings"`
	AcknowledgmentMS     float64                         `json:"command_roundtrip_ms"`
	FreshResultMS        float64                         `json:"edit_start_to_fresh_result_ms"`
	VerifyMS             float64                         `json:"verify_admission_to_completion_ms"`
	VerifyDuration       float64                         `json:"verified_duration_s"`
	VerifySampleStep     float64                         `json:"verified_sample_step_s"`
	graph                workbench.Graph
}
type PerformanceSummary struct {
	Count   int     `json:"count"`
	P50     float64 `json:"p50_ms"`
	P95     float64 `json:"p95_ms"`
	Maximum float64 `json:"maximum_ms"`
}
type PerformanceCase struct {
	Name               string              `json:"name"`
	Condition          string              `json:"condition"`
	Samples            []PerformanceSample `json:"samples"`
	Acknowledgment     PerformanceSummary  `json:"command_roundtrip"`
	FreshResult        PerformanceSummary  `json:"edit_to_fresh_result"`
	Verify             PerformanceSummary  `json:"verify_completion"`
	FreshTargetPassed  bool                `json:"backend_fresh_500ms_target_passed"`
	VerifyTargetPassed bool                `json:"verify_2s_target_passed"`
}
type PerformanceResult struct {
	Families      []string                   `json:"circuit_families"`
	Repetitions   int                        `json:"repetitions"`
	TargetsPassed bool                       `json:"backend_targets_passed"`
	Cases         []PerformanceCase          `json:"cases"`
	Circuits      map[string]workbench.Graph `json:"circuits_by_sha256"`
	Limitations   []string                   `json:"limitations"`
}

func summarizePerformance(values []float64) (PerformanceSummary, error) {
	if len(values) == 0 {
		return PerformanceSummary{}, fmt.Errorf("performance evidence missing")
	}
	values = append([]float64(nil), values...)
	for _, v := range values {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return PerformanceSummary{}, fmt.Errorf("invalid duration")
		}
	}
	sort.Float64s(values)
	return PerformanceSummary{len(values), values[int(math.Ceil(.5*float64(len(values))))-1], values[int(math.Ceil(.95*float64(len(values))))-1], values[len(values)-1]}, nil
}

// RunPerformance measures only owned sessions and public domain commands.
// Wall timings must be recorded in an ordinary build, outside race/other loads.
func RunPerformance(ctx context.Context, factory Factory, dir string, repetitions int) (out PerformanceResult, err error) {
	return runPerformance(ctx, factory, dir, repetitions, []string{"led", "rc", "npn", "npn-pulse", "flasher"})
}

// RunInductorPerformance keeps coil timing separate from the original families.
// All four cases capture 200 ms at 1 ms outputs, including two switching periods.
func RunInductorPerformance(ctx context.Context, factory Factory, dir string, repetitions int) (out PerformanceResult, err error) {
	return runPerformance(ctx, factory, dir, repetitions, []string{"rl", "rlc", "flyback", "npn-coil"})
}

// RunOpAmpPerformance measures the guided amplifiers and their mixed RC/device
// loads separately. Each Verify captures 200 ms at 1 ms output intervals.
func RunOpAmpPerformance(ctx context.Context, factory Factory, dir string, repetitions int) (PerformanceResult, error) {
	return runPerformance(ctx, factory, dir, repetitions, opAmpCaseNames)
}

func runPerformance(ctx context.Context, factory Factory, dir string, repetitions int, families []string) (out PerformanceResult, err error) {
	if repetitions < 20 || repetitions > 100 {
		return out, fmt.Errorf("performance repetitions must be 20–100")
	}
	out.Repetitions = repetitions
	out.Families = append([]string(nil), families...)
	out.TargetsPassed = true
	out.Circuits = make(map[string]workbench.Graph)
	out.Limitations = []string{"command start includes admission/transport/JSON overhead and is a conservative backend upper bound; it is not input-to-visible acknowledgment", "fresh means a fresh owned session with the reference circuit prepared; OS/filesystem caches are not purged and process-launch time is excluded", "warm means one owned session after an unmeasured edit/Verify; successful records are excluded from project retention between repetitions and may be evicted by the ordinary bounded store", "only the explicitly listed circuit families are measured on this workstation; no universal hardware or UI performance claim"}
	for _, name := range families {
		for _, condition := range []string{"fresh-session", "warm-session"} {
			entry := PerformanceCase{Name: name, Condition: condition}
			if condition == "fresh-session" {
				for i := 0; i < repetitions; i++ {
					var sample PerformanceSample
					err = electricalSession(ctx, factory, filepath.Join(dir, name, condition, fmt.Sprintf("%03d", i)), func(s *session) error {
						if e := preparePerformance(s, name); e != nil {
							return e
						}
						var e error
						sample, e = measurePerformance(s, name, i, condition)
						return e
					})
					if err != nil {
						return out, fmt.Errorf("%s %s #%d: %w", name, condition, i, err)
					}
					entry.Samples = append(entry.Samples, sample)
				}
			} else {
				err = electricalSession(ctx, factory, filepath.Join(dir, name, condition), func(s *session) error {
					if e := preparePerformance(s, name); e != nil {
						return e
					}
					if _, e := measurePerformance(s, name, -1, condition); e != nil {
						return e
					}
					for i := 0; i < repetitions; i++ {
						sample, e := measurePerformance(s, name, i, condition)
						if e != nil {
							return e
						}
						entry.Samples = append(entry.Samples, sample)
					}
					return nil
				})
				if err != nil {
					return out, fmt.Errorf("%s warm: %w", name, err)
				}
			}
			ack, fresh, verify := []float64{}, []float64{}, []float64{}
			for _, sample := range entry.Samples {
				out.Circuits[sample.GraphSHA256] = sample.graph
				ack = append(ack, sample.AcknowledgmentMS)
				fresh = append(fresh, sample.FreshResultMS)
				verify = append(verify, sample.VerifyMS)
			}
			entry.Acknowledgment, err = summarizePerformance(ack)
			if err != nil {
				return out, err
			}
			entry.FreshResult, err = summarizePerformance(fresh)
			if err != nil {
				return out, err
			}
			entry.Verify, err = summarizePerformance(verify)
			if err != nil {
				return out, err
			}
			entry.FreshTargetPassed = entry.FreshResult.P95 <= 500
			entry.VerifyTargetPassed = entry.Verify.Maximum <= 2000
			out.TargetsPassed = out.TargetsPassed && entry.FreshTargetPassed && entry.VerifyTargetPassed
			out.Cases = append(out.Cases, entry)
		}
	}
	return out, nil
}
func preparePerformance(s *session, name string) error {
	if isCapacityPerformance(name) {
		if _, err := prepareCapacity(s, name); err != nil {
			return err
		}
	} else if isOpAmpPerformance(name) {
		if err := prepareOpAmpCase(s, name); err != nil {
			return err
		}
		if _, err := restart(s); err != nil {
			return err
		}
	} else if isInductorPerformance(name) {
		id := name
		if name == "npn-coil" {
			id = "flyback"
		}
		if err := s.ask("example.load", workbench.ExampleLoadParams{ID: id}, true, nil); err != nil {
			return err
		}
		if err := s.ask("simulation.pause", nil, true, nil); err != nil {
			return err
		}
		if name == "npn-coil" {
			if err := buildInductorNPNExperiment(s); err != nil {
				return err
			}
		} else if _, err := restart(s); err != nil {
			return err
		}
	} else if name == "flasher" {
		if err := s.ask("example.flasher", nil, true, nil); err != nil {
			return err
		}
	} else {
		for _, c := range circuits {
			if c.name == name || name == "npn-pulse" && c.name == "npn" {
				if name == "npn-pulse" {
					c.parts = append([]part(nil), c.parts...)
					for i := range c.parts {
						if c.parts[i].id == "Drive" {
							c.parts[i].kind = "pulse-source"
						}
					}
				}
				if err := buildElectrical(s, c); err != nil {
					return err
				}
				break
			}
		}
	}
	if err := s.ask("scope.probes.set", workbench.ProbeParams{IDs: performanceProbes(name)}, true, nil); err != nil {
		return err
	}
	if err := s.ask("simulation.resume", nil, true, nil); err != nil {
		return err
	}
	_, err := s.state("simulation.wait", nil, false)
	return err
}
func measurePerformance(s *session, name string, index int, condition string) (sample PerformanceSample, err error) {
	sample = PerformanceSample{Index: index, Condition: condition}
	before, err := s.state("project.inspect", nil, false)
	if err != nil {
		return sample, err
	}
	id, a, b := "Limit", "1k", "2k"
	switch name {
	case "rc":
		a, b = "10k", "20k"
	case "npn", "npn-pulse":
		id, a, b = "BaseR", "10k", "20k"
	case "flasher":
		id, a, b = "RB1", "47k", "48k"
	case "rl":
		id, a, b = "R1", "100 Ω", "120 Ω"
	case "rlc":
		id, a, b = "R1", "50 Ω", "60 Ω"
	case "flyback", "npn-coil":
		id, a, b = "RL", "100 Ω", "120 Ω"
	case "opamp-follower":
		id, a, b = "RL", "10k", "20k"
	case "opamp-inverting":
		id, a, b = "RF", "20k", "22k"
	case "opamp-rc-npn", "opamp-rc-mosfet":
		id, a, b = "RI", "10k", "12k"
	case "capacity-100-nodes":
		id, a, b = "P1", "10k", "20k"
	case "capacity-64-parts":
		id, a, b = "R1", "1k", "2k"
	case "capacity-rc-mosfet":
		id, a, b = "RL", "1k", "2k"
	}
	high, err := workbench.ParseValue(b, "Ω")
	if err != nil {
		return sample, err
	}
	value := b
	for _, part := range before.Components {
		if part.ID == id && part.Value == high {
			value = a
		}
	}
	started := time.Now()
	accepted, err := s.state("component.set", workbench.ValueParams{ID: id, Value: value}, true)
	if err != nil {
		return sample, err
	}
	sample.AcknowledgmentMS = float64(time.Since(started)) / float64(time.Millisecond)
	state := accepted
	if !performanceFresh(state, name) {
		state, err = s.state("simulation.wait", nil, false)
		if err != nil {
			return sample, err
		}
	}
	sample.FreshResultMS = float64(time.Since(started)) / float64(time.Millisecond)
	if !performanceFresh(state, name) || state.ElectricalRevision <= before.ElectricalRevision || state.RunID <= before.RunID {
		return sample, fmt.Errorf("edit did not produce a fresh changed run")
	}
	sample.RunID, sample.ElectricalRevision = state.RunID, state.ElectricalRevision
	sample.ProbeIDs = append([]string(nil), state.Scope.Selected...)
	sample.Settings = state.TransientSettings
	sample.Models = map[string]string{}
	for _, part := range state.Components {
		sample.Models[part.ID] = part.Model
	}
	if len(sample.ProbeIDs) != 4 {
		return sample, fmt.Errorf("performance fixture must retain four live probes")
	}
	switch name {
	case "led":
		sample.Engine = state.Result.Engine
	case "npn":
		sample.Engine = state.NPN.Engine
	default:
		sample.Engine = state.Transient.Engine
	}
	graph, e := json.Marshal(workbench.Graph{Components: state.Components, Wires: state.Wires})
	if e != nil {
		return sample, e
	}
	sample.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(graph))
	sample.graph = workbench.Graph{Components: state.Components, Wires: state.Wires}
	if err = s.ask("simulation.pause", nil, true, nil); err != nil {
		return sample, err
	}
	duration := 0.
	probes := performanceProbes(name)
	switch name {
	case "rc":
		duration = .2
	case "npn", "npn-pulse":
		if name == "npn-pulse" {
			duration = .2
		}
	case "flasher":
		duration = .2
	case "rl", "rlc", "flyback", "npn-coil":
		duration = .2
	case "opamp-follower", "opamp-inverting", "opamp-rc-npn", "opamp-rc-mosfet":
		duration = .2
	case "capacity-100-nodes", "capacity-64-parts", "capacity-rc-mosfet":
		duration = .2
	}
	step := .001
	meta, rows, elapsed, e := electricalVerify(s, duration, step, probes)
	if e != nil {
		return sample, e
	}
	sample.VerifyMS = elapsed * 1000
	sample.VerifyDuration = meta.Duration
	sample.VerifySampleStep = meta.Settings.StepSeconds
	if isCapacityPerformance(name) {
		sample.CapacityVerification = &workbench.VerificationDocument{Metadata: *meta, Rows: rows}
		if err := validateCapacityPerformanceSample(name, sample); err != nil {
			return sample, err
		}
	}
	if err = s.ask("recording.keep.set", workbench.RecordingKeepParams{ID: meta.OperationID, Keep: false}, true, nil); err != nil {
		return sample, err
	}
	if err = s.ask("simulation.resume", nil, true, nil); err != nil {
		return sample, err
	}
	return sample, nil
}
func performanceFresh(s workbench.Snapshot, name string) bool {
	if isCapacityPerformance(name) {
		return capacityPerformanceFresh(s, name)
	}
	if isOpAmpPerformance(name) {
		count := 1
		if name == "opamp-rc-mosfet" {
			count = 2
		}
		return s.Transient != nil && !s.Transient.Stale && s.Transient.RunID == s.RunID && s.Transient.ElectricalRevision == s.ElectricalRevision && s.Mixed != nil && !s.Mixed.Stale && s.Mixed.RunID == s.RunID && s.Mixed.ElectricalRevision == s.ElectricalRevision && len(s.Mixed.OpAmps) == count && s.OpAmpModel != nil && *s.OpAmpModel == workbench.OpAmpDeviceModel()
	}
	switch name {
	case "led":
		return s.Result != nil && !s.Result.Stale && s.Result.RunID == s.RunID && s.Result.ElectricalRevision == s.ElectricalRevision
	case "npn":
		return s.NPN != nil && !s.NPN.Stale && s.NPN.RunID == s.RunID && s.NPN.ElectricalRevision == s.ElectricalRevision
	case "flasher":
		return s.Transient != nil && !s.Transient.Stale && s.Transient.RunID == s.RunID && s.Transient.ElectricalRevision == s.ElectricalRevision && s.Mixed != nil && !s.Mixed.Stale
	case "rc", "npn-pulse":
		return s.Transient != nil && !s.Transient.Stale && s.Transient.RunID == s.RunID && s.Transient.ElectricalRevision == s.ElectricalRevision
	case "rl", "rlc", "flyback", "npn-coil":
		return s.Transient != nil && !s.Transient.Stale && s.Transient.Engine == workbench.InductorEngineID && s.Transient.RunID == s.RunID && s.Transient.ElectricalRevision == s.ElectricalRevision && s.Mixed != nil && !s.Mixed.Stale && s.Mixed.Engine == workbench.InductorEngineID && s.Mixed.RunID == s.RunID && s.Mixed.ElectricalRevision == s.ElectricalRevision
	}
	return false
}

func performanceProbes(name string) []string {
	if isCapacityPerformance(name) {
		return capacityProbes(name)
	}
	if isOpAmpPerformance(name) {
		// The numerical RC/NPN reference uses a fifth base-voltage boundary
		// probe. Timing keeps four probes in every family for comparability.
		return opAmpCaseProbes(name)[:4]
	}
	switch name {
	case "led":
		return []string{"v:Light.anode", "i:Light", "i:Limit", "v:Supply.positive"}
	case "rc":
		return []string{"v:Storage.1", "i:Limit", "i:Storage", "v:Supply.positive"}
	case "npn", "npn-pulse":
		return []string{"v:Transistor.collector", "i:Transistor.collector", "i:Transistor.base", "v:Drive.positive"}
	case "rl":
		return []string{"i:L1", "v:L1", "v:V1.positive", "i:R1"}
	case "rlc":
		return []string{"i:L1", "v:L1", "v:C1", "v:V1.positive"}
	case "flyback":
		return []string{"i:L1", "v:L1", "v:M1.drain", "i:D1"}
	case "npn-coil":
		return []string{"i:L1", "v:L1", "v:CB", "v:QCOIL.collector"}
	case "flasher":
		return []string{"i:D1", "i:D2", "v:C1", "v:C2"}
	}
	return nil
}

func isInductorPerformance(name string) bool {
	return name == "rl" || name == "rlc" || name == "flyback" || name == "npn-coil"
}

func isOpAmpPerformance(name string) bool {
	for _, known := range opAmpCaseNames {
		if name == known {
			return true
		}
	}
	return false
}
