//go:build offscreen

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"testing"

	"github.com/dshills/elab/workbench"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedVerificationStart(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	if s.service.Snapshot().InspectorSection != "simulation" {
		uiAction(t, s, "inspector-section-simulation", nil)
	}
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "verify-duration", Sequence: 1}}
		for i, ids := range [][]string{{"verify-duration", "verify-step"}, {"verify-initial", "verify-cap-C1"}, {"verify-start", "verify-project-startup"}} {
			row := ui.Node{ID: fmt.Sprintf("verify-test-row-%d", i), Kind: ui.Row}
			for _, id := range ids {
				node, ok := findNode(assembled, id)
				if !ok {
					t.Fatalf("missing Verify control %s", id)
				}
				row.Children = append(row.Children, node)
			}
			root.Children = append(root.Children, row)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedVerificationStartUsesUnfinishedDuration(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "simulation.restart", struct{}{})
	m := mountedVerificationStart(t, s)
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "200 ms" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if field := mountedControlElement(t, m, "verify-duration"); field.Value != "200 ms" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("fixture must retain private duration without changing accepted settings", field.Value)
	}
	clickMountedControl(t, m, "verify-start")
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Operations) != len(before.Operations)+1 || after.VerificationSettings.Duration != .2 {
		t.Fatalf("one Verify click did not capture typed duration: duration=%g operations=%d→%d revision=%d→%d error=%s", after.VerificationSettings.Duration, len(before.Operations), len(after.Operations), before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	if after.Operations[len(after.Operations)-1].RequestedDuration != .2 {
		t.Fatal("Verify captured the old duration")
	}
	if before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) {
		t.Fatal("independent verification changed the live experiment")
	}
}

// Exercise private native text/number buffers, then compare the completed job
// with an independent agent using the public commands and the RC equation.
func TestMountedVerificationFieldsMatchAgentCapture(t *testing.T) {
	for _, edit := range []struct{ id, text string }{
		{"verify-duration", "20 ms"}, {"verify-step", "1 ms"},
		{"verify-initial", "2"}, {"verify-cap-C1", "3"},
	} {
		t.Run(edit.id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			reference := workbench.New()
			t.Cleanup(reference.Close)
			for _, service := range []*workbench.Service{s.service, reference} {
				agentAction(t, service, "example.rc", workbench.ReplaceParams{})
				agentAction(t, service, "simulation.pause", struct{}{})
				agentAction(t, service, "simulation.restart", struct{}{})
				cfg := service.Snapshot().VerificationSettings
				cfg.InputMode = "charge"
				agentAction(t, service, "verification.settings.set", cfg)
			}
			m := mountedVerificationStart(t, s)
			before := s.service.Snapshot()
			if edit.id == "verify-initial" || edit.id == "verify-cap-C1" {
				clickMountedControl(t, m, edit.id)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
				code := key.Code2
				if edit.text == "3" {
					code = key.Code3
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: code, Rune: rune(edit.text[0])})
			} else {
				typeMountedCursor(t, m, edit.id, edit.text)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("typing committed early")
			}
			if edit.id == "verify-initial" || edit.id == "verify-cap-C1" {
				// Accessibility activation does not first transfer pointer focus.
				button := verificationControl(t, s, "verify-start")
				s.app.Do(button.Activate)
				m.settle()
			} else {
				clickMountedControl(t, m, "verify-start")
			}
			after := s.service.Snapshot()
			if s.errorMessage != "" || len(after.Operations) != 1 {
				t.Fatal("first click failed", s.errorMessage)
			}
			settings := reference.Snapshot().VerificationSettings
			switch edit.id {
			case "verify-duration":
				settings.Duration = .02
			case "verify-step":
				settings.Step = .001
			case "verify-initial":
				v := 2.
				settings.InitialVoltage = &v
			case "verify-cap-C1":
				settings.Capacitors = []workbench.CapacitorInitial{{ID: "C1", Voltage: 3}}
			}
			agentAction(t, reference, "verification.settings.set", settings)
			duration := settings.Duration
			agentAction(t, reference, "simulation.verify", workbench.VerifyParams{Duration: &duration})
			got := completedVerification(t, s.service, after.LastOperation)
			want := completedVerification(t, reference, reference.Snapshot().LastOperation)
			if !reflect.DeepEqual(got.Metadata.Graph, want.Metadata.Graph) || got.Metadata.Settings != want.Metadata.Settings || !reflect.DeepEqual(got.Metadata.Statistics, want.Metadata.Statistics) {
				t.Fatal("mounted capture differs from independent public agent")
			}
			compareVerificationSamples(t, s.service, after.LastOperation, reference, reference.Snapshot().LastOperation)
			initial := 0.
			if edit.id == "verify-initial" {
				initial = 2
			}
			if edit.id == "verify-cap-C1" {
				initial = 3
			}
			state := s.service.Snapshot()
			expected := 5 + (initial-5)*math.Exp(-duration/.01)
			if state.Verification == nil || math.Abs(state.Verification.Readings[0].Value-expected) > 1e-10 {
				t.Fatal("capture differs from analytic RC charge", expected, state.Verification)
			}
			if !reflect.DeepEqual(before.Transient, state.Transient) || before.RunID != state.RunID || !reflect.DeepEqual(before.Components, state.Components) || !reflect.DeepEqual(before.Wires, state.Wires) {
				t.Fatal("verification changed live experiment")
			}
		})
	}
}

