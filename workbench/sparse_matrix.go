package workbench

import (
	"fmt"
	"math"
)

const minimumSparseSize = 224
const maximumSparseTermsPerRow = 16 // Average input occupancy before falling back to dense elimination.

type sparseTerm struct {
	column int
	value  float64
}

type sparseRow struct {
	terms, scratch []sparseTerm
}

// Each numerical worker owns its rows. Their storage survives successive
// Newton corrections, while the coefficients and pivot order are rebuilt.
// Nonlinear devices can change their nonzero positions between corrections.
type sparseLinearWorkspace struct {
	rows     []sparseRow
	order    []int
	position []int
	heads    []int
	next     []int
	rhs      []float64
}

func (w *sparseLinearWorkspace) prepare(matrix [][]float64, rhs []float64) bool {
	size := len(rhs)
	if len(w.rows) != size {
		w.rows = make([]sparseRow, size)
		w.order = make([]int, size)
		w.position = make([]int, size)
		w.heads = make([]int, size)
		w.next = make([]int, size)
		w.rhs = make([]float64, size)
		const initialTerms = 8
		terms := make([]sparseTerm, 2*size*initialTerms)
		for i := range w.rows {
			start := 2 * i * initialTerms
			middle, end := start+initialTerms, start+2*initialTerms
			w.rows[i].terms, w.rows[i].scratch = terms[start:start:middle], terms[middle:middle:end]
		}
	}
	copy(w.rhs, rhs)
	for i := range w.heads {
		w.heads[i] = -1
	}
	count := 0
	for i, row := range matrix {
		w.order[i], w.position[i] = i, i
		terms := w.rows[i].terms[:0]
		for j, value := range row {
			if value != 0 {
				terms = append(terms, sparseTerm{j, value})
			}
		}
		w.rows[i].terms = terms
		count += len(terms)
		if count > size*maximumSparseTermsPerRow {
			return false
		}
		w.queue(i)
	}
	return true
}

func (w *sparseLinearWorkspace) queue(row int) {
	terms := w.rows[row].terms
	if len(terms) != 0 {
		col := terms[0].column
		w.next[row], w.heads[col] = w.heads[col], row
	}
}

func (r *sparseRow) eliminate(pivot []sparseTerm, factor float64) {
	local, pivot := r.terms[1:], pivot[1:]
	out := r.scratch[:0]
	for len(local) > 0 && len(pivot) > 0 {
		a, b := local[0], pivot[0]
		switch {
		case a.column < b.column:
			out = append(out, a)
			local = local[1:]
		case a.column > b.column:
			if value := -factor * b.value; value != 0 {
				out = append(out, sparseTerm{b.column, value})
			}
			pivot = pivot[1:]
		default:
			if value := a.value - factor*b.value; value != 0 {
				out = append(out, sparseTerm{a.column, value})
			}
			local, pivot = local[1:], pivot[1:]
		}
	}
	out = append(out, local...)
	for _, term := range pivot {
		if value := -factor * term.value; value != 0 {
			out = append(out, sparseTerm{term.column, value})
		}
	}
	r.terms, r.scratch = out, r.terms[:0]
}

func (w *sparseLinearWorkspace) factor(check func() error) ([]float64, error) {
	for col := range w.order {
		if check != nil && col%16 == 0 {
			if err := check(); err != nil {
				return nil, err
			}
		}
		pivot, magnitude := col, 0.
		for index := w.heads[col]; index >= 0; index = w.next[index] {
			value := math.Abs(w.rows[index].terms[0].value)
			if !finite(value) {
				return nil, fmt.Errorf("nonfinite mixed matrix coefficient")
			}
			position := w.position[index]
			if value > magnitude || value == magnitude && position < pivot {
				pivot, magnitude = position, value
			}
		}
		if magnitude < 1e-30 {
			return nil, fmt.Errorf("singular mixed circuit; check floating nodes or inconsistent ideal constraints")
		}
		pivotRow, replaced := w.order[pivot], w.order[col]
		w.order[col], w.order[pivot] = pivotRow, replaced
		w.position[pivotRow], w.position[replaced] = col, pivot
		values := w.rows[pivotRow].terms
		for index := w.heads[col]; index >= 0; {
			current := index
			index = w.next[current]
			if current == pivotRow {
				continue
			}
			row := &w.rows[current]
			factor := row.terms[0].value / values[0].value
			row.eliminate(values, factor)
			w.rhs[current] -= factor * w.rhs[pivotRow]
			w.queue(current)
		}
	}
	x := make([]float64, len(w.order))
	for i := len(w.order) - 1; i >= 0; i-- {
		index := w.order[i]
		row := w.rows[index].terms
		sum := w.rhs[index]
		for _, term := range row[1:] {
			sum -= term.value * x[term.column]
		}
		x[i] = sum / row[0].value
		if !finite(x[i]) {
			return nil, fmt.Errorf("nonfinite mixed solution")
		}
	}
	return x, nil
}

func (w *sparseLinearWorkspace) solve(matrix [][]float64, rhs []float64, skipZeros bool, check func() error) ([]float64, error) {
	if len(rhs) >= minimumSparseSize && w.prepare(matrix, rhs) {
		return w.factor(check)
	}
	return linearSolveMode(matrix, rhs, skipZeros)
}
