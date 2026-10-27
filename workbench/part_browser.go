package workbench

import "strings"

// PartCategory is a stable browser group, shared by native and agent clients.
// Counts describe the whole catalog, independently of the current search.
type PartCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type PartCategoryParams struct {
	Category string `json:"category"`
}

func PartCategories() []PartCategory {
	groups := []PartCategory{{ID: "common", Name: "Common"}, {ID: "all", Name: "All components"}, {ID: "sources", Name: "Sources"}, {ID: "motors", Name: "Motors"}, {ID: "passives", Name: "Passives"}, {ID: "semiconductors", Name: "Semiconductors"}, {ID: "amplifiers", Name: "Amplifiers / ICs"}, {ID: "logic", Name: "Logic"}, {ID: "switches", Name: "Switches"}, {ID: "connections", Name: "Connections"}}
	for _, part := range libraryParts() {
		category, common := partGroup(part.Kind)
		for i := range groups {
			if groups[i].ID == "all" || groups[i].ID == category || groups[i].ID == "common" && common {
				groups[i].Count++
			}
		}
	}
	return groups
}

func browserCategory(category string) string {
	if category == "" {
		return "common"
	}
	return category
}

func partGroup(kind string) (category string, common bool) {
	switch kind {
	case "shift-register", "bcd-decoder", "seven-segment", "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter", "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		category = "logic"
	case "hbridge", "dc-motor", "pwm-controller":
		category = "motors"
	case "current-source", "electronic-load", "battery", "dc-source", "sine-source", "finite-pulse-source", "pulse-source":
		category = "sources"
	case "speaker", "piezo", "active-buzzer", "transformer", "thermistor", "ldr", "resistor", "capacitor", "inductor", "potentiometer":
		category = "passives"
	case "njfet", "pjfet", "optocoupler", "timer555", "led", "diode", "zener", "schottky", "npn", "pnp", "nmos", "pmos":
		category = "semiconductors"
	case "linear-regulator", "ldo-regulator", "opamp", "comparator":
		category = "amplifiers"
	case "relay", "switch", "pushbutton", "bouncing-pushbutton":
		category = "switches"
	case "ground", "junction", "net-label":
		category = "connections"
	}
	switch kind {
	case "dc-source", "resistor", "capacitor", "led", "switch", "ground":
		common = true
	}
	return
}

func browserParts(category, query string) []Part {
	parts := filteredParts(query)
	if strings.TrimSpace(query) != "" {
		return parts
	}
	out := []Part{}
	category = browserCategory(category)
	for _, part := range parts {
		if category == "all" || category == "common" && part.Common || part.Category == category {
			out = append(out, part)
		}
	}
	return out
}
