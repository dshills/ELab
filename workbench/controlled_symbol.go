package workbench

func ControlSymbolPaths(kind string, m ControlMemory) [][]Point {
	if IsButtonKind(kind) {
		y := -16.
		if m.ContactClosed {
			y = 0
		}
		paths := [][]Point{{{-55, 0}, {-20, 0}}, {{20, 0}, {55, 0}}, {{-22, y}, {22, y}}, {{0, y}, {0, y - 20}}, {{-12, y - 20}, {12, y - 20}}}
		if kind == "bouncing-pushbutton" {
			paths = append(paths, []Point{{25, -30}, {31, -36}, {37, -30}, {43, -36}})
		}
		return paths
	}
	return nil
}
