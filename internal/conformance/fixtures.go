package conformance

type part struct {
	id, kind, value string
	x, y            float64
}
type circuit struct {
	name   string
	parts  []part
	wires  [][2]string
	target string
}

var circuits = []circuit{
	{
		name: "led",
		parts: []part{
			{"Supply", "dc-source", "5 V", -200, 0},
			{"Limit", "resistor", "1k", 0, -110},
			{"Light", "led", "", 200, 0},
			{"Earth", "ground", "", 0, 110},
		},
		wires: [][2]string{
			{"Supply.positive", "Limit.1"},
			{"Limit.2", "Light.anode"},
			{"Light.cathode", "Earth.reference"},
			{"Earth.reference", "Supply.negative"},
		},
		target: "Light",
	},
	{
		name: "rc",
		parts: []part{
			{"Supply", "pulse-source", "5 V", -200, 0},
			{"Limit", "resistor", "10k", 0, -110},
			{"Storage", "capacitor", "1uF", 200, 0},
			{"Earth", "ground", "", 0, 110},
		},
		wires: [][2]string{
			{"Supply.positive", "Limit.1"},
			{"Limit.2", "Storage.1"},
			{"Storage.2", "Earth.reference"},
			{"Earth.reference", "Supply.negative"},
		},
		target: "Storage",
	},
	{
		name: "npn",
		parts: []part{
			{"Supply", "dc-source", "5 V", 260, -100},
			{"Drive", "dc-source", "5 V", -260, 60},
			{"BaseR", "resistor", "10k", -100, 60},
			{"Limit", "resistor", "1k", 100, -180},
			{"Transistor", "npn", "", 100, 60},
			{"Earth", "ground", "", 100, 180},
		},
		wires: [][2]string{
			{"Supply.positive", "Limit.1"},
			{"Limit.2", "Transistor.collector"},
			{"Drive.positive", "BaseR.1"},
			{"BaseR.2", "Transistor.base"},
			{"Transistor.emitter", "Earth.reference"},
			{"Supply.negative", "Earth.reference"},
			{"Drive.negative", "Earth.reference"},
		},
		target: "Transistor",
	},
}
