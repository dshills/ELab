package workbench

// Larger electrical matrices benefit from omitting zero elimination updates.
// Keep the small-circuit path free of the extra column-list allocation: the
// ladder crossover and complete flasher/capacity fixtures are benchmarked.
const minimumZeroSkipSize = 16
const minimumReusableAssemblySize = 128

// Each row has its own disjoint, capacity-limited span. Two allocations avoid
// the per-row allocator/collector cost whenever Newton rebuilds a dense matrix.
// Matrices belong to one solve; no mutable workspace is shared between workers.
func denseMatrix(size int) [][]float64 {
	rows, values := make([][]float64, size), make([]float64, size*size)
	for row := range rows {
		rows[row] = values[row*size : (row+1)*size : (row+1)*size]
	}
	return rows
}

// Reuse assembly storage only in larger systems. The same work can assemble
// a different numerical matrix after any device state change; reset clears
// coefficients and restores row order even after dense elimination.
type matrixAssembly struct {
	rows   [][]float64
	values []float64
	rhs    []float64
}

func (a *matrixAssembly) reset(size int) ([][]float64, []float64) {
	if a == nil {
		return denseMatrix(size), make([]float64, size)
	}
	if len(a.rows) != size {
		a.rows = make([][]float64, size)
		a.values = make([]float64, size*size)
		a.rhs = make([]float64, size)
	} else {
		clear(a.values)
		clear(a.rhs)
	}
	for i := range a.rows {
		a.rows[i] = a.values[i*size : (i+1)*size : (i+1)*size]
	}
	return a.rows, a.rhs
}
