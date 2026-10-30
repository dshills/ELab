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
	node    ui.Node
	choices menuChoices
	index   int
}

// Capture the assembled popover selector and its original bound callback.
// This supplies a descriptor completion; mounted routing is tested separately.
func captureWireChoice(t *testing.T, s *screen, id, target string) capturedWireChoice {
	t.Helper()
	var result capturedWireChoice
	s.app.Do(func() {
		node, ok := findNode(s.view(), id)
		if !ok {
			t.Fatalf("missing native wiring control %s", id)
		}
		if node.Kind != ui.Selector || node.Change == nil {
			t.Fatalf("%s did not retain its scene selector", id)
		}
		choices := menuChoices{Kind: node.Kind, Options: node.Options, Change: node.Change}
		result = capturedWireChoice{node: node, choices: choices, index: -1}
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
	s.app.Do(func() {
		if choice.index < 0 || choice.index >= len(choice.choices.Options) {
			return
		}
		choice.choices.Change(ui.ChangeEvent{Kind: choice.choices.Kind, SelectedID: choice.choices.Options[choice.index].ID})
	})
}

func TestNamedWiringPopoverBindingsMatchAgentLED(t *testing.T) {
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
	recordNamedWiringObservation(t, "popover-binding-complete", actual)
	if actual.Result == nil || actual.Result.Stale || math.Abs(actual.Result.Current-.002916796577441) > 5e-12 {
		t.Fatal("popover binding LED current differs from reference", actual.Result)
	}
	if s.fieldContext != nil || s.menuCommandRevision != nil {
		t.Fatal("popover binding leaked field/menu context")
	}
}

func TestNamedWiringPopoverRefusesRetiredChoices(t *testing.T) {
	for _, scenario := range []string{"agent-endpoints", "document-replaced", "agent-tool", "invalid-index"} {
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
			case "agent-tool":
				agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "select"})
			case "invalid-index":
				choice.index = len(choice.choices.Options)
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

func TestNamedWiringPopoverPlaceholderClearsOnlyItsEndpoint(t *testing.T) {
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

// Managed selectors intentionally continue their own accepted choices. Unlike
// an AppKit completion, a retained field callback is not a one-shot token.
func TestNamedWiringPopoverContinuesOnlyItsOwnChoices(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareNamedWiringLED(t, s.service)
	choice := captureWireChoice(t, s, "wire-from", "V1.positive")
	completeWireChoice(s, choice)
	for i, o := range choice.choices.Options {
		if o.ID == "R1.1" {
			choice.index = i
		}
	}
	completeWireChoice(s, choice)
	if s.service.Snapshot().Drawing.From != "R1.1" {
		t.Fatal("selector did not continue its own accepted choice", s.errorMessage)
	}
	agentAction(t, s.service, "wire.terminals.set", workbench.TerminalsParams{From: "D1.anode", To: "R1.2"})
	before := s.service.Snapshot()
	completeWireChoice(s, choice)
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("selector borrowed an agent revision")
	}
}
