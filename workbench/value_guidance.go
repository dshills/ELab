package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Explain only an actual value refusal on a known numeric part. Malformed
// requests, missing targets and other refusal causes keep the generic guidance.
// This runs under the service lock; SetValue validates a copy without mutation.
func componentValueRequestHelp(graph Graph, raw json.RawMessage, help *DiagnosticHelp) {
	var p ValueParams
	if params(raw, &p, "id", "value") != nil {
		return
	}
	part, ok := graph.Component(p.ID)
	if !ok {
		return
	}
	domain, ok := ValueDomainForKind(part.Kind)
	if !ok {
		return
	}
	if _, err := SetValue(part, p.Value); err == nil {
		return
	}

	help.Summary = fmt.Sprintf("%s needs a valid %s.", part.ID, strings.ReplaceAll(domain.Parameter, "_", " "))
	help.Action = fmt.Sprintf("Enter %s. %s.", domain.Range, domain.Syntax)
	switch part.Kind {
	case "sine-source":
		help.Action += fmt.Sprintf(" With the current %s offset, amplitude cannot exceed %s.", FormatValue(part.Sine.Offset, "V"), FormatValue(domain.Maximum-math.Abs(part.Sine.Offset), "V"))
	case "finite-pulse-source":
		help.Action += fmt.Sprintf(" High level must be at least the current low level of %s.", FormatValue(part.Pulse.LowVoltage, "V"))
	}
	help.Action += fmt.Sprintf(" The previous value (%s) is unchanged.", FormatValue(part.Value, part.Unit))
}
