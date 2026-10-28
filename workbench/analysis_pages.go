package workbench

import (
	"fmt"
	"slices"
)

const MaxAnalysisPageRows = 50

type AnalysisSummary struct {
	Kind              string             `json:"kind"`
	ID                string             `json:"id"`
	Status            string             `json:"status"`
	Rows              int                `json:"rows"`
	Columns           []Probe            `json:"columns"`
	NoiseContributors []NoiseContributor `json:"noise_contributors,omitempty"`
}
type AnalysisCatalog struct {
	Revision           uint64            `json:"revision"`
	CommandRevision    uint64            `json:"command_revision"`
	DocumentGeneration uint64            `json:"document_generation"`
	Analyses           []AnalysisSummary `json:"analyses"`
}
type AnalysisPageParams struct {
	Kind               string `json:"kind"`
	ID                 string `json:"id"`
	DocumentGeneration uint64 `json:"document_generation"`
	Offset             int    `json:"offset,omitempty"`
	Limit              int    `json:"limit,omitempty"`
}
type AnalysisPage struct {
	DocumentGeneration uint64            `json:"document_generation"`
	Analysis           AnalysisSummary   `json:"analysis"`
	Offset             int               `json:"offset"`
	NextOffset         int               `json:"next_offset"`
	Done               bool              `json:"done"`
	Points             []SweepPoint      `json:"points,omitempty"`
	Samples            []VerificationRow `json:"samples,omitempty"`
	Bins               []SpectrumBin     `json:"bins,omitempty"`
	Trials             []ToleranceTrial  `json:"trials,omitempty"`
}

// Views borrow immutable published records only while the service mutex is held.
// Returned summaries/pages own all slices; no full snapshot or compile is needed.
func (s *Service) analysisViewLocked(kind, id string) (AnalysisSummary, any, error) {
	summary := AnalysisSummary{Kind: kind, ID: id, Columns: []Probe{}}
	switch kind {
	case "sweep":
		if s.sweeps != nil {
			if job := s.sweeps.operations[id]; job != nil {
				summary.Status = job.summary.Status
				if job.record != nil {
					r := &job.record.Result
					summary.Rows = len(r.Points)
					summary.Columns = append([]Probe{}, r.Probes...)
					if r.Noise != nil {
						summary.NoiseContributors = append([]NoiseContributor{}, r.Noise.Contributors...)
						for k := range summary.NoiseContributors {
							summary.NoiseContributors[k].RMS = append([]float64{}, r.Noise.Contributors[k].RMS...)
						}
					}
					return summary, r, nil
				}
				return summary, nil, nil
			}
		}
	case "recording":
		if job := s.operations[id]; job != nil {
			summary.Status = job.summary.Status
			if job.record != nil {
				summary.Rows = len(job.record.rows)
				summary.Columns = append([]Probe{}, job.record.metadata.Probes...)
				return summary, job.record, nil
			}
			return summary, nil, nil
		}
	case "spectrum":
		if s.spectrum != nil && s.spectrum.Result != nil && s.spectrum.Result.ID == id {
			r := s.spectrum.Result
			summary.Status = "completed"
			summary.Rows = len(r.Bins)
			summary.Columns = []Probe{r.Probe}
			return summary, r, nil
		}
	case "tolerance":
		if s.tolerance != nil && s.tolerance.view.ID == id {
			summary.Status = s.tolerance.view.Status
			if r := s.tolerance.view.Result; r != nil {
				summary.Rows = len(r.Trials)
				summary.Columns = []Probe{r.Probe}
				return summary, r, nil
			}
			return summary, nil, nil
		}
	default:
		return summary, nil, fmt.Errorf("analysis kind must be sweep, recording, spectrum or tolerance")
	}
	return summary, nil, &actionError{"not-found", "Analysis is unknown, replaced or no longer retained"}
}
func (s *Service) analysisCatalogLocked() AnalysisCatalog {
	out := AnalysisCatalog{Revision: s.revision, CommandRevision: s.commandRevision, DocumentGeneration: s.documentGeneration, Analyses: []AnalysisSummary{}}
	add := func(kind, id string) {
		summary, _, err := s.analysisViewLocked(kind, id)
		if err == nil {
			out.Analyses = append(out.Analyses, summary)
		}
	}
	if s.sweeps != nil {
		for _, id := range s.sweeps.order {
			add("sweep", id)
		}
	}
	for _, id := range s.operationOrder {
		add("recording", id)
	}
	if s.spectrum != nil && s.spectrum.Result != nil {
		add("spectrum", s.spectrum.Result.ID)
	}
	if s.tolerance != nil && s.tolerance.view.ID != "" {
		add("tolerance", s.tolerance.view.ID)
	}
	return out
}
func (s *Service) analysisPageLocked(p AnalysisPageParams) (AnalysisPage, error) {
	out := AnalysisPage{DocumentGeneration: s.documentGeneration}
	if p.DocumentGeneration != s.documentGeneration {
		return out, &actionError{"conflict", "Document changed; refresh analysis.list before paging"}
	}
	if p.ID == "" || len(p.ID) > 128 {
		return out, fmt.Errorf("analysis id must have 1–128 bytes")
	}
	if p.Limit == 0 {
		p.Limit = 25
	}
	if p.Offset < 0 || p.Limit < 1 || p.Limit > MaxAnalysisPageRows {
		return out, fmt.Errorf("analysis pages require offset >= 0 and limit 1–50")
	}
	summary, record, err := s.analysisViewLocked(p.Kind, p.ID)
	if err != nil {
		return out, err
	}
	if record == nil {
		return out, &actionError{"not-ready", "Analysis has no completed result"}
	}
	if p.Offset > summary.Rows {
		return out, fmt.Errorf("offset exceeds analysis rows")
	}
	end := p.Offset + min(p.Limit, summary.Rows-p.Offset)
	out.Analysis = summary
	out.Offset = p.Offset
	out.NextOffset = end
	out.Done = end == summary.Rows
	switch r := record.(type) {
	case *SweepResult:
		for _, point := range r.Points[p.Offset:end] {
			v := point
			v.Values = slices.Clone(point.Values)
			v.AC = slices.Clone(point.AC)
			v.Noise = cloneNoisePoint(point.Noise)
			out.Points = append(out.Points, v)
		}
	case *verificationRecord:
		for _, row := range r.rows[p.Offset:end] {
			out.Samples = append(out.Samples, VerificationRow{Time: row.Time, Values: slices.Clone(row.Values)})
		}
	case *SpectrumResult:
		out.Bins = append([]SpectrumBin{}, r.Bins[p.Offset:end]...)
	case *ToleranceResult:
		for _, trial := range r.Trials[p.Offset:end] {
			v := trial
			v.Parameters = slices.Clone(trial.Parameters)
			if trial.Value != nil {
				value := *trial.Value
				v.Value = &value
			}
			out.Trials = append(out.Trials, v)
		}
	default:
		return AnalysisPage{DocumentGeneration: s.documentGeneration}, fmt.Errorf("unsupported analysis record %T", record)
	}
	return out, nil
}
