package workbench

import "math"

// MOSFETModel pins a deliberately small, educational three-terminal model.
// Beta already includes W/L; the body is internally connected to source.
type MOSFETModel struct {
	ID          string     `json:"id"`
	Threshold   float64    `json:"threshold_v"`
	Beta        float64    `json:"beta_a_per_v2"`
	Lambda      float64    `json:"lambda_per_v"`
	BodyDiode   DiodeModel `json:"body_diode"`
	Limitations string     `json:"limitations"`
}

type MOSFETResult struct {
	ID            string      `json:"id"`
	Model         MOSFETModel `json:"model"`
	GateCurrent   float64     `json:"gate_current_a"`
	DrainCurrent  float64     `json:"drain_current_a"`
	SourceCurrent float64     `json:"source_current_a"`
	VGS           float64     `json:"vgs_v"`
	VDS           float64     `json:"vds_v"`
	Power         float64     `json:"power_w"`
	State         string      `json:"state"`
}

// Retain the original Go names and exact NMOS serialized shape.
type NMOSModel = MOSFETModel
type NMOSResult = MOSFETResult
type PMOSModel = MOSFETModel

// IsMOSFETKind identifies the supported three-pin enhancement MOSFETs.
func IsMOSFETKind(kind string) bool { return kind == "nmos" || kind == "pmos" }

func mosfetModel(kind string) MOSFETModel {
	if kind == "pmos" {
		return PMOSDeviceModel()
	}
	return NMOSDeviceModel()
}

func (n *mixedNetwork) mosfetResult(p Component, x []float64) MOSFETResult {
	vgs := voltage(x, n.pin(p, "gate")) - voltage(x, n.pin(p, "source"))
	vds := voltage(x, n.pin(p, "drain")) - voltage(x, n.pin(p, "source"))
	id, _, _ := mosfetLinear(p.Kind, vgs, vds)
	return MOSFETResult{ID: p.ID, Model: mosfetModel(p.Kind), DrainCurrent: id, SourceCurrent: id, VGS: vgs, VDS: vds, Power: vds * id, State: mosfetState(p.Kind, vgs, vds)}
}

// positiveMOSChannel uses polarity-normalized voltages; beta includes W/L.
func positiveMOSChannel(m MOSFETModel, vgs, vds float64) (i, gm, gd float64) {
	u := vgs - math.Abs(m.Threshold)
	if u <= 0 {
		return 0, 0, 0
	}
	mod := 1 + m.Lambda*vds
	if vds < u {
		base := m.Beta * (u*vds - .5*vds*vds)
		return base * mod, m.Beta * vds * mod, m.Beta*(u-vds)*mod + base*m.Lambda
	}
	base := .5 * m.Beta * u * u
	return base * mod, m.Beta * u * mod, base * m.Lambda
}

// ID is positive into physical drain, IS positive out of physical source.
// Two sign changes cancel in the derivatives; PMOS gm/gd retain their
// normalized values while current and voltages reverse polarity.
func mosfetLinear(kind string, vgs, vds float64) (id, gm, gd float64) {
	sign := 1.
	if kind == "pmos" {
		sign = -1
	}
	m := mosfetModel(kind)
	vg, vd := sign*vgs, sign*vds
	if vd >= 0 {
		id, gm, gd = positiveMOSChannel(m, vg, vd)
	} else {
		i, g, d := positiveMOSChannel(m, vg-vd, -vd)
		id, gm, gd = -i, -g, g+d
	}
	body, g := siliconDiodeCurrent(-vd)
	return sign * (id - body), gm, gd + g
}

func mosfetState(kind string, vgs, vds float64) string {
	if kind == "pmos" {
		vgs, vds = -vgs, -vds
	}
	threshold := math.Abs(mosfetModel(kind).Threshold)
	if vds < 0 {
		return "reverse"
	}
	if vgs <= threshold {
		return "cutoff"
	}
	if vds < vgs-threshold {
		return "triode"
	}
	return "saturation"
}
