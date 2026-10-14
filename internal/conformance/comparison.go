package conformance

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
)

// Compare requires identical command coverage and semantic persisted documents,
// then checks every measured SI value; missing measurements are failures.
func Compare(reference, other []Result) error {
	if len(reference) != len(circuits) || len(reference) != len(other) {
		return fmt.Errorf("workflow count differs")
	}
	for i, a := range reference {
		b := other[i]
		if a.Name != circuits[i].name || len(a.Commands) == 0 || len(a.Calls) == 0 || len(a.Measurements) == 0 {
			return fmt.Errorf("%s missing reference evidence", a.Name)
		}
		left, err := semanticDocument(a.Project)
		if err != nil {
			return err
		}
		right, err := semanticDocument(b.Project)
		if err != nil {
			return err
		}
		if a.Name != b.Name || !reflect.DeepEqual(left, right) {
			return fmt.Errorf("%s semantic project differs", a.Name)
		}
		// Poll counts may vary with job scheduling; compare ordered commands with
		// repeated adjacent operation.inspect polls collapsed.
		if !reflect.DeepEqual(collapsePolls(a.Commands), collapsePolls(b.Commands)) || !reflect.DeepEqual(collapseCalls(a.Calls), collapseCalls(b.Calls)) {
			return fmt.Errorf("%s command sequence differs", a.Name)
		}
		if len(a.Measurements) != len(b.Measurements) {
			return fmt.Errorf("%s measurement coverage differs", a.Name)
		}
		for key, value := range a.Measurements {
			other, ok := b.Measurements[key]
			if !ok || math.IsNaN(value) || math.IsInf(value, 0) || math.IsNaN(other) || math.IsInf(other, 0) || math.Abs(value-other) > 1e-10*math.Max(1, math.Abs(value)) {
				return fmt.Errorf("%s measurement %s differs: %.17g / %.17g", a.Name, key, value, other)
			}
		}
	}
	return nil
}

// Independent sessions have different creation times and publication revisions.
// Retain originals in artifacts; normalize only runtime provenance for cross-
// session semantic comparison. Graphs, models/hashes, all rows and settings stay
// exact. Within-session save/reopen above requires unnormalized equality.
func semanticDocument(document workbench.Document) (workbench.Document, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return document, err
	}
	var out workbench.Document
	if err = json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	if out.Recordings != nil {
		for i := range out.Recordings.Entries {
			m := &out.Recordings.Entries[i].Data.Metadata
			if _, err = time.Parse(time.RFC3339Nano, m.CreatedAt); err != nil {
				return out, fmt.Errorf("invalid recording creation provenance: %w", err)
			}
			m.CreatedAt = ""
			m.OperationID = ""
			m.InputRevision = 0
			m.ElectricalRevision = 0
			m.ActuationRevision = 0
			m.DocumentGeneration = 0
			if m.LED != nil {
				m.LED.RunID = 0
				m.LED.ElectricalRevision = 0
			}
			if m.NPN != nil {
				m.NPN.RunID = 0
				m.NPN.ElectricalRevision = 0
			}
			if m.Mixed != nil {
				m.Mixed.RunID = 0
				m.Mixed.ElectricalRevision = 0
			}
		}
	}
	return out, nil
}
func collapsePolls(commands []string) []string {
	var out []string
	for _, command := range commands {
		if command == "operation.inspect" && len(out) > 0 && out[len(out)-1] == command {
			continue
		}
		out = append(out, command)
	}
	return out
}

func collapseCalls(calls []Call) []Call {
	var out []Call
	for _, call := range calls {
		if call.Command == "operation.inspect" && len(out) > 0 && reflect.DeepEqual(out[len(out)-1], call) {
			continue
		}
		out = append(out, call)
	}
	return out
}
