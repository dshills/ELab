package workbench

const NMOSModelID = "elab.generic-nmos.v1"

// NMOSModel pins a deliberately small, educational three-terminal model.
// Beta already includes W/L; the body is internally connected to source.
type NMOSModel struct {
	ID          string     `json:"id"`
	Threshold   float64    `json:"threshold_v"`
	Beta        float64    `json:"beta_a_per_v2"`
	Lambda      float64    `json:"lambda_per_v"`
	BodyDiode   DiodeModel `json:"body_diode"`
	Limitations string     `json:"limitations"`
}

func NMOSDeviceModel() NMOSModel {
	return NMOSModel{NMOSModelID, 1.5, .1, .02, SiliconDiodeModel(), "Generic enhancement NMOS with square-law channel, channel-length modulation and source-to-drain body diode. Body tied to source; zero gate current. No subthreshold conduction, body effect, gate/junction capacitance, charge storage, switching loss, breakdown, heating, ratings or named device identity. External capacitors provide circuit timing."}
}

type NMOSResult struct {
	ID            string    `json:"id"`
	Model         NMOSModel `json:"model"`
	GateCurrent   float64   `json:"gate_current_a"`
	DrainCurrent  float64   `json:"drain_current_a"`
	SourceCurrent float64   `json:"source_current_a"`
	VGS           float64   `json:"vgs_v"`
	VDS           float64   `json:"vds_v"`
	Power         float64   `json:"power_w"`
	State         string    `json:"state"`
}

// nmosChannel returns drain current and its gate/drain derivatives for VDS>=0.
// This is the long-channel square law, with no numerical off conductance.
func nmosChannel(vgs, vds float64) (i, gm, gd float64) {
	m := NMOSDeviceModel()
	u := vgs - m.Threshold
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

// Positive ID enters drain and leaves source. Derivatives use VGS and VDS;
// source derivative is -(gm+gd). Reverse channel swaps the intrinsic ends,
// while the physical body diode retains its source-to-drain orientation.
func nmosLinear(vgs, vds float64) (id, gm, gd float64) {
	if vds >= 0 {
		id, gm, gd = nmosChannel(vgs, vds)
	} else {
		i, g, d := nmosChannel(vgs-vds, -vds)
		id, gm, gd = -i, -g, g+d
	}
	body, conductance := siliconDiodeCurrent(-vds)
	return id - body, gm, gd + conductance
}

func nmosState(vgs, vds float64) string {
	if vds < 0 {
		return "reverse"
	}
	if vgs <= NMOSDeviceModel().Threshold {
		return "cutoff"
	}
	if vds < vgs-NMOSDeviceModel().Threshold {
		return "triode"
	}
	return "saturation"
}
