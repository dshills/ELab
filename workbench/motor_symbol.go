package workbench

import "math"

func MotorSymbolPaths() [][]Point {
	circle := make([]Point, 49)
	for k := range circle {
		a := float64(k) * 2 * math.Pi / 48
		circle[k] = Point{32 * math.Cos(a), 32 * math.Sin(a)}
	}
	return [][]Point{circle, {{0, -55}, {0, -32}}, {{0, 32}, {0, 55}}, {{-17, 13}, {-17, -13}, {0, 7}, {17, -13}, {17, 13}}}
}
