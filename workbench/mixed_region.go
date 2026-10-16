package workbench

// MixedRegion identifies the device providing the network trace's state.
// Existing traces prefer the first NPN, then the first MOSFET, independent of
// selection. Keeping identity and state together prevents a display from
// attributing one device family's state to another family in a mixed circuit.
func MixedRegion(result *MixedResult) (deviceID, state string) {
	if result != nil {
		if len(result.Transistors) > 0 {
			q := result.Transistors[0]
			return q.TransistorID, q.State
		}
		if len(result.MOSFETs) > 0 {
			m := result.MOSFETs[0]
			return m.ID, m.State
		}
	}
	return "", "network"
}
