package workbench

import "context"

// Source stepping is only a cold-start fallback for the newly admitted larger
// discrete envelope. It never replaces a successful ordinary start, changes a
// stored reactive condition, or publishes an intermediate operating point.
func (n *mixedNetwork) canStepBipolarSources() bool {
	for _, p := range n.c.MixedGraph.Components {
		switch p.Kind {
		case "dc-source", "sine-source", "pulse-source", "finite-pulse-source", "resistor", "diode", "schottky", "zener", "led", "npn", "pnp", "ground", "junction", "net-label":
		default:
			return false
		}
	}
	return true
}

func (n *mixedNetwork) bipolarSourceStep(ctx context.Context, old []float64, dt float64, t *Transient, at float64) (*mixedState, error) {
	trial := *n
	var seed []float64
	for step := 1; step <= 8; step++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scale := float64(step) / 8
		trial.startupSourceScale = &scale
		if step == 1 {
			seed = trial.initialConstraintGuess(old, dt, t, at)
		}
		state, err := trial.newton(ctx, seed, old, dt, t, at)
		if err != nil {
			return nil, err
		}
		seed = state.x
	}
	// Recheck the full, unmodified authored problem using the ordinary strict
	// residual, uniqueness and voltage guards, with unchanged Newton budgets.
	return n.newton(ctx, seed, old, dt, t, at)
}
