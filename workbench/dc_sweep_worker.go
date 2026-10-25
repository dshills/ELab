package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const MaxDCSweepOperations = 8

// DCCircuit captures only model-bearing source data. It cannot recursively
// contain other analysis records, and is independent of project view/preferences.
type DCCircuit struct {
	Graph              Graph               `json:"graph"`
	Settings           TransientSettings   `json:"transient_settings"`
	LEDModel           LEDModel            `json:"led_model"`
	NPNModel           *NPNModel           `json:"npn_model,omitempty"`
	DiodeModel         *DiodeModel         `json:"diode_model,omitempty"`
	SchottkyModel      *DiodeModel         `json:"schottky_model,omitempty"`
	ZenerModel         *ZenerModel         `json:"zener_model,omitempty"`
	SwitchModel        *SwitchModel        `json:"switch_model,omitempty"`
	NMOSModel          *NMOSModel          `json:"nmos_model,omitempty"`
	PMOSModel          *PMOSModel          `json:"pmos_model,omitempty"`
	PNPModel           *PNPModel           `json:"pnp_model,omitempty"`
	OpAmpModel         *OpAmpModel         `json:"opamp_model,omitempty"`
	Timer555Model      *Timer555Model      `json:"timer555_model,omitempty"`
	DeviceModel        *ExtendedModel      `json:"device_model,omitempty"`
	HBridgeModel       *HBridgeModel       `json:"hbridge_model,omitempty"`
	LogicModel         *LogicModel         `json:"logic_model,omitempty"`
	SequentialModel    *SequentialModel    `json:"sequential_model,omitempty"`
	DisplayModel       *DisplayModel       `json:"display_model,omitempty"`
	RegulatorModel     *RegulatorModel     `json:"regulator_model,omitempty"`
	SensorModel        *SensorModel        `json:"sensor_model,omitempty"`
	PowerModel         *PowerModel         `json:"power_model,omitempty"`
	ControlModel       *ControlModel       `json:"control_model,omitempty"`
	PotentiometerModel *PotentiometerModel `json:"potentiometer_model,omitempty"`
	WaveformModel      *WaveformModel      `json:"waveform_model,omitempty"`
}

func dcCircuitSnapshot(g Graph, cfg TransientSettings) DCCircuit {
	out := DCCircuit{Graph: g.Clone(), Settings: cfg, LEDModel: Model(), PNPModel: pnpModelForGraph(g), OpAmpModel: opAmpModelForGraph(g), SequentialModel: sequentialModelForGraph(g), DisplayModel: displayModelForGraph(g), RegulatorModel: regulatorModelForGraph(g), PowerModel: powerModelForGraph(g), SensorModel: sensorModelForGraph(g), ControlModel: controlModelForGraph(g), DeviceModel: extendedModelForGraph(g), HBridgeModel: hbridgeModelForGraph(g), LogicModel: logicModelForGraph(g), Timer555Model: timer555ModelForGraph(g)}
	for _, p := range g.Components {
		switch p.Kind {
		case "pmos":
			m := PMOSDeviceModel()
			out.PMOSModel = &m
		case "nmos":
			m := NMOSDeviceModel()
			out.NMOSModel = &m
		case "npn":
			m := TransistorModel()
			out.NPNModel = &m
		case "zener":
			m := ZenerDeviceModel()
			out.ZenerModel = &m
		case "schottky":
			m := SchottkyDeviceModel()
			out.SchottkyModel = &m
		case "diode":
			m := SiliconDiodeModel()
			out.DiodeModel = &m
		case "switch":
			m := SwitchDeviceModel()
			out.SwitchModel = &m
		case "potentiometer":
			m := LinearPotentiometerModel()
			out.PotentiometerModel = &m
		case "sine-source", "finite-pulse-source":
			m := SourceWaveformModel()
			out.WaveformModel = &m
		}
	}
	return out
}
func (c DCCircuit) document() Document {
	paused := true
	cfg := c.Settings
	return Document{Format: DocumentFormat, Version: DocumentVersion, Graph: c.Graph.Clone(), View: View{Zoom: 1}, Paused: &paused, ScopeProbes: &ProbeParams{IDs: []string{}}, TransientSettings: &cfg, LEDModel: c.LEDModel, NPNModel: c.NPNModel, DiodeModel: c.DiodeModel, SchottkyModel: c.SchottkyModel, ZenerModel: c.ZenerModel, SwitchModel: c.SwitchModel, NMOSModel: c.NMOSModel, PMOSModel: c.PMOSModel, PNPModel: c.PNPModel, OpAmpModel: c.OpAmpModel, SequentialModel: c.SequentialModel, DisplayModel: c.DisplayModel, RegulatorModel: c.RegulatorModel, PowerModel: c.PowerModel, SensorModel: c.SensorModel, ControlModel: c.ControlModel, DeviceModel: c.DeviceModel, HBridgeModel: c.HBridgeModel, LogicModel: c.LogicModel, Timer555Model: c.Timer555Model, PotentiometerModel: c.PotentiometerModel, WaveformModel: c.WaveformModel, Recordings: &ProjectRecordings{Entries: []ProjectRecording{}, Selected: -1}}
}

