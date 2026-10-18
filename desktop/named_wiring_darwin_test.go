//go:build darwin && cgo

package main

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type capturedWireChoice struct {
	node             ui.Node
	choices          menuChoices
	index            int
	revision, serial uint64
}

// Capture the assembled macOS adapter's real options and bound callback. The
// test supplies its completion; it does not open or interact with an OS menu.
func captureWireChoice(t *testing.T, s *screen, id, target string) capturedWireChoice {
	t.Helper()
	var result capturedWireChoice
	s.app.Do(func() {
		state := s.service.Snapshot()
		node, ok := findNode(s.view(), id)
		if !ok {
			t.Fatalf("missing native wiring control %s", id)
		}
		choices, ok := node.Custom.(menuChoices)
		if !ok || node.Kind != ui.Button {
			t.Fatalf("%s did not use native menu adapter", id)
		}
		result = capturedWireChoice{node: node, choices: choices, index: -1, revision: state.CommandRevision, serial: s.menuSerial}
		for i, option := range choices.Options {
			if option.ID == target {
				result.index = i
				break
			}
		}
		if result.index < 0 {
			t.Fatalf("native menu %s lacks option %s", id, target)
		}
	})
	return result
}

func completeWireChoice(s *screen, choice capturedWireChoice) {
	s.app.Do(func() { s.acceptMenuChoice(choice.node, choice.choices, choice.index, choice.serial, choice.revision) })
}

func TestNativeNamedWiringMenuBindingsMatchAgentLED(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareNamedWiringLED(t, service)
	}
	for _, pair := range namedLEDConnections() {
		completeWireChoice(s, captureWireChoice(t, s, "wire-from", pair.From))
		if s.errorMessage != "" {
			t.Fatal(s.errorMessage)
		}
		completeWireChoice(s, captureWireChoice(t, s, "wire-to", pair.To))
		if s.errorMessage != "" {
			t.Fatal(s.errorMessage)
		}
		agentAction(t, agent, "wire.terminals.set", workbench.TerminalsParams{From: pair.From, To: pair.To})
		assertNamedWiringParity(t, s.service.Snapshot(), agent.Snapshot())
		uiAction(t, s, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", pair)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	actual, err := s.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := agent.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertNamedWiringParity(t, actual, expected)
	recordNamedWiringObservation(t, "native-adapter-complete", actual)
	if actual.Result == nil || actual.Result.Stale || math.Abs(actual.Result.Current-.002916796577441) > 5e-12 {
		t.Fatal("native adapter LED current differs from reference", actual.Result)
	}
	if s.fieldContext != nil || s.menuCommandRevision != nil {
		t.Fatal("native adapter leaked field/menu context")
	}
}

func TestNativeNamedWiringMenuRefusesRetiredChoices(t *testing.T) {
	for _, scenario := range []string{"agent-endpoints", "document-replaced", "newer-menu", "invalid-index", "duplicate-completion"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareNamedWiringLED(t, s.service)
			choice := captureWireChoice(t, s, "wire-from", "V1.positive")
			switch scenario {
			case "agent-endpoints":
				agentAction(t, s.service, "wire.terminals.set", workbench.TerminalsParams{From: "R1.2", To: "D1.anode"})
			case "document-replaced":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "newer-menu":
				s.app.Do(func() { s.menuSerial++ })
			case "invalid-index":
				choice.index = len(choice.choices.Options)
			case "duplicate-completion":
				completeWireChoice(s, choice)
			}
			before := s.service.Snapshot()
			focus := s.keyboardFocus
			completeWireChoice(s, choice)
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.keyboardFocus != focus || s.fieldContext != nil || s.menuCommandRevision != nil {
				t.Fatal("retired menu choice changed state or leaked context", scenario, s.errorMessage)
			}
			if scenario == "agent-endpoints" {
				completeWireChoice(s, captureWireChoice(t, s, "wire-from", "V1.positive"))
				state := s.service.Snapshot()
				if s.errorMessage != "" || state.Drawing.From != "V1.positive" || state.Drawing.To != "D1.anode" {
					t.Fatal("fresh menu choice overwrote agent destination", s.errorMessage, state.Drawing)
				}
			}
		})
	}
}

func TestNativeNamedWiringPlaceholderClearsOnlyItsEndpoint(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareNamedWiringLED(t, s.service)
	agentAction(t, s.service, "wire.terminals.set", workbench.TerminalsParams{From: "V1.positive", To: "R1.1"})
	completeWireChoice(s, captureWireChoice(t, s, "wire-from", "__none"))
	state := s.service.Snapshot()
	var connect ui.Node
	s.app.Do(func() { connect, _ = findNode(s.view(), "connect-terminals") })
	if s.errorMessage != "" || state.Drawing.From != "" || state.Drawing.To != "R1.1" || !connect.Disabled || len(state.Wires) != 0 {
		t.Fatal("placeholder cleared wrong endpoint or enabled Connect", state.Drawing, s.errorMessage)
	}
}
