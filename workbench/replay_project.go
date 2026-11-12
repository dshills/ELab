package workbench

import (
	"encoding/json"
	"fmt"
)

// CapturedProjectBytes creates a paused, editable schematic from a recording's
// captured startup circuit. Recording data stays in the independent replay.
func CapturedProjectBytes(metadata VerificationMetadata) ([]byte, error) {
	settings := metadata.Settings
	// Offline verification can use a tighter step than the live playback speed
	// permits. Keep that sampling step and choose a compatible playback speed.
	if err := settings.validatePlayback(); err != nil {
		settings.PlaybackSpeed = .001
	}
	document := dcCircuitSnapshot(metadata.Graph, settings).document()
	document.ScopeProbes = &ProbeParams{IDs: []string{}}
	for _, probe := range metadata.Probes {
		document.ScopeProbes.IDs = append(document.ScopeProbes.IDs, probe.ID)
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	if err := ValidateProject(data); err != nil {
		return nil, fmt.Errorf("could not create an editable copy: %w", err)
	}
	return append(data, '\n'), nil
}
