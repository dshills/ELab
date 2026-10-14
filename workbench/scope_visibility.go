package workbench

// Scope visibility changes presentation only; data acquisition and all live or
// recorded measurement state remain independent of the mounted plot.
type ScopeVisibilityParams struct {
	Visible bool `json:"visible"`
}