func TestMountedVerificationResetSettlesDuration(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	cfg := s.service.Snapshot().VerificationSettings
	v := 2.
	cfg.InitialVoltage = &v
	cfg.Capacitors = []workbench.CapacitorInitial{{ID: "C1", Voltage: 3}}
	cfg.InputMode, cfg.Step = "discharge", .001
	agentAction(t, s.service, "verification.settings.set", cfg)
	m := mountedVerificationStart(t, s)
	typeMountedCursor(t, m, "verify-duration", "30 ms")
	clickMountedControl(t, m, "verify-project-startup")
	state := s.service.Snapshot()
	if s.errorMessage != "" || state.VerificationSettings.Duration != .03 || state.VerificationSettings.InitialVoltage != nil || len(state.VerificationSettings.Capacitors) != 0 || state.VerificationSettings.InputMode != "auto" || state.VerificationSettings.Step != 0 || len(state.Operations) != 0 {
		t.Fatal("reset lost duration or retained overrides", s.errorMessage, state.VerificationSettings)
	}
}

func TestMountedVerificationHeldClickRefusesAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	m := mountedVerificationStart(t, s)
	typeMountedCursor(t, m, "verify-duration", "20 ms")
	button := mountedControlElement(t, m, "verify-start")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	m.frame()
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	afterHeld := s.service.Snapshot()
	// Focus loss can commit the field against its fresh binding after the
	// agent edit. The held action must still refuse its obsolete circuit.
	if s.errorMessage == "" || len(afterHeld.Operations) != 0 || !reflect.DeepEqual(before.Components, afterHeld.Components) || !reflect.DeepEqual(before.Wires, afterHeld.Wires) || before.ElectricalRevision != afterHeld.ElectricalRevision {
		t.Fatalf("held Verify adopted agent edit: error=%s revision=%d→%d settings=%+v→%+v operations=%d→%d", s.errorMessage, before.CommandRevision, s.service.Snapshot().CommandRevision, before.VerificationSettings, s.service.Snapshot().VerificationSettings, len(before.Operations), len(s.service.Snapshot().Operations))
	}
	clickMountedControl(t, m, "verify-start")
	if s.errorMessage != "" || len(s.service.Snapshot().Operations) != 1 {
		t.Fatal("fresh Verify failed", s.errorMessage)
	}
}

