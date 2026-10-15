package workbench

import (
	"math"
	"strconv"
	"unicode/utf8"
)

// FormatValue produces editable V/F/Ω text without losing a stored float's bits.
// Shorter engineering forms are used only after ParseValue verifies them.
// The SI fallback uses the shortest round-trip decimal representation.
func FormatValue(value float64, unit string) string {
	best := strconv.FormatFloat(value, 'g', -1, 64) + " " + unit
	if !finite(value) {
		return best
	}
	bestLength := utf8.RuneCountInString(best)
	normalized := func(n float64) bool { v := math.Abs(n); return v >= 1 && v < 1000 }
	bestNormalized := value == 0 || normalized(value)
	for _, prefix := range valuePrefixes(unit) {
		// Prefer the canonical display spellings while the parser retains all
		// accepted aliases, including K, u and Greek-mu variants.
		if prefix.suffix == "K" || prefix.suffix == "u" || prefix.suffix == "μ" {
			continue
		}
		scaled := value / prefix.scale
		if !finite(scaled) || value != 0 && scaled == 0 {
			continue
		}
		candidate := strconv.FormatFloat(scaled, 'g', -1, 64) + " " + prefix.suffix + unit
		parsed, err := ParseValue(candidate, unit)
		if err != nil || math.Float64bits(parsed) != math.Float64bits(value) {
			continue
		}
		length := utf8.RuneCountInString(candidate)
		isNormalized := normalized(scaled)
		if length < bestLength || length == bestLength && isNormalized && !bestNormalized {
			best, bestLength, bestNormalized = candidate, length, isNormalized
		}
	}
	return best
}
