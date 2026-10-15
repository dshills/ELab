package workbench

import (
	"fmt"
	"time"
)

const (
	MaxComponents            = 100 // Drawing objects, including reference/connectivity helpers.
	MaxWires                 = 300
	MaxSimulatedComponents   = 64
	MaxElectricalNodes       = 100 // Includes the ground reference.
	MaxMixedNodes            = MaxElectricalNodes - 1
	MaxNetworkSources        = 3
	MaxNetworkCapacitors     = 4
	MaxNetworkTransistors    = 2
	MaxNetworkMOSFETs        = 2
	MaxNetworkResistors      = 64
	MaxNetworkSwitches       = 8
	MaxNetworkDiodes         = 8
	MaxNetworkPotentiometers = 64
	MaxNetworkLEDs           = 4
)

type CircuitLimits struct {
	IntegrationAttempts int `json:"integration_attempts_per_sample"`
	IntegrationWallMS   int `json:"integration_wall_ms_per_sample"`
	PinnedReadings      int `json:"pinned_readings"`
	DrawingObjects      int `json:"drawing_objects"`
	Wires               int `json:"wires"`
	SimulatedComponents int `json:"simulated_components"`
	ElectricalNodes     int `json:"electrical_nodes_including_ground"`
	Sources             int `json:"sources"`
	Capacitors          int `json:"capacitors"`
	Transistors         int `json:"npn_transistors"`
	MOSFETs             int `json:"nmos_transistors"`
	PMOS                int `json:"pmos_transistors"`
	CombinedMOSFETs     int `json:"mosfets_total"`
	Resistors           int `json:"resistors"`
	Switches            int `json:"switches"`
	Diodes              int `json:"silicon_diodes"`
	Potentiometers      int `json:"potentiometers"`
	LEDs                int `json:"leds"`
	ScopeTraces         int `json:"scope_traces"`
}

func Limits() CircuitLimits {
	return CircuitLimits{IntegrationAttempts: MaxIntegrationAttempts, IntegrationWallMS: int(MaxIntegrationWallTime / time.Millisecond), PinnedReadings: MaxPinnedReadings, DrawingObjects: MaxComponents, Wires: MaxWires, SimulatedComponents: MaxSimulatedComponents, ElectricalNodes: MaxElectricalNodes, Sources: MaxNetworkSources, Capacitors: MaxNetworkCapacitors, Transistors: MaxNetworkTransistors, MOSFETs: MaxNetworkMOSFETs, PMOS: MaxNetworkMOSFETs, CombinedMOSFETs: MaxNetworkMOSFETs, Resistors: MaxNetworkResistors, Switches: MaxNetworkSwitches, Diodes: MaxNetworkDiodes, Potentiometers: MaxNetworkPotentiometers, LEDs: MaxNetworkLEDs, ScopeTraces: MaxScopeProbes}
}
func SimulatedComponentCount(g Graph) int {
	count := 0
	for _, p := range g.Components {
		if p.Kind != "ground" && !IsConnectivityPrimitive(p.Kind) {
			count++
		}
	}
	return count
}
func networkLimitMessage() string {
	return fmt.Sprintf("Network simulation supports at most %d simulated parts and %d electrical nodes including ground: %d sources, %d capacitors, %d NPNs, %d total N/P-channel MOSFETs, %d resistors, %d switches, %d diodes, %d potentiometers and %d LEDs. Excess drawings remain editable and savable.", MaxSimulatedComponents, MaxElectricalNodes, MaxNetworkSources, MaxNetworkCapacitors, MaxNetworkTransistors, MaxNetworkMOSFETs, MaxNetworkResistors, MaxNetworkSwitches, MaxNetworkDiodes, MaxNetworkPotentiometers, MaxNetworkLEDs)
}