func TestMountedVerificationRejectsThenCorrectsDuration(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	m := mountedVerificationStart(t, s)
	typeMountedCursor(t, m, "verify-duration", "nope")
	clickMountedControl(t, m, "verify-start")
	if s.errorMessage == "" || len(s.service.Snapshot().Operations) != 0 {
		t.Fatal("invalid quantity started capture")
	}
	typeMountedCursor(t, m, "verify-duration", "20 ms")
	clickMountedControl(t, m, "verify-start")
	if s.errorMessage != "" || len(s.service.Snapshot().Operations) != 1 {
		t.Fatal("corrected first click failed", s.errorMessage)
	}
}

func compareVerificationSamples(t *testing.T, a *workbench.Service, aid string, b *workbench.Service, bid string) {
	t.Helper()
	for offset := 0; ; {
		pages := []workbench.VerificationPage{}
		for i, svc := range []*workbench.Service{a, b} {
			id := aid
			if i == 1 {
				id = bid
			}
			raw, _ := json.Marshal(workbench.VerificationPageParams{ID: id, Offset: offset, Limit: 200})
			response := svc.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "samples", Command: "operation.samples", Params: raw}, "test")
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			pages = append(pages, response.Result.(workbench.VerificationPage))
		}
		if !reflect.DeepEqual(pages[0].Rows, pages[1].Rows) || !reflect.DeepEqual(pages[0].Probes, pages[1].Probes) || pages[0].Total != pages[1].Total || pages[0].NextOffset != pages[1].NextOffset {
			t.Fatal("captured sample page differs from independent agent", offset)
		}
		if pages[0].NextOffset >= pages[0].Total {
			break
		}
		if pages[0].NextOffset <= offset {
			t.Fatal("sample pagination did not advance")
		}
		offset = pages[0].NextOffset
	}
}

func TestMountedVerificationInvalidDurationSurvivesStepEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	m := mountedVerificationStart(t, s)
	typeMountedCursor(t, m, "verify-duration", "nope")
	clickMountedControl(t, m, "verify-start")
	if s.errorMessage == "" || len(s.service.Snapshot().Operations) != 0 {
		t.Fatal("first invalid duration started capture")
	}
	typeMountedCursor(t, m, "verify-step", "1 ms")
	clickMountedControl(t, m, "verify-start")
	if s.errorMessage == "" || len(s.service.Snapshot().Operations) != 0 {
		t.Fatal("editing sample step forgot the invalid duration and captured old settings")
	}
	typeMountedCursor(t, m, "verify-duration", "20 ms")
	button := verificationControl(t, s, "verify-start")
	s.app.Do(button.Activate)
	m.settle()
	if s.errorMessage != "" || len(s.service.Snapshot().Operations) != 1 || s.service.Snapshot().Operations[0].RequestedDuration != .02 {
		t.Fatal("corrected accessibility activation did not capture once", s.errorMessage)
	}

}

func TestMountedVerificationRefusalOwnership(t *testing.T) {
	for _, scenario := range []string{"agent-circuit", "agent-duration", "document"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			agentAction(t, s.service, "simulation.pause", struct{}{})
			m := mountedVerificationStart(t, s)
			typeMountedCursor(t, m, "verify-duration", "nope")
			clickMountedControl(t, m, "verify-start")
			if len(s.service.Snapshot().Operations) != 0 || s.errorMessage == "" {
				t.Fatal("invalid quantity started capture")
			}
			switch scenario {
			case "agent-circuit":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "agent-duration":
				cfg := s.service.Snapshot().VerificationSettings
				cfg.Duration = .02
				agentAction(t, s.service, "verification.settings.set", cfg)
			case "document":
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
				agentAction(t, s.service, "simulation.pause", struct{}{})
			}
			m.frame()
			clickMountedControl(t, m, "verify-start")
			state := s.service.Snapshot()
			if scenario == "agent-circuit" {
				if s.errorMessage == "" || len(state.Operations) != 0 {
					t.Fatal("agent circuit edit forgot invalid duration")
				}
				typeMountedCursor(t, m, "verify-duration", "20 ms")
				button := verificationControl(t, s, "verify-start")
				s.app.Do(button.Activate)
				m.settle()
				if s.errorMessage != "" || len(s.service.Snapshot().Operations) != 1 {
					t.Fatal("corrected field did not capture after agent edit", s.errorMessage)
				}
			} else if s.errorMessage != "" || len(state.Operations) != 1 {
				t.Fatal("replacement controlled value/document inherited old refusal", s.errorMessage)
			}
		})
	}
}

