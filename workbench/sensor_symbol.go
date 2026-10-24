package workbench

import "math"

func SensorSymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{-55, 0}, {-43, 0}, {-35, -12}, {-21, 12}, {-7, -12}, {7, 12}, {21, -12}, {35, 12}, {43, 0}, {55, 0}}}
	if kind == "thermistor" {
		return append(paths, []Point{{-34, 27}, {34, -27}}, []Point{{-34, 27}, {-45, 27}})
	}
	circle := []Point{}
	for k := 0; k <= 48; k++ {
		a := 2 * math.Pi * float64(k) / 48
		circle = append(circle, Point{46 * math.Cos(a), 46 * math.Sin(a)})
	}
	return append(paths, circle, []Point{{-62, -62}, {-29, -29}, {-31, -42}}, []Point{{-29, -29}, {-42, -31}}, []Point{{-40, -72}, {-7, -39}, {-9, -52}}, []Point{{-7, -39}, {-20, -41}})
}
