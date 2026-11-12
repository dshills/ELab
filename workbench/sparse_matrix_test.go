package workbench

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
)

func sparseCircuitMatrix(size int) ([][]float64, []float64) {
	matrix, rhs := denseMatrix(size), make([]float64, size)
	for i := range matrix {
		matrix[i][i] = 4
		for _, j := range []int{i - 13, i - 1, i + 1, i + 13} {
			if j >= 0 && j < size {
				matrix[i][j] = -.75
			}
		}
		for j, value := range matrix[i] {
			rhs[i] += value * math.Sin(float64(j))
		}
	}
	// Force a row pivot and retain the same known solution.
	matrix[0], matrix[size-1] = matrix[size-1], matrix[0]
	rhs[0], rhs[size-1] = rhs[size-1], rhs[0]
	return matrix, rhs
}

func TestSparseLargeCircuitPivotFillAndReuse(t *testing.T) {
	var workspace sparseLinearWorkspace
	for _, size := range []int{128, 201, 375} {
		matrix, rhs := sparseCircuitMatrix(size)
		for attempt := range 2 {
			if attempt == 1 {
				// A nonlinear stamp can introduce a new nonzero at the same size.
				matrix[8][30] = .35
				rhs[8] += .35 * math.Sin(30)
			}
			if !workspace.prepare(matrix, rhs) {
				t.Fatal("test circuit unexpectedly chose the dense path")
			}
			got, err := workspace.factor(nil)
			if err != nil {
				t.Fatal(err)
			}
			copyOf := denseMatrix(size)
			for i := range matrix {
				copy(copyOf[i], matrix[i])
			}
			dense, err := linearSolveMode(copyOf, append([]float64(nil), rhs...), true)
			if err != nil {
				t.Fatal(err)
			}
			for i, value := range got {
				if value != dense[i] || math.Abs(value-math.Sin(float64(i))) > 1e-12 {
					t.Fatalf("size %d, column %d: got %.16g", size, i, value)
				}
			}
		}
	}
}

func TestSparseLargeInductorCircuitMatchesAnalyticStep(t *testing.T) {
	b := exampleBuilder{graph: capacityChain(t, "pulse-source", 60)}
	b.part("RL", "resistor", "100 Ω", -1050, -160, 0)
	b.part("L1", "inductor", "1 H", -1050, 0, 0)
	b.wire("V1.positive", "RL.1")
	b.wire("RL.2", "L1.1")
	b.wire("L1.2", "GND1.reference")
	compiled, _, diagnostic := b.graph.Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	if size := 3 * newMixedNetwork(compiled).matrixSize(.001); size < minimumSparseSize {
		t.Fatalf("%d unknowns did not exercise large factorization", size)
	}
	s := newGraphService(SolveLED, b.graph)
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	const at = .01
	call(t, s, "simulation.step", StepParams{Duration: at}, "step")
	result := s.Snapshot().Mixed
	want := .05 * (1 - math.Exp(-at/.01))
	if result == nil || result.Stale || len(result.Inductors) != 1 || result.KCLResidual > 1e-8 || math.Abs(result.Inductors[0].Current-want) > 2e-6 {
		t.Fatalf("large RL step did not match %g A: %+v", want, result)
	}
}

func TestSparseLargeCircuitRejectsCancellationAndSingularity(t *testing.T) {
	matrix, rhs := sparseCircuitMatrix(minimumSparseSize)
	var workspace sparseLinearWorkspace
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := workspace.solve(matrix, rhs, true, ctx.Err); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled factorization returned", err)
	}
	clear(matrix[len(matrix)-1])
	if _, err := workspace.solve(matrix, rhs, true, nil); err == nil {
		t.Fatal("singular large circuit was accepted")
	}
}

func TestDenseLargeCircuitFallsBackToDenseFactorization(t *testing.T) {
	size := minimumSparseSize
	matrix, rhs := denseMatrix(size), make([]float64, size)
	for i, row := range matrix {
		for j := range row {
			row[j] = .5
		}
		row[i] = float64(size)
		rhs[i] = float64(size) + .5*float64(size-1)
	}
	var workspace sparseLinearWorkspace
	if workspace.prepare(matrix, rhs) {
		t.Fatal("a dense circuit should use the dense factorization")
	}
	got, err := workspace.solve(matrix, rhs, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range got {
		if math.Abs(value-1) > 1e-12 {
			t.Fatalf("column %d: got %g, want 1", i, value)
		}
	}
}

func BenchmarkCircuitMatrixFactorization(b *testing.B) {
	for _, size := range []int{128, 201, 375} {
		matrix, rhs := sparseCircuitMatrix(size)
		for _, sparse := range []bool{false, true} {
			b.Run(fmt.Sprintf("%d/sparse=%t", size, sparse), func(b *testing.B) {
				var workspace sparseLinearWorkspace
				copyOf := denseMatrix(size)
				values := make([]float64, size)
				b.ReportAllocs()
				for b.Loop() {
					var err error
					if sparse {
						if !workspace.prepare(matrix, rhs) {
							b.Fatal("benchmark matrix is too dense")
						}
						_, err = workspace.factor(nil)
					} else {
						for i := range matrix {
							copy(copyOf[i], matrix[i])
						}
						copy(values, rhs)
						_, err = linearSolveMode(copyOf, values, true)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
