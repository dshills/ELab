package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dshills/elab/internal/audioplay"

	"github.com/dshills/elab/internal/printspool"
)

type Snapshot struct {
	Audio                  *AudioPreview          `json:"audio_preview,omitempty"`
	InspectorSection       string                 `json:"inspector_section"`
	WorkspaceLayout        WorkspaceLayout        `json:"workspace_layout"`
	Viewport               ViewportState          `json:"viewport"`
	PulseTimings           []PulseTiming          `json:"pulse_timings"`
	Shutdown               *ShutdownWork          `json:"shutdown,omitempty"`
	Printing               *PrintState            `json:"printing,omitempty"`
	VerificationInputModes []string               `json:"verification_input_modes"`
	LiveInputModes         []string               `json:"live_input_modes"`
	DCSweep                *SweepState            `json:"dc_sweep,omitempty"`
	PrintPreview           *PrintPreview          `json:"print_preview,omitempty"`
	SourceAnnotations      []SourceAnnotation     `json:"source_annotations"`
	ProjectWork            *ProjectWork           `json:"project_work,omitempty"`
	InputWork              *InputWork             `json:"input_work,omitempty"`
	Example                *ExampleContext        `json:"example,omitempty"`
	DiagnosticPresentation DiagnosticPresentation `json:"diagnostic_presentation"`
	Display                CircuitDisplay         `json:"display"`
	MeasurementInspection  MeasurementInspection  `json:"measurement_inspection"`
	PinnedReadings         []PinnedReading        `json:"pinned_readings"`
	TraceFocus             *TraceFocus            `json:"trace_focus,omitempty"`
	Limits                 CircuitLimits          `json:"limits"`
	SimulatedComponents    int                    `json:"simulated_component_count"`
	StepWork               *StepWork              `json:"step_work,omitempty"`
	UndoEntry              *HistoryTip            `json:"undo_entry,omitempty"`
	RedoEntry              *HistoryTip            `json:"redo_entry,omitempty"`
	ActuationRevision      uint64                 `json:"actuation_revision"`
	SwitchStates           []SwitchState          `json:"switch_states,omitempty"`
	SourceSampling         []SourceSamplingAdvice `json:"source_sampling,omitempty"`
	SourceWarnings         []Diagnostic           `json:"source_warnings,omitempty"`
	LCSampling             *LCSamplingAdvice      `json:"lc_sampling,omitempty"`
	WaveformModel          *WaveformModel         `json:"waveform_model,omitempty"`
	PotentiometerModel     *PotentiometerModel    `json:"potentiometer_model,omitempty"`
	DiodeModel             *DiodeModel            `json:"diode_model,omitempty"`
	SchottkyModel          *DiodeModel            `json:"schottky_model,omitempty"`
	ZenerModel             *ZenerModel            `json:"zener_model,omitempty"`
	SwitchModel            *SwitchModel           `json:"switch_model,omitempty"`
	NMOSModel              *NMOSModel             `json:"nmos_model,omitempty"`
	PMOSModel              *PMOSModel             `json:"pmos_model,omitempty"`
	PNPModel               *PNPModel              `json:"pnp_model,omitempty"`
	OpAmpModel             *OpAmpModel            `json:"opamp_model,omitempty"`
	Timer555Model          *Timer555Model         `json:"timer555_model,omitempty"`
	DeviceModel            *ExtendedModel         `json:"device_model,omitempty"`
	HBridgeModel           *HBridgeModel          `json:"hbridge_model,omitempty"`
	LogicModel             *LogicModel            `json:"logic_model,omitempty"`
	SequentialModel        *SequentialModel       `json:"sequential_model,omitempty"`
	DisplayModel           *DisplayModel          `json:"display_model,omitempty"`
	RegulatorModel         *RegulatorModel        `json:"regulator_model,omitempty"`
	SensorModel            *SensorModel           `json:"sensor_model,omitempty"`
	PowerModel             *PowerModel            `json:"power_model,omitempty"`
	ControlModel           *ControlModel          `json:"control_model,omitempty"`
	Recording              *RecordingView         `json:"recording,omitempty"`
	Recovery               RecoveryState          `json:"recovery"`
	VerificationSettings   VerificationSettings   `json:"verification_settings"`
	Operations             []OperationSummary     `json:"operations"`
	SelectedOperation      string                 `json:"selected_operation"`
	LastOperation          string                 `json:"last_operation,omitempty"`
	Verification           *VerificationPreview   `json:"verification,omitempty"`
	LastExport             *Artifact              `json:"last_export,omitempty"`
	SchematicPage          SchematicPage          `json:"schematic_page"`
	PartCategory           string                 `json:"part_category"`
	PartCategories         []PartCategory         `json:"part_categories"`
	PartSearch             string                 `json:"part_search"`
	Theme                  string                 `json:"theme"`
	Appearance             AppearanceState        `json:"appearance"`
	AppearanceWork         *AppearanceWork        `json:"appearance_work,omitempty"`
	MatchingParts          []Part                 `json:"matching_parts"`
	UndoName               string                 `json:"undo_name"`
	RedoName               string                 `json:"redo_name"`
	Scope                  Scope                  `json:"scope"`
	ScopeVisible           bool                   `json:"scope_visible"`
	Mixed                  *MixedResult           `json:"mixed,omitempty"`
	NPN                    *NPNResult             `json:"npn,omitempty"`
	Transient              *Transient             `json:"transient,omitempty"`
	TransientSettings      TransientSettings      `json:"transient_settings"`
	Document               DocumentState          `json:"document"`
	CommandRevision        uint64                 `json:"command_revision"`
	Revision               uint64                 `json:"revision"`
	ElectricalRevision     uint64                 `json:"electrical_revision"`
	Circuit                Circuit                `json:"circuit"`
	Components             []Component            `json:"components"`
	Wires                  []Wire                 `json:"wires"`
	Nets                   []Net                  `json:"nets"`
	Drawing                DrawingState           `json:"drawing"`
	Model                  LEDModel               `json:"model"`
	Selected               string                 `json:"selected"`
	View                   View                   `json:"view"`
	Status                 string                 `json:"status"`
	Analysis               string                 `json:"analysis"`
	Paused                 bool                   `json:"paused"`
	RunID                  uint64                 `json:"run_id"`
	Result                 *Result                `json:"result,omitempty"`
	Diagnostics            []Diagnostic           `json:"diagnostics"`
	CanUndo                bool                   `json:"can_undo"`
	CanRedo                bool                   `json:"can_redo"`
	LastActor              string                 `json:"last_actor"`
	Closed                 bool                   `json:"closed"`
}
type savedResponse struct {
	fingerprint [32]byte
	response    Response
	project     *projectJob
}
type Service struct {
	audio                            *audioJob
	audioWorkers                     int
	audioSerial                      uint64
	audioPlayer                      audioplay.Player
	audioPlaybackCancel              context.CancelFunc
	audioPlaybackSerial              uint64
	inspectorSection                 string
	workspaceLayout                  WorkspaceLayout
	shutdown                         *shutdownJob
	printState                       *PrintState
	printWork                        *printWork
	printJobs                        []*PrintJob
	printSerial, printJobSerial      uint64
	printBackend                     printspool.Backend
	printRender                      func(context.Context, exportCapture) ([]byte, error)
	sweeps                           *sweepStore
	sweepSerial                      uint64
	sweepViewerRevision              uint64
	sweepImport                      sweepImportFunc
	sweepWorkers                     int
	sweepCompute                     func(context.Context, sweepPlan, func(int)) (*SweepResult, error)
	preview                          *previewJob
	previewSerial                    uint64
	previewWorkers                   int
	previewRender                    func(context.Context, exportCapture) ([]byte, error)
	projectJob                       *projectJob
	projectWorkers                   int
	projectJobs                      map[string]*projectJob
	projectOrder                     []string
	projectSerial                    uint64
	projectLoad                      projectLoadFunc
	projectSave                      projectSaveFunc
	projectExport                    projectExportFunc
	projectImport                    projectImportFunc
	recoveryQueryRead                recoveryRestoreFunc
	recoveryReadSerial               uint64
	recoveryReadJobs                 map[string]*recoveryReadJob
	recoveryReadOrder                []string
	recoveryReaders                  map[uint64]context.CancelFunc
	recoveryDismissPrepare           recoveryDismissPrepareFunc
	recoveryDismissExecute           recoveryDismissExecuteFunc
	recoveryConfigure                recoveryConfigurationPrepareFunc
	recoveryRestore                  recoveryRestoreFunc
	recoveryPrepare                  recoveryPrepareFunc
	recoveryExecute                  recoveryExecuteFunc
	recoveryCatalogRevision          uint64
	viewerRevision                   uint64
	projectPublish                   projectPublishFunc
	inputJob                         *inputJob
	inputSerial                      uint64
	inputSolve                       inputSolveFunc
	exampleID                        string
	diagnosticDetails                bool
	displayPreferences               DisplayPreferences
	measurementTarget                *MeasurementTarget
	pins                             []PinSpec
	savedPreferences                 [32]byte
	dcCompiled                       *Compiled
	focusProbe                       string
	stepSerial                       uint64
	stepJob                          *stepJob
	actuationRevision                uint64
	liveSwitchEdit                   bool
	savedRecordings                  [32]byte
	restorePaused                    bool
	recording                        recordingState
	recovery                         RecoveryState
	recoverySelectionCleared         bool
	recoveryStore                    *recoveryStore
	recoveryCancel                   context.CancelFunc
	recoveredFrom                    string
	discardRecoveryOnClose           bool
	operations                       map[string]*verificationOperation
	operationOrder                   []string
	operationSerial                  uint64
	selectedOperation, lastOperation string
	verificationSettings             VerificationSettings
	lastExport                       *Artifact
	exportPage                       SchematicPage
	scopeWindow                      *ScopeWindow
	partCategory                     string
	partSearch                       string
	nextHistoryName                  string
	scopeProbes                      []string
	scopeConfigured                  bool
	cursorA, cursorB                 *float64
	activeCursor                     string

	mixed                       *MixedResult
	mixedState                  *mixedState
	npnHigh, npnLow             *NPNResult
	npn                         *NPNResult
	mu                          sync.Mutex
	liveWorkers                 sync.WaitGroup
	documentPath                string
	documentGeneration          uint64
	savedGraph, diskHash        [32]byte
	revision, electrical, run   uint64
	commandRevision             uint64
	circuit                     Circuit
	graph                       Graph
	drawing                     DrawingState
	usedIDs                     map[string]bool
	selected                    string
	viewport                    ViewportState
	view                        View
	status                      string
	paused, closed              bool
	result                      *Result
	diagnostics                 []Diagnostic
	undo, redo                  []historyEntry
	transient                   *Transient
	transientSettings           TransientSettings
	inputMode, pendingInputMode string
	actor                       string
	cache                       map[string]savedResponse
	cacheOrder                  []string
	changed                     chan struct{}
	eventRevisions              map[string]uint64
	eventWaiters                int
	scopeVisible                bool
	theme                       string
	themeRevision               uint64
	appearance                  AppearanceState
	appearanceStore             *appearanceStore
	appearanceWorkers           int
	appearanceRead              appearanceReadFunc
	appearancePublish           projectPublishFunc
	appearanceJob               *appearanceJob
	appearanceJobs              map[string]*appearanceJob
	appearanceOrder             []string
	appearanceSerial            uint64
	cancel                      context.CancelFunc
	solver                      func(context.Context, Circuit) (Result, error)
	networkAdvance              networkAdvanceFunc
	initialSolver               initialSolverFunc
	initialTimeout              time.Duration
}

