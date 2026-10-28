package workbench

import "math"

const PNPModelID = "elab.generic-pnp.v1"

// PNPModel has the same parameter representation as the existing NPN model.
// These are illustrative complementary parameters, not a physical matched pair.
type PNPModel = NPNModel

func IsBipolarKind(kind string) bool { return kind == "npn" || kind == "pnp" }

func hasBipolarPolarity(parts []Component, kind string) bool {
	for _, p := range parts {
		if p.Kind == kind {
			return true
		}
	}
	return false
}

func pnpModelForGraph(g Graph) *PNPModel {
	if !hasBipolarPolarity(g.Components, "pnp") {
		return nil
	}
	m := PNPDeviceModel()
	return &m
}

func PNPDeviceModel() PNPModel {
	m := TransistorModel()
	m.ID = PNPModelID
	m.Limitations = "Educational complementary PNP Ebers–Moll model; same illustrative parameter magnitudes as the generic NPN, not a named or physically matched device. No breakdown, Early effect, heating, junction capacitance, charge storage or switching delay. External capacitors provide circuit timing."
	return m
}

func bipolarModel(kind string) NPNModel {
	if kind == "pnp" {
		return PNPDeviceModel()
	}
	return TransistorModel()
}

func bipolarPolarity(kind string) float64 {
	if kind == "pnp" {
		return -1
	}
	return 1
}

// IB and IC enter the physical base/collector; IE leaves the physical emitter.
// Voltages are always physical V(B)-V(E) and V(C)-V(E). PNP currents and
// voltages therefore reverse sign. The two polarity changes cancel in the
// derivatives with respect to physical VBE/VCE.
func bipolarLinear(kind string, vbe, vce float64) (ib, ic, bb, bc, cb, cc float64) {
	sign := bipolarPolarity(kind)
	ib, ic, bb, bc, cb, cc = npnLinear(sign*vbe, sign*vce)
	return sign * ib, sign * ic, bb, bc, cb, cc
}

// Stamp all three terminals, including their emitter derivatives. No pin is
// assumed to be grounded; add and inject omit the actual reference node.
func (n *mixedNetwork) stampBipolar(p Component, x []float64, add func(int, int, float64), f []float64) {
	b, c, e := n.pin(p, "base"), n.pin(p, "collector"), n.pin(p, "emitter")
	ib, ic, bb, bc, cb, cc := componentBipolarLinear(p, voltage(x, b)-voltage(x, e), voltage(x, c)-voltage(x, e))
	inject := func(row int, current float64) {
		if row >= 0 {
			f[row] += current
		}
	}
	inject(b, ib)
	inject(c, ic)
	inject(e, -ib-ic)
	add(b, b, bb)
	add(b, c, bc)
	add(b, e, -bb-bc)
	add(c, b, cb)
	add(c, c, cc)
	add(c, e, -cb-cc)
	add(e, b, -bb-cb)
	add(e, c, -bc-cc)
	add(e, e, bb+bc+cb+cc)
}

func (n *mixedNetwork) bipolarResult(p Component, x []float64) MixedTransistor {
	vbe := voltage(x, n.pin(p, "base")) - voltage(x, n.pin(p, "emitter"))
	vce := voltage(x, n.pin(p, "collector")) - voltage(x, n.pin(p, "emitter"))
	ib, ic, _, _, _, _ := componentBipolarLinear(p, vbe, vce)
	if hasThermal(p) {
		e := n.evaluateThermal(p, x)
		ib, ic = e.currents[0], e.currents[1]
	}
	state := "active"
	sign := bipolarPolarity(p.Kind)
	if math.Abs(ib) < 1e-8 && math.Abs(ic) < 1e-8 {
		state = "cutoff"
	} else if sign*vbe > 0 && sign*(vbe-vce) > 0 {
		state = "saturation"
	} else if sign*vce < 0 {
		state = "reverse-active"
	}
	be, bc, energy := n.junctionReading(p, x)
	heat := vbe*ib + vce*ic
	// Optional profile-only reading; omit it for historical generic-v1 results.
	if !hasTransistorProfile(p) && !hasThermal(p) {
		heat = 0
	}
	ib += be + bc
	ic -= bc
	return MixedTransistor{JunctionBECurrent: be, JunctionBCCurrent: bc, JunctionEnergy: energy, ConductiveHeat: heat, TransistorID: p.ID, Model: componentBipolarModel(p), BaseCurrent: ib, CollectorCurrent: ic, EmitterCurrent: ib + ic, VBE: vbe, VCE: vce, TransistorPower: vbe*ib + vce*ic, State: state}
}

// Old saved NPN parameters are still the same model. Accept only the exact
// historical description as an alternative; never relax numeric/model checks
// or rewrite captured provenance and hashes when reading old files.
func supportedNPNModel(m NPNModel) bool {
	current := TransistorModel()
	if m == current {
		return true
	}
	current.Limitations = "Educational Ebers–Moll DC model; no named physical device, breakdown, Early effect, heating, junction capacitance or switching delay. LED reverse current is neglected. Supports a grounded-emitter switch only."
	return m == current
}
