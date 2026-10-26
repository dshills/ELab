package workbench

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Plot one live source as a single guarded presentation intent. Never evict a
// trace silently or mistake an existing recorded viewer for the live scope.
func (s *Service) plotProbeLocked(raw json.RawMessage) error {
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return err
	}
	if s.transient == nil {
		return &actionError{"not-ready", "A transient circuit is required to plot a waveform; DC readings remain available in the inspector"}
	}
	available := false
	for _, probe := range probeInventory(s.graph) {
		available = available || probe.ID == p.ID
	}
	if !available {
		return &actionError{"not-found", "Choose a current probe ID from snapshot.scope.available"}
	}
	// Resolve defaults exactly as the visible scope does before adding a trace.
	selected := s.plotProbeSelectionLocked()
	if !slices.Contains(selected, p.ID) {
		if len(selected) >= MaxScopeProbes {
			return fmt.Errorf("six traces are already plotted; remove one before plotting %s", p.ID)
		}
		selected = append(selected, p.ID)
	}
	s.scopeProbes = append([]string{}, selected...)
	s.scopeConfigured = true
	s.focusProbe = p.ID
	s.scopeVisible = true
	s.recording.viewing, s.recording.playing = false, false
	s.viewerRevision++ // A later import must not steal this explicit live choice.
	return nil
}

func (s *Service) plotProbeSelectionLocked() []string {
	if s.scopeConfigured {
		return append([]string{}, s.scopeProbes...)
	}
	compiled, _, _ := s.graph.Compile()
	return append([]string{}, defaultProbes(compiled)...)
}

func (s *Service) plotProbeRequestHelpLocked(raw json.RawMessage, help *DiagnosticHelp) {
	var p IDParams
	if params(raw, &p, "id") != nil || s.transient == nil {
		return
	}
	known := false
	for _, probe := range probeInventory(s.graph) {
		known = known || probe.ID == p.ID
	}
	selected := s.plotProbeSelectionLocked()
	if known && len(selected) >= MaxScopeProbes && !slices.Contains(selected, p.ID) {
		help.Summary = "All six scope trace slots are in use."
		help.Action = "Remove a trace in Scope probes, then plot this source again. Your existing traces and circuit were kept."
	}
}
