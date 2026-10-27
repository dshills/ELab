package workbench

import "math"

func CurrentSymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{0, -34}, {0, -25}}, {{0, 25}, {0, 34}}}
	circle := []Point{}
	for k := 0; k <= 32; k++ {
		a := 2 * math.Pi * float64(k) / 32
		circle = append(circle, Point{25 * math.Cos(a), 25 * math.Sin(a)})
	}
	paths = append(paths, circle)
	if kind == "current-source" {
		return append(paths, []Point{{0, 15}, {0, -15}, {-6, -7}}, []Point{{0, -15}, {6, -7}})
	}
	return append(paths, []Point{{0, -15}, {0, 15}, {-6, 7}}, []Point{{0, 15}, {6, 7}}, []Point{{-17, -17}, {17, 17}})
}
