package conformance

import "math"

// Collapse the 55 unloaded series resistors into four stages. The insulated
// MOS gate draws zero current, so the RC subsystem is exactly linear and
// independent of the drain/LED load. Evaluate its 4x4 matrix exponential by
// scaling/Taylor/squaring, without ELab's MNA or time-stepping implementation.
func capacityRCReference(at float64) [4]float64 {
	var a [4][4]float64
	r := [4]float64{14000, 14000, 14000, 13000}
	for i := range a {
		a[i][i] = -1 / (r[i] * 1e-7)
		if i > 0 {
			a[i][i-1] = 1 / (r[i] * 1e-7)
		}
		if i < 3 {
			a[i][i] -= 1 / (r[i+1] * 1e-7)
			a[i][i+1] = 1 / (r[i+1] * 1e-7)
		}
	}
	propagate := func(value [4]float64, elapsed, source float64) [4]float64 {
		if elapsed == 0 {
			return value
		}
		var matrix [4][4]float64
		norm := 0.
		for i := range a {
			row := 0.
			for j := range a[i] {
				matrix[i][j] = a[i][j] * elapsed
				row += math.Abs(matrix[i][j])
			}
			norm = math.Max(norm, row)
		}
		squares := max(0, int(math.Ceil(math.Log2(norm/.5))))
		for i := range matrix {
			for j := range matrix[i] {
				matrix[i][j] = math.Ldexp(matrix[i][j], -squares)
			}
		}
		var exponential, term [4][4]float64
		for i := range term {
			term[i][i], exponential[i][i] = 1, 1
		}
		multiply := func(left, right [4][4]float64) (product [4][4]float64) {
			for i := range product {
				for j := range product[i] {
					for k := range left[i] {
						product[i][j] += left[i][k] * right[k][j]
					}
				}
			}
			return product
		}
		for order := 1; order <= 32; order++ {
			term = multiply(term, matrix)
			for i := range term {
				for j := range term[i] {
					term[i][j] /= float64(order)
					exponential[i][j] += term[i][j]
				}
			}
		}
		for range squares {
			exponential = multiply(exponential, exponential)
		}
		var result [4]float64
		for i := range result {
			result[i] = source
			for j := range value {
				result[i] += exponential[i][j] * (value[j] - source)
			}
		}
		return result
	}
	charged := propagate([4]float64{}, math.Min(at, .1), 5)
	return propagate(charged, math.Max(0, at-.1), 0)
}
