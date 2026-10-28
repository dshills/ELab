package workbench

type Snapshot struct {
	ScopeView               string                 `json:"scope_view"`
	BlockClipboardAvailable bool                   `json:"block_clipboard_available"`
	Tolerance               *ToleranceState        `json:"tolerance,omitempty"`
	Spectrum                *SpectrumState         `json:"spectrum,omitempty"`
	Proposal                *EditProposal          `json:"edit_proposal,omitempty"`
	Explanation             *ComponentExplanation  `json:"component_explanation,omitempty"`
	ExampleBrowser          ExampleBrowser         `json:"example_browser"`
	Audio                   *AudioPreview          `json:"audio_preview,omitempty"`
	InspectorSection        string                 `json:"inspector_section"`
	WorkspaceLayout         WorkspaceLayout        `json:"workspace_layout"`
	Viewport                ViewportState          `json:"viewport"`
	PulseTimings            []PulseTiming          `json:"pulse_timings"`
	Shutdown                *ShutdownWork          `json:"shutdown,omitempty"`
	Printing                *PrintState            `json:"printing,omitempty"`
	VerificationInputModes  []string               `json:"verification_input_modes"`
	LiveInputModes          []string               `json:"live_input_modes"`
	DCSweep                 *SweepState            `json:"dc_sweep,omitempty"`
	PrintPreview            *PrintPreview          `json:"print_preview,omitempty"`
	SourceAnnotations       []SourceAnnotation     `json:"source_annotations"`
	ProjectWork             *ProjectWork           `json:"project_work,omitempty"`
	InputWork               *InputWork             `json:"input_work,omitempty"`
	Example                 *ExampleContext        `json:"example,omitempty"`
	DiagnosticPresentation  DiagnosticPresentation `json:"diagnostic_presentation"`
	Display                 CircuitDisplay         `json:"display"`
	MeasurementInspection   MeasurementInspection  `json:"measurement_inspection"`
	PinnedReadings          []PinnedReading        `json:"pinned_readings"`
	TraceFocus              *TraceFocus            `json:"trace_focus,omitempty"`
	Limits                  CircuitLimits          `json:"limits"`
	SimulatedComponents     int                    `json:"simulated_component_count"`
	StepWork                *StepWork              `json:"step_work,omitempty"`
	UndoEntry               *HistoryTip            `json:"undo_entry,omitempty"`
	RedoEntry               *HistoryTip            `json:"redo_entry,omitempty"`
	ActuationRevision       uint64                 `json:"actuation_revision"`
	SwitchStates            []SwitchState          `json:"switch_states,omitempty"`
	SourceSampling          []SourceSamplingAdvice `json:"source_sampling,omitempty"`
	SourceWarnings          []Diagnostic           `json:"source_warnings,omitempty"`
	LCSampling              *LCSamplingAdvice      `json:"lc_sampling,omitempty"`
	WaveformModel           *WaveformModel         `json:"waveform_model,omitempty"`
	PotentiometerModel      *PotentiometerModel    `json:"potentiometer_model,omitempty"`
	DiodeModel              *DiodeModel            `json:"diode_model,omitempty"`
	SchottkyModel           *DiodeModel            `json:"schottky_model,omitempty"`
	ZenerModel              *ZenerModel            `json:"zener_model,omitempty"`
	SwitchModel             *SwitchModel           `json:"switch_model,omitempty"`
	NMOSModel               *NMOSModel             `json:"nmos_model,omitempty"`
	PMOSModel               *PMOSModel             `json:"pmos_model,omitempty"`
	PNPModel                *PNPModel              `json:"pnp_model,omitempty"`
	OpAmpModel              *OpAmpModel            `json:"opamp_model,omitempty"`
	Timer555Model           *Timer555Model         `json:"timer555_model,omitempty"`
	DeviceModel             *ExtendedModel         `json:"device_model,omitempty"`
	HBridgeModel            *HBridgeModel          `json:"hbridge_model,omitempty"`
	LogicModel              *LogicModel            `json:"logic_model,omitempty"`
	SequentialModel         *SequentialModel       `json:"sequential_model,omitempty"`
	DisplayModel            *DisplayModel          `json:"display_model,omitempty"`
	RegulatorModel          *RegulatorModel        `json:"regulator_model,omitempty"`
	SensorModel             *SensorModel           `json:"sensor_model,omitempty"`
	PowerModel              *PowerModel            `json:"power_model,omitempty"`
	ControlModel            *ControlModel          `json:"control_model,omitempty"`
	Recording               *RecordingView         `json:"recording,omitempty"`
	Recovery                RecoveryState          `json:"recovery"`
	VerificationSettings    VerificationSettings   `json:"verification_settings"`
	Operations              []OperationSummary     `json:"operations"`
	SelectedOperation       string                 `json:"selected_operation"`
	LastOperation           string                 `json:"last_operation,omitempty"`
	Verification            *VerificationPreview   `json:"verification,omitempty"`
	LastExport              *Artifact              `json:"last_export,omitempty"`
	SchematicPage           SchematicPage          `json:"schematic_page"`
	PartCategory            string                 `json:"part_category"`
	PartCategories          []PartCategory         `json:"part_categories"`
	PartSearch              string                 `json:"part_search"`
	Theme                   string                 `json:"theme"`
	Appearance              AppearanceState        `json:"appearance"`
	AppearanceWork          *AppearanceWork        `json:"appearance_work,omitempty"`
	MatchingParts           []Part                 `json:"matching_parts"`
	UndoName                string                 `json:"undo_name"`
	RedoName                string                 `json:"redo_name"`
	Scope                   Scope                  `json:"scope"`
	ScopeVisible            bool                   `json:"scope_visible"`
	Mixed                   *MixedResult           `json:"mixed,omitempty"`
	NPN                     *NPNResult             `json:"npn,omitempty"`
	Transient               *Transient             `json:"transient,omitempty"`
	TransientSettings       TransientSettings      `json:"transient_settings"`
	Document                DocumentState          `json:"document"`
	CommandRevision         uint64                 `json:"command_revision"`
	Revision                uint64                 `json:"revision"`
	ElectricalRevision      uint64                 `json:"electrical_revision"`
	Circuit                 Circuit                `json:"circuit"`
	Components              []Component            `json:"components"`
	Wires                   []Wire                 `json:"wires"`
	Nets                    []Net                  `json:"nets"`
	Drawing                 DrawingState           `json:"drawing"`
	Model                   LEDModel               `json:"model"`
	Selected                string                 `json:"selected"`
	View                    View                   `json:"view"`
	Status                  string                 `json:"status"`
	Analysis                string                 `json:"analysis"`
	Paused                  bool                   `json:"paused"`
	RunID                   uint64                 `json:"run_id"`
	Result                  *Result                `json:"result,omitempty"`
	Diagnostics             []Diagnostic           `json:"diagnostics"`
	CanUndo                 bool                   `json:"can_undo"`
	CanRedo                 bool                   `json:"can_redo"`
	LastActor               string                 `json:"last_actor"`
	Closed                  bool                   `json:"closed"`
}

