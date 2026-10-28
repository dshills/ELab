package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

const PhotodiodeModelID = "elab.generic-photodiode.v1"
const photodiodeDescription = "Generic 27 C photodiode: PN Shockley junction (n=1.5, configured saturation dark current, continuous linear continuation above exponent40), fixed 1 Gohm shunt, photocurrent=responsivity*incident optical watts directed cathode to anode, and optional constant junction capacitance initially uncharged. No spectral/lux conversion, avalanche, voltage-dependent capacitance, transit delay, thermal behavior or manufacturer fit. Terminal power may be negative: light is an external energy source. Junction capacitance consumes existing storage/part limits."

type PhotodiodeConfig struct {
	Version      string  `json:"version"`
	Responsivity float64 `json:"responsivity_a_per_w"`
	Light        float64 `json:"incident_light_uw"`
	DarkCurrent  float64 `json:"dark_saturation_current_a"`
	Capacitance  float64 `json:"junction_capacitance_f"`
}
type PhotodiodeParams struct {
	ID         string           `json:"id"`
	Parameters PhotodiodeConfig `json:"parameters"`
}
type OpticalLightParams struct {
	ID    string  `json:"id"`
	Light float64 `json:"incident_light_uw"`
}
type PhotodiodeResult struct {
	ID              string           `json:"id"`
	Model           string           `json:"model"`
	Parameters      PhotodiodeConfig `json:"parameters"`
	Voltage         float64          `json:"voltage_v"`
	Current         float64          `json:"terminal_current_a"`
	Photocurrent    float64          `json:"photocurrent_a"`
	JunctionCurrent float64          `json:"junction_current_a"`
	StorageCurrent  float64          `json:"storage_current_a"`
	Energy          float64          `json:"stored_energy_j"`
	Power           float64          `json:"terminal_power_w"`
}

