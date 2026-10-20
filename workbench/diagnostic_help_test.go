package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDiagnosticTerminalAssociationDisclosureAndUndo(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "object.delete", IDParams{ID: "W1"}, "open")
	before := s.Snapshot()
	help := before.DiagnosticPresentation.Items[0]
	if help.Code != "open-terminal" || !reflect.DeepEqual(help.Objects, []string{"V1"}) || !reflect.DeepEqual(help.Terminals, []string{"V1.positive"}) || help.Action == "" || help.Details == "" || before.DiagnosticPresentation.DetailsVisible {
		t.Fatal("known open terminal is missing actionable source associations")
	}
	call(t, s, "diagnostics.details.set", DiagnosticDetailsParams{Visible: true}, "details")
	after := s.Snapshot()
	if !after.DiagnosticPresentation.DetailsVisible || after.RunID != before.RunID || !reflect.DeepEqual(after.Result, before.Result) || after.Document != before.Document || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || after.Drawing != before.Drawing {
		t.Fatal("disclosing technical data mutates editing/physics/history")
	}
	after.DiagnosticPresentation.Items[0].Terminals[0] = "mutated"
	if s.Snapshot().DiagnosticPresentation.Items[0].Terminals[0] == "mutated" {
		t.Fatal("diagnostic associations expose shared state")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if len(s.Snapshot().DiagnosticPresentation.Items) != 0 {
		t.Fatal("fixed connection retains a ghost highlight")
	}
}

func TestShortAndRenamedSolverSourcesIdentifyActualComponents(t *testing.T) {
	g := ExampleGraph()
	g.Wires = append(g.Wires, Wire{ID: "WS", From: "R1.1", To: "R1.2", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	help := s.Snapshot().DiagnosticPresentation.Items[0]
	if help.Code != "shorted-component" || !reflect.DeepEqual(help.Objects, []string{"R1"}) {
		t.Fatal("short diagnostic hides known component identity")
	}
	g = ExampleGraph()
	for i, p := range g.Components {
		if p.ID == "D1" {
			g.Components[i].ID = "Light"
		}
	}
	for i, w := range g.Wires {
		g.Wires[i].From = strings.Replace(w.From, "D1.", "Light.", 1)
		g.Wires[i].To = strings.Replace(w.To, "D1.", "Light.", 1)
	}
	failed := newGraphService(func(context.Context, Circuit) (Result, error) {
		return Result{}, fmt.Errorf("Newton residual detail 1e-7")
	}, g)
	defer failed.Close()
	state := settled(t, failed)
	help = state.DiagnosticPresentation.Items[0]
	if state.Status != "error" || !reflect.DeepEqual(help.Objects, []string{"Light"}) || strings.Contains(help.Summary, "Newton") || !strings.Contains(help.Details, "Newton") || help.Action == "" {
		t.Fatal("solver failure leaks technical summary or invents D1 association")
	}
}

func TestDiagnosticNetWireAndUnknownAssociationsAreConservative(t *testing.T) {
	g := ExampleGraph()
	c, _, _ := g.Compile()
	net := makeDiagnosticHelp(g, c, Diagnostic{Code: "floating-island", Message: "Connect this island", ObjectID: c.TerminalNet["R1.2"]}, "simulation", nil)
	if len(net.Terminals) != 2 || len(net.Wires) != 1 || net.Wires[0] != "W2" {
		t.Fatal("known net identity loses its wire/terminal membership")
	}
	wire := makeDiagnosticHelp(g, c, Diagnostic{Code: "invalid-terminal", Message: "Missing pin", ObjectID: "W1"}, "simulation", nil)
	if !reflect.DeepEqual(wire.Wires, []string{"W1"}) {
		t.Fatal("known wire is not associated")
	}
	unknown := makeDiagnosticHelp(g, c, Diagnostic{Code: "solve-failed", Message: "no known location", ObjectID: "unknown"}, "simulation", nil)
	if len(unknown.Objects)+len(unknown.Terminals)+len(unknown.Wires) != 0 {
		t.Fatal("unknown solver cause fabricates a location")
	}
}

func TestFailedRequestHelpIsContextualWithoutMutatingState(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	raw, _ := json.Marshal(ValueParams{ID: "R1", Value: "zero"})
	revision := before.CommandRevision
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad", Command: "component.set", Params: raw, ExpectedCommandRevision: &revision, Key: "bad"}, "test")
	if r.Error == nil || r.Help == nil || r.Help.Source != "request" || !reflect.DeepEqual(r.Help.Objects, []string{"R1"}) || r.Help.Details != r.Error.Message || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("invalid value help modifies state or loses attempted target")
	}
	clone := cloneMutationResponse(r)
	clone.Help.Objects[0] = "changed"
	if r.Help.Objects[0] == "changed" {
		t.Fatal("response help cloning aliases context")
	}
	raw, _ = json.Marshal(ValueParams{ID: "missing", Value: "1k"})
	r = s.Dispatch(context.Background(), Request{Version: 1, ID: "missing", Command: "component.set", Params: raw, ExpectedCommandRevision: &revision, Key: "missing"}, "test")
	if r.Help == nil || len(r.Help.Objects) != 0 {
		t.Fatal("failed unknown target invents a diagram object")
	}
}

func TestDiagnosticDetailsStrictInputsAndProjectLifecycle(t *testing.T) {
	s := New()
	defer s.Close()
	refuse(t, s, "diagnostics.details.set", map[string]any{}, "invalid-params")
	call(t, s, "diagnostics.details.set", DiagnosticDetailsParams{Visible: true}, "show")
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "new")
	if s.Snapshot().DiagnosticPresentation.DetailsVisible {
		t.Fatal("new project retained technical disclosure preference")
	}
}

