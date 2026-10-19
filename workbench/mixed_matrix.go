package workbench

// Larger electrical matrices benefit from omitting zero elimination updates.
// Keep the small-circuit path free of the extra column-list allocation: the
// ladder crossover and complete flasher/capacity fixtures are benchmarked.
const minimumZeroSkipSize = 16

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
