package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const MaxVerificationSteps = 1000
const MaxVerificationFrames = 4096
const MaxVerificationOperations = 8

// verificationOutputSteps counts the actual binary64 output grid, whose last
// time is min(duration, float64(n)*step). Division can round an exact endpoint
// above an integer (1 ms / 1 µs, for example). Correct its estimate using the
// same multiplication as the worker, without merging any distinct final time.
// Callers validate positive finite duration/step and the bounded ratio first.
func verificationOutputSteps(duration, step float64) int {
	steps := max(1, int(math.Ceil(duration/step)))
	for steps > 1 && float64(steps-1)*step >= duration {
		steps--
	}
	for float64(steps)*step < duration {
		steps++
	}
	return steps
}

func validateVerificationSettings(settings VerificationSettings, g Graph) error {
	if !finite(settings.Duration) || settings.Duration < 0 || settings.Duration > 10 {
		return fmt.Errorf("verify duration must be 0–10 seconds; DC uses zero")
	}
	if !finite(settings.Step) || settings.Step < 0 || settings.Step > 1 || settings.Step != 0 && settings.Step < 1e-6 {
		return fmt.Errorf("verify sample step must be 0 (project default) or 1 µs–1 s")
	}
	switch settings.InputMode {
	case "auto", "high", "low", "charge", "discharge":
	default:
		return fmt.Errorf("unknown verify input mode")
	}
	if settings.InitialVoltage != nil && (!finite(*settings.InitialVoltage) || *settings.InitialVoltage < 0 || *settings.InitialVoltage > 12) {
		return fmt.Errorf("default verify initial voltage must be 0–12 V")
	}
	seen := map[string]bool{}
	for _, initial := range settings.Inductors {
		p, ok := g.Component(initial.ID)
		if !ok || p.Kind != "inductor" || seen[initial.ID] || !finite(initial.Current) || math.Abs(initial.Current) > MaxInductorInitialCurrent {
			return fmt.Errorf("invalid or duplicate inductor initial condition %q", initial.ID)
		}
		seen[initial.ID] = true
	}
	for _, initial := range settings.Capacitors {
		p, ok := g.Component(initial.ID)
		if !ok || p.Kind != "capacitor" || seen[initial.ID] || !finite(initial.Voltage) || math.Abs(initial.Voltage) > 12 {
			return fmt.Errorf("invalid or duplicate capacitor initial condition %q", initial.ID)
		}
		seen[initial.ID] = true
	}
	return nil
}
func (s *Service) startVerificationLocked(p VerifyParams) error {
	for _, id := range s.operationOrder {
		if operationActive(s.operations[id].summary.Status) {
			return &actionError{"busy", "A verification is already active; inspect or cancel it first"}
		}
	}
	if err := s.requireRecordingSlotLocked(); err != nil {
		return err
	}
	settings := cloneVerificationSettings(s.verificationSettings)
	if p.Duration != nil {
		settings.Duration = *p.Duration
	}
	if p.Step != nil {
		settings.Step = *p.Step
	}
	if p.InputMode != "" {
		settings.InputMode = p.InputMode
	}
	if p.InitialVoltage != nil {
		settings.InitialVoltage = p.InitialVoltage
	}
	if p.Capacitors != nil {
		settings.Capacitors = p.Capacitors
	}
	if p.Inductors != nil {
		settings.Inductors = p.Inductors
	}
	graph := s.graph.Clone()
	if err := validateVerificationSettings(settings, graph); err != nil {
		return err
	}
	cfg := s.transientSettings
	if settings.Step != 0 {
		cfg.StepSeconds = settings.Step
	}
	if settings.InitialVoltage != nil {
		cfg.InitialVoltage = *settings.InitialVoltage
	}
	for _, initial := range settings.Capacitors {
		for i := range graph.Components {
			if graph.Components[i].ID == initial.ID {
				v := initial.Voltage
				graph.Components[i].InitialVoltage = &v
			}
		}
	}
	for _, initial := range settings.Inductors {
		for k := range graph.Components {
			if graph.Components[k].ID == initial.ID {
				v := initial.Current
				graph.Components[k].InitialCurrent = &v
			}
		}
	}
	compiled, _, diagnostic := graph.Compile()
	if diagnostic != nil {
		return &actionError{"unavailable", "Complete a supported circuit before Verify: " + diagnostic.Message}
	}
	transient := IsTransientAnalysis(compiled.Analysis)
	if transient {
		if settings.Duration <= 0 {
			return fmt.Errorf("transient Verify requires a positive duration")
		}
		if err := cfg.validateForGraph(graph, false); err != nil {
			return err
		}
	} else {
		if p.Duration == nil {
			settings.Duration = 0
		}
		if settings.Duration != 0 {
			return fmt.Errorf("DC Verify is an operating point: duration_s must be zero")
		}
		if settings.InitialVoltage != nil || len(settings.Capacitors) > 0 || len(settings.Inductors) > 0 {
			return fmt.Errorf("initial capacitor/inductor conditions require a transient circuit")
		}
	}
	if !verificationInputModeValid(compiled.Analysis, settings.InputMode) {
		return fmt.Errorf("verify input mode does not match the circuit analysis")
	}
	steps := 1
	if transient {
		steps = verificationOutputSteps(settings.Duration, cfg.StepSeconds)
		if steps > MaxVerificationSteps {
			return fmt.Errorf("verify supports at most %d output steps; shorten duration or increase its sample step", MaxVerificationSteps)
		}
	}
	ids := p.ProbeIDs
	if ids == nil {
		ids = s.scopeLocked(compiled).Selected
		if len(ids) == 0 && !transient {
			if IsNetworkAnalysis(compiled.Analysis) {
				ids = defaultProbes(compiled)
			} else if compiled.Analysis == NPNAnalysis {
				ids = []string{"i:" + compiled.Transistor.ID + ".collector", "v:" + compiled.Transistor.ID + ".collector", "i:" + compiled.Transistor.ID + ".base", "v:" + compiled.Transistor.ID + ".base"}
			} else {
				ids = []string{"i:" + compiled.LED.ID, "v:" + compiled.LED.ID, "v:" + compiled.Resistor.ID, "v:" + compiled.Source.ID}
			}
		}
	}
	if len(ids) > MaxScopeProbes {
		return fmt.Errorf("verify supports at most six probes")
	}
	available := map[string]Probe{}
	for _, probe := range probeInventory(graph) {
		available[probe.ID] = probe
	}
	probes := []Probe{}
	seen := map[string]bool{}
	for _, id := range ids {
		probe, ok := available[id]
		if !ok || seen[id] {
			return fmt.Errorf("unknown or duplicate verify probe %q", id)
		}
		seen[id] = true
		probes = append(probes, probe)
	}
	s.operationSerial++
	id := fmt.Sprintf("verify-%d", s.operationSerial)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	operation := &verificationOperation{sourceActuationRevision: s.actuationRevision, done: make(chan struct{}), ctx: ctx, cancel: cancel, summary: OperationSummary{ID: id, Kind: "verify", Status: "queued", InputRevision: s.revision, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration, RequestedDuration: settings.Duration, TotalSteps: steps}}
	s.retainOperationLocked(operation)
	s.selectRecordingLocked(id)
	s.lastOperation = id
	metadata := VerificationMetadata{ActuationRevision: s.actuationRevision, Format: "elab.verification", Version: 1, OperationID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), InputRevision: s.revision, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration, Graph: graph.Clone(), Settings: cfg, InputMode: settings.InputMode, Analysis: compiled.Analysis, LEDModel: Model(), Probes: probes, Duration: settings.Duration, Statistics: []ScopeStatistics{}}
	if hasBipolarPolarity(graph.Components, "npn") {
		model := TransistorModel()
		metadata.NPNModel = &model
	}
	for _, part := range graph.Components {
		if IsWaveformSource(part.Kind) {
			m := SourceWaveformModel()
			metadata.WaveformModel = &m
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "potentiometer" {
			model := LinearPotentiometerModel()
			metadata.PotentiometerModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "zener" {
			m := ZenerDeviceModel()
			metadata.ZenerModel = &m
		}
		if part.Kind == "schottky" {
			m := SchottkyDeviceModel()
			metadata.SchottkyModel = &m
		}
		if part.Kind == "diode" {
			model := SiliconDiodeModel()
			metadata.DiodeModel = &model
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "switch" {
			model := SwitchDeviceModel()
			metadata.SwitchModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "nmos" {
			model := NMOSDeviceModel()
			metadata.NMOSModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "pmos" {
			model := PMOSDeviceModel()
			metadata.PMOSModel = &model
			break
		}
	}
	metadata.PNPModel = pnpModelForGraph(graph)
	metadata.OpAmpModel = opAmpModelForGraph(graph)
	metadata.Timer555Model = timer555ModelForGraph(graph)
	metadata.DeviceModel = extendedModelForGraph(graph)
	metadata.HBridgeModel = hbridgeModelForGraph(graph)
	metadata.LogicModel = logicModelForGraph(graph)
	metadata.SequentialModel = sequentialModelForGraph(graph)
	metadata.DisplayModel = displayModelForGraph(graph)
	metadata.RegulatorModel = regulatorModelForGraph(graph)
	metadata.SensorModel = sensorModelForGraph(graph)
	metadata.PowerModel = powerModelForGraph(graph)
	metadata.ControlModel = controlModelForGraph(graph)
	metadata.Engine = verificationEngineID(compiled.Analysis)
	operation.inputMetadata = cloneVerificationMetadata(metadata)
	input := verificationInput{compiled: compiled, settings: cfg, mode: settings.InputMode, probes: probes, duration: settings.Duration, steps: steps, serial: s.operationSerial, metadata: metadata}
	s.liveWorkers.Add(1)
	go func() { defer s.liveWorkers.Done(); s.runVerification(operation, input) }()
	return nil
}

// VerificationInputModes returns an owned list for the current analysis. Stored
// defaults may be configured earlier, but admission requires a compatible mode.
func VerificationInputModes(analysis string) []string {
	modes := []string{"auto"}
	if analysis == "RC transient" {
		return append(modes, "charge", "discharge")
	}
	if IsNPNTransientAnalysis(analysis) {
		return append(modes, "high", "low")
	}
	return modes
}
func verificationInputModeValid(analysis, mode string) bool {
	for _, allowed := range VerificationInputModes(analysis) {
		if mode == allowed {
			return true
		}
	}
	return false
}

func operationActive(status string) bool {
	return status == "queued" || status == "running" || status == "canceling"
}
func (s *Service) operationSummaryLocked(operation *verificationOperation) OperationSummary {
	out := operation.summary
	out.KeptInProject = operation.keep
	out.SourceUnchanged = out.Kind == "verify" && out.ElectricalRevision == s.electrical && out.DocumentGeneration == s.documentGeneration && operation.sourceActuationRevision == s.actuationRevision
	if out.Error != nil {
		failure := *out.Error
		out.Error = &failure
	}
	out.FailureHelp = verificationFailureHelp(out)
	return out
}
func (s *Service) cancelOperationLocked(id string) error {
	operation, ok := s.operations[id]
	if !ok {
		return &actionError{"not-found", "Operation not found or no longer retained"}
	}
	if !operationActive(operation.summary.Status) {
		return &actionError{"unavailable", "Operation already finished"}
	}
	operation.summary.Status = "canceling"
	operation.cancel()
	return nil
}
func (s *Service) verificationQueryLocked(command string, raw json.RawMessage) (any, error) {
	if command == "operation.inspect" {
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return nil, err
		}
		operation, ok := s.operations[p.ID]
		if !ok {
			return nil, &actionError{"not-found", "Operation not found or no longer retained"}
		}
		captured := cloneVerificationMetadata(operation.inputMetadata)
		view := OperationView{Operation: s.operationSummaryLocked(operation), Metadata: &captured}
		if operation.record != nil {
			metadata := cloneVerificationMetadata(operation.record.metadata)
			view.Metadata = &metadata
		}
		return view, nil
	}
	var p VerificationPageParams
	if err := params(raw, &p, "id"); err != nil {
		return nil, err
	}
	operation, ok := s.operations[p.ID]
	if !ok {
		return nil, &actionError{"not-found", "Operation not found or no longer retained"}
	}
	if operation.record == nil {
		return nil, &actionError{"not-ready", "Only completed verification has recorded samples"}
	}
	return recordedSamplePage(p.ID, operation.record.metadata.Probes, operation.record.rows, p.Offset, p.Limit)
}
func recordedSamplePage(id string, probes []Probe, rows []VerificationRow, offset, limit int) (VerificationPage, error) {
	if limit == 0 {
		limit = 100
	}
	if offset < 0 || offset > len(rows) || limit < 1 || limit > 200 {
		return VerificationPage{}, fmt.Errorf("sample pages require offset within the record and limit 1–200")
	}
	end := min(len(rows), offset+limit)
	page := VerificationPage{ID: id, Offset: offset, NextOffset: end, Total: len(rows), Probes: append([]Probe{}, probes...), Rows: []VerificationRow{}}
	for _, row := range rows[offset:end] {
		page.Rows = append(page.Rows, VerificationRow{Time: row.Time, Values: append([]float64{}, row.Values...)})
	}
	return page, nil
}
func cloneVerificationMetadata(in VerificationMetadata) VerificationMetadata {
	out := in
	out.Graph = in.Graph.Clone()
	out.Probes = append([]Probe{}, in.Probes...)
	out.Statistics = append([]ScopeStatistics{}, in.Statistics...)
	if in.WaveformModel != nil {
		m := *in.WaveformModel
		out.WaveformModel = &m
	}
	if in.PotentiometerModel != nil {
		model := *in.PotentiometerModel
		out.PotentiometerModel = &model
	}
	if in.SchottkyModel != nil {
		m := *in.SchottkyModel
		out.SchottkyModel = &m
	}
	if in.ZenerModel != nil {
		m := *in.ZenerModel
		out.ZenerModel = &m
	}
	if in.DiodeModel != nil {
		model := *in.DiodeModel
		out.DiodeModel = &model
	}
	if in.NMOSModel != nil {
		model := *in.NMOSModel
		out.NMOSModel = &model
	}
	if in.Timer555Model != nil {
		m := *in.Timer555Model
		out.Timer555Model = &m
	}
	if in.DeviceModel != nil {
		m := *in.DeviceModel
		out.DeviceModel = &m
	}
	if in.HBridgeModel != nil {
		m := *in.HBridgeModel
		out.HBridgeModel = &m
	}
	if in.LogicModel != nil {
		m := *in.LogicModel
		out.LogicModel = &m
	}
	if in.SequentialModel != nil {
		m := *in.SequentialModel
		out.SequentialModel = &m
	}
	if in.DisplayModel != nil {
		m := *in.DisplayModel
		out.DisplayModel = &m
	}
	if in.RegulatorModel != nil {
		m := *in.RegulatorModel
		out.RegulatorModel = &m
	}
	if in.PowerModel != nil {
		m := *in.PowerModel
		out.PowerModel = &m
	}
	if in.SensorModel != nil {
		m := *in.SensorModel
		out.SensorModel = &m
	}
	if in.ControlModel != nil {
		m := *in.ControlModel
		out.ControlModel = &m
	}
	if in.OpAmpModel != nil {
		model := *in.OpAmpModel
		out.OpAmpModel = &model
	}
	if in.PNPModel != nil {
		model := *in.PNPModel
		out.PNPModel = &model
	}
	if in.PMOSModel != nil {
		model := *in.PMOSModel
		out.PMOSModel = &model
	}
	if in.SwitchModel != nil {
		model := *in.SwitchModel
		out.SwitchModel = &model
	}
	out.NPN = cloneNPN(in.NPN)
	out.Mixed = cloneMixed(in.Mixed)
	if in.NPNModel != nil {
		model := *in.NPNModel
		out.NPNModel = &model
	}
	if in.LED != nil {
		r := *in.LED
		r.Nodes = map[string]float64{}
		for k, v := range in.LED.Nodes {
			r.Nodes[k] = v
		}
		out.LED = &r
	}
	return out
}

func validateCapacitorInitialFields(raw json.RawMessage) error {
	var object map[string]json.RawMessage
	_ = json.Unmarshal(raw, &object)
	field, exists := object["capacitor_initials"]
	if !exists {
		return nil
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(field, &rows); err != nil {
		return err
	}
	if rows == nil {
		return fmt.Errorf("capacitor_initials must be an array")
	}
	for _, row := range rows {
		if err := params(row, &CapacitorInitial{}, "id", "voltage_v"); err != nil {
			return err
		}
	}
	return nil
}

func verificationEngineID(analysis string) string {
	switch analysis {
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis, InductorAnalysis, Timer555Analysis, SequentialAnalysis, ControlAnalysis:
		return networkEngine(analysis)
	case "RC transient":
		return RCEngineID
	case NPNTransientAnalysis:
		return NPNTransientEngineID
	case NPNAnalysis:
		return NPNEngineID
	default:
		return EngineID
	}
}