func TestRecordedPositionRefusalExplainsSelectedCaptureWithoutMutation(t *testing.T) {
	for _, kind := range []string{"rc", "dc"} {
		t.Run(kind, func(t *testing.T) {
			var s *Service
			params := VerifyParams{}
			if kind == "rc" {
				s = pausedRC(t)
				duration := .02
				params.Duration = &duration
			} else {
				s = NewExample()
				settled(t, s)
				call(t, s, "simulation.pause", struct{}{}, "pause")
			}
			t.Cleanup(s.Close)
			_, view := verify(t, s, params, "capture")
			call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "show")
			index := 0
			if kind == "rc" {
				index = 4
			}
			call(t, s, "recording.seek", RecordingSeekParams{Index: &index}, "position")
			before := s.Snapshot()
			for i, bad := range []any{
				map[string]any{"time_s": "oops"},
				map[string]any{"index": 1.5},
				map[string]any{"index": view.Metadata.SampleCount},
				map[string]any{"time_s": view.Metadata.Duration + 1},
				map[string]any{"index": 0, "time_s": 0},
			} {
				raw, err := json.Marshal(bad)
				if err != nil {
					t.Fatal(err)
				}
				revision := before.CommandRevision
				response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint("bad-", i), Command: "recording.seek", Params: raw, ExpectedCommandRevision: &revision}, "agent.test")
				if response.Error == nil || response.Error.Code != "invalid-params" || response.Help == nil {
					t.Fatal("missing structured position refusal", response)
				}
				help := response.Help
				if help.Source != "request" || help.Code != response.Error.Code || help.Details != response.Error.Message || len(help.Objects)+len(help.Terminals)+len(help.Wires) != 0 || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("position guidance lost details, invented a location or mutated state", help)
				}
				if kind == "rc" {
					if !strings.Contains(help.Summary, "recording") || !strings.Contains(help.Action, "0 to 20") || !strings.Contains(help.Action, "0.02 s") || !strings.Contains(help.Action, "ms") {
						t.Fatalf("position refusal hides valid range or units: %+v", help)
					}
				} else if !strings.Contains(help.Action, "sample index 0 or time 0 s") || strings.Contains(help.Action, "3 ms") {
					t.Fatalf("DC guidance suggests an unavailable position: %+v", help)
				}
			}
		})
	}
}
