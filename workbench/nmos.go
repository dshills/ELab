package workbench

const NMOSModelID = "elab.generic-nmos.v1"

func NMOSDeviceModel() NMOSModel {
	return NMOSModel{NMOSModelID, 1.5, .1, .02, SiliconDiodeModel(), "Generic enhancement NMOS with square-law channel, channel-length modulation and source-to-drain body diode. Body tied to source; zero gate current. No subthreshold conduction, body effect, gate/junction capacitance, charge storage, switching loss, breakdown, heating, ratings or named device identity. External capacitors provide circuit timing."}
}

// Compatibility helpers pin the original NMOS test/kernel entry points.
func nmosChannel(vgs, vds float64) (i, gm, gd float64) {
	return positiveMOSChannel(NMOSDeviceModel(), vgs, vds)
}
func nmosLinear(vgs, vds float64) (id, gm, gd float64) { return mosfetLinear("nmos", vgs, vds) }
func nmosState(vgs, vds float64) string                { return mosfetState("nmos", vgs, vds) }
