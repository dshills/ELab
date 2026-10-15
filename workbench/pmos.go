package workbench

const PMOSModelID = "elab.generic-pmos.v1"

func PMOSDeviceModel() PMOSModel {
	return PMOSModel{PMOSModelID, -1.5, .1, .02, SiliconDiodeModel(), "Generic enhancement PMOS with square-law channel, channel-length modulation and drain-to-source body diode. Body tied to source; zero gate current. No subthreshold conduction, body effect, gate/junction capacitance, charge storage, switching loss, breakdown, heating, ratings or named device identity. External capacitors provide circuit timing. Same illustrative beta magnitude as the generic NMOS, not a physical matched pair."}
}
