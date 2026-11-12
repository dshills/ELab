package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const ExperimentDocumentVersion = 7
const MaxExperimentChecks = 16

// ExperimentCheck describes a repeatable test against the project's current
// circuit and sampling settings. Definitions may outlive a deleted probe.
type ExperimentCheck struct {
	Name     string `json:"name"`
	Probe    string `json:"probe"`
	Unit     string `json:"unit"`
	Rule     string `json:"rule"`
	Level    string `json:"level"`
	Duration string `json:"duration"`
}

type ExperimentChecksParams struct {
	Checks []ExperimentCheck `json:"checks"`
}

func decodeExperimentChecks(raw json.RawMessage) ([]ExperimentCheck, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || len(items) > MaxExperimentChecks {
		return nil, fmt.Errorf("choose up to %d experiment checks", MaxExperimentChecks)
	}
	var checks []ExperimentCheck
	names := make(map[string]bool, len(items))
	for _, item := range items {
		var check ExperimentCheck
		if err := params(item, &check, "name", "probe", "unit", "rule", "level", "duration"); err != nil {
			return nil, err
		}
		if _, _, err := check.Quantities(); err != nil {
			return nil, err
		}
		name := strings.ToLower(strings.TrimSpace(check.Name))
		if names[name] {
			return nil, fmt.Errorf("experiment check names must be unique")
		}
		names[name] = true
		check.Name = strings.TrimSpace(check.Name)
		checks = append(checks, check)
	}
	return checks, nil
}

func (c ExperimentCheck) Quantities() (duration, level float64, err error) {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 120 || strings.IndexFunc(c.Name, unicode.IsControl) >= 0 {
		return 0, 0, fmt.Errorf("enter a test name of up to 120 bytes")
	}
	if c.Probe == "" || len(c.Probe) > 160 || strings.IndexFunc(c.Probe, unicode.IsControl) >= 0 || c.Unit != "A" && c.Unit != "V" {
		return 0, 0, fmt.Errorf("choose a voltage or current measurement")
	}
	duration, err = ParseTimeSeconds(c.Duration)
	if err != nil || duration < 0 || duration > 10 {
		return 0, 0, fmt.Errorf("choose a duration between 0 and 10 seconds")
	}
	switch c.Rule {
	case "rises", "falls", "max":
		level, err = ParseValue(c.Level, c.Unit)
	case "reverses":
		if c.Unit != "A" || strings.TrimSpace(c.Level) != "" {
			return 0, 0, fmt.Errorf("direction reversal requires a current measurement and no threshold")
		}
	default:
		return 0, 0, fmt.Errorf("choose a test condition")
	}
	return duration, level, err
}

func addExperimentCheckDetails(id string, schema map[string]any) bool {
	if id != "experiment.checks.set" {
		return false
	}
	inputProperty(schema, "checks")["maxItems"] = MaxExperimentChecks
	schema["x-command-contract"] = commandContractDetail{
		Version: 1, Scope: "experiment-tests",
		Preconditions: []string{"Use current command and document guards and a retry key", "Choose distinct names and use a voltage or current probe ID. Rules are rises, falls, max and reverses. Reverses requires current and an empty level"},
		Effects:       []string{"Replace the project's saved definitions as one undoable edit without restarting simulation. An empty array removes all saved tests"},
		Completion:    &commandCompletionDetail{Mode: "immediate command admission", Observe: []string{"project.inspect"}},
	}
	return true
}