func TestMountedVerificationCorrectionToAcceptedTime(t *testing.T) {
	for _, action := range []string{"verify-start", "verify-project-startup"} {
		for _, field := range []string{"verify-duration", "verify-step"} {
			for _, mode := range []string{"pointer", "without-blur"} {
				for _, correction := range []string{"none", "accepted", "different"} {
					t.Run(action+"/"+field+"/"+mode+"/"+correction, func(t *testing.T) {
						s, cleanup := testScreen(t)
						t.Cleanup(cleanup)
						uiAction(t, s, "example.rc", nil)
						agentAction(t, s.service, "simulation.pause", struct{}{})
						cfg := s.service.Snapshot().VerificationSettings
						cfg.Duration, cfg.Step, cfg.InputMode = .02, .001, "charge"
						agentAction(t, s.service, "verification.settings.set", cfg)
						m := mountedVerificationStart(t, s)
						activate := func() {
							if mode == "pointer" {
								clickMountedControl(t, m, action)
							} else {
								button := verificationControl(t, s, action)
								s.app.Do(button.Activate)
								m.settle()
							}
						}
						before := s.service.Snapshot()
						typeMountedCursor(t, m, field, "bad")
						activate()
						if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
							t.Fatal("invalid timing entry did not preserve accepted state")
						}
						if correction != "none" {
							text := timeText(verificationTextValue(cfg, field))
							if correction == "different" {
								if field == "verify-step" {
									cfg.Step = .002
								} else {
									cfg.Duration = .03
								}
								text = timeText(verificationTextValue(cfg, field))
							}
							typeMountedCursor(t, m, field, text)
						}
						activate()
						after := s.service.Snapshot()
						if correction == "none" {
							if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
								t.Fatal("untouched invalid timing entry was silently accepted")
							}
							return
						}
						wantOperations := 1
						if action == "verify-project-startup" {
							wantOperations = 0
							duration := cfg.Duration
							cfg = workbench.DefaultVerificationSettings()
							cfg.Duration = duration
						}
						if s.errorMessage != "" || len(after.Operations) != wantOperations || !reflect.DeepEqual(after.VerificationSettings, cfg) || after.CommandRevision != before.CommandRevision+2 || len(s.verificationRejected) != 0 {
							t.Fatalf("correction failed: settings=%+v want=%+v operations=%d error=%q", after.VerificationSettings, cfg, len(after.Operations), s.errorMessage)
						}
						assertAnalysisSettingsPreserveLiveCircuit(t, before, after)
						if wantOperations != 0 {
							completedVerification(t, s.service, after.LastOperation)
							agent := workbench.New()
							t.Cleanup(agent.Close)
							agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
							agentAction(t, agent, "simulation.pause", struct{}{})
							agentAction(t, agent, "verification.settings.set", cfg)
							agentAction(t, agent, "simulation.verify", workbench.VerifyParams{Duration: &cfg.Duration})
							completedVerification(t, agent, agent.Snapshot().LastOperation)
							compareVerificationSamples(t, s.service, after.LastOperation, agent, agent.Snapshot().LastOperation)
						}
					})
				}
			}
		}
	}
}

// Analysis settings and kept captures can change preference/recording dirtiness;
// they must leave the live circuit, physical state and circuit history intact.
func assertAnalysisSettingsPreserveLiveCircuit(t *testing.T, before, after workbench.Snapshot) {
	t.Helper()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document.Generation != after.Document.Generation || before.Document.Path != after.Document.Path || before.Document.SchematicDirty != after.Document.SchematicDirty ||
		!reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) ||
		!reflect.DeepEqual(before.Result, after.Result) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) ||
		!reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) {
		t.Fatal("analysis settings or capture changed the live circuit or circuit history")
	}
}