// SnapshotIfChanged avoids cloning/compiling unchanged state for presentation
// polling. A changed snapshot retains the same detached ownership as Snapshot.
func (s *Service) SnapshotIfChanged(after uint64) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revision == after {
		return Snapshot{}, false
	}
	return s.snapshotLocked(), true
}

func (s *Service) Snapshot() Snapshot { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshotLocked() }
func (s *Service) snapshotLocked() Snapshot {
	page, _, _ := normalizeSchematicPage(&s.exportPage)
	graph := s.graph.Clone()
	compiled, _, diagnostic := s.compileGraphLocked()
	liveModes := []string{}
	if diagnostic == nil {
		liveModes = LiveInputModes(compiled.Analysis)
		if compiled.Source.Kind == "battery" || compiled.Source.Kind == "current-source" {
			liveModes = []string{}
		}
	}
	analysis := compiled.Analysis
	if analysis == "" {
		analysis = "DC operating point"
	}
	out := Snapshot{
		BlockClipboardAvailable: s.blockClipboard != nil,
		Tolerance:               s.toleranceViewLocked(),
		Spectrum:                s.spectrumViewLocked(),
		Proposal:                s.proposalViewLocked(),
		WorkspaceLayout:         s.workspaceLayout,
		InspectorSection:        s.inspectorSection,
		Viewport:                cloneViewport(s.viewport),
		PulseTimings:            s.pulseTimingsLocked(),
		Audio:                   s.audioViewLocked(),
		Shutdown:                s.shutdownViewLocked(),
		SourceAnnotations:       s.sourceAnnotationsLocked(),
		Printing:                s.printingStateLocked(),
		PrintPreview:            s.printPreviewLocked(),
		DCSweep:                 s.sweepStateLocked(),
		SchematicPage:           page,
		ProjectWork:             s.projectWorkLocked(),
		InputWork:               s.inputWorkLocked(),
		Example:                 s.exampleContextLocked(),
		DiagnosticPresentation:  s.diagnosticPresentationLocked(compiled),
		Display:                 s.circuitDisplayLocked(),
		MeasurementInspection:   s.measurementInspectionLocked(compiled, s.currentMeasurementTargetLocked()),
		PinnedReadings:          s.pinnedReadingsLocked(),
		TraceFocus:              s.traceFocusLocked(compiled),
		Limits:                  Limits(),
		SimulatedComponents:     SimulatedComponentCount(graph),
		StepWork:                s.stepWorkLocked(),
		ActuationRevision:       s.actuationRevision,
		SwitchStates:            s.switchStatesLocked(),
		SourceSampling:          sourceSamplingAdvice(graph, s.transientSettings),
		SourceWarnings:          waveformSamplingWarnings(graph, s.transientSettings),
		LCSampling:              lcSamplingAdvice(graph, s.transientSettings),
		Recording:               s.recordingViewLocked(),
		Recovery:                s.recoveryStateLocked(),
		VerificationSettings:    cloneVerificationSettings(s.verificationSettings),
		VerificationInputModes:  VerificationInputModes(analysis),
		LiveInputModes:          liveModes,
		Operations:              []OperationSummary{},
		SelectedOperation:       s.selectedOperation,
		LastOperation:           s.lastOperation,
		ExampleBrowser:          s.exampleBrowserLocked(),
		PartCategory:            browserCategory(s.partCategory),
		PartCategories:          PartCategories(),
		PartSearch:              s.partSearch,
		Theme:                   s.theme,
		Appearance:              s.appearance,
		AppearanceWork:          s.appearanceWorkLocked(),
		MatchingParts:           browserParts(s.partCategory, s.partSearch),
		Scope:                   s.scopeLocked(compiled),
		ScopeView:               s.scopeViewLocked(),
		ScopeVisible:            s.scopeVisible,
		Mixed:                   cloneMixed(s.mixed),
		NPN:                     cloneNPN(s.npn),
		CommandRevision:         s.commandRevision,
		Transient:               cloneTransient(s.transient),
		TransientSettings:       s.transientSettings,
		Document:                DocumentState{PreferencesDirty: preferencesHash(s.pins, s.displayPreferences, s.exampleID) != s.savedPreferences, SchematicDirty: contentHash(s.graph, s.transientSettings) != s.savedGraph, RecordingsDirty: s.recordingsDirtyLocked(), RecordingCount: s.recordingCountLocked(), Path: s.documentPath, Dirty: s.dirtyLocked(), FormatVersion: s.currentDocumentVersionLocked(), Generation: s.documentGeneration, RecoveredFrom: s.recoveredFrom},
		Revision:                s.revision,
		ElectricalRevision:      s.electrical,
		Circuit:                 s.circuit,
		Components:              graph.Components,
		Wires:                   graph.Wires,
		Nets:                    compiled.Nets,
		Drawing:                 s.drawing,
		Model:                   Model(),
		Selected:                s.selected,
		View:                    s.view,
		Status:                  s.status,
		Analysis:                analysis,
		Paused:                  s.paused,
		RunID:                   s.run,
		Diagnostics:             append([]Diagnostic{}, s.diagnostics...),
		CanUndo:                 len(s.undo) > 0,
		CanRedo:                 len(s.redo) > 0,
		LastActor:               s.actor,
		Closed:                  s.closed,
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
	if _, ok := graph.Component(s.selected); ok {
		var inspection *MeasurementInspection
		if target := out.MeasurementInspection.Target; target != nil && target.Kind == "object" && target.ID == s.selected {
			inspection = &out.MeasurementInspection
		}
		out.Explanation, _ = s.componentExplanationLocked(compiled, s.selected, inspection)
	}
	return out
}
func cloneSnapshot(state Snapshot) Snapshot {
	state.Tolerance = cloneTolerance(state.Tolerance)
	state.Spectrum = cloneSpectrum(state.Spectrum)
	state.Proposal = cloneEditProposal(state.Proposal)
	state.Explanation = cloneComponentExplanation(state.Explanation)
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
	state.ExampleBrowser = cloneExampleBrowser(state.ExampleBrowser)
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
