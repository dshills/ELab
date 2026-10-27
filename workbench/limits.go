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
	MaxNetworkTransistors    = 16
	MaxNetworkMOSFETs        = 2
	MaxNetworkOpAmps         = 2
	MaxNetworkResistors      = 64
	MaxNetworkSwitches       = 8
	MaxNetworkDiodes         = 8
	MaxNetworkPotentiometers = 64
	MaxNetworkLEDs           = 4
)

type CircuitLimits struct {
	JFETs               int `json:"jfets"`
	CombinedFETs        int `json:"field_effect_transistors_total"`
	Timers555           int `json:"timers555,omitempty"`
	PNP                 int `json:"pnp_transistors,omitempty"`
	CombinedBipolar     int `json:"bipolar_transistors_total,omitempty"`
	IntegrationAttempts int `json:"integration_attempts_per_sample"`
	IntegrationWallMS   int `json:"integration_wall_ms_per_sample"`
	PinnedReadings      int `json:"pinned_readings"`
	DrawingObjects      int `json:"drawing_objects"`
	Wires               int `json:"wires"`
	SimulatedComponents int `json:"simulated_components"`
	ElectricalNodes     int `json:"electrical_nodes_including_ground"`
	Sources             int `json:"sources"`
	Capacitors          int `json:"capacitors"`
	Inductors           int `json:"inductors"`
	Transistors         int `json:"npn_transistors"`
	MOSFETs             int `json:"nmos_transistors"`
	PMOS                int `json:"pmos_transistors"`
	CombinedMOSFETs     int `json:"mosfets_total"`
	OpAmps              int `json:"opamps"`
	Resistors           int `json:"resistors"`
	Switches            int `json:"switches"`
	// Diodes is the combined silicon/Schottky/Zener budget. Keep the legacy
	// JSON key for existing capacity reports and clients.
	Diodes         int `json:"silicon_diodes"`
	Potentiometers int `json:"potentiometers"`
	LEDs           int `json:"leds"`
	ScopeTraces    int `json:"scope_traces"`
}

func Limits() CircuitLimits {
	return CircuitLimits{JFETs: MaxNetworkMOSFETs, CombinedFETs: MaxNetworkMOSFETs, Timers555: MaxNetworkTimers555, PNP: MaxNetworkTransistors, CombinedBipolar: MaxNetworkTransistors, IntegrationAttempts: MaxIntegrationAttempts, IntegrationWallMS: int(MaxIntegrationWallTime / time.Millisecond), PinnedReadings: MaxPinnedReadings, DrawingObjects: MaxComponents, Wires: MaxWires, SimulatedComponents: MaxSimulatedComponents, ElectricalNodes: MaxElectricalNodes, Sources: MaxNetworkSources, Capacitors: MaxNetworkCapacitors, Inductors: MaxNetworkInductors, Transistors: MaxNetworkTransistors, MOSFETs: MaxNetworkMOSFETs, PMOS: MaxNetworkMOSFETs, CombinedMOSFETs: MaxNetworkMOSFETs, OpAmps: MaxNetworkOpAmps, Resistors: MaxNetworkResistors, Switches: MaxNetworkSwitches, Diodes: MaxNetworkDiodes, Potentiometers: MaxNetworkPotentiometers, LEDs: MaxNetworkLEDs, ScopeTraces: MaxScopeProbes}
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
	return fmt.Sprintf("Network simulation supports at most %d simulated parts and %d electrical nodes including ground: %d sources, %d capacitors, %d inductors, %d total NPN/PNP transistors, %d total N/P-channel MOSFETs/JFETs, %d op-amps, %d 555 timers, %d resistors, %d switches, %d total silicon/Schottky/Zener diodes, %d potentiometers and %d LEDs. Excess drawings remain editable and savable.", MaxSimulatedComponents, MaxElectricalNodes, MaxNetworkSources, MaxNetworkCapacitors, MaxNetworkInductors, MaxNetworkTransistors, MaxNetworkMOSFETs, MaxNetworkOpAmps, MaxNetworkTimers555, MaxNetworkResistors, MaxNetworkSwitches, MaxNetworkDiodes, MaxNetworkPotentiometers, MaxNetworkLEDs)
}
