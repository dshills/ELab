package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
)

const SweepDocumentVersion = 5

type ProjectSweeps struct {
	Entries  []SweepRecord `json:"entries"`
	Selected int           `json:"selected_index"`
	TraceID  string        `json:"trace_id"`
	Cursor   *int          `json:"cursor_index,omitempty"`
	Viewing  bool          `json:"viewing"`
}

func cloneSweepRecord(in *SweepRecord) *SweepRecord {
	if in == nil {
		return nil
	}
	return &SweepRecord{Metadata: cloneSweepMetadata(in.Metadata), Result: *cloneSweepResult(&in.Result)}
}
func validateSweepRecord(data []byte) (*SweepRecord, error) {
	if len(data) > MaxProjectBytes {
		return nil, fmt.Errorf("sweep record exceeds 8 MiB")
	}
	var record SweepRecord
	if err := params(data, &record, "metadata", "result"); err != nil {
		return nil, err
	}
	// Required numeric zeros must be present, rather than silently supplied by
	// Go's decoder. The remaining checks validate their semantic relationships.
	if err := validateSweepFields(data); err != nil {
		return nil, err
	}
	m := record.Metadata
	format, engine := acFormat(m.Settings.Parameter)
	if m.Format != format || m.Version != 1 || m.Engine != engine || m.OperationID == "" {
		return nil, fmt.Errorf("unsupported sweep format, version or engine")
	}
	// Reuse strict model/graph validation, using a minimal nonrecursive source
	// document. Saved values are recorded data, not a certification of a solve.
	raw, err := json.Marshal(m.Source.document())
	if err != nil {
		return nil, err
	}
	if _, err = decodeDocument(raw); err != nil {
		return nil, fmt.Errorf("sweep source: %w", err)
	}
	plan, err := newSweepPlan(m.Source.Graph, m.Settings)
	if err != nil {
		return nil, err
	}
	r := &record.Result
	probes := plan.probes
	if m.Settings.Parameter == "frequency" {
		probes = acProbes(probes)
	}
	if !reflect.DeepEqual(m.Sources, plan.sources) || r.Target != plan.target || r.Spacing != m.Settings.Spacing || r.Start != plan.start || r.Stop != plan.stop || !reflect.DeepEqual(r.Probes, probes) || len(r.Points) != m.Settings.Points || len(r.Statistics) != len(r.Probes) {
		return nil, fmt.Errorf("sweep metadata, probes or point count do not match its source/settings")
	}
	stats := make([]SweepStatistics, len(r.Probes))
	for i, p := range r.Probes {
		stats[i] = SweepStatistics{p.ID, math.Inf(1), math.Inf(-1)}
	}
	for i, point := range r.Points {
		expected := sweepAxisPoint(plan.start, plan.stop, len(r.Points), i, m.Settings.Spacing)
		if point.Input != expected || len(point.Values) != len(r.Probes) || !finite(point.KCLResidual) || point.KCLResidual < 0 || !finite(point.PowerResidual) {
			return nil, fmt.Errorf("invalid sweep point %d", i)
		}
		for j, value := range point.Values {
			if !finite(value) {
				return nil, fmt.Errorf("nonfinite sweep reading")
			}
			stats[j].Minimum = math.Min(stats[j].Minimum, value)
			stats[j].Maximum = math.Max(stats[j].Maximum, value)
		}
	}
	if err := validateACRecord(plan, r); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(stats, r.Statistics) {
		return nil, fmt.Errorf("sweep extrema do not match recorded points")
	}
	return &record, nil
}
func validateSweepFields(data []byte) error {
	var raw struct {
		Metadata json.RawMessage `json:"metadata"`
		Result   json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var metadata SweepMetadata
	if err := params(raw.Metadata, &metadata, "format", "version", "engine", "operation_id", "created_at", "document_generation", "electrical_revision", "actuation_revision", "source", "settings", "dc_sources"); err != nil {
		return err
	}
	var source DCCircuit
	if err := paramsNested(raw.Metadata, "source", &source, "graph", "transient_settings", "led_model"); err != nil {
		return err
	}
	var settings SweepSettings
	if err := paramsNested(raw.Metadata, "settings", &settings, "component_id", "parameter", "start", "stop", "points"); err != nil {
		return err
	}
	var result SweepResult
	if err := params(raw.Result, &result, "target", "start_si", "stop_si", "probes", "points", "statistics"); err != nil {
		return err
	}
	var arrays struct {
		Points     []json.RawMessage `json:"points"`
		Statistics []json.RawMessage `json:"statistics"`
	}
	if err := json.Unmarshal(raw.Result, &arrays); err != nil {
		return err
	}
	for i, point := range arrays.Points {
		var out SweepPoint
		if err := params(point, &out, "input", "values", "kcl_residual_a", "power_residual_w"); err != nil {
			return fmt.Errorf("sweep point %d: %w", i, err)
		}
	}
	if metadata.Settings.Parameter == "frequency" {
		if err := validateACFields(raw.Result, arrays.Points); err != nil {
			return err
		}
	}
	for i, statistic := range arrays.Statistics {
		var out SweepStatistics
		if err := params(statistic, &out, "probe_id", "minimum", "maximum"); err != nil {
			return fmt.Errorf("sweep statistic %d: %w", i, err)
		}
	}
	return nil
}

func (s *Service) projectSweepsLocked() *ProjectSweeps {
	if s.sweeps == nil {
		return nil
	}
	out := &ProjectSweeps{Entries: []SweepRecord{}, Selected: -1}
	for _, id := range s.sweeps.order {
		job := s.sweeps.operations[id]
		if !job.keep || job.record == nil {
			continue
		}
		if s.sweeps.selected == id {
			out.Selected = len(out.Entries)
			out.TraceID = s.sweeps.traceID
			out.Viewing = s.sweeps.visible
			if s.sweeps.cursor != nil {
				v := *s.sweeps.cursor
				out.Cursor = &v
			}
		}
		out.Entries = append(out.Entries, *cloneSweepRecord(job.record))
	}
	if len(out.Entries) == 0 {
		return nil
	}
	return out
}
func validateProjectSweeps(raw json.RawMessage) (*ProjectSweeps, error) {
	var out ProjectSweeps
	if err := params(raw, &out, "entries", "selected_index", "trace_id", "viewing"); err != nil {
		return nil, err
	}
	if len(out.Entries) < 1 || len(out.Entries) > MaxDCSweepOperations || out.Selected < -1 || out.Selected >= len(out.Entries) || out.Viewing && out.Selected < 0 {
		return nil, fmt.Errorf("invalid saved sweep count/selection")
	}
	// Preserve original field presence through validation. Re-marshaling typed
	// entries here would silently fill omitted zero-valued measurements.
	var original struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(raw, &original); err != nil {
		return nil, err
	}
	for i, data := range original.Entries {
		record, err := validateSweepRecord(data)
		if err != nil {
			return nil, fmt.Errorf("sweep %d: %w", i, err)
		}
		out.Entries[i] = *record
	}
	if out.Selected < 0 {
		if out.Cursor != nil || out.TraceID != "" {
			return nil, fmt.Errorf("unselected sweep must not have cursor/trace")
		}
		return &out, nil
	}
	selected := out.Entries[out.Selected].Result
	found := false
	for _, p := range selected.Probes {
		if p.ID == out.TraceID {
			found = true
		}
	}
	if !found || out.Cursor != nil && (*out.Cursor < 0 || *out.Cursor >= len(selected.Points)) {
		return nil, fmt.Errorf("invalid saved sweep trace/cursor")
	}
	return &out, nil
}
func (s *Service) installProjectSweepsLocked(project *ProjectSweeps) {
	s.clearSweepsLocked()
	if project == nil {
		return
	}
	store := s.ensureSweepsLocked()
	for i, entry := range project.Entries {
		record := cloneSweepRecord(&entry)
		s.sweepSerial++
		id := fmt.Sprintf("saved-sweep-%d", s.sweepSerial)
		job := &sweepJob{keep: true, record: record, metadata: record.Metadata, sourceHash: sweepGraphHash(record.Metadata.Source.Graph), summary: SweepSummary{ID: id, Kind: "project-sweep", Status: "completed", Actor: "project", Completed: len(record.Result.Points), Total: len(record.Result.Points)}}
		store.operations[id] = job
		store.order = append(store.order, id)
		if project.Selected == i {
			cfg := cloneSweepSettings(record.Metadata.Settings)
			store.settings = &cfg
			store.selected = id
			store.traceID = project.TraceID
			store.visible = project.Viewing
			if project.Cursor != nil {
				v := *project.Cursor
				store.cursor = &v
			}
		}
	}
}
func (s *Service) keptSweepCountLocked() int {
	count := 0
	if s.sweeps != nil {
		for _, job := range s.sweeps.operations {
			if job.keep && job.record != nil {
				count++
			}
		}
	}
	return count
}
func (s *Service) currentDocumentVersionLocked() int {
	if s.keptSweepCountLocked() > 0 {
		return SweepDocumentVersion
	}
	return DocumentVersion
}
