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
	m := record.Metadata
	if m.Format != "elab.dc-sweep" || m.Version != 1 || m.Engine != DCSweepEngineID || m.OperationID == "" {
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
	if !reflect.DeepEqual(m.Sources, plan.sources) || r.Target != plan.target || r.Start != plan.start || r.Stop != plan.stop || !reflect.DeepEqual(r.Probes, plan.probes) || len(r.Points) != m.Settings.Points || len(r.Statistics) != len(r.Probes) {
		return nil, fmt.Errorf("sweep metadata, probes or point count do not match its source/settings")
	}
	stats := make([]SweepStatistics, len(r.Probes))
	for i, p := range r.Probes {
		stats[i] = SweepStatistics{p.ID, math.Inf(1), math.Inf(-1)}
	}
	for i, point := range r.Points {
		expected := plan.start + (plan.stop-plan.start)*float64(i)/float64(len(r.Points)-1)
		if i == 0 {
			expected = plan.start
		}
		if i == len(r.Points)-1 {
			expected = plan.stop
		}
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
	if !reflect.DeepEqual(stats, r.Statistics) {
		return nil, fmt.Errorf("sweep extrema do not match recorded points")
	}
	return &record, nil
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
	for i := range out.Entries {
		data, err := json.Marshal(out.Entries[i])
		if err != nil {
			return nil, err
		}
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
		job := &sweepJob{keep: true, record: record, metadata: record.Metadata, sourceHash: sweepGraphHash(record.Metadata.Source.Graph), summary: SweepSummary{ID: id, Status: "completed", Actor: "project", Completed: len(record.Result.Points), Total: len(record.Result.Points)}}
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
