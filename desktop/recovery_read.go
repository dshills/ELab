package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type recoveryReadCache struct {
	id      string
	loading bool
	result  *workbench.RecoveryReadResult
	err     string
}

func latestRecoveryRead(state workbench.Snapshot, choice workbench.RecoveryCandidate, kind string) *workbench.RecoveryReadWork {
	for i := len(state.Recovery.Reads) - 1; i >= 0; i-- {
		w := state.Recovery.Reads[i]
		if w.Input.Kind == kind && w.Input.ID == choice.ID && w.Input.SHA256 == choice.SHA256 && w.Directory == state.Recovery.Directory {
			return &w
		}
	}
	return nil
}
func (s *screen) recoveryCache(kind string) *recoveryReadCache {
	if kind == "inspection" {
		return &s.recoveryInspection
	}
	return &s.recoverySamples
}
func (s *screen) loadRecoveryRead(work *workbench.RecoveryReadWork) {
	cache := s.recoveryCache(work.Input.Kind)
	if work.Status != "completed" || cache.id == work.ID {
		return
	}
	*cache = recoveryReadCache{id: work.ID, loading: true}
	captured := *work
	go func() {
		ctx, cancel := context.WithTimeout(s.app.Context(), 10*time.Second)
		defer cancel()
		result, err := readRecoveryResult(ctx, s.service, captured)
		_ = s.app.PostWindow("main", func() {
			cache := s.recoveryCache(captured.Input.Kind)
			// Selection may change while decoding. Keep this immutable result so
			// returning to the same copy cannot strand a loading cache. Rendering
			// separately checks candidate hash, directory and latest read ID.
			if cache.id != captured.ID {
				return
			}
			cache.loading = false
			cache.result = result
			if err != nil {
				cache.err = err.Error()
			}
			s.app.Invalidate("main")
		})
	}()
}
func readRecoveryResult(ctx context.Context, service *workbench.Service, work workbench.RecoveryReadWork) (*workbench.RecoveryReadResult, error) {
	if work.ResultBytes <= 0 || work.ResultBytes > workbench.MaxRecoveryReadResultBytes {
		return nil, fmt.Errorf("invalid recovery result size")
	}
	data := make([]byte, 0, work.ResultBytes)
	for len(data) < work.ResultBytes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(workbench.RecoveryReadChunkParams{ID: work.ID, Offset: len(data)})
		response := service.Dispatch(ctx, workbench.Request{Version: 1, ID: fmt.Sprintf("recovery-result-%s-%d", work.ID, len(data)), Command: "recovery.read.chunk", Params: raw}, "user")
		if response.Error != nil {
			return nil, fmt.Errorf("%s", response.Error.Message)
		}
		chunk, ok := response.Result.(workbench.RecoveryReadChunk)
		if !ok || chunk.ID != work.ID || chunk.Offset != len(data) || chunk.Total != work.ResultBytes || chunk.SHA256 != work.ResultSHA256 || len(chunk.JSON) == 0 || chunk.Next != len(data)+len(chunk.JSON) || chunk.Next > work.ResultBytes {
			return nil, fmt.Errorf("invalid recovery result chunk")
		}
		data = append(data, chunk.JSON...)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != work.ResultSHA256 {
		return nil, fmt.Errorf("recovery result checksum mismatch")
	}
	var result workbench.RecoveryReadResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result.ID != work.ID || result.Kind != work.Input.Kind {
		return nil, fmt.Errorf("recovery result identity mismatch")
	}
	return &result, ctx.Err()
}
func (s *screen) recoveryReadControls(state workbench.Snapshot, choice workbench.RecoveryCandidate) []ui.Node {
	params := workbench.RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}
	nodes := []ui.Node{s.action("recovery-inspect-copy", "Inspect saved copy", "recovery.read.start", params)}
	for _, work := range state.Recovery.Reads {
		if work.Input.ID != choice.ID || work.Input.SHA256 != choice.SHA256 || work.Directory != state.Recovery.Directory {
			continue
		}
		if work.Status == "queued" || work.Status == "running" || work.Status == "canceling" {
			nodes = append(nodes, quiet("recovery-read-progress-"+work.ID, work.Input.Kind+" · "+work.Status+" · "+work.Phase))
			cancel := s.action("recovery-read-cancel-"+work.ID, "Cancel read", "recovery.read.cancel", workbench.IDParams{ID: work.ID})
			cancel.Disabled = work.Status == "canceling"
			nodes = append(nodes, cancel)
		}
	}
	for _, kind := range []string{"inspection", "samples"} {
		work := latestRecoveryRead(state, choice, kind)
		if work == nil {
			continue
		}
		if work.Error != nil {
			nodes = append(nodes, quiet("recovery-read-error-"+kind, work.Error.Message))
			continue
		}
		s.loadRecoveryRead(work)
		cache := s.recoveryCache(kind)
		if cache.id != work.ID || cache.loading {
			continue
		}
		if cache.err != "" {
			nodes = append(nodes, quiet("recovery-result-error-"+kind, cache.err))
			continue
		}
		if cache.result == nil {
			continue
		}
		if inspection := cache.result.Inspection; inspection != nil {
			doc := inspection.Document
			nodes = append(nodes, quiet("recovery-inspected-copy", fmt.Sprintf("Captured copy · %d components\n%d embedded time recordings\nReading this copy does not restore it.", len(doc.Graph.Components), len(doc.Recordings))))
			for _, part := range doc.Graph.Components[:min(40, len(doc.Graph.Components))] {
				if part.Unit != "" {
					nodes = append(nodes, quiet("recovery-part-"+part.ID, part.ID+" · "+workbench.FormatValue(part.Value, part.Unit)))
				}
			}
			if len(doc.Graph.Components) > 40 {
				nodes = append(nodes, quiet("recovery-parts-truncated", "Showing the first 40 components. Restore a copy to explore the full schematic."))
			}
			if len(doc.Recordings) > 0 {
				options := []ui.Option{{ID: "", Label: "Read a recording…"}}
				for i, m := range doc.Recordings {
					options = append(options, ui.Option{ID: strconv.Itoa(i), Label: fmt.Sprintf("%d · %s · %.4g s", i+1, m.Analysis, m.Duration)})
				}
				nodes = append(nodes, ui.Node{ID: "recovery-recording-read", Kind: ui.Selector, Options: options, SelectedID: "", Accessibility: ui.Accessibility{Name: "Read captured recovery recording"}, Change: func(e ui.ChangeEvent) {
					i, err := strconv.Atoi(e.SelectedID)
					if err != nil {
						return
					}
					s.invoke("recovery.read.start", workbench.RecoveryReadParams{Kind: "samples", ID: choice.ID, SHA256: choice.SHA256, Recording: &i, Limit: 20})
				}})
			}
		} else if page := cache.result.Samples; page != nil {
			nodes = append(nodes, quiet("recovery-recorded-page", fmt.Sprintf("Recording %d · rows %d–%d of %d", *work.Input.Recording+1, page.Offset, page.NextOffset, page.Total)))
			for i, row := range page.Rows {
				values := []string{fmt.Sprintf("t %.5g s", row.Time)}
				for j, value := range row.Values {
					values = append(values, fmt.Sprintf("%s %.5g %s", page.Probes[j].ID, value, page.Probes[j].Unit))
				}
				nodes = append(nodes, quiet(fmt.Sprintf("recovery-sample-%d", i), strings.Join(values, "\n")))
			}
			previous, next := work.Input, work.Input
			previous.Offset = max(0, page.Offset-20)
			previous.Limit = 20
			next.Offset = page.NextOffset
			next.Limit = 20
			prev := s.action("recovery-samples-previous", "Previous rows", "recovery.read.start", previous)
			prev.Disabled = page.Offset == 0
			nextButton := s.action("recovery-samples-next", "Next rows", "recovery.read.start", next)
			nextButton.Disabled = page.NextOffset >= page.Total
			nodes = append(nodes, prev, nextButton)
		}
	}
	return nodes
}
