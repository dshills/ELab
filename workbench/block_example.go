package workbench

func RCBlockExampleGraph() Graph {
	s := Service{graph: StepResponseExampleGraph(), usedIDs: map[string]bool{}}
	p := BlockCreateParams{ID: "FILTER", Name: "RC low-pass", Position: Point{X: 0, Y: 300}, Members: []string{"R1", "C1"}, Ports: []BlockPort{{Name: "in", Terminal: "R1.1"}, {Name: "ground", Terminal: "C1.2"}, {Name: "out", Terminal: "C1.1"}}, Parameters: []BlockParameter{{Name: "resistance", ComponentID: "R1"}, {Name: "capacitance", ComponentID: "C1"}}}
	if err := s.createBlockLocked(p); err != nil {
		panic("invalid built-in RC block: " + err.Error())
	}
	return s.graph
}