func DefaultPhotodiodeConfig() PhotodiodeConfig {
	return PhotodiodeConfig{PhotodiodeModelID, .5, 10, 1e-9, 100e-12}
}
func photodiodeFields() []string {
	return []string{"version", "responsivity_a_per_w", "incident_light_uw", "dark_saturation_current_a", "junction_capacitance_f"}
}
func validatePhotodiode(c PhotodiodeConfig) error {
	if c.Version != PhotodiodeModelID || !finite(c.Responsivity) || c.Responsivity < .01 || c.Responsivity > 1 || !finite(c.Light) || c.Light < 0 || c.Light > 1000 || !finite(c.DarkCurrent) || c.DarkCurrent < 1e-15 || c.DarkCurrent > 1e-6 || !finite(c.Capacitance) || c.Capacitance < 0 || c.Capacitance > 1e-6 {
		return fmt.Errorf("photodiode requires supported version, responsivity .01..1 A/W, light 0..1000 µW, dark saturation 1 fA..1 µA, capacitance 0..1 µF")
	}
	return nil
}
func hasPhotodiodes(g Graph) bool {
	for _, p := range g.Components {
		if p.Kind == "photodiode" {
			return true
		}
	}
	return false
}
func photoCap(id string) string { return "$" + id + "_photo_storage" }
func lowerPhotodiode(c *Compiled, p Component) []Component {
	out := []Component{p}
	if p.Device.Photo.Capacitance == 0 {
		return out
	}
	zero := 0.
	cap := Component{ID: photoCap(p.ID), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: p.Device.Photo.Capacitance, InitialVoltage: &zero}
	c.TerminalNet[cap.ID+".1"] = c.TerminalNet[p.ID+".anode"]
	c.TerminalNet[cap.ID+".2"] = c.TerminalNet[p.ID+".cathode"]
	return append(out, cap)
}
func photodiodeLaw(p Component, v float64) (current, g, junction, photo float64) {
	c := p.Device.Photo
	nv := 1.5 * 1.380649e-23 * 300.15 / 1.602176634e-19
	z := v / nv
	limited := math.Min(40, z)
	junction = c.DarkCurrent * math.Expm1(limited)
	g = c.DarkCurrent * math.Exp(limited) / nv
	if z > 40 {
		junction += g * (v - 40*nv)
	}
	photo = c.Responsivity * c.Light * 1e-6
	return junction + v*1e-9 - photo, g + 1e-9, junction, photo
}
func (n *mixedNetwork) photoResult(p Component, x []float64) PhotodiodeResult {
	v := voltage(x, n.pin(p, "anode")) - voltage(x, n.pin(p, "cathode"))
	i, _, j, photo := photodiodeLaw(p, v)
	storage := 0.
	for k, c := range n.caps {
		if c.ID == photoCap(p.ID) && !n.dc {
			storage = x[n.capacitorIndex(k)]
		}
	}
	return PhotodiodeResult{ID: p.ID, Model: PhotodiodeModelID, Parameters: *p.Device.Photo, Voltage: v, Current: i + storage, Photocurrent: photo, JunctionCurrent: j, StorageCurrent: storage, Energy: .5 * p.Device.Photo.Capacitance * v * v, Power: v * (i + storage)}
}
func (s *Service) configurePhotodiodeLocked(raw json.RawMessage) error {
	var v PhotodiodeParams
	if err := params(raw, &v, "id", "parameters"); err != nil {
		return err
	}
	if err := paramsNested(raw, "parameters", &PhotodiodeConfig{}, photodiodeFields()...); err != nil {
		return err
	}
	if err := validatePhotodiode(v.Parameters); err != nil {
		return err
	}
	for k, p := range s.graph.Components {
		if p.ID == v.ID && p.Kind == "photodiode" {
			s.graph.Components[k].Device = &DeviceParameters{Photo: &v.Parameters}
			return nil
		}
	}
	return fmt.Errorf("choose a photodiode")
}
func (s *Service) setOpticalLightLocked(v OpticalLightParams) error {
	p, ok := s.graph.Component(v.ID)
	if !ok || p.Kind != "photodiode" {
		return fmt.Errorf("choose a photodiode")
	}
	c := *p.Device.Photo
	c.Light = v.Light
	if err := validatePhotodiode(c); err != nil {
		return err
	}
	if p.Device.Photo.Light == v.Light {
		if s.activeInputWorkLocked() && sensorInputID(&s.inputJob.work) == v.ID {
			s.cancelInputWorkLocked("superseded")
		}
		return nil
	}
	g := s.graph.Clone()
	for k, p := range g.Components {
		if p.ID == v.ID {
			g.Components[k].Device.Photo.Light = v.Light
		}
	}
	if s.transient != nil && !s.transient.Stale && s.mixedState != nil && IsNetworkAnalysis(s.mixedState.net.c.Analysis) {
		return s.startInputWorkLocked("", nil, g, fmt.Sprintf("sensor:photodiode:%s:%.17g", v.ID, v.Light))
	}
	s.graph = g
	return nil
}
func (n *mixedNetwork) restorePhotodiodes(r *MixedResult, x []float64) error {
	seen := map[string]bool{}
	for _, a := range r.Photodiodes {
		p, ok := n.authored.Component(a.ID)
		if !ok || p.Kind != "photodiode" || seen[a.ID] || a.Model != PhotodiodeModelID || a.Parameters != *p.Device.Photo || !finite(a.StorageCurrent) {
			return fmt.Errorf("invalid recorded photodiode identity/model")
		}
		seen[a.ID] = true
		found := p.Device.Photo.Capacitance == 0
		for k, c := range n.caps {
			if c.ID == photoCap(p.ID) {
				x[n.capacitorIndex(k)] = a.StorageCurrent
				found = true
			}
		}
		if !found {
			return fmt.Errorf("missing photodiode storage")
		}
		e := n.photoResult(p, x)
		for _, pair := range [][2]float64{{a.Voltage, e.Voltage}, {a.Current, e.Current}, {a.Photocurrent, e.Photocurrent}, {a.JunctionCurrent, e.JunctionCurrent}, {a.StorageCurrent, e.StorageCurrent}, {a.Energy, e.Energy}, {a.Power, e.Power}} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("photodiode readings disagree with stored charge/illumination")
			}
		}
		if p.Device.Photo.Capacitance == 0 && a.StorageCurrent != 0 {
			return fmt.Errorf("photodiode has no storage current")
		}
	}
	for _, p := range n.authored.Components {
		if p.Kind == "photodiode" && !seen[p.ID] {
			return fmt.Errorf("missing photodiode reading")
		}
	}
	return nil
}
func PhotodiodeSymbolPaths() [][]Point {
	return [][]Point{{{0, -32}, {0, -24}}, {{-20, -24}, {20, -24}, {0, 17}, {-20, -24}}, {{-22, 19}, {22, 19}}, {{0, 19}, {0, 32}}, {{40, -25}, {21, -6}, {21, -14}, {21, -6}, {29, -6}}, {{48, -12}, {29, 7}, {29, -1}, {29, 7}, {37, 7}}}
}