func New() *Service        { return newGraphService(SolveLED, EmptyGraph()) }
func NewExample() *Service { return newService(SolveLED) }
func newService(solver func(context.Context, Circuit) (Result, error)) *Service {
	s := newGraphService(solver, ExampleGraph())
	s.mu.Lock()
	s.exampleID = "led"
	s.mu.Unlock()
	return s
}
func newGraphService(solver func(context.Context, Circuit) (Result, error), graph Graph) *Service {
	s := &Service{
		workspaceLayout:        DefaultWorkspaceLayout(),
		inspectorSection:       "component",
		revision:               1,
		sweepImport:            prepareSweepImport,
		sweepCompute:           computeDCSweep,
		previewRender:          renderSchematicPage,
		printRender:            renderSchematicPage,
		printBackend:           printspool.System(),
		electrical:             1,
		graph:                  graph.Clone(),
		drawing:                DefaultDrawing(),
		usedIDs:                map[string]bool{},
		view:                   View{Zoom: 1},
		cache:                  map[string]savedResponse{},
		changed:                make(chan struct{}),
		eventRevisions:         map[string]uint64{"state": 1},
		solver:                 solver,
		networkAdvance:         advanceLiveNetwork,
		inputSolve:             solveInputWork,
		projectLoad:            prepareProjectLoad,
		projectSave:            prepareProjectSave,
		projectExport:          prepareProjectExport,
		projectImport:          prepareRecordingImport,
		recoveryPrepare:        prepareManualRecovery,
		recoveryQueryRead:      prepareRecoveryRestore,
		recoveryDismissPrepare: prepareRecoveryDismiss,
		recoveryDismissExecute: executeRecoveryDismiss,
		recoveryConfigure:      prepareRecoveryConfiguration,
		recoveryRestore:        prepareRecoveryRestore,
		recoveryExecute:        executeManualRecovery,
		projectPublish:         publishPreparedFile,
		initialSolver:          solveInitial,
		initialTimeout:         5 * time.Second,
		actor:                  "system",
		scopeVisible:           true,
		theme:                  "light",
		appearance:             AppearanceState{Status: "disabled"},
		appearanceRead:         readAppearance,
		appearancePublish:      publishPreparedFile,
	}
	for _, c := range graph.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range graph.Wires {
		s.usedIDs[w.ID] = true
	}
	s.displayPreferences = DefaultDisplayPreferences()
	s.commandRevision = 1
	s.inputMode = "auto"
	s.transientSettings = DefaultTransientSettings()
	s.verificationSettings = DefaultVerificationSettings()
	s.recovery = RecoveryState{Interval: 2, Candidates: []RecoveryCandidate{}, Problems: []RecoveryProblem{}}
	s.documentGeneration = 1
	s.savedGraph = contentHash(EmptyGraph(), s.transientSettings)
	s.savedPreferences = preferencesHash(nil, DefaultDisplayPreferences(), "")
	s.savedRecordings = s.recordingMembershipHashLocked()
	s.startLocked()
	return s
}
func (s *Service) Snapshot() Snapshot { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshotLocked() }
func (s *Service) snapshotLocked() Snapshot {
	page, _, _ := normalizeSchematicPage(&s.exportPage)
	graph := s.graph.Clone()
	compiled, _, diagnostic := graph.Compile()
	liveModes := []string{}
	if diagnostic == nil {
		liveModes = LiveInputModes(compiled.Analysis)
		if compiled.Source.Kind == "battery" {
			liveModes = []string{}
		}
	}
	analysis := compiled.Analysis
	if analysis == "" {
		analysis = "DC operating point"
	}
	out := Snapshot{
		WorkspaceLayout:        s.workspaceLayout,
		InspectorSection:       s.inspectorSection,
		Viewport:               cloneViewport(s.viewport),
		PulseTimings:           s.pulseTimingsLocked(),
		Audio:                  s.audioViewLocked(),
		Shutdown:               s.shutdownViewLocked(),
		SourceAnnotations:      s.sourceAnnotationsLocked(),
		Printing:               s.printingStateLocked(),
		PrintPreview:           s.printPreviewLocked(),
		DCSweep:                s.sweepStateLocked(),
		SchematicPage:          page,
		ProjectWork:            s.projectWorkLocked(),
		InputWork:              s.inputWorkLocked(),
		Example:                s.exampleContextLocked(),
		DiagnosticPresentation: s.diagnosticPresentationLocked(compiled),
		Display:                s.circuitDisplayLocked(),
		MeasurementInspection:  s.measurementInspectionLocked(compiled, s.currentMeasurementTargetLocked()),
		PinnedReadings:         s.pinnedReadingsLocked(),
		TraceFocus:             s.traceFocusLocked(compiled),
		Limits:                 Limits(),
		SimulatedComponents:    SimulatedComponentCount(graph),
		StepWork:               s.stepWorkLocked(),
		ActuationRevision:      s.actuationRevision,
		SwitchStates:           s.switchStatesLocked(),
		SourceSampling:         sourceSamplingAdvice(graph, s.transientSettings),
		SourceWarnings:         waveformSamplingWarnings(graph, s.transientSettings),
		LCSampling:             lcSamplingAdvice(graph, s.transientSettings),
		Recording:              s.recordingViewLocked(),
		Recovery:               s.recoveryStateLocked(),
		VerificationSettings:   cloneVerificationSettings(s.verificationSettings),
		VerificationInputModes: VerificationInputModes(analysis),
		LiveInputModes:         liveModes,
		Operations:             []OperationSummary{},
		SelectedOperation:      s.selectedOperation,
		LastOperation:          s.lastOperation,
		PartCategory:           browserCategory(s.partCategory),
		PartCategories:         PartCategories(),
		PartSearch:             s.partSearch,
		Theme:                  s.theme,
		Appearance:             s.appearance,
		AppearanceWork:         s.appearanceWorkLocked(),
		MatchingParts:          browserParts(s.partCategory, s.partSearch),
		Scope:                  s.scopeLocked(compiled),
		ScopeVisible:           s.scopeVisible,
		Mixed:                  cloneMixed(s.mixed),
		NPN:                    cloneNPN(s.npn),
		CommandRevision:        s.commandRevision,
		Transient:              cloneTransient(s.transient),
		TransientSettings:      s.transientSettings,
		Document:               DocumentState{PreferencesDirty: preferencesHash(s.pins, s.displayPreferences, s.exampleID) != s.savedPreferences, SchematicDirty: contentHash(s.graph, s.transientSettings) != s.savedGraph, RecordingsDirty: s.recordingsDirtyLocked(), RecordingCount: s.recordingCountLocked(), Path: s.documentPath, Dirty: s.dirtyLocked(), FormatVersion: s.currentDocumentVersionLocked(), Generation: s.documentGeneration, RecoveredFrom: s.recoveredFrom},
		Revision:               s.revision,
		ElectricalRevision:     s.electrical,
		Circuit:                s.circuit,
		Components:             graph.Components,
		Wires:                  graph.Wires,
		Nets:                   compiled.Nets,
		Drawing:                s.drawing,
		Model:                  Model(),
		Selected:               s.selected,
		View:                   s.view,
		Status:                 s.status,
		Analysis:               analysis,
		Paused:                 s.paused,
		RunID:                  s.run,
		Diagnostics:            append([]Diagnostic{}, s.diagnostics...),
		CanUndo:                len(s.undo) > 0,
		CanRedo:                len(s.redo) > 0,
		LastActor:              s.actor,
		Closed:                 s.closed,
	}
	for i := len(s.operationOrder) - 1; i >= 0; i-- {
		out.Operations = append(out.Operations, s.operationSummaryLocked(s.operations[s.operationOrder[i]]))
	}
	if operation := s.operations[s.selectedOperation]; operation != nil && operation.record != nil {
		record := operation.record
		preview := &VerificationPreview{ID: operation.summary.ID, Engine: record.metadata.Engine, InputMode: record.metadata.InputMode, Duration: record.metadata.Duration, SampleCount: record.metadata.SampleCount, Readings: []VerificationReading{}}
		for i, probe := range record.metadata.Probes {
			stat := record.metadata.Statistics[i]
			preview.Readings = append(preview.Readings, VerificationReading{Probe: probe, Value: record.rows[len(record.rows)-1].Values[i], Minimum: stat.Minimum, Maximum: stat.Maximum})
		}
		out.Verification = preview
	}
	for _, part := range graph.Components {
		if IsWaveformSource(part.Kind) {
			m := SourceWaveformModel()
			out.WaveformModel = &m
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "potentiometer" {
			model := LinearPotentiometerModel()
			out.PotentiometerModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "zener" {
			m := ZenerDeviceModel()
			out.ZenerModel = &m
		}
		if part.Kind == "schottky" {
			m := SchottkyDeviceModel()
			out.SchottkyModel = &m
		}
		if part.Kind == "diode" {
			model := SiliconDiodeModel()
			out.DiodeModel = &model
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "switch" {
			model := SwitchDeviceModel()
			out.SwitchModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "nmos" {
			model := NMOSDeviceModel()
			out.NMOSModel = &model
			break
		}
	}
	for _, part := range graph.Components {
		if part.Kind == "pmos" {
			model := PMOSDeviceModel()
			out.PMOSModel = &model
			break
		}
	}
	out.PNPModel = pnpModelForGraph(graph)
	out.OpAmpModel = opAmpModelForGraph(graph)
	out.Timer555Model = timer555ModelForGraph(graph)
	out.DeviceModel = extendedModelForGraph(graph)
	out.HBridgeModel = hbridgeModelForGraph(graph)
	out.LogicModel = logicModelForGraph(graph)
	out.SequentialModel = sequentialModelForGraph(graph)
	out.DisplayModel = displayModelForGraph(graph)
	out.RegulatorModel = regulatorModelForGraph(graph)
	out.SensorModel = sensorModelForGraph(graph)
	out.PowerModel = powerModelForGraph(graph)
	out.ControlModel = controlModelForGraph(graph)
	out.LastExport = cloneArtifact(s.lastExport)
	if len(s.undo) > 0 {
		out.UndoName = s.undo[len(s.undo)-1].Name
		out.UndoEntry = historyTip(s.undo[len(s.undo)-1])
	}
	if len(s.redo) > 0 {
		out.RedoName = s.redo[len(s.redo)-1].Name
		out.RedoEntry = historyTip(s.redo[len(s.redo)-1])
	}
	if out.Transient != nil {
		out.Transient.Sample.frame = nil
		for i := range out.Transient.Samples {
			out.Transient.Samples[i].frame = nil
		}
	}
	if s.result != nil {
		r := *s.result
		r.Nodes = map[string]float64{}
		for k, v := range s.result.Nodes {
			r.Nodes[k] = v
		}
		out.Result = &r
	}
	return out
}
func (s *Service) startLocked() {
	s.cancelInputWorkLocked("superseded")
	s.cancelStepLocked("superseded")
	allowPaused := s.restorePaused
	s.restorePaused = false
	s.cursorA = nil
	s.cursorB = nil
	s.scopeWindow = nil
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.run++
	if s.mixed != nil {
		s.mixed.Stale = true
	}
	if s.npn != nil {
		s.npn.Stale = true
	}
	if s.transient != nil {
		s.transient.Stale = true
	}
	if s.result != nil {
		s.result.Stale = true
	}
	s.diagnostics = nil
	s.circuit = s.graph.Summary()
	compiled, status, diagnostic := s.graph.Compile()
	if diagnostic != nil {
		s.diagnostics = []Diagnostic{*diagnostic}
		if s.paused {
			s.status = "paused"
		} else {
			s.status = status
		}
		return
	}
	if s.paused && !allowPaused {
		s.status = "paused"
		return
	}
	if IsNetworkAnalysis(compiled.Analysis) {
		s.startMixedLocked(compiled)
		return
	}
	s.mixed = nil
	s.mixedState = nil
	if compiled.Analysis == NPNTransientAnalysis {
		s.startNPNTransientLocked(compiled)
		return
	}
	if compiled.Analysis == NPNAnalysis {
		s.startNPNLocked(compiled, allowPaused)
		return
	}
	s.npn = nil
	if compiled.Analysis == "RC transient" {
		s.startRCLocked(compiled)
		return
	}
	s.transient = nil
	s.status = "updating"
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	c, revision, run := s.circuit, s.electrical, s.run
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		result, err := s.solver(ctx, c)
		if err == nil {
			result = compiled.Annotate(result)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.paused && !allowPaused || s.electrical != revision || s.run != run || ctx.Err() != nil {
			return
		}
		s.cancel = nil
		cancel()
		if err != nil {
			s.status = "error"
			s.diagnostics = []Diagnostic{{"solve-failed", err.Error(), compiled.LED.ID}}
		} else {
			result.RunID = run
			result.ElectricalRevision = revision
			s.result = &result
			s.dcCompiled = &compiled
			if s.paused {
				s.status = "paused"
			} else {
				s.status = "running"
			}
		}
		s.revision++
		s.signalLocked("simulation", "diagnostics")
	}()
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closeLocked()
	job := s.shutdown
	s.mu.Unlock()
	<-job.done
}
func (s *Service) closeLocked() {
	if s.closed {
		return
	}
	store := s.recoveryStore
	capture := recoveryCapture{generation: s.documentGeneration}
	if store != nil && !s.discardRecoveryOnClose && s.recovery.Enabled {
		capture = s.recoveryCaptureLocked()
	}
	job := &shutdownJob{done: make(chan struct{}), work: ShutdownWork{ID: "shutdown-1", Status: "queued", Phase: "queued", Actor: "system", InputRevision: s.revision, CommandRevision: s.commandRevision, DocumentGeneration: s.documentGeneration, DiscardRecovery: s.discardRecoveryOnClose, CheckpointRecovery: s.recovery.Enabled}}
	if store != nil {
		job.work.RecoveryDirectory = store.directory
	}
	s.shutdown = job
	for _, cancel := range s.recoveryReaders {
		cancel()
	}
	if s.recoveryStore != nil {
		if s.recoveryCancel != nil {
			s.recoveryCancel()
		}
	}
	if s.sweeps != nil {
		for _, job := range s.sweeps.operations {
			if operationActive(job.summary.Status) {
				job.cancel()
			}
		}
	}
	if s.preview != nil {
		s.preview.cancel()
		s.preview.data = nil
		s.preview.view.Visible = false
	}
	if s.printWork != nil {
		s.printWork.cancel()
	}
	s.cancelProjectWorkLocked("canceled")
	s.cancelInputWorkLocked("canceled")
	s.cancelStepLocked("canceled")
	s.cancelAudioLocked()
	s.closed = true
	if s.appearanceStore != nil {
		select {
		case s.appearanceStore.wake <- struct{}{}:
		default:
		}
	}
	if s.recording.cancel != nil {
		s.recording.cancel()
	}
	s.recording.playing = false
	for _, operation := range s.operations {
		if operationActive(operation.summary.Status) {
			operation.cancel()
			operation.summary.Status = "canceled"
		}
	}
	s.status = "closed"
	if s.cancel != nil {
		s.cancel()
	}
	s.revision++
	s.signalLocked("lifecycle", "simulation", "jobs", "recordings", "recovery")
	go s.runShutdown(job, store, capture)
}
func (s *Service) Wait(ctx context.Context) (Snapshot, error) {
	for {
		s.mu.Lock()
		state := s.snapshotLocked()
		changed := s.changed
		s.mu.Unlock()
		if state.Status != "updating" && (state.StepWork == nil || !operationActive(state.StepWork.Status)) && (state.InputWork == nil || !operationActive(state.InputWork.Status)) && (state.ProjectWork == nil || !operationActive(state.ProjectWork.Status)) {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-changed:
		}
	}
}

// Dispatch is the sole semantic entry point used by widgets and agent clients.
// Mutation preconditions and retry keys apply identically to both.
func (s *Service) Dispatch(ctx context.Context, r Request, actor string) Response {
	if r.Command == "simulation.step" && s.legacyStepNeedsInitialization(r, actor) {
		if _, err := s.Wait(ctx); err != nil {
			return s.failure(r, "canceled", err.Error())
		}
	}
	out := s.dispatch(ctx, r, actor)
	if r.Command == "simulation.step" {
		return s.awaitStep(ctx, r, actor, out)
	}
	if r.Command == "app.close" {
		return s.awaitShutdown(ctx, out)
	}
	if waitingProjectCommand(r.Command) {
		return s.awaitProjectWork(ctx, r, actor, out)
	}
	return out
}
func (s *Service) dispatch(ctx context.Context, r Request, actor string) Response {
	if r.Command == "events.poll" {
		return s.pollEvents(ctx, r)
	}
	if r.Command == "simulation.wait" || r.Command == "appearance.wait" {
		if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 {
			return s.failure(r, "invalid-request", "version 1 and a nonempty ID (max 128 bytes) required")
		}
		if err := params(r.Params, &struct{}{}); err != nil {
			return s.failure(r, "invalid-params", err.Error())
		}
		bounded, cancel := context.WithTimeout(ctx, SimulationWaitTimeout)
		defer cancel()
		var state Snapshot
		var err error
		if r.Command == "appearance.wait" {
			state, err = s.waitAppearance(bounded)
		} else {
			state, err = s.Wait(bounded)
		}
		if err != nil {
			return Response{Version: 1, ID: r.ID, Revision: state.Revision, Error: &Failure{"timeout", err.Error()}}
		}
		return Response{Version: 1, ID: r.ID, Revision: state.Revision, Result: state}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Response{Version: ProtocolVersion, ID: r.ID, Revision: s.revision}
	fail := func(code, message string) Response {
		out.Error = &Failure{code, message}
		out.Help = s.requestDiagnosticLocked(r, code, message)
		return out
	}
	if err := ctx.Err(); err != nil {
		return fail("canceled", err.Error())
	}
	if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 {
		return fail("invalid-request", "version 1 and a nonempty ID (max 128 bytes) required")
	}
	cap, known := capabilityIndex[r.Command]
	if !known {
		return fail("unknown-command", "enumerate capabilities.list for supported commands")
	}
	if !cap.Mutation {
		if r.Command == "appearance.work.inspect" {
			var p IDParams
			if err := params(r.Params, &p, "id"); err != nil {
				return fail("invalid-params", err.Error())
			}
			job := s.appearanceJobs[p.ID]
			if job == nil {
				return fail("operation-unavailable", "Appearance operation expired or is unknown")
			}
			out.Result = cloneAppearanceWork(&job.work)
			return out
		}
		if r.Command == "app.close.inspect" {
			result, err := s.shutdownQueryLocked(r.Params)
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = result
			return out
		}
		if r.Command == "sweep.inspect" {
			var p IDParams
			if err := params(r.Params, &p, "id"); err != nil {
				return fail("invalid-params", err.Error())
			}
			result, err := s.inspectSweepLocked(p.ID)
			if err != nil {
				return fail("not-found", err.Error())
			}
			out.Result = result
			return out
		}
		if r.Command == "print.preview.image" {
			var p PreviewImageParams
			if err := params(r.Params, &p, "id"); err != nil {
				return fail("invalid-params", err.Error())
			}
			result, err := s.previewImageLocked(p)
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = result
			return out
		}
		if r.Command == "project.work.inspect" {
			var p IDParams
			if err := params(r.Params, &p, "id"); err != nil {
				return fail("invalid-params", err.Error())
			}
			job := s.projectJobs[p.ID]
			if job == nil {
				return fail("operation-unavailable", "Project operation expired or is unknown")
			}
			out.Result = cloneProjectWork(&job.work)
			return out
		}
		if r.Command == "audio.read" {
			var p IDParams
			if err := params(r.Params, &p, "id"); err != nil {
				return fail("invalid-params", err.Error())
			}
			if s.audio == nil || s.audio.view.ID != p.ID {
				return fail("not-found", "Audio preview is not retained")
			}
			out.Result = AudioData{Preview: *s.audioViewLocked(), WAV: append([]byte(nil), s.audio.wav...)}
			return out
		}
		if r.Command == "examples.list" {
			if err := params(r.Params, &struct{}{}); err != nil {
				return fail("invalid-params", err.Error())
			}
			out.Result = ExampleCatalog()
			return out
		}
		if r.Command == "display.inspect" {
			frame, err := s.displayQueryLocked(r.Params)
			if err != nil {
				return fail("invalid-params", err.Error())
			}
			out.Result = frame
			return out
		}
		if r.Command == "measurement.inspect" {
			inspection, err := s.measurementQueryLocked(r.Params)
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = inspection
			return out
		}
		if r.Command == "recovery.inspect" || r.Command == "recovery.samples" || r.Command == "recovery.read.inspect" || r.Command == "recovery.read.result" || r.Command == "recovery.read.chunk" {
			var result any
			var err error
			if r.Command == "recovery.read.inspect" || r.Command == "recovery.read.result" || r.Command == "recovery.read.chunk" {
				result, err = s.recoveryReadQueryLocked(ctx, r.Command, r.Params)
			} else {
				result, err = s.recoveryQueryLocked(ctx, r.Command, r.Params)
			}
			out.Revision = s.revision
			if err != nil {
				var action *actionError
				switch {
				case errors.As(err, &action):
					return fail(action.code, action.message)
				case errors.Is(err, context.Canceled):
					return fail("canceled", "Recovery query canceled")
				case errors.Is(err, context.DeadlineExceeded):
					return fail("timeout", "Recovery query exceeded its cooperative read budget")
				default:
					return fail("invalid-params", err.Error())
				}
			}
			out.Result = result
			return out
		}
		if r.Command == "recording.compare" || r.Command == "recording.timing" || r.Command == "recording.statistics" {
			var result any
			var err error
			if r.Command == "recording.statistics" {
				result, err = s.recordingStatisticsQueryLocked(r.Params)
			} else if r.Command == "recording.timing" {
				result, err = s.recordingTimingQueryLocked(r.Params)
			} else {
				result, err = s.recordingComparisonQueryLocked(r.Params)
			}
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = result
			return out
		}
		if r.Command == "operation.inspect" || r.Command == "operation.samples" {
			result, err := s.verificationQueryLocked(r.Command, r.Params)
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = result
			return out
		}
		var capabilities []Capability
		if r.Command == "capabilities.list" {
			var p CapabilityQuery
			if err := params(r.Params, &p); err != nil {
				return fail("invalid-params", err.Error())
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(r.Params, &fields)
			if _, present := fields["command"]; present {
				if err := params(r.Params, &p, "command"); err != nil {
					return fail("invalid-params", err.Error())
				}
			}
			capabilities = Catalog()
			if p.Command != "" {
				var err error
				capabilities, err = detailedCapabilities(p.Command)
				if err != nil {
					var action *actionError
					if errors.As(err, &action) {
						return fail(action.code, action.message)
					}
					return fail("invalid-params", err.Error())
				}
			}
		} else if err := params(r.Params, &struct{}{}); err != nil {
			return fail("invalid-params", err.Error())
		}
		if r.Command == "capabilities.list" {
			out.Result = map[string]any{"version": ProtocolVersion, "schema_version": DiscoverySchemaVersion, "schemas": SchemaRegistry(), "capabilities": capabilities, "mutation_preconditions": map[string]string{"expected_revision": "Guards all state publication, including live samples", "expected_command_revision": "Guards admitted commands, including other actors; live solver publication does not invalidate it", "expected_document_generation": "Optional mutation context guard; prevents applying an old rendered action to a replacement document, including background publication", "combination": "A revision guard is required; all supplied revision/document guards must match"}, "events": map[string]any{"command": "events.poll", "mode": "coalesced-latest-state", "max_waiters": MaxEventWaiters, "max_timeout_ms": MaxEventWaitMS, "default_timeout_ms": 1000, "topics": append([]string{}, eventTopics...)}, "retry_retention": "Latest 256 successful mutation keys per session; reconnect to the same process preserves them. Expired keys have no deduplication guarantee."}
		} else if r.Command == "parts.list" {
			out.Result = Library()
		} else {
			out.Result = s.snapshotLocked()
		}
		return out
	}
	if (r.ExpectedRevision == nil && r.ExpectedCommandRevision == nil) || r.Key == "" || len(r.Key) > 128 {
		return fail("invalid-request", "mutations require expected_revision or expected_command_revision and key (1–128 bytes)")
	}
	normalized, err := normalizeTimeInput(r.Command, r.Params)
	if err != nil {
		return fail("invalid-params", err.Error())
	}
	r.Params = normalized
	canonical := map[string]any{}
	if err := DecodeStrict(r.Params, &canonical); err != nil {
		return fail("invalid-params", err.Error())
	}
	if canonical == nil {
		return fail("invalid-params", "params must be a JSON object")
	}
	data, _ := json.Marshal(struct {
		Command string
		Params  map[string]any
	}{r.Command, canonical})
	fingerprint := sha256.Sum256(data)
	key := actor + "\x00" + r.Key
	if old, ok := s.cache[key]; ok {
		if old.fingerprint != fingerprint {
			return fail("key-reused", "retry key already belongs to a different command")
		}
		out = cloneMutationResponse(old.response)
		out.ID = r.ID
		out.Duplicate = true
		return out
	}
	if err := s.mutationContextFailureLocked(r); err != nil {
		return fail(err.code, err.message)
	}
	if r.Command == "edit.undo" || r.Command == "edit.redo" {
		if err := s.checkHistoryTargetLocked(r.Command, r.Params, actor); err != nil {
			var action *actionError
			if errors.As(err, &action) {
				return fail(action.code, action.message)
			}
			return fail("invalid-params", err.Error())
		}
	}

	s.liveSwitchEdit = false
	before := s.historyLocked()
	s.nextHistoryName = ""
	if err := s.applyLocked(r.Command, r.Params); err != nil {
		var action *actionError
		if errors.As(err, &action) {
			return fail(action.code, action.message)
		}
		return fail("invalid-params", err.Error())
	}
	changedGraph := !reflect.DeepEqual(before.Graph, s.graph)
	changedSettings := before.Settings != s.transientSettings
	changedPins := !reflect.DeepEqual(clonePins(before.Pins), clonePins(s.pins))
	changedDisplay := before.Display != s.displayPreferences
	changedExample := before.ExampleID != s.exampleID
	changedDocument := changedGraph || changedSettings || changedPins || changedDisplay || changedExample
	forceRestart := r.Command == "example.load" || r.Command == "example.current.reset" || legacyExampleID(r.Command) != ""
	if changedDocument {
		if changedGraph {
			s.retainInterruptedDefaultProbesLocked(before.Graph)
		}
		if r.Command != "edit.undo" && r.Command != "edit.redo" {
			before.Actor = actor
			before.CommandRevision = s.commandRevision + 1
			before.Name = historyName(r.Command)
			if s.nextHistoryName != "" {
				before.Name = s.nextHistoryName
			}
			s.undo = append(s.undo, before)
			s.redo = nil
			if len(s.undo) > 100 {
				s.undo = append([]historyEntry(nil), s.undo[len(s.undo)-100:]...)
			}
		}
		s.normalizeDrawingLocked()
		if !ElectricalEqual(before.Graph, s.graph) || changedSettings || forceRestart {
			if s.liveSwitchEdit {
				s.cancelStepLocked("superseded")
				s.actuationRevision++
				s.cancelInputWorkLocked("superseded")
			} else {
				s.electrical++
				s.startLocked()
			}
		}
	}
	if forceRestart && !changedDocument {
		s.electrical++
		s.startLocked()
	}
	if r.Command == "project.clear" || r.Command == "example.reset" || forceRestart && r.Command != "simulation.restart" && r.Command != "project.load" && r.Command != "recovery.restore" {
		s.verificationSettings = DefaultVerificationSettings()
		s.scopeConfigured = false
		s.focusProbe = ""
		s.scopeProbes = nil
		if definition, ok := findExample(s.exampleID); ok && len(definition.probes) > 0 {
			s.scopeProbes = append([]string{}, definition.probes...)
			s.scopeConfigured = true
		}
		s.activeCursor = "a"
		s.cursorA = nil
		s.cursorB = nil
	}
	if r.Command == "sweep.start" && s.sweeps != nil {
		s.sweeps.operations[s.sweeps.selected].summary.Actor = actor
	}
	if s.printWork != nil && s.printWork.view.Kind == r.Command && s.printWork.view.Status == "queued" {
		s.printWork.view.Actor = actor
		if s.printWork.view.Kind == "print.submit" {
			s.printWork.job.Actor = actor
		}
	}
	if (r.Command == "print.preview.start" || r.Command == "print.preview.page.set") && s.preview != nil && s.preview.view.Actor == "" {
		s.preview.view.Actor = actor
	}
	if r.Command == "recovery.read.start" && len(s.recoveryReadOrder) > 0 {
		s.recoveryReadJobs[s.recoveryReadOrder[len(s.recoveryReadOrder)-1]].work.Actor = actor
	}
	if (r.Command == "app.close" || r.Command == "app.close.start") && s.shutdown != nil {
		s.shutdown.work.Actor = actor
		s.shutdown.work.CommandRevision = s.commandRevision + 1
	}
	if r.Command == "simulation.verify" {
		s.operations[s.lastOperation].summary.Actor = actor
	}
	if s.appearanceJob != nil && s.appearanceJob.work.CommandRevision == s.commandRevision+1 {
		s.appearanceJob.work.Actor = actor
	}
	if s.projectJob != nil && s.projectJob.work.CommandRevision == s.commandRevision+1 {
		s.projectJob.work.Actor = actor
		s.projectJob.requestID = r.ID
		if waitingProjectCommand(r.Command) {
			s.projectJob.legacyKey = key
		}
	} else if s.activeProjectWorkLocked() && s.projectWorkSupersededLocked(s.projectJob) {
		s.cancelProjectWorkLocked("superseded")
	}
	if s.inputJob != nil && s.inputJob.work.CommandRevision == s.commandRevision+1 {
		s.inputJob.work.Actor = actor
	}
	if r.Command == "simulation.step" || r.Command == "simulation.step.start" {
		s.stepJob.work.Actor = actor
		s.stepJob.requestID = r.ID
		if r.Command == "simulation.step" {
			s.stepJob.legacyKey = key
		}
	}
	if strings.HasPrefix(r.Command, "recording.") && r.Command != "recording.keep.set" {
		s.viewerRevision++
	}
	if s.audio != nil && !s.audioViewLocked().Current {
		s.cancelAudioLocked()
	}
	s.actor = actor
	s.commandRevision++
	s.revision++
	s.signalLocked(commandEventTopics(r.Command, changedDocument)...)
	out.Revision = s.revision
	out.Result = s.snapshotLocked()
	cached := out
	cached.Result = cloneSnapshot(out.Result.(Snapshot))
	entry := savedResponse{fingerprint: fingerprint, response: cached}
	if waitingProjectCommand(r.Command) && s.projectJob.work.CommandRevision == s.commandRevision {
		entry.project = s.projectJob
	}
	s.cache[key] = entry
	s.cacheOrder = append(s.cacheOrder, key)
	if len(s.cacheOrder) > 256 {
		delete(s.cache, s.cacheOrder[0])
		s.cacheOrder = s.cacheOrder[1:]
	}
	return out
}

func cloneSnapshot(state Snapshot) Snapshot {
	if state.Audio != nil {
		v := *state.Audio
		if v.Error != nil {
			e := *v.Error
			v.Error = &e
		}
		state.Audio = &v
	}
	state.Viewport = cloneViewport(state.Viewport)
	state.Shutdown = cloneShutdown(state.Shutdown)
	state.SourceAnnotations = cloneSourceAnnotations(state.SourceAnnotations)
	state.PrintPreview = clonePrintPreview(state.PrintPreview)
	state.Printing = clonePrintState(state.Printing)
	state.DCSweep = cloneSweepState(state.DCSweep)
	state.SchematicPage = cloneSchematicPage(state.SchematicPage)
	state.ProjectWork = cloneProjectWork(state.ProjectWork)
	state.AppearanceWork = cloneAppearanceWork(state.AppearanceWork)
	state.InputWork = cloneInputWork(state.InputWork)
	state.Example = cloneExampleContext(state.Example)
	state.DiagnosticPresentation = cloneDiagnosticPresentation(state.DiagnosticPresentation)
	state.Display = cloneCircuitDisplay(state.Display)
	state.MeasurementInspection = cloneMeasurementInspection(state.MeasurementInspection)
	state.PinnedReadings = clonePinnedReadings(state.PinnedReadings)
	state.TraceFocus = cloneTraceFocus(state.TraceFocus)
	state.StepWork = cloneStepWork(state.StepWork)
	if state.UndoEntry != nil {
		tip := *state.UndoEntry
		state.UndoEntry = &tip
	}
	if state.RedoEntry != nil {
		tip := *state.RedoEntry
		state.RedoEntry = &tip
	}
	state.SwitchStates = cloneSwitchStates(state.SwitchStates)
	state.SourceSampling = append([]SourceSamplingAdvice{}, state.SourceSampling...)
	state.SourceWarnings = append([]Diagnostic{}, state.SourceWarnings...)
	state.LCSampling = cloneLCSamplingAdvice(state.LCSampling)
	if state.WaveformModel != nil {
		m := *state.WaveformModel
		state.WaveformModel = &m
	}
	if state.PotentiometerModel != nil {
		model := *state.PotentiometerModel
		state.PotentiometerModel = &model
	}
	if state.SchottkyModel != nil {
		m := *state.SchottkyModel
		state.SchottkyModel = &m
	}
	if state.ZenerModel != nil {
		m := *state.ZenerModel
		state.ZenerModel = &m
	}
	if state.DiodeModel != nil {
		model := *state.DiodeModel
		state.DiodeModel = &model
	}
	if state.NMOSModel != nil {
		model := *state.NMOSModel
		state.NMOSModel = &model
	}
	if state.Timer555Model != nil {
		m := *state.Timer555Model
		state.Timer555Model = &m
	}
	if state.DeviceModel != nil {
		m := *state.DeviceModel
		state.DeviceModel = &m
	}
	if state.HBridgeModel != nil {
		m := *state.HBridgeModel
		state.HBridgeModel = &m
	}
	if state.LogicModel != nil {
		m := *state.LogicModel
		state.LogicModel = &m
	}
	if state.SequentialModel != nil {
		m := *state.SequentialModel
		state.SequentialModel = &m
	}
	if state.DisplayModel != nil {
		m := *state.DisplayModel
		state.DisplayModel = &m
	}
	if state.RegulatorModel != nil {
		m := *state.RegulatorModel
		state.RegulatorModel = &m
	}
	if state.PowerModel != nil {
		m := *state.PowerModel
		state.PowerModel = &m
	}
	if state.SensorModel != nil {
		m := *state.SensorModel
		state.SensorModel = &m
	}
	if state.ControlModel != nil {
		m := *state.ControlModel
		state.ControlModel = &m
	}
	if state.OpAmpModel != nil {
		model := *state.OpAmpModel
		state.OpAmpModel = &model
	}
	if state.PNPModel != nil {
		model := *state.PNPModel
		state.PNPModel = &model
	}
	if state.PMOSModel != nil {
		model := *state.PMOSModel
		state.PMOSModel = &model
	}
	if state.SwitchModel != nil {
		model := *state.SwitchModel
		state.SwitchModel = &model
	}
	state.Recording = cloneRecordingView(state.Recording)
	state.Recovery = cloneRecovery(state.Recovery)
	state.VerificationSettings = cloneVerificationSettings(state.VerificationSettings)
	state.VerificationInputModes = append([]string{}, state.VerificationInputModes...)
	state.LiveInputModes = append([]string{}, state.LiveInputModes...)
	state.Operations = append([]OperationSummary{}, state.Operations...)
	state.PulseTimings = append([]PulseTiming{}, state.PulseTimings...)
	for i := range state.Operations {
		state.Operations[i].FailureHelp = cloneDiagnosticHelp(state.Operations[i].FailureHelp)
		if state.Operations[i].Error != nil {
			failure := *state.Operations[i].Error
			state.Operations[i].Error = &failure
		}
	}
	if state.Verification != nil {
		preview := *state.Verification
		preview.Readings = append([]VerificationReading{}, preview.Readings...)
		state.Verification = &preview
	}
	state.LastExport = cloneArtifact(state.LastExport)
	state.PartCategories = PartCategories()
	state.MatchingParts = browserParts(state.PartCategory, state.PartSearch)
	state.Scope = cloneScope(state.Scope)
	state.Mixed = cloneMixed(state.Mixed)
	state.NPN = cloneNPN(state.NPN)
	state.Transient = cloneTransient(state.Transient)
	state.Components = append([]Component{}, state.Components...)
	for i := range state.Components {
		state.Components[i].Power = clonePower(state.Components[i].Power)
		state.Components[i].Device = cloneDeviceParameters(state.Components[i].Device)
		state.Components[i].Terminals = append([]string(nil), state.Components[i].Terminals...)
		if state.Components[i].Sine != nil {
			w := *state.Components[i].Sine
			state.Components[i].Sine = &w
		}
		if state.Components[i].Pulse != nil {
			w := *state.Components[i].Pulse
			state.Components[i].Pulse = &w
		}
		if v := state.Components[i].TemperatureC; v != nil {
			c := *v
			state.Components[i].TemperatureC = &c
		}
		if v := state.Components[i].IlluminanceLux; v != nil {
			c := *v
			state.Components[i].IlluminanceLux = &c
		}
		if state.Components[i].WiperPosition != nil {
			v := *state.Components[i].WiperPosition
			state.Components[i].WiperPosition = &v
		}
		if state.Components[i].SwitchClosed != nil {
			copy := *state.Components[i].SwitchClosed
			state.Components[i].SwitchClosed = &copy
		}
		if v := state.Components[i].InitialVoltage; v != nil {
			copy := *v
			state.Components[i].InitialVoltage = &copy
		}
	}
	state.Wires = append([]Wire{}, state.Wires...)
	state.Diagnostics = append([]Diagnostic{}, state.Diagnostics...)
	state.Nets = append([]Net{}, state.Nets...)
	for i := range state.Nets {
		state.Nets[i].Terminals = append([]string{}, state.Nets[i].Terminals...)
		state.Nets[i].Labels = append([]string{}, state.Nets[i].Labels...)
	}
	if state.Result != nil {
		r := *state.Result
		r.Nodes = map[string]float64{}
		for k, v := range state.Result.Nodes {
			r.Nodes[k] = v
		}
		state.Result = &r
	}
	return state
}
func (s *Service) failure(r Request, code, message string) Response {
	return Response{Version: 1, ID: r.ID, Revision: s.Snapshot().Revision, Error: &Failure{code, message}}
}

// Cached mutation replies own their snapshot and failure independently of callers.
func cloneMutationResponse(out Response) Response {
	out.Help = cloneDiagnosticHelp(out.Help)
	if out.Result != nil {
		out.Result = cloneSnapshot(out.Result.(Snapshot))
	}
	if out.Error != nil {
		failure := *out.Error
		out.Error = &failure
	}
	return out
}

// Invalid legacy step requests must not wait behind an unrelated initializer.
func legacyStepMayWait(r Request) bool {
	if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 || r.Key == "" || len(r.Key) > 128 || r.ExpectedRevision == nil && r.ExpectedCommandRevision == nil {
		return false
	}
	normalized, err := normalizeTimeInput("simulation.step", r.Params)
	if err != nil {
		return false
	}
	var p StepParams
	return params(normalized, &p, "duration_s") == nil && finite(p.Duration) && p.Duration > 0 && p.Duration <= MaxManualStepSeconds
}

func (s *Service) legacyStepNeedsInitialization(r Request, actor string) bool {
	if !legacyStepMayWait(r) {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, cached := s.cache[actor+"\x00"+r.Key]
	return !cached && s.mutationContextFailureLocked(r) == nil && s.status == "updating"
}
