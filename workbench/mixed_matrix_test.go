package workbench

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func TestLinearZeroSkippingMatchesDenseUpdatesAndPivoting(t *testing.T) {
	random := rand.New(rand.NewSource(17))
	for _, size := range []int{1, 3, 8, 16, 32, 61} {
		for _, sparse := range []bool{false, true} {
			matrix, rhs, want := denseMatrix(size), make([]float64, size), make([]float64, size)
			for i := range want {
				want[i] = random.Float64()*2 - 1
			}
			for i := range matrix {
				sum := 1.
				for j := range matrix[i] {
					if i != j && (!sparse || random.Intn(5) == 0) {
						matrix[i][j] = random.Float64() - .5
						sum += math.Abs(matrix[i][j])
					}
				}
				matrix[i][i] = sum
				for j := range matrix[i] {
					rhs[i] += matrix[i][j] * want[j]
				}
			}
			// Moving the strongest diagonal away from its column exercises
			// real row pivoting for both update policies.
			matrix[0], matrix[size-1] = matrix[size-1], matrix[0]
			rhs[0], rhs[size-1] = rhs[size-1], rhs[0]
			copyMatrix := func() [][]float64 {
				out := denseMatrix(size)
				for i := range matrix {
					copy(out[i], matrix[i])
				}
				return out
			}
			for _, skip := range []bool{false, true} {
				got, err := linearSolveMode(copyMatrix(), append([]float64{}, rhs...), skip)
				if err != nil {
					t.Fatal(size, sparse, skip, err)
				}
				for i := range got {
					if math.Abs(got[i]-want[i]) > 1e-12 {
						t.Fatal("zero skipping changed the solution", size, sparse, skip, i, got[i], want[i])
					}
				}
			}
			selected, err := linearSolve(copyMatrix(), append([]float64{}, rhs...))
			if err != nil {
				t.Fatal(err)
			}
			for i := range selected {
				if math.Abs(selected[i]-want[i]) > 1e-12 {
					t.Fatal("size-selected solver changed the solution", size, sparse)
				}
			}
		}
	}
}

func TestLinearZeroSkippingStillRejectsInvalidSystems(t *testing.T) {
	for _, size := range []int{3, 16, 32} {
		for _, invalid := range []string{"singular", "nan", "infinite"} {
			for _, skip := range []bool{false, true} {
				matrix, rhs := denseMatrix(size), make([]float64, size)
				for i := range matrix {
					matrix[i][i], rhs[i] = 1, 1
				}
				switch invalid {
				case "singular":
					matrix[size-1][size-1] = 0
				case "nan":
					matrix[0][0] = math.NaN()
				case "infinite":
					rhs[size-1] = math.Inf(1)
				}
				if _, err := linearSolveMode(matrix, rhs, skip); err == nil {
					t.Fatal("invalid system accepted", size, invalid, skip)
				}
			}
		}
	}
}

func TestDenseMatrixRowsAndCallsAreIndependent(t *testing.T) {
	for _, size := range []int{0, 1, 8, 30} {
		first, second := denseMatrix(size), denseMatrix(size)
		if len(first) != size || len(second) != size {
			t.Fatal("wrong matrix dimension", size)
		}
		for row := range first {
			if len(first[row]) != size {
				t.Fatal("wrong row dimension", size, row)
			}
			for column, value := range first[row] {
				if value != 0 {
					t.Fatal("matrix is not zero initialized")
				}
				first[row][column] = float64(row*size + column + 1)
			}
		}
		for row := range first {
			for column, value := range first[row] {
				if value != float64(row*size+column+1) || second[row][column] != 0 {
					t.Fatal("rows or separate matrix calls share mutable values")
				}
			}
			grown := append(first[row], -1)
			grown[len(grown)-1] = -2
		}
		for row := range first {
			for column, value := range first[row] {
				if value != float64(row*size+column+1) {
					t.Fatal("growing a row overwrote another row")
				}
			}
		}
	}
}

func TestDenseMatrixPivotingPreservesLinearSolution(t *testing.T) {
	values := [][]float64{{0, 2, 1}, {1, -2, -3}, {2, 3, 1}}
	matrix := denseMatrix(len(values))
	for row := range values {
		copy(matrix[row], values[row])
	}
	solution, err := linearSolve(matrix, []float64{7, -12, 11})
	if err != nil {
		t.Fatal(err)
	}
	for i, expected := range []float64{1, 2, 3} {
		if math.Abs(solution[i]-expected) > 1e-12 {
			t.Fatal("row pivoting changed the linear solution", solution)
		}
	}
}

func TestMixedAssemblyDoesNotShareNewtonWorkspace(t *testing.T) {
	compiled, _, diagnostic := FlasherExampleGraph().Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	n := newMixedNetwork(compiled)
	trace := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	state, err := n.solve(nil, []float64{.1, 0}, 0, trace, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, firstResidual := n.equations(state.x, state.capV, .001, trace, .001)
	second, secondResidual := n.equations(state.x, state.capV, .001, trace, .001)
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstResidual, secondResidual) {
		t.Fatal("identical equations differ")
	}
	want, wantResidual := second[0][0], secondResidual[0]
	first[0][0] += 100
	firstResidual[0] += 100
	if second[0][0] != want || secondResidual[0] != wantResidual {
		t.Fatal("one Newton trial changed another trial's matrix or residual")
	}
}
