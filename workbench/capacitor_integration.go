package workbench

import "math"

// Two-stage, stiffly accurate SDIRK with gamma = 1 - 1/sqrt(2).
// Each diagonal stage is an ordinary implicit capacitor companion solve.
// The second stage's history includes the first stage derivative, giving a
// second-order result without a three-times-larger collocation matrix.
const capacitorStageGamma = 1 - 1/math.Sqrt2

func (state *mixedState) capacitorStep(trace *Transient, h, end float64, budget *integrationBudget) (*mixedState, error) {
	if err := budget.checkTime(); err != nil {
		return nil, err
	}
	diagonal := capacitorStageGamma * h
	first, err := state.net.solveContext(budget.solverContext(), state.x, state.capV, diagonal, trace, end-h+diagonal)
	if err != nil {
		return nil, err
	}
	if err := budget.checkTime(); err != nil {
		return nil, err
	}
	history := make([]float64, len(state.capV))
	for i, previous := range state.capV {
		history[i] = previous + (1-capacitorStageGamma)/capacitorStageGamma*(first.capV[i]-previous)
	}
	next, err := state.net.solveContext(budget.solverContext(), first.x, history, diagonal, trace, end)
	if err != nil {
		return nil, err
	}
	if err := budget.checkTime(); err != nil {
		return nil, err
	}
	return next, nil
}