type SweepMetadata struct {
	Format             string        `json:"format"`
	Version            int           `json:"version"`
	Engine             string        `json:"engine"`
	OperationID        string        `json:"operation_id"`
	CreatedAt          string        `json:"created_at"`
	DocumentGeneration uint64        `json:"document_generation"`
	ElectricalRevision uint64        `json:"electrical_revision"`
	ActuationRevision  uint64        `json:"actuation_revision"`
	Source             DCCircuit     `json:"source"`
	Settings           SweepSettings `json:"settings"`
	Sources            []SweepSource `json:"dc_sources"`
	Assumptions        []string      `json:"assumptions"`
}
type SweepRecord struct {
	Metadata SweepMetadata `json:"metadata"`
	Result   SweepResult   `json:"result"`
}
type SweepSummary struct {
	Kind            string   `json:"kind"`
	SourcePath      string   `json:"source_path,omitempty"`
	SourceSHA256    string   `json:"source_sha256,omitempty"`
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Actor           string   `json:"actor"`
	Completed       int      `json:"completed_points"`
	Total           int      `json:"total_points"`
	SourceUnchanged bool     `json:"source_unchanged"`
	Kept            bool     `json:"kept_in_project"`
	Error           *Failure `json:"error,omitempty"`
}
type SweepInspection struct {
	Operation SweepSummary  `json:"operation"`
	Metadata  SweepMetadata `json:"metadata"`
	Result    *SweepResult  `json:"result,omitempty"`
}
type SweepState struct {
	Settings   SweepSettings  `json:"settings"`
	Targets    []SweepTarget  `json:"targets"`
	Available  []Probe        `json:"available_probes"`
	Operations []SweepSummary `json:"operations"`
	Selected   string         `json:"selected_operation"`
	Visible    bool           `json:"visible"`
	TraceID    string         `json:"trace_id"`
	Cursor     *int           `json:"cursor_index,omitempty"`
	Result     *SweepResult   `json:"result,omitempty"`
}
type SweepStartParams struct {
	Settings *SweepSettings `json:"settings,omitempty"`
}
type SweepVisibilityParams struct {
	Visible bool `json:"visible"`
}
type SweepCursorParams struct {
	OperationID string `json:"operation_id,omitempty"`
	Index       *int   `json:"index"`
}
type sweepJob struct {
	summary    SweepSummary
	metadata   SweepMetadata
	sourceHash [32]byte
	ctx        context.Context
	cancel     context.CancelFunc
	record     *SweepRecord
	keep       bool
}
type sweepStore struct {
	settings          *SweepSettings
	operations        map[string]*sweepJob
	order             []string
	selected, traceID string
	visible           bool
	cursor            *int
}

