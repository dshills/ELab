package workbench

// Imported runs retain the engine that actually produced their data. Accept
// earlier backward-Euler captures for the same analysis family without
// relabeling or replaying them through the newer live integration method.
func recordingEngineCompatible(analysis, engine string) bool {
	if engine == verificationEngineID(analysis) {
		return true
	}
	switch analysis {
	case MixedAnalysis:
		return engine == "elab.mixed-mna-be.v1"
	case NetworkDCAnalysis, NetworkTransientAnalysis:
		return engine == "elab.network-mna-be.v1"
	case WaveformAnalysis:
		return engine == "elab.network-waveforms-mna-be.v1"
	}
	return false
}
