package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dshills/elab/internal/schematicpdf"
)

// ExportStartParams selects the same capture/encoding used by native exports.
type ExportStartParams struct {
	PageIndex          *int           `json:"page_index,omitempty"`
	PreviewID          string         `json:"preview_id,omitempty"`
	DPI                int            `json:"dpi,omitempty"`
	Page               *SchematicPage `json:"page,omitempty"`
	Kind               string         `json:"kind"`
	ID                 string         `json:"id,omitempty"`
	Path               string         `json:"path"`
	Overwrite          bool           `json:"overwrite,omitempty"`
	ExpectedGeneration *uint64        `json:"expected_document_generation,omitempty"`
}
type exportCapture struct {
	protectedPath string
	previewData   []byte
	annotations   []SourceAnnotation
	settings      TransientSettings
	page          schematicpdf.Page
	kind          string
	generation    uint64
	overwrite     bool
	graph         Graph
	display       CircuitDisplay
	readings      []PinnedReading
	trace         *Transient
	scope         Scope
	record        *verificationRecord
	sweep         *SweepRecord
	artifact      Artifact
}
type projectExportFunc func(context.Context, exportCapture, func(string, int)) (*preparedFile, error)

func (s *Service) startExportLocked(p ExportStartParams) error {
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	if path == s.documentPath {
		return fmt.Errorf("export must use a different path from the schematic")
	}
	capture := exportCapture{kind: p.Kind, generation: s.documentGeneration, overwrite: p.Overwrite, artifact: Artifact{Path: path, RunID: s.run, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration}}
	if p.Kind == "schematic-svg" || p.Kind == "schematic-pdf" || p.Kind == "schematic-png" {
		capture.annotations = s.sourceAnnotationsLocked()
		capture.settings = s.transientSettings
	}
	command := ""
	if p.Page != nil && p.Kind != "schematic-pdf" && p.Kind != "schematic-png" {
		return fmt.Errorf("page settings apply only to schematic PDF/PNG")
	}
	if p.DPI != 0 && p.Kind != "schematic-png" {
		return fmt.Errorf("dpi applies only to schematic PNG")
	}
	if p.PageIndex != nil && (p.Kind != "schematic-png" || *p.PageIndex < 0) {
		return fmt.Errorf("page_index is a nonnegative PNG page selection")
	}
	if p.PreviewID != "" && (p.Kind != "schematic-png" || p.Page != nil || p.PageIndex != nil || p.DPI != 0) {
		return fmt.Errorf("preview_id exports an exact captured PNG; no page/index/dpi override")
	}
	if p.PreviewID != "" {
		if s.preview == nil || s.preview.view.ID != p.PreviewID || !s.preview.view.Visible || s.preview.view.Status != "completed" || len(s.preview.data) == 0 {
			return &actionError{"not-ready", "Choose the exact visible completed preview page"}
		}
		if p.ID != "" {
			return fmt.Errorf("preview PNG does not use a recording ID")
		}
		capture = s.preview.capture
		capture.generation = s.documentGeneration
		capture.overwrite = p.Overwrite
		capture.artifact.Path = path
		capture.artifact.OperationID = p.PreviewID
		capture.previewData = s.preview.data // Immutable owned bytes, retained even if preview closes.
		command = "export.schematic.png"
	} else {
		switch p.Kind {
		case "schematic-pdf", "schematic-png":
			if !schematicpdf.Supported() {
				return &actionError{"unsupported", "schematic PDF/PNG requires the macOS CoreGraphics build"}
			}
			if p.ID != "" {
				return fmt.Errorf("schematic PDF/PNG does not use a recording ID")
			}
			settings := p.Page
			if settings == nil {
				settings = &s.exportPage
			}
			page, layout, err := normalizeSchematicPage(settings)
			if err != nil {
				return err
			}
			command = "export.schematic.pdf"
			capture.artifact.Format = "pdf"
			if p.Kind == "schematic-png" {
				dpi := p.DPI
				if dpi == 0 {
					dpi = 150
				}
				if dpi < 72 || dpi > 300 {
					return fmt.Errorf("dpi must be 72–300, or omitted for 150")
				}
				if p.PageIndex != nil {
					index := *p.PageIndex
					layout.PageIndex = &index
					capture.artifact.PageIndex = &index
				}
				layout.RasterDPI = dpi
				capture.artifact.DPI = dpi
				command = "export.schematic.png"
				capture.artifact.Format = "png"
			}
			capture.artifact.Page = &page
			capture.page = layout
			capture.graph = s.graph.Clone()
			capture.display = CircuitDisplay{Preferences: DefaultDisplayPreferences()}
			if page.Measurements {
				capture.display = cloneCircuitDisplay(s.circuitDisplayLocked())
				capture.display.Animate = false
				capture.readings = clonePinnedReadings(s.pinnedReadingsLocked())
			}
		case "schematic-svg":
			if p.ID != "" {
				return fmt.Errorf("schematic export does not use a recording ID")
			}
			command = "export.schematic.svg"
			capture.artifact.Format = "svg"
			capture.graph = s.graph.Clone()
			capture.display = cloneCircuitDisplay(s.circuitDisplayLocked())
			capture.display.Animate = false
			capture.readings = clonePinnedReadings(s.pinnedReadingsLocked())
		case "audio-wav":
			if s.audio == nil || s.audio.view.ID != p.ID || s.audio.view.Status != "completed" || !s.audioViewLocked().Current {
				return fmt.Errorf("export needs the exact current completed audio preview")
			}
			command = "export.audio.wav"
			capture.previewData = append([]byte(nil), s.audio.wav...)
			capture.artifact.Format = "wav"
			capture.artifact.OperationID = p.ID
			capture.artifact.RunID = 0
		case "scope-csv":
			if p.ID != "" {
				return fmt.Errorf("live scope export does not use a recording ID")
			}
			command = "export.scope.csv"
			capture.artifact.Format = "csv"
			compiled, _, _ := s.graph.Compile()
			scope := s.scopeLocked(compiled)
			t := s.transient
			if t == nil || t.Stale || !scope.WindowValid || len(scope.Traces) == 0 || len(scope.Traces[0].Points) == 0 {
				return &actionError{"unavailable", "CSV needs selected traces in a fresh retained scope interval"}
			}
			capture.trace = cloneTransient(t)
			capture.scope = cloneScope(scope)
			capture.artifact.RunID = t.RunID
			capture.artifact.ElectricalRevision = t.ElectricalRevision
		case "sweep-json", "sweep-csv":
			command = "sweep.export"
			if s.sweeps == nil || s.sweeps.operations[p.ID] == nil {
				return &actionError{"not-found", "Choose a retained sweep"}
			}
			job := s.sweeps.operations[p.ID]
			if job.record == nil {
				return &actionError{"not-ready", "Export requires a completed sweep"}
			}
			capture.sweep = cloneSweepRecord(job.record)
			capture.artifact.Format = "dc-sweep.json"
			if p.Kind == "sweep-csv" {
				capture.artifact.Format = "dc-sweep.csv"
			}
			capture.artifact.OperationID = p.ID
			capture.artifact.RunID = 0
			capture.artifact.ElectricalRevision = job.metadata.ElectricalRevision
		case "verification-json", "verification-csv":
			command = "verification.export"
			capture.artifact.Format = "verification.json"
			if p.Kind == "verification-csv" {
				capture.artifact.Format = "verification.csv"
			}
			operation, ok := s.operations[p.ID]
			if !ok {
				return &actionError{"not-found", "Operation not found or no longer retained"}
			}
			if operation.record == nil {
				return &actionError{"not-ready", "Export requires a completed verification"}
			}
			// Record rows are immutable. Copy metadata and retain that independent asset
			// even if its live session entry is subsequently removed/replaced.
			capture.record = &verificationRecord{metadata: cloneVerificationMetadata(operation.record.metadata), rows: operation.record.rows}
			capture.artifact.OperationID = p.ID
			capture.artifact.RunID = 0
			capture.artifact.ElectricalRevision = operation.record.metadata.ElectricalRevision
		default:
			return fmt.Errorf("export kind must be schematic-svg, schematic-pdf, schematic-png, scope-csv, verification-json, verification-csv, sweep-json, sweep-csv or audio-wav")
		}
	}
	// Keep protection tied to this admission even for an older preview capture.
	// Filesystem alias checks run in the worker, outside the editor mutex.
	capture.protectedPath = s.documentPath
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), export: &capture, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: command, Status: "queued", Phase: "queued", Cancelable: true, Path: path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	prepare, publish := s.projectExport, s.projectPublish
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runProjectExport(job, prepare, publish)
	return nil
}
func prepareProjectExport(ctx context.Context, capture exportCapture, progress func(string, int)) (*preparedFile, error) {
	if err := checkExportDestination(capture.artifact.Path, capture.protectedPath); err != nil {
		return nil, err
	}
	progress("encoding", 0)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var data []byte
	var err error
	switch capture.kind {
	case "schematic-pdf", "schematic-png":
		if capture.previewData != nil {
			data = capture.previewData
		} else {
			data, err = renderSchematicPage(ctx, capture)
		}
	case "sweep-json":
		data, err = encodeSweepExport(capture.sweep, "json")
	case "sweep-csv":
		data, err = encodeSweepExport(capture.sweep, "csv")
	case "schematic-svg":
		data = schematicSVGWithAnnotations(capture.graph, capture.artifact.RunID, capture.artifact.ElectricalRevision, capture.display, capture.annotations, capture.settings, capture.readings...)
	case "audio-wav":
		data = append([]byte(nil), capture.previewData...)
	case "scope-csv":
		data, err = encodeScopeCSV(capture.trace, capture.scope)
	case "verification-json":
		data, err = encodeVerificationExport(capture.record, "json")
	case "verification-csv":
		data, err = encodeVerificationExport(capture.record, "csv")
	default:
		return nil, &actionError{"invalid-params", "Unsupported export capture kind"}
	}
	if err != nil {
		return nil, err
	}
	if len(data) > MaxProjectBytes {
		return nil, &actionError{"invalid-params", "export exceeds 8 MiB"}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return prepareFileData(ctx, data, capture.artifact.Path, "", [32]byte{}, capture.overwrite, ".elab-export-*", progress)
}
func (s *Service) exportCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.documentGeneration == job.export.generation && s.documentPath != job.work.Path
}
func (s *Service) runProjectExport(job *projectJob, prepare projectExportFunc, publish projectPublishFunc) {
	defer s.projectWorkerDone()
	prepared, err := prepare(job.ctx, *job.export, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	if !s.exportCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason == "" && err == nil && job.ctx.Err() == nil {
		job.publishing = true
		job.work.Phase = "publishing"
		job.work.Status = "running"
		job.work.Cancelable = false
		s.revision++
		s.signalLocked("jobs")
		s.mu.Unlock()
		err = checkExportDestination(prepared.path, job.export.protectedPath)
		if err == nil {
			err = publish(prepared)
		}
		s.mu.Lock()
	}
	if prepared != nil {
		job.work.Phase = "cleanup"
		s.revision++
		s.signalLocked("jobs")
		s.mu.Unlock()
		removeErr := os.Remove(prepared.temp)
		s.mu.Lock()
		if removeErr != nil && !os.IsNotExist(removeErr) {
			job.work.CleanupWarning = removeErr.Error()
		}
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Export discarded before publication; the existing destination was preserved"}
	} else if err != nil || !job.publishing {
		if err == nil {
			err = job.ctx.Err()
		}
		s.failProjectWorkLocked(job, err)
	} else {
		artifact := job.export.artifact
		artifact.Bytes = len(prepared.data)
		digest := sha256.Sum256(prepared.data)
		artifact.SHA256 = hex.EncodeToString(digest[:])
		s.lastExport = &artifact
		associated := s.exportCurrentLocked(job)
		job.work.Associated = &associated
		job.work.ArtifactSHA256 = artifact.SHA256
		job.work.ResultGeneration = job.export.generation
		job.work.ResultRun = artifact.RunID
		job.work.Status = "completed"
		job.work.Phase = "completed"
	}
	job.export.previewData = nil
	job.export.protectedPath = ""
	job.export.graph = Graph{}
	job.export.trace = nil
	job.export.record = nil
	job.export.sweep = nil
	job.export.scope = Scope{}
	job.export.readings = nil
	job.export.annotations = nil
	job.export.display = CircuitDisplay{}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}

func renderSchematicPage(ctx context.Context, capture exportCapture) ([]byte, error) {
	identity := map[string]any{"format": "elab.schematic." + capture.artifact.Format + ".v1", "run_id": capture.artifact.RunID, "electrical_revision": capture.artifact.ElectricalRevision, "document_generation": capture.artifact.DocumentGeneration, "page": capture.artifact.Page, "measurement_source": "captured-live-snapshot; unavailable/stale labels retained when overlays requested"}
	if capture.artifact.PageIndex != nil {
		identity["page_index"] = *capture.artifact.PageIndex
	}
	identity["source_annotations"] = capture.annotations
	identity["source_settings"] = capture.settings
	sources := []Component{}
	for _, p := range capture.graph.Components {
		if p.Kind == "pulse-source" || IsWaveformSource(p.Kind) {
			sources = append(sources, p)
		}
	}
	identity["source_components"] = sources
	if capture.artifact.DPI != 0 {
		identity["dpi"] = capture.artifact.DPI
	}
	metadata, _ := json.Marshal(identity)
	capture.page.Metadata = string(metadata)
	return schematicpdf.Render(ctx, schematicPageSVG(capture), capture.page)
}

func schematicPageSVG(c exportCapture) []byte {
	return schematicSVGWithAnnotations(c.graph, c.artifact.RunID, c.artifact.ElectricalRevision, c.display, c.annotations, c.settings, c.readings...)
}
