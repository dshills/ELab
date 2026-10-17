package conformance

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dshills/elab/workbench"
)

func reopenCapacity(ctx context.Context, factory Factory, directory, path string, recording []byte, graph workbench.Graph, probes []string, excess bool, observe func(workbench.Snapshot) error) (out CapacityReopen, err error) {
	out.Kind = "valid"
	if excess {
		out.Kind = "excess"
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	err = electricalSession(ctx, factory, directory, func(s *session) error {
		if err := s.ask("project.load", workbench.LoadParams{Path: path}, true, nil); err != nil {
			return err
		}
		state, err := s.state("simulation.wait", nil, false)
		if err != nil {
			return err
		}
		out.Status = state.Status
		out.GraphAndProbesRestored = reflect.DeepEqual(state.Components, graph.Components) && reflect.DeepEqual(state.Wires, graph.Wires) && reflect.DeepEqual(state.Scope.Selected, probes)
		if !out.GraphAndProbesRestored || !state.Paused || state.Document.Dirty || state.Document.RecordingCount != 1 || len(state.Operations) != 1 {
			return fmt.Errorf("capacity reopen lost graph, probes, clean saved state or retained capture")
		}
		if excess {
			if state.Status != "paused" || len(state.Diagnostics) != 1 || state.Diagnostics[0].Code != "mixed-limit" {
				return fmt.Errorf("excess drawing did not reopen as an editable unsupported circuit")
			}
		} else {
			if state.Transient == nil || state.Transient.Sample.Time != 0 || state.Mixed == nil || state.Mixed.Time != 0 {
				return fmt.Errorf("valid capacity reopen did not restart from configured startup")
			}
			if err := observe(state); err != nil {
				return err
			}
		}
		copyPath := filepath.Join(directory, "recording-copy.json")
		if err := s.ask("verification.export", workbench.VerificationExportParams{ID: state.Operations[0].ID, Path: copyPath, Format: "json"}, true, nil); err != nil {
			return err
		}
		data, err := os.ReadFile(copyPath)
		if err != nil {
			return err
		}
		out.RecordingByteExact = bytes.Equal(data, recording)
		if !out.RecordingByteExact {
			return fmt.Errorf("capacity reopen changed retained recording JSON")
		}
		out.Calls = normalizedCapacityCalls(s.calls, path)
		return nil
	})
	if err != nil {
		return out, err
	}
	untouched, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	out.OriginalFileUnchanged = bytes.Equal(original, untouched)
	if !out.OriginalFileUnchanged {
		return out, fmt.Errorf("capacity reopen changed original project bytes")
	}
	return out, nil
}

func capacityPhysicsEqual(a, b workbench.Snapshot) bool {
	return a.RunID == b.RunID && a.ElectricalRevision == b.ElectricalRevision && reflect.DeepEqual(a.Mixed, b.Mixed) && reflect.DeepEqual(a.Transient, b.Transient)
}

func exerciseCapacityBoundary(s *session, name string) (out CapacityBoundary, err error) {
	id := "Overflow"
	out.Kind = "65th-simulated-part"
	if name == "capacity-100-nodes" {
		out.Kind, id = "101st-electrical-node", "P1"
		err = s.ask("object.delete", workbench.IDParams{ID: "W51"}, true, nil)
	} else {
		err = s.ask("component.place", workbench.PlaceParams{ID: id, Kind: "resistor", Value: "1k", Position: workbench.Point{X: 2000, Y: 1400}}, true, nil)
	}
	if err != nil {
		return out, err
	}
	before, err := s.state("project.inspect", nil, false)
	if err != nil {
		return out, err
	}
	if !before.Paused || before.Status != "paused" || len(before.Diagnostics) != 1 || before.Diagnostics[0].Code != "mixed-limit" {
		return out, fmt.Errorf("excess capacity lacks explicit unsupported guidance: status %q, %d parts, %d nets, diagnostics %+v", before.Status, before.SimulatedComponents, len(before.Nets), before.Diagnostics)
	}
	response, err := s.request("simulation.step", workbench.StepParams{Duration: .001}, true)
	if err != nil {
		return out, err
	}
	if response.Error == nil || response.Error.Code != "unavailable" {
		return out, fmt.Errorf("excess circuit unexpectedly accepted Step")
	}
	out.StepErrorCode = response.Error.Code
	after, err := s.state("project.inspect", nil, false)
	if err != nil {
		return out, err
	}
	out.RejectedStepPreservedState = capacityPhysicsEqual(before, after) && reflect.DeepEqual(before.Components, after.Components) && reflect.DeepEqual(before.Wires, after.Wires) && before.Document == after.Document && reflect.DeepEqual(before.UndoEntry, after.UndoEntry) && reflect.DeepEqual(before.Operations, after.Operations)
	if !out.RejectedStepPreservedState {
		return out, fmt.Errorf("rejected over-limit Step changed circuit, physics, document or history")
	}
	moved, err := s.state("component.move", workbench.MoveParams{ID: id, Position: workbench.Point{X: 2200, Y: 1400}}, true)
	if err != nil {
		return out, err
	}
	out.GeometryPreservedPhysics = capacityPhysicsEqual(after, moved)
	if !out.GeometryPreservedPhysics {
		return out, fmt.Errorf("over-limit drawing movement changed accepted physics")
	}
	out.Graph = workbench.Graph{Components: moved.Components, Wires: moved.Wires}
	if moved.Status != "paused" || len(moved.Diagnostics) != 1 || moved.Diagnostics[0].Code != "mixed-limit" {
		return out, fmt.Errorf("over-limit movement lost paused lifecycle or diagnostic")
	}
	_, out.CompileStatus, _ = out.Graph.Compile()
	out.SimulatedParts, out.Nodes, out.Status, out.DiagnosticCode = moved.SimulatedComponents, len(moved.Nets), moved.Status, moved.Diagnostics[0].Code
	return out, nil
}
