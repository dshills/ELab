package workbench

import "time"

const (
	MaxLiveSamplesPerWallSecond = 1000.
	MaxManualStepSeconds        = 1.
	MaxManualStepSamples        = 1000
	MinCapacitorStepSeconds     = 1e-10
	ManualStepDeadline          = 30 * time.Second
	InputWorkDeadline           = 5 * time.Second
	SimulationWaitTimeout       = 5 * time.Second
)

// LiveInputModes is the owned mode inventory for a supported transient analysis.
// Membership does not replace fresh-result and lifecycle admission checks.
func LiveInputModes(analysis string) []string {
	if !IsTransientAnalysis(analysis) {
		return []string{}
	}
	return VerificationInputModes(analysis)
}

func liveInputModeValid(analysis, mode string) bool {
	for _, allowed := range LiveInputModes(analysis) {
		if mode == allowed {
			return true
		}
	}
	return false
}