func cloneSweepSettings(in SweepSettings) SweepSettings {
	out := in
	out.ProbeIDs = append([]string{}, in.ProbeIDs...)
	if in.ProbeIDs == nil {
		out.ProbeIDs = nil
	}
	return out
}
func cloneSweepMetadata(in SweepMetadata) SweepMetadata {
	// Captured documents contain detached typed data; this also owns optional
	// model/preference structures without exposing job-held pointers to callers.
	data, _ := json.Marshal(in)
	var out SweepMetadata
	_ = json.Unmarshal(data, &out)
	return out
}
func (s *Service) ensureSweepsLocked() *sweepStore {
	if s.sweeps == nil {
		s.sweeps = &sweepStore{operations: map[string]*sweepJob{}}
	}
	return s.sweeps
}
func (s *Service) sweepSettingsLocked() SweepSettings {
	if s.sweeps != nil && s.sweeps.settings != nil {
		return cloneSweepSettings(*s.sweeps.settings)
	}
	if s.exampleID == "ac-rc" || s.exampleID == "ac-rlc" || s.exampleID == "ac-opamp" {
		source, probe := "V1", "v:C1"
		if s.exampleID == "ac-opamp" {
			source, probe = "VIN", "v:U1"
		}
		return SweepSettings{ComponentID: source, Parameter: "frequency", Spacing: "logarithmic", Start: "1 Hz", Stop: "100 kHz", Points: 201, ProbeIDs: []string{probe}}
	}
	cfg := SweepSettings{Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 51}
	targets := sweepTargets(s.graph)
	if len(targets) > 0 {
		cfg.ComponentID = targets[0].ComponentID
		cfg.Parameter = targets[0].Parameter
		switch cfg.Parameter {
		case "resistance":
			cfg.Start, cfg.Stop = "100 Ω", "10k Ω"
		case "wiper":
			cfg.Start, cfg.Stop = "0", "1"
		}
	}
	return cfg
}
func sweepGraphHash(g Graph) [32]byte { data, _ := json.Marshal(g); return sha256.Sum256(data) }
func (s *Service) sweepSummaryLocked(job *sweepJob) SweepSummary {
	out := job.summary
	out.Kept = job.keep
	out.SourceUnchanged = (job.summary.Kind == "dc-sweep" || job.summary.Kind == "ac-sweep") && job.metadata.DocumentGeneration == s.documentGeneration && job.sourceHash == sweepGraphHash(s.graph)
	if out.Error != nil {
		e := *out.Error
		out.Error = &e
	}
	return out
}
func (s *Service) sweepStateLocked() *SweepState {
	targets := sweepTargets(s.graph)
	if len(targets) == 0 && s.sweeps == nil {
		return nil
	}
	state := &SweepState{Settings: s.sweepSettingsLocked(), Targets: targets, Available: probeInventory(s.graph), Operations: []SweepSummary{}}
	if s.sweeps != nil {
		store := s.sweeps
		state.Selected, state.Visible, state.TraceID = store.selected, store.visible, store.traceID
		if store.cursor != nil {
			v := *store.cursor
			state.Cursor = &v
		}
		for _, id := range store.order {
			state.Operations = append(state.Operations, s.sweepSummaryLocked(store.operations[id]))
		}
		if job := store.operations[store.selected]; job != nil && job.record != nil {
			state.Result = cloneSweepResult(&job.record.Result)
		}
	}
	return state
}
func cloneSweepState(in *SweepState) *SweepState {
	if in == nil {
		return nil
	}
	out := *in
	out.Settings = cloneSweepSettings(in.Settings)
	out.Targets = append([]SweepTarget{}, in.Targets...)
	out.Available = append([]Probe{}, in.Available...)
	out.Operations = append([]SweepSummary{}, in.Operations...)
	for i := range out.Operations {
		if e := out.Operations[i].Error; e != nil {
			f := *e
			out.Operations[i].Error = &f
		}
	}
	if in.Cursor != nil {
		v := *in.Cursor
		out.Cursor = &v
	}
	out.Result = cloneSweepResult(in.Result)
	return &out
}
func (s *Service) startDCSweepLocked(p SweepStartParams) error {
	if s.sweepWorkers != 0 {
		return &actionError{"busy", "Finish or cancel the active sweep before starting another"}
	}
	cfg := s.sweepSettingsLocked()
	if p.Settings != nil {
		cfg = cloneSweepSettings(*p.Settings)
	}
	plan, err := newSweepPlan(s.graph, cfg)
	if err != nil {
		return err
	}
	if err := s.requireSweepSlotLocked(); err != nil {
		return err
	}
	store := s.ensureSweepsLocked()

	s.sweepSerial++
	id := fmt.Sprintf("sweep-%d", s.sweepSerial)
	source := dcCircuitSnapshot(s.graph, s.transientSettings)
	// This capture is a source document, not a recursive collection of other runs.
	metadata := SweepMetadata{Format: "elab.dc-sweep", Version: 1, Engine: DCSweepEngineID, OperationID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), DocumentGeneration: s.documentGeneration, ElectricalRevision: s.electrical, ActuationRevision: s.actuationRevision, Source: source, Settings: cloneSweepSettings(plan.settings), Sources: append([]SweepSource{}, plan.sources...), Assumptions: []string{"Independent DC operating points; no time advancement or hysteresis continuation", "Capacitors open; startup voltages ignored", "Sine sources use offset; finite pulses use low level; ideal pulses use zero; swept source uses the requested DC bias", "Configured switch states and accepted wiper positions; live held inputs and pending actuation are not applied", "Existing generic device assumptions and limits remain in force"}}
	metadata.Format, metadata.Engine = acFormat(cfg.Parameter)
	if cfg.Parameter == "frequency" {
		metadata.Assumptions = acAssumptions()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &sweepJob{summary: SweepSummary{ID: id, Kind: "dc-sweep", Status: "queued", Total: cfg.Points}, metadata: metadata, sourceHash: sweepGraphHash(s.graph), ctx: ctx, cancel: cancel}
	s.retainSweepJobLocked(job)
	s.sweepViewerRevision++
	traceID := plan.probes[0].ID
	if cfg.Parameter == "frequency" {
		traceID = "gain:" + traceID
		job.summary.Kind = "ac-sweep"
	}
	store.selected, store.visible, store.traceID, store.cursor = id, true, traceID, nil
	s.dismissPrintPreviewLocked()
	s.sweepWorkers++
	s.liveWorkers.Add(1)
	compute := s.sweepCompute
	go s.runDCSweep(job, plan, compute)
	return nil
}
func (s *Service) dismissPrintPreviewLocked() {
	if s.preview != nil && s.preview.view.Visible {
		_ = s.cancelPrintPreviewLocked(IDParams{ID: s.preview.view.ID}, true)
	}
}
func (s *Service) runDCSweep(job *sweepJob, plan sweepPlan, compute func(context.Context, sweepPlan, func(int)) (*SweepResult, error)) {
	defer s.liveWorkers.Done()
	defer job.cancel()
	s.mu.Lock()
	if job.ctx.Err() == nil {
		job.summary.Status = "running"
		s.revision++
		s.signalLocked("sweeps", "jobs")
	}
	s.mu.Unlock()
	last := time.Now()
	result, err := compute(job.ctx, plan, func(done int) {
		if done != plan.settings.Points && time.Since(last) < 100*time.Millisecond {
			return
		}
		last = time.Now()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || job.ctx.Err() != nil || s.sweeps == nil || s.sweeps.operations[job.summary.ID] != job {
			return
		}
		job.summary.Completed = done
		s.revision++
		s.signalLocked("sweeps", "jobs")
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepWorkers--
	if s.sweeps == nil || s.sweeps.operations[job.summary.ID] != job {
		return
	}
	if job.ctx.Err() != nil {
		err = job.ctx.Err()
	}
	switch {
	case s.closed || errors.Is(err, context.Canceled):
		job.summary.Status = "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		job.summary.Status = "failed"
		job.summary.Error = &Failure{Code: "timeout", Message: "Sweep exceeded its cooperative 30-second budget"}
	case err != nil:
		job.summary.Status = "failed"
		job.summary.Error = &Failure{Code: "sweep-failed", Message: err.Error()}
	default:
		job.summary.Status = "completed"
		job.summary.Completed = job.summary.Total
		job.record = &SweepRecord{Metadata: job.metadata, Result: *result}
		job.keep = true
	}
	s.revision++
	s.signalLocked("sweeps", "jobs", "recordings")
}
func (s *Service) inspectSweepLocked(id string) (SweepInspection, error) {
	if s.sweeps == nil || s.sweeps.operations[id] == nil {
		return SweepInspection{}, &actionError{"not-found", "Sweep is unknown or was evicted"}
	}
	job := s.sweeps.operations[id]
	out := SweepInspection{Operation: s.sweepSummaryLocked(job), Metadata: cloneSweepMetadata(job.metadata)}
	if job.record != nil {
		out.Result = cloneSweepResult(&job.record.Result)
	}
	return out, nil
}
func (s *Service) cancelSweepLocked(id string) error {
	if s.sweeps == nil || s.sweeps.operations[id] == nil {
		return &actionError{"not-found", "Sweep is unknown or was evicted"}
	}
	job := s.sweeps.operations[id]
	if !operationActive(job.summary.Status) {
		return &actionError{"unavailable", "Only an active sweep can be canceled"}
	}
	job.summary.Status = "canceling"
	job.cancel()
	return nil
}
func (s *Service) clearSweepsLocked() {
	s.sweepViewerRevision++
	if s.sweeps != nil {
		for _, job := range s.sweeps.operations {
			if operationActive(job.summary.Status) {
				job.cancel()
			}
		}
	}
	s.sweeps = nil
}

func (s *Service) requireSweepSlotLocked() error {
	if s.sweeps == nil || len(s.sweeps.order) < MaxDCSweepOperations {
		return nil
	}
	for _, id := range s.sweeps.order {
		job := s.sweeps.operations[id]
		if !job.keep && !operationActive(job.summary.Status) {
			return nil
		}
	}
	return &actionError{"record-limit", "Eight sweeps are kept; exclude one before creating or importing another"}
}

// Admission/publication check capacity first. Eviction never removes kept or
// active work; an expired selection cannot keep pointing at a removed asset.
func (s *Service) retainSweepJobLocked(job *sweepJob) {
	store := s.ensureSweepsLocked()
	if len(store.order) >= MaxDCSweepOperations {
		for i, id := range store.order {
			old := store.operations[id]
			if old.keep || operationActive(old.summary.Status) {
				continue
			}
			delete(store.operations, id)
			store.order = append(store.order[:i], store.order[i+1:]...)
			if store.selected == id {
				store.selected, store.traceID, store.visible, store.cursor = "", "", false, nil
			}
			break
		}
	}
	store.operations[job.summary.ID] = job
	store.order = append(store.order, job.summary.ID)
}
