package workbench

// initialConstraintGuess initializes only voltages fixed by an ideal-constraint
// path to ground. It does not solve a substitute circuit or accept a result:
// Newton still checks every nonlinear branch, KCL and voltage/work limit.
// An existing accepted guess is never passed here, preserving continuation.
func (n *mixedNetwork) initialConstraintGuess(old []float64, dt float64, t *Transient, at float64) []float64 {
	x := make([]float64, n.matrixSize(dt))
	type edge struct {
		to   int
		drop float64
	}
	// Shift node indices by one so the ground index (-1) becomes zero.
	adjacent := make([][]edge, len(n.nodes)+1)
	constraint := func(positive, negative int, value float64) {
		p, q := positive+1, negative+1
		adjacent[q] = append(adjacent[q], edge{p, value})
		adjacent[p] = append(adjacent[p], edge{q, -value})
	}
	for _, p := range n.sources {
		constraint(n.pin(p, "positive"), n.pin(p, "negative"), n.sourceStimulus(p, t, at, dt))
	}
	if dt == 0 {
		for i, p := range n.caps {
			constraint(n.pin(p, "1"), n.pin(p, "2"), old[i])
		}
	}
	known := make([]bool, len(adjacent))
	known[0] = true
	queue := []int{0}
	for next := 0; next < len(queue); next++ {
		from := queue[next]
		for _, e := range adjacent[from] {
			if known[e.to] {
				continue
			}
			known[e.to] = true
			x[e.to-1] = voltage(x, from-1) + e.drop
			queue = append(queue, e.to)
		}
	}
	return x
}
