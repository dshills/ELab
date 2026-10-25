package workbench

func ACRCExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "0", -250, 0, 0)
	b.ground("GND", "V1.negative", -250, 180)
	b.part("R1", "resistor", "1k", 0, -100, 0)
	b.part("C1", "capacitor", "1u", 200, 0, 0)
	b.wire("V1.positive", "R1.1")
	b.wire("R1.2", "C1.1")
	b.ground("GNDC", "C1.2", 200, 180)
	return b.graph
}

func ACOpAmpExampleGraph() Graph {
	b := exampleBuilder{graph: OpAmpInvertingExampleGraph()}
	b.part("CF", "capacitor", "10n", 80, 340, 0)
	b.wire("CF.1", "RF.1")
	b.wire("CF.2", "RF.2")
	return b.graph
}
