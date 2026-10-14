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
	DPI                int            `json:"dpi,omitempty"`
	Page               *SchematicPage `json:"page,omitempty"`
	Kind               string         `json:"kind"`
	ID                 string         `json:"id,omitempty"`
	Path               string         `json:"path"`
	Overwrite          bool           `json:"overwrite,omitempty"`
	ExpectedGeneration *uint64        `json:"expected_document_generation,omitempty"`
}
type exportCapture struct {
	annotations []SourceAnnotation
	settings    TransientSettings
	page        schematicpdf.Page
	kind        string
	generation  uint64
	overwrite   bool
	graph       Graph
	display     CircuitDisplay
	readings    []PinnedReading
	trace       *Transient
	scope       Scope
	record      *verificationRecord
	artifact    Artifact
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
		return fmt.Errorf("export kind must be schematic-svg, schematic-pdf, schematic-png, scope-csv, verification-json or verification-csv")
	}
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
	progress("encoding", 0)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var data []byte
	var err error
	switch capture.kind {
	case "schematic-pdf", "schematic-png":
		identity := map[string]any{"format": "elab.schematic." + capture.artifact.Format + ".v1", "run_id": capture.artifact.RunID, "electrical_revision": capture.artifact.ElectricalRevision, "document_generation": capture.artifact.DocumentGeneration, "page": capture.artifact.Page, "measurement_source": "captured-live-snapshot; unavailable/stale labels retained when overlays requested"}
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
		data, err = schematicpdf.Render(ctx, schematicSVGWithAnnotations(capture.graph, capture.artifact.RunID, capture.artifact.ElectricalRevision, capture.display, capture.annotations, capture.settings, capture.readings...), capture.page)
	case "schematic-svg":
		data = schematicSVGWithAnnotations(capture.graph, capture.artifact.RunID, capture.artifact.ElectricalRevision, capture.display, capture.annotations, capture.settings, capture.readings...)
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
		err = publish(prepared)
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
	job.export.graph = Graph{}
	job.export.trace = nil
	job.export.record = nil
	job.export.scope = Scope{}
	job.export.readings = nil
	job.export.annotations = nil
	job.export.display = CircuitDisplay{}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}
