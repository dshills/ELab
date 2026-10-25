package workbench

// networkReferencePair distinguishes conductive paths from control coupling.
// Preserve legacy compilation for networks without op-amps. With op-amps,
// an ideal MOS gate cannot supply a common-mode reference to their input pair.
func networkReferencePair(p Component, one, two string, hasOpAmp bool) bool {
	if p.Kind == "transformer" {
		primary := func(pin string) bool { return pin == "primary1" || pin == "primary2" }
		return primary(one) == primary(two)
	}
	if p.Kind == "optocoupler" {
		input := func(pin string) bool { return pin == "anode" || pin == "cathode" }
		return input(one) == input(two)
	}
	if p.Kind == "relay" {
		coil := func(pin string) bool { return pin == "coil1" || pin == "coil2" }
		return coil(one) == coil(two)
	}
	if p.Kind == "opamp" {
		group := func(pin string) int {
			switch pin {
			case "noninverting", "inverting":
				return 0 // The declared differential input resistor.
			case "out", "vminus":
				return 1 // The declared output bleed remains at every supply state.
			case "vplus":
				return 2 // Require an external reference for the positive rail.
			default:
				return -1
			}
		}
		a, b := group(one), group(two)
		return a >= 0 && a == b
	}
	if hasOpAmp && IsMOSFETKind(p.Kind) && (one == "gate" || two == "gate") {
		return false
	}
	return true
}
