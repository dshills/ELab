package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
)

// Result preserves one circuit's original project, command intent and readings.
type Result struct {
	Calls        []Call             `json:"calls"`
	Name         string             `json:"name"`
	Commands     []string           `json:"commands"`
	Project      workbench.Document `json:"project"`
	Measurements map[string]float64 `json:"measurements_si"`
	Checks       []string           `json:"checks"`
}

// Run builds all three PRD reference families using identical public commands.
// Every circuit and its reopening get separate owned, empty sessions.
func Run(ctx context.Context, factory Factory, directory string) ([]Result, error) {
	var results []Result
	for _, c := range circuits {
		result, err := runCircuit(ctx, factory, filepath.Join(directory, c.name), c)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.name, err)
		}
		results = append(results, result)
	}
	return results, nil
}
func runCircuit(ctx context.Context, factory Factory, dir string, c circuit) (out Result, err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return out, err
	}
	client, err := factory(ctx, filepath.Join(dir, "initial"))
	if err != nil {
		return out, err
	}
	defer func() { err = errors.Join(err, client.Close()) }()
	s := &session{client: client, ctx: ctx, directory: dir}
	out = Result{Name: c.name, Measurements: map[string]float64{}}
	initial, err := s.state("project.inspect", nil, false)
	if err != nil {
		return out, err
	}
	if len(initial.Components) != 0 || len(initial.Wires) != 0 || initial.Document.Path != "" {
		return out, fmt.Errorf("factory did not create an empty session")
	}
	var catalog struct {
		Capabilities []workbench.Capability `json:"capabilities"`
	}
	if err = s.ask("capabilities.list", nil, false, &catalog); err != nil {
		return out, err
	}
	var library []workbench.Part
	if err = s.ask("parts.list", nil, false, &library); err != nil {
		return out, err
	}
	for _, p := range c.parts {
		found := false
		for _, available := range library {
			if available.Kind == p.kind && available.Model != "" && len(available.Terminals) > 0 {
				found = true
			}
		}
		if !found {
			return out, fmt.Errorf("missing discoverable model/pins for %s", p.kind)
		}
		if err = s.ask("component.place", workbench.PlaceParams{ID: p.id, Kind: p.kind, Value: p.value, Position: workbench.Point{X: p.x, Y: p.y}}, true, nil); err != nil {
			return out, err
		}
	}
	// Paused construction prevents wall-clock sampling from entering comparison.
	if err = s.ask("simulation.pause", nil, true, nil); err != nil {
		return out, err
	}
	for _, wire := range c.wires {
		if err = s.ask("wire.connect", map[string]string{"from": wire[0], "to": wire[1]}, true, nil); err != nil {
			return out, err
		}
	}
	ready, err := restart(s)
	if err != nil {
		return out, err
	}
	if len(ready.Nets) == 0 || len(ready.Diagnostics) != 0 {
		return out, fmt.Errorf("complete circuit lacks valid connectivity: %+v", ready.Diagnostics)
	}
	if err = experiment(s, c.name, &out); err != nil {
		return out, err
	}
	beforeFault, err := s.state("project.inspect", nil, false)
	if err != nil {
		return out, err
	}
	broken, err := s.state("object.delete", workbench.IDParams{ID: beforeFault.Wires[0].ID}, true)
	if err != nil {
		return out, err
	}
	if len(broken.Diagnostics) == 0 || fresh(broken, c.name) {
		return out, fmt.Errorf("broken wire still presents fresh readings")
	}
	if err = s.ask("edit.undo", nil, true, nil); err != nil {
		return out, err
	}
	ready, err = restart(s)
	if err != nil {
		return out, err
	}
	if !fresh(ready, c.name) || !reflect.DeepEqual(beforeFault.Wires, ready.Wires) {
		return out, fmt.Errorf("undo failed to recover complete fresh circuit")
	}
	var inspection workbench.MeasurementInspection
	if err = s.ask("measurement.inspect", workbench.MeasurementQueryParams{Target: &workbench.MeasurementSelectParams{Kind: "object", ID: c.target}}, false, &inspection); err != nil {
		return out, err
	}
	if !inspection.Available || len(inspection.Readings) == 0 {
		return out, fmt.Errorf("live source measurement unavailable")
	}
	for _, reading := range inspection.Readings {
		if reading.Value != nil {
			out.Measurements["inspect:"+reading.Probe.ID] = *reading.Value
		}
	}
	if c.name == "rc" {
		if err = s.ask("scope.probes.set", workbench.ProbeParams{IDs: []string{"v:Storage.1", "i:Limit"}}, true, nil); err != nil {
			return out, err
		}
		ready, err = s.state("simulation.step", workbench.StepParams{Duration: .01}, true)
		if err != nil {
			return out, err
		}
		if err = s.ask("scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: 0}, true, nil); err != nil {
			return out, err
		}
		if err = s.ask("scope.cursor.set", workbench.CursorParams{Cursor: "b", Time: .01}, true, nil); err != nil {
			return out, err
		}
		ready, err = s.state("project.inspect", nil, false)
		if err != nil {
			return out, err
		}
		if ready.Scope.DeltaSeconds == nil || math.Abs(*ready.Scope.DeltaSeconds-.01) > 1e-12 {
			return out, fmt.Errorf("scope cursors lose exact sample interval")
		}
		out.Measurements["cursor_interval_s"] = *ready.Scope.DeltaSeconds
		if err = s.ask("export.scope.csv", map[string]any{"path": filepath.Join(dir, "scope.csv")}, true, nil); err != nil {
			return out, err
		}
	}
	duration := 0.
	if c.name == "rc" {
		duration = .2
	}
	verified, err := s.state("simulation.verify", workbench.VerifyParams{Duration: &duration}, true)
	if err != nil {
		return out, err
	}
	var operation workbench.OperationView
	for {
		if err = s.ask("operation.inspect", workbench.IDParams{ID: verified.LastOperation}, false, &operation); err != nil {
			return out, err
		}
		if operation.Operation.Status == "completed" {
			break
		}
		if operation.Operation.Status == "failed" || operation.Operation.Status == "canceled" {
			return out, fmt.Errorf("verification: %+v", operation.Operation)
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	if operation.Metadata == nil || operation.Metadata.Engine == "" || operation.Metadata.LEDModel.ID == "" {
		return out, fmt.Errorf("verification lacks pinned engine/model identity")
	}
	if operation.Metadata.ElectricalRevision != verified.ElectricalRevision || operation.Metadata.DocumentGeneration != verified.Document.Generation || !reflect.DeepEqual(operation.Metadata.Graph.Components, verified.Components) || !reflect.DeepEqual(operation.Metadata.Graph.Wires, verified.Wires) {
		return out, fmt.Errorf("verification captured unrelated source state")
	}
	for _, stat := range operation.Metadata.Statistics {
		out.Measurements["verify-min:"+stat.ProbeID] = stat.Minimum
		out.Measurements["verify-max:"+stat.ProbeID] = stat.Maximum
	}
	if err = s.ask("verification.export", workbench.VerificationExportParams{ID: verified.LastOperation, Path: filepath.Join(dir, "verification.csv"), Format: "csv"}, true, nil); err != nil {
		return out, err
	}
	if err = s.ask("export.schematic.svg", map[string]any{"path": filepath.Join(dir, "schematic.svg")}, true, nil); err != nil {
		return out, err
	}
	projectPath := filepath.Join(dir, "project.elab")
	saved, err := s.state("project.save", workbench.SaveParams{Path: projectPath}, true)
	if err != nil {
		return out, err
	}
	if saved.Document.Dirty {
		return out, fmt.Errorf("saved project remains dirty")
	}
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(data, &out.Project); err != nil {
		return out, err
	}
	closed, err := s.state("app.close", nil, true)
	if err != nil {
		return out, err
	}
	if !closed.Closed {
		return out, fmt.Errorf("close was not acknowledged")
	}
	freshClient, err := factory(ctx, filepath.Join(dir, "reopened"))
	if err != nil {
		return out, err
	}
	defer func() { err = errors.Join(err, freshClient.Close()) }()
	reopened := &session{client: freshClient, ctx: ctx, directory: dir}
	if _, err = reopened.state("project.inspect", nil, false); err != nil {
		return out, err
	}
	loaded, err := reopened.state("project.load", workbench.LoadParams{Path: projectPath}, true)
	if err != nil {
		return out, err
	}
	loaded, err = reopened.state("simulation.wait", nil, false)
	if err != nil {
		return out, err
	}
	if loaded.Document.Dirty || !loaded.Paused || !reflect.DeepEqual(saved.Components, loaded.Components) || !reflect.DeepEqual(saved.Wires, loaded.Wires) || !fresh(loaded, c.name) {
		return out, fmt.Errorf("reopen loses saved state/fresh pause semantics")
	}
	if c.name == "rc" && (loaded.Transient.Sample.Time != 0 || loaded.Transient.Sample.Voltage != 0) {
		return out, fmt.Errorf("reopen resumes old capacitor state")
	}
	if c.name != "rc" {
		if c.name == "led" && loaded.Result.Current != saved.Result.Current {
			return out, fmt.Errorf("LED reopen differs")
		}
		if c.name == "npn" && loaded.NPN.CollectorCurrent != saved.NPN.CollectorCurrent {
			return out, fmt.Errorf("NPN reopen differs")
		}
	}
	if err = reopened.ask("project.save", workbench.SaveParams{Path: filepath.Join(dir, "reopened.elab")}, true, nil); err != nil {
		return out, err
	}
	reopenedData, err := os.ReadFile(filepath.Join(dir, "reopened.elab"))
	if err != nil {
		return out, err
	}
	var reopenedDocument workbench.Document
	if err = json.Unmarshal(reopenedData, &reopenedDocument); err != nil {
		return out, err
	}
	if !reflect.DeepEqual(out.Project, reopenedDocument) {
		return out, fmt.Errorf("saved semantic project changes across fresh reopen")
	}
	if err = reopened.ask("app.close", nil, true, nil); err != nil {
		return out, err
	}
	out.Commands = append(s.commands, reopened.commands...)
	out.Calls = append(s.calls, reopened.calls...)
	for _, command := range out.Commands {
		found := false
		for _, capability := range catalog.Capabilities {
			if capability.ID == command && len(capability.Input) > 0 && len(capability.OutputSchema) > 0 {
				found = true
			}
		}
		if !found {
			return out, fmt.Errorf("workflow command %s lacks discovery schemas", command)
		}
	}
	out.Checks = []string{"empty owned session; discovered models/pins/connectivity", "value experiment and accepted SI measurements", "introduced disconnected wire; stale diagnostics; undo repair", "bounded recorded verification and SVG/CSV exports", "save; structured close; fresh session reopen; semantic document equality"}
	return out, nil
}
