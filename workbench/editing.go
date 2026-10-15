package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

type actionError struct{ code, message string }

func (e *actionError) Error() string { return e.message }

type FitParams struct {
	Width  float64 `json:"width_px,omitempty"`
	Height float64 `json:"height_px,omitempty"`
}
type PlaceParams struct {
	ID       string `json:"id,omitempty"`
	Kind     string `json:"kind"`
	Position Point  `json:"position"`
	Value    string `json:"value,omitempty"`
}
type MoveParams struct {
	ID               string `json:"id"`
	Position         Point  `json:"position"`
	PreviousPosition *Point `json:"previous_position,omitempty"`
}
type IDParams struct {
	ID string `json:"id"`
}
type ValueParams struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}
type ConnectParams struct {
	ID   string `json:"id,omitempty"`
	From string `json:"from"`
	To   string `json:"to"`
}
type ToolParams struct {
	Tool string `json:"tool"`
	Kind string `json:"kind,omitempty"`
}
type PositionParams struct {
	Position Point `json:"position"`
}
type TerminalsParams struct {
	From string `json:"from"`
	To   string `json:"to"`
}
type DraftParams struct {
	Value string `json:"value"`
}

func (s *Service) nextID(prefix string) string {
	for i := 1; ; i++ {
		id := fmt.Sprintf("%s%d", prefix, i)
		if !s.usedIDs[id] && !s.graph.HasObject(id) {
			return id
		}
	}
}
func validatePointParam(raw json.RawMessage, field string) error {
	var object map[string]json.RawMessage
	json.Unmarshal(raw, &object)
	return params(object[field], &Point{}, "x", "y")
}
func (s *Service) normalizeDrawingLocked() {
	if s.selected != "" && !s.graph.HasObject(s.selected) {
		s.selected = ""
	}
	for _, ref := range []*string{&s.drawing.WireStart, &s.drawing.From, &s.drawing.To} {
		if *ref != "" {
			if _, _, ok := s.graph.Terminal(*ref); !ok {
				*ref = ""
			}
		}
	}
	if c, ok := s.graph.Component(s.selected); ok && c.Unit != "" {
		s.drawing.Value = FormatValue(c.Value, c.Unit)
	} else if c, ok := s.graph.Component(s.selected); ok && c.Kind == "net-label" {
		s.drawing.Value = c.Label
	} else {
		s.drawing.Value = ""
	}
}
func (s *Service) connectLocked(p ConnectParams) error {
	if len(s.graph.Wires) >= MaxWires {
		return fmt.Errorf("drawing supports at most %d wires", MaxWires)
	}
	if _, _, ok := s.graph.Terminal(p.From); !ok {
		return fmt.Errorf("unknown terminal %q", p.From)
	}
	if _, _, ok := s.graph.Terminal(p.To); !ok {
		return fmt.Errorf("unknown terminal %q", p.To)
	}
	if p.From == p.To {
		return fmt.Errorf("choose two different terminals")
	}
	for _, w := range s.graph.Wires {
		if w.From == p.From && w.To == p.To || w.From == p.To && w.To == p.From {
			return fmt.Errorf("these terminals are already connected")
		}
	}
	if p.ID == "" {
		p.ID = s.nextID("W")
	}
	if !ValidID(p.ID) || s.usedIDs[p.ID] || s.graph.HasObject(p.ID) {
		return fmt.Errorf("wire ID is invalid or already used in this session")
	}
	s.graph.Wires = append(s.graph.Wires, Wire{ID: p.ID, From: p.From, To: p.To, Connected: true})
	s.usedIDs[p.ID] = true
	s.drawing.WireStart = ""
	s.drawing.From = ""
	s.drawing.To = ""
	return nil
}
func (s *Service) applyLocked(command string, raw json.RawMessage) error {
	switch command {
	case "simulation.sampling.fit":
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
		return s.fitWaveformSamplingLocked()
	case "source.sine.set":
		var p SineParams
		if err := params(raw, &p, "id", "waveform"); err != nil {
			return err
		}
		if err := paramsNested(raw, "waveform", &SineWaveform{}, "amplitude_v", "offset_v", "frequency_hz", "phase_degrees"); err != nil {
			return err
		}
		return s.setSineLocked(p)
	case "source.pulse.set":
		var p PulseParams
		if err := params(raw, &p, "id", "waveform"); err != nil {
			return err
		}
		if err := paramsNested(raw, "waveform", &PulseWaveform{}, "high_v", "low_v", "high_s", "low_s", "rise_s", "fall_s"); err != nil {
			return err
		}
		return s.setPulseLocked(p)
	case "source.pulse.timing.set":
		return s.setPulseTimingLocked(raw)
	case "simulation.pulse.timing.set":
		return s.setIdealPulseTimingLocked(raw)
	case "potentiometer.wiper.set":
		var p WiperParams
		if err := params(raw, &p, "id", "position"); err != nil {
			return err
		}
		return s.setWiperLocked(p)
	case "switch.set":
		var p SwitchParams
		if err := params(raw, &p, "id", "closed"); err != nil {
			return err
		}
		return s.setSwitchLocked(p)
	case "wire.junction.insert":
		var p JunctionInsertParams
		if err := params(raw, &p, "wire_ids"); err != nil {
			return err
		}
		if p.Position != nil {
			if err := validatePointParam(raw, "position"); err != nil {
				return err
			}
		}
		return s.insertJunctionLocked(p)
	case "recording.timing.set":
		return s.setRecordingTimingLocked(raw)
	case "recording.compare.set":
		return s.setRecordingComparisonLocked(raw)
	case "recording.keep.set":
		var p RecordingKeepParams
		if err := params(raw, &p, "id", "keep"); err != nil {
			return err
		}
		return s.setRecordingKeepLocked(p)
	case "verification.import", "verification.import.start":
		var p RecordingImportParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.startRecordingImportLocked(p)
	case "recording.view.set", "recording.seek", "recording.step", "recording.play", "recording.speed.set", "recording.pause", "recording.window.set", "recording.probes.set", "recording.cursor.set", "recording.cursor.select", "recording.cursor.clear":
		return s.recordingCommandLocked(command, raw)
	case "app.close.acknowledge":
		return s.acknowledgeShutdownLocked(raw)
	case "recovery.read.start":
		return s.startRecoveryReadLocked(raw)
	case "recovery.read.cancel":
		return s.cancelRecoveryReadLocked(raw)
	case "recovery.work.start", "recovery.configure", "recovery.configure.start", "recovery.scan", "recovery.checkpoint", "recovery.select", "recovery.restore", "recovery.restore.start", "recovery.dismiss", "recovery.dismiss.start":
		return s.recoveryCommandLocked(command, raw)
	case "sweep.import", "sweep.import.start":
		var p RecordingImportParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.startSweepImportLocked(p)
	case "sweep.export":
		var p VerificationExportParams
		if err := params(raw, &p, "id", "path", "format"); err != nil {
			return err
		}
		if p.Format != "json" && p.Format != "csv" {
			return fmt.Errorf("sweep export format must be json or csv")
		}
		return s.startExportLocked(ExportStartParams{Kind: "sweep-" + p.Format, ID: p.ID, Path: p.Path, Overwrite: p.Overwrite, ExpectedGeneration: p.ExpectedGeneration})
	case "verification.export":
		var p VerificationExportParams
		if err := params(raw, &p, "id", "path", "format"); err != nil {
			return err
		}
		if p.Format != "json" && p.Format != "csv" {
			return fmt.Errorf("verification format must be json or csv")
		}
		return s.startExportLocked(ExportStartParams{Kind: "verification-" + p.Format, ID: p.ID, Path: p.Path, Overwrite: p.Overwrite, ExpectedGeneration: p.ExpectedGeneration})
	case "verification.settings.set":
		var p VerificationSettings
		if err := params(raw, &p, "duration_s", "sample_step_s", "input_mode", "capacitor_initials"); err != nil {
			return err
		}
		if err := validateCapacitorInitialFields(raw); err != nil {
			return err
		}
		if err := validateVerificationSettings(p, s.graph); err != nil {
			return err
		}
		s.verificationSettings = cloneVerificationSettings(p)
		return nil
	case "simulation.verify":
		var p VerifyParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if err := validateCapacitorInitialFields(raw); err != nil {
			return err
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		if _, present := fields["probe_ids"]; present && p.ProbeIDs == nil {
			return fmt.Errorf("probe_ids must be an array; omit to inherit defaults")
		}
		return s.startVerificationLocked(p)
	case "operation.cancel", "operation.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if command == "operation.cancel" {
			return s.cancelOperationLocked(p.ID)
		}
		if p.ID != "" && s.operations[p.ID] == nil {
			return &actionError{"not-found", "Operation not found or no longer retained"}
		}
		s.selectRecordingLocked(p.ID)
		return nil
	case "sweep.settings.set":
		var p SweepSettings
		if err := params(raw, &p, "component_id", "parameter", "start", "stop", "points"); err != nil {
			return err
		}
		if _, err := buildSweepPlan(s.graph, p, false); err != nil {
			return err
		}
		store := s.ensureSweepsLocked()
		cfg := cloneSweepSettings(p)
		store.settings = &cfg
		return nil
	case "sweep.start":
		var p SweepStartParams
		if err := params(raw, &p); err != nil {
			return err
		}
		return s.startDCSweepLocked(p)
	case "sweep.cancel":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		return s.cancelSweepLocked(p.ID)
	case "sweep.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if s.sweeps == nil || s.sweeps.operations[p.ID] == nil {
			return &actionError{"not-found", "Choose a retained sweep"}
		}
		store := s.sweeps
		s.sweepViewerRevision++
		store.selected = p.ID
		store.cursor = nil
		if record := store.operations[p.ID].record; record != nil {
			store.traceID = record.Result.Probes[0].ID
		}
		return nil
	case "sweep.visibility.set":
		var p SweepVisibilityParams
		if err := params(raw, &p, "visible"); err != nil {
			return err
		}
		store := s.ensureSweepsLocked()

		s.sweepViewerRevision++
		store.visible = p.Visible
		if p.Visible {
			s.dismissPrintPreviewLocked()
		}
		return nil
	case "sweep.trace.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if s.sweeps == nil {
			return &actionError{"unavailable", "Complete a sweep first"}
		}
		job := s.sweeps.operations[s.sweeps.selected]
		if job == nil || job.record == nil {
			return &actionError{"not-ready", "Complete a sweep first"}
		}
		for _, probe := range job.record.Result.Probes {
			if probe.ID == p.ID {
				s.sweepViewerRevision++
				s.sweeps.traceID = p.ID
				return nil
			}
		}
		return fmt.Errorf("choose a captured sweep probe")
	case "sweep.cursor.set":
		var p SweepCursorParams
		if err := params(raw, &p); err != nil {
			return err
		}
		// The index is required, but explicit null means clear. The ordinary
		// required-field decoder deliberately refuses null for other commands.
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return err
		}
		if _, present := fields["index"]; !present {
			return fmt.Errorf("required field %q is missing", "index")
		}
		if s.sweeps == nil {
			return &actionError{"not-ready", "Complete a sweep first"}
		}
		if p.OperationID != "" && p.OperationID != s.sweeps.selected {
			return &actionError{"conflict", "Sweep selection changed; inspect before setting a cursor"}
		}
		job := s.sweeps.operations[s.sweeps.selected]
		if job == nil || job.record == nil {
			return &actionError{"not-ready", "Complete a sweep first"}
		}
		if p.Index != nil && (*p.Index < 0 || *p.Index >= len(job.record.Result.Points)) {
			return fmt.Errorf("cursor index is outside sweep points")
		}
		s.sweepViewerRevision++
		s.sweeps.cursor = nil
		if p.Index != nil {
			v := *p.Index
			s.sweeps.cursor = &v
		}
		return nil
	case "sweep.keep.set":
		var p RecordingKeepParams
		if err := params(raw, &p, "id", "keep"); err != nil {
			return err
		}
		if s.sweeps == nil || s.sweeps.operations[p.ID] == nil || s.sweeps.operations[p.ID].record == nil {
			return &actionError{"not-ready", "Choose a completed sweep"}
		}
		s.sweeps.operations[p.ID].keep = p.Keep
		return nil
	case "print.printers.refresh", "print.destination.set", "print.submit", "print.job.refresh", "print.job.cancel", "print.work.cancel":
		return s.printCommandLocked(command, raw)
	case "print.preview.start":
		var p PreviewParams
		if err := params(raw, &p); err != nil {
			return err
		}
		return s.startPrintPreviewLocked(p)
	case "print.preview.cancel", "print.preview.close":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		return s.cancelPrintPreviewLocked(p, command == "print.preview.close")
	case "export.start":
		var p ExportStartParams
		if err := params(raw, &p, "kind", "path"); err != nil {
			return err
		}
		return s.startExportLocked(p)
	case "export.page.set":
		var p SchematicPage
		if err := params(raw, &p); err != nil {
			return err
		}
		page, _, err := normalizeSchematicPage(&p)
		if err != nil {
			return err
		}
		s.exportPage = page
		return nil
	case "export.schematic.pdf":
		var p PDFExportParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.startExportLocked(ExportStartParams{Kind: "schematic-pdf", Path: p.Path, Overwrite: p.Overwrite, ExpectedGeneration: p.ExpectedGeneration, Page: p.Page})
	case "export.schematic.png":
		var p PNGExportParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.startExportLocked(ExportStartParams{Kind: "schematic-png", Path: p.Path, Overwrite: p.Overwrite, ExpectedGeneration: p.ExpectedGeneration, Page: p.Page, DPI: p.DPI})
	case "export.scope.csv", "export.schematic.svg":
		var p ExportParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		kind := "schematic-svg"
		if command == "export.scope.csv" {
			kind = "scope-csv"
		}
		return s.startExportLocked(ExportStartParams{Kind: kind, Path: p.Path, Overwrite: p.Overwrite, ExpectedGeneration: p.ExpectedGeneration})
	case "scope.window.set":
		var p ScopeWindow
		if err := params(raw, &p, "mode"); err != nil {
			return err
		}
		if p.Mode == "fixed" {
			if err := params(raw, &p, "mode", "start_s", "end_s"); err != nil {
				return err
			}
		}
		return s.setScopeWindowLocked(p)
	case "parts.search.set":
		var p SearchParams
		if err := params(raw, &p, "query"); err != nil {
			return err
		}
		if len(p.Query) > 128 {
			return fmt.Errorf("search is limited to 128 bytes")
		}
		s.partSearch = p.Query
		return nil
	case "component.duplicate":
		var p DuplicateParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if p.Position != nil {
			if err := validatePointParam(raw, "position"); err != nil {
				return err
			}
		}
		return s.duplicateLocked(p)
	case "edit.batch":
		var p BatchParams
		if err := params(raw, &p, "name", "edits"); err != nil {
			return err
		}
		for _, edit := range p.Edits {
			if edit.Command == "" || edit.Params == nil {
				return fmt.Errorf("every batch edit needs command and params")
			}
		}
		return s.batchLocked(p)
	case "example.load":
		var p ExampleLoadParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
			return err
		}
		return s.loadExampleLocked(p.ID, p.DiscardChanges)
	case "example.current.reset":
		var p ExampleResetParams
		if err := params(raw, &p); err != nil {
			return err
		}
		return s.resetCurrentExampleLocked(p)
	case "scope.visibility.set":
		var p ScopeVisibilityParams
		if err := params(raw, &p, "visible"); err != nil {
			return err
		}
		s.scopeVisible = p.Visible
		return nil
	case "diagnostics.details.set":
		var p DiagnosticDetailsParams
		if err := params(raw, &p, "visible"); err != nil {
			return err
		}
		s.diagnosticDetails = p.Visible
		return nil
	case "appearance.configure":
		var p AppearanceConfigureParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.configureAppearanceLocked(p)
	case "appearance.theme.set":
		var p struct {
			Theme string `json:"theme"`
		}
		if err := params(raw, &p, "theme"); err != nil {
			return err
		}
		if !validTheme(p.Theme) {
			return fmt.Errorf("theme must be light, dark, or system")
		}
		// An explicit choice, including the current value, supersedes a pending
		// profile load. Retried commands are deduplicated before reaching here.
		s.theme = p.Theme
		s.appearanceChangedLocked()
		return nil
	case "display.preferences.set":
		p, err := decodeDisplayPreferences(raw)
		if err != nil {
			return err
		}
		s.displayPreferences = p
		return nil
	case "measurement.select":
		return s.selectMeasurementLocked(raw)
	case "measurement.pin.add", "measurement.pin.move", "measurement.pin.remove":
		return s.pinCommandLocked(command, raw)
	case "scope.trace.select", "scope.probe.remove", "scope.probes.set", "scope.cursor.set", "scope.cursor.select", "scope.cursor.clear":
		return s.scopeCommandLocked(command, raw)
	case "capacitor.initial.set":
		var p ValueParams
		if err := params(raw, &p, "id", "value"); err != nil {
			return err
		}
		return s.capacitorInitialLocked(p)
	case "simulation.transient.set", "simulation.input.set", "simulation.step", "simulation.step.start", "simulation.work.cancel":
		return s.transientCommandLocked(command, raw)
	case "project.work.cancel":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if !s.activeProjectWorkLocked() || s.projectJob.work.ID != p.ID {
			return &actionError{"operation-unavailable", "Choose the active project work to cancel"}
		}
		if s.projectJob.publishing {
			return &actionError{"operation-committing", "The file operation is publishing atomically and can no longer be canceled"}
		}
		s.cancelProjectWorkLocked("canceled")
		return nil
	case "project.save", "project.save.start":
		var p SaveParams
		if err := params(raw, &p); err != nil {
			return err
		}
		return s.startProjectSaveLocked(p)
	case "project.load", "project.load.start":
		var p LoadParams
		if err := params(raw, &p, "path"); err != nil {
			return err
		}
		return s.startProjectLoadLocked(p)
	case "component.place":
		var p PlaceParams
		if err := params(raw, &p, "kind", "position"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "position"); err != nil {
			return err
		}
		if len(s.graph.Components) >= MaxComponents {
			return fmt.Errorf("drawing supports at most %d components", MaxComponents)
		}
		if p.ID == "" {
			prefix := map[string]string{"dc-source": "V", "resistor": "R", "led": "D", "diode": "D", "potentiometer": "P", "ground": "GND", "capacitor": "C", "pulse-source": "V", "sine-source": "V", "finite-pulse-source": "V", "npn": "Q", "junction": "J", "net-label": "L", "switch": "S"}[p.Kind]
			if prefix == "" {
				return fmt.Errorf("unknown part kind")
			}
			p.ID = s.nextID(prefix)
		}
		if s.usedIDs[p.ID] || s.graph.HasObject(p.ID) {
			return fmt.Errorf("component ID is already used in this session")
		}
		c, err := MakeComponent(p.ID, p.Kind, p.Position, p.Value)
		if err != nil {
			return err
		}
		for _, other := range s.graph.Components {
			if other.Position == c.Position {
				return fmt.Errorf("this grid position is occupied; choose another")
			}
		}
		s.graph.Components = append(s.graph.Components, c)
		s.usedIDs[p.ID] = true
		s.selected = c.ID
		s.drawing.Tool = "select"
		return nil
	case "component.set":
		var p ValueParams
		if err := params(raw, &p, "id", "value"); err != nil {
			return err
		}
		for i, c := range s.graph.Components {
			if c.ID == p.ID {
				next, err := SetValue(c, p.Value)
				if err != nil {
					return err
				}
				s.graph.Components[i] = next
				return nil
			}
		}
		return fmt.Errorf("unknown component ID")
	case "component.move":
		var p MoveParams
		if err := params(raw, &p, "id", "position"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "position"); err != nil {
			return err
		}
		position, err := snapPoint(p.Position)
		if err != nil {
			return err
		}
		for _, other := range s.graph.Components {
			if other.ID != p.ID && other.Position == position {
				return fmt.Errorf("this grid position is occupied")
			}
		}
		for i, c := range s.graph.Components {
			if c.ID == p.ID {
				if p.PreviousPosition != nil && c.Position != *p.PreviousPosition {
					return &actionError{"conflict", "component moved during this gesture; inspect its current position"}
				}
				s.graph.Components[i].Position = position
				return nil
			}
		}
		return fmt.Errorf("unknown component ID")
	case "component.rotate":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		for i, c := range s.graph.Components {
			if c.ID == p.ID {
				s.graph.Components[i].Rotation = (c.Rotation + 90) % 360
				return nil
			}
		}
		return fmt.Errorf("unknown component ID")
	case "object.delete":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if !s.graph.HasObject(p.ID) {
			return fmt.Errorf("unknown object ID")
		}
		graph := s.graph.Clone()
		graph.Components = graph.Components[:0]
		for _, c := range s.graph.Components {
			if c.ID != p.ID {
				graph.Components = append(graph.Components, c)
			}
		}
		graph.Wires = graph.Wires[:0]
		for _, w := range s.graph.Wires {
			from, _, _ := s.graph.Terminal(w.From)
			to, _, _ := s.graph.Terminal(w.To)
			if w.ID != p.ID && from.ID != p.ID && to.ID != p.ID {
				graph.Wires = append(graph.Wires, w)
			}
		}
		s.graph = graph
		return nil
	case "component.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if p.ID != "" && !s.graph.HasObject(p.ID) {
			return fmt.Errorf("unknown object ID")
		}
		s.selected = p.ID
		s.measurementTarget = nil
		s.normalizeDrawingLocked()
		return nil
	case "wire.connect":
		var p ConnectParams
		if err := params(raw, &p, "from", "to"); err != nil {
			return err
		}
		return s.connectLocked(p)
	case "wire.begin":
		var p struct {
			Terminal string `json:"terminal"`
		}
		if err := params(raw, &p, "terminal"); err != nil {
			return err
		}
		if _, _, ok := s.graph.Terminal(p.Terminal); !ok {
			return fmt.Errorf("unknown terminal")
		}
		s.drawing.Tool = "wire"
		s.drawing.WireStart = p.Terminal
		return nil
	case "drawing.tool.set":
		var p ToolParams
		if err := params(raw, &p, "tool"); err != nil {
			return err
		}
		if p.Tool != "select" && p.Tool != "place" && p.Tool != "wire" && p.Tool != "pan" {
			return fmt.Errorf("tool must be select, place, wire, or pan")
		}
		kind := s.drawing.Kind
		if p.Kind != "" {
			kind = p.Kind
		}
		valid := false
		for _, part := range Library() {
			if part.Kind == kind {
				valid = true
			}
		}
		if !valid {
			return fmt.Errorf("unknown part kind")
		}
		if p.Tool == "place" && kind == "net-label" {
			s.drawing.Value = ""
		}
		s.drawing.Tool = p.Tool
		s.drawing.Kind = kind
		s.drawing.WireStart = ""
		return nil
	case "drawing.position.set":
		var p PositionParams
		if err := params(raw, &p, "position"); err != nil {
			return err
		}
		if err := validatePointParam(raw, "position"); err != nil {
			return err
		}
		position, err := snapPoint(p.Position)
		if err != nil {
			return err
		}
		s.drawing.Position = position
		return nil
	case "wire.terminals.set":
		var p TerminalsParams
		if err := params(raw, &p, "from", "to"); err != nil {
			return err
		}
		for _, ref := range []string{p.From, p.To} {
			if ref != "" {
				if _, _, ok := s.graph.Terminal(ref); !ok {
					return fmt.Errorf("unknown terminal %q", ref)
				}
			}
		}
		s.drawing.From = p.From
		s.drawing.To = p.To
		return nil
	case "editor.value.set":
		var p DraftParams
		if err := params(raw, &p, "value"); err != nil {
			return err
		}
		if len(p.Value) > 128 {
			return fmt.Errorf("value entry is limited to 128 bytes")
		}
		s.drawing.Value = p.Value
		return nil
	case "circuit.connection.set":
		// Compatibility for the original prepared fixture; new drawing uses wire.connect/object.delete.
		var p struct {
			Connected bool `json:"connected"`
		}
		if err := params(raw, &p, "connected"); err != nil {
			return err
		}
		if _, _, ok := s.graph.Terminal("R1.2"); !ok {
			return fmt.Errorf("legacy fixture terminal R1.2 is absent")
		}
		if _, _, ok := s.graph.Terminal("D1.anode"); !ok {
			return fmt.Errorf("legacy fixture terminal D1.anode is absent")
		}
		for i, w := range s.graph.Wires {
			if w.From == "R1.2" && w.To == "D1.anode" || w.From == "D1.anode" && w.To == "R1.2" {
				if p.Connected {
					return nil
				}
				s.graph.Wires = append(s.graph.Wires[:i], s.graph.Wires[i+1:]...)
				return nil
			}
		}
		if !p.Connected {
			return nil
		}
		return s.connectLocked(ConnectParams{From: "R1.2", To: "D1.anode"})
	case "view.fit":
		p := FitParams{Width: 650, Height: 400}
		if err := params(raw, &p); err != nil {
			return err
		}
		if !finite(p.Width) || !finite(p.Height) || p.Width < 64 || p.Height < 64 || p.Width > 100000 || p.Height > 100000 {
			return fmt.Errorf("fit dimensions must be 64–100000 pixels")
		}
		view, err := s.fitViewLocked(p)
		if err != nil {
			return err
		}
		s.view = view
		return nil
	case "selection.navigate":
		var p NavigateParams
		if err := params(raw, &p, "direction"); err != nil {
			return err
		}
		return s.navigateLocked(p)
	case "view.selection.center":
		var p CenterSelectionParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if p.ID != "" && p.ID != s.selected {
			return &actionError{"conflict", "Selected object changed"}
		}
		center, err := s.objectCenterLocked(s.selected)
		if err != nil {
			return err
		}
		s.view.Center = center
		return nil
	case "view.pan":
		var p struct {
			DX float64 `json:"dx"`
			DY float64 `json:"dy"`
		}
		if err := params(raw, &p, "dx", "dy"); err != nil {
			return err
		}
		x, y := s.view.Center.X+p.DX, s.view.Center.Y+p.DY
		if !finite(x) || !finite(y) || math.Abs(x) > 1e6 || math.Abs(y) > 1e6 {
			return fmt.Errorf("view center exceeds ±1000000 units")
		}
		s.view.Center = Point{x, y}
		return nil
	case "view.zoom":
		var p struct {
			Factor float64 `json:"factor"`
			Anchor *Point  `json:"anchor"`
		}
		if err := params(raw, &p, "factor"); err != nil {
			return err
		}
		if p.Anchor != nil {
			if err := validatePointParam(raw, "anchor"); err != nil {
				return err
			}
		}
		if !finite(p.Factor) || p.Factor < .25 || p.Factor > 4 {
			return fmt.Errorf("zoom factor must be 0.25–4")
		}
		anchor := s.view.Center
		if p.Anchor != nil {
			anchor = *p.Anchor
		}
		if !finite(anchor.X) || !finite(anchor.Y) || math.Abs(anchor.X) > 1e6 || math.Abs(anchor.Y) > 1e6 {
			return fmt.Errorf("invalid zoom anchor")
		}
		zoom := math.Max(.01, math.Min(4, s.view.Zoom*p.Factor))
		factor := zoom / s.view.Zoom
		center := Point{anchor.X + (s.view.Center.X-anchor.X)/factor, anchor.Y + (s.view.Center.Y-anchor.Y)/factor}
		if math.Abs(center.X) > 1e6 || math.Abs(center.Y) > 1e6 {
			return fmt.Errorf("zoom exceeds view bounds")
		}
		s.view = View{center, zoom}
		return nil
	}
	if command != "project.clear" && command != "example.reset" && command != "example.rc" && command != "example.npn" && command != "example.npn-pulse" && command != "example.flasher" && command != "example.switch-npn" && command != "example.rectifier" && command != "app.close" && command != "app.close.start" && command != "edit.undo" && command != "edit.redo" {
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
	}
	switch command {
	case "project.clear":
		var p ReplaceParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
			return err
		}
		if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
			return err
		}
		s.detachDocumentLocked()
		s.graph = EmptyGraph()
		s.selected = ""
		s.drawing = DefaultDrawing()
		s.view = View{Zoom: 1}
	case "example.reset", "example.rc", "example.npn", "example.npn-pulse", "example.flasher", "example.switch-npn", "example.rectifier":
		var p ReplaceParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
			return err
		}
		if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
			return err
		}
		ids := map[string]string{"example.reset": "led", "example.rc": "rc", "example.npn": "npn", "example.npn-pulse": "blink", "example.flasher": "flasher", "example.switch-npn": "switch-npn", "example.rectifier": "rectifier"}
		return s.loadExampleLocked(ids[command], p.DiscardChanges)
	case "edit.undo":
		if len(s.undo) == 0 {
			return &actionError{"unavailable", "nothing to undo"}
		}
		s.cancelInputWorkLocked("superseded")
		h := s.historyLocked()
		tip := s.undo[len(s.undo)-1]
		h.Name = tip.Name
		h.Actor = tip.Actor
		h.CommandRevision = tip.CommandRevision
		s.redo = append(s.redo, h)
		s.restoreLocked(s.undo[len(s.undo)-1])
		s.undo = s.undo[:len(s.undo)-1]
	case "edit.redo":
		if len(s.redo) == 0 {
			return &actionError{"unavailable", "nothing to redo"}
		}
		s.cancelInputWorkLocked("superseded")
		h := s.historyLocked()
		tip := s.redo[len(s.redo)-1]
		h.Name = tip.Name
		h.Actor = tip.Actor
		h.CommandRevision = tip.CommandRevision
		s.undo = append(s.undo, h)
		s.restoreLocked(s.redo[len(s.redo)-1])
		s.redo = s.redo[:len(s.redo)-1]
	case "simulation.pause":
		s.cancelInputWorkLocked("canceled")
		s.cancelStepLocked("canceled")
		s.paused = true
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.status = "paused"
	case "simulation.resume":
		s.cancelStepLocked("superseded")
		if !s.paused && (s.status == "running" || s.status == "updating") {
			return nil // already running; preserve the existing clock and work
		}
		s.paused = false
		if s.transient != nil && !s.transient.Stale {
			c, _, d := s.graph.Compile()
			if d == nil && IsTransientAnalysis(c.Analysis) {
				if s.transient.PendingMode != "" {
					var err error
					if IsNetworkAnalysis(c.Analysis) {
						err = s.startInputWorkLocked(s.transient.PendingMode, nil, s.graph, "")
					} else {
						err = s.applyTransientInputLocked(c, s.transient.PendingMode)
					}
					if err != nil {
						s.paused = true
						return err
					}
				}
				s.status = "running"
				s.launchTransientLocked(c)
				break
			}
		}
		s.startLocked()
	case "simulation.restart":
		s.restorePaused = s.paused
		s.startLocked()
	case "drawing.cancel":
		s.drawing.Tool = "select"
		s.drawing.WireStart = ""
	case "app.close", "app.close.start":
		var p ReplaceParams
		if err := params(raw, &p); err != nil {
			return err
		}
		if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
			return err
		}
		if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
			return err
		}
		s.discardRecoveryOnClose = p.DiscardChanges
		s.closeLocked()
	default:
		return fmt.Errorf("unknown command")
	}
	return nil
}
