//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedTimingControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	if s.service.Snapshot().InspectorSection != "simulation" {
		uiAction(t, s, "inspector-section-simulation", nil)
	}
	if s.service.Snapshot().SimulationTool != "recordings" {
		uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "recordings"})
	}
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "recorded-timing-level", Sequence: 1}}
		for row, ids := range [][]string{{"recorded-timing-level", "recorded-timing-second-level"}, {"recorded-timing-start", "recorded-timing-end"}, {"recorded-timing-next"}} {
			group := ui.Node{ID: []string{"levels", "interval", "pages"}[row], Kind: ui.Row}
			for _, id := range ids {
				node, ok := findNode(assembled, id)
				if !ok {
					timing := s.service.Snapshot().Recording.Timing
					if id == "recorded-timing-next" && timing.NextOffset >= timing.Total {
						continue
					}
					t.Fatalf("missing production timing control %s", id)
				}
				node.MinWidth, node.MinHeight = 190, 40
				group.Children = append(group.Children, node)
			}
			root.Children = append(root.Children, group)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	return m
}

func TestMountedRecordingTimingTypedThresholdThenNextMatchesAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareRecordedTiming(t, service)
	}
	m := mountedTimingControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "recorded-timing-level", "3 V")
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("private threshold draft changed accepted measurements")
	}
	clickMountedControl(t, m, "recorded-timing-next")
	cfg := workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Level = "3 V"
	agentAction(t, agent, "recording.timing.set", cfg)
	cfg = workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Offset = agent.Snapshot().Recording.Timing.NextOffset
	agentAction(t, agent, "recording.timing.set", cfg)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || !reflect.DeepEqual(after.Recording.Timing, agent.Snapshot().Recording.Timing) {
		t.Fatalf("one Next click failed after private threshold: level=%s offset=%d revision=%d→%d error=%s", after.Recording.Timing.Settings.Level, after.Recording.Timing.Settings.Offset, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedRecordingTimingInvalidThresholdDoesNotAdvancePage(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	m := mountedTimingControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "recorded-timing-level", "bad")
	clickMountedControl(t, m, "recorded-timing-next")
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatalf("invalid threshold advanced page or cleared validation: offset=%d error=%s", s.service.Snapshot().Recording.Timing.Settings.Offset, s.errorMessage)
	}
	typeMountedCursor(t, m, "recorded-timing-level", "3 V")
	clickMountedControl(t, m, "recorded-timing-next")
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.Recording.Timing.Settings.Level != "3 V" || after.Recording.Timing.Settings.Offset != 8 || after.CommandRevision != before.CommandRevision+2 {
		t.Fatal("corrected threshold did not page on the first click", s.errorMessage)
	}
}

func TestMountedRecordingTimingCorrectionAfterRefusal(t *testing.T) {
	for _, field := range []struct{ id, accepted, different string }{
		{"recorded-timing-level", "2.5 V", "3 V"}, {"recorded-timing-second-level", "4 V", "4.5 V"},
		{"recorded-timing-start", "0 s", "1 ms"}, {"recorded-timing-end", "500 ms", "450 ms"},
	} {
		for _, action := range []string{"page-pointer", "page-without-blur", "direction-without-blur"} {
			for _, correction := range []string{"none", "accepted", "different"} {
				t.Run(field.id+"/"+action+"/"+correction, func(t *testing.T) {
					s, cleanup := testScreen(t)
					t.Cleanup(cleanup)
					prepareRecordedTiming(t, s.service)
					cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
					cfg.SecondLevel = "4 V"
					agentAction(t, s.service, "recording.timing.set", cfg)
					m := mountedTimingControls(t, s)
					activate := func() {
						switch action {
						case "page-pointer":
							clickMountedControl(t, m, "recorded-timing-next")
						case "page-without-blur":
							button := verificationControl(t, s, "recorded-timing-next")
							s.app.Do(button.Activate)
							m.settle()
						default:
							changeMountedTimingDirection(t, s, m)
						}
					}
					before := s.service.Snapshot()
					typeMountedCursor(t, m, field.id, "bad")
					activate()
					if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
						t.Fatal("invalid timing input changed the accepted measurement")
					}
					if correction != "none" {
						text := field.accepted
						if correction == "different" {
							text = field.different
						}
						typeMountedCursor(t, m, field.id, text)
					}
					activate()
					after := s.service.Snapshot()
					if correction == "none" {
						if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
							t.Fatal("untouched invalid timing input was silently accepted")
						}
						return
					}
					want := cfg
					if correction == "different" {
						switch field.id {
						case "recorded-timing-level":
							want.Level = field.different
						case "recorded-timing-second-level":
							want.SecondLevel = field.different
						case "recorded-timing-start":
							want.Start = .001
						case "recorded-timing-end":
							want.End = .45
						}
					}
					if action == "direction-without-blur" {
						want.Direction = "rising"
					} else {
						want.Offset = before.Recording.Timing.NextOffset
					}
					if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || workbench.RecordingTimingParams(after.Recording.Timing.Settings) != want {
						t.Fatalf("fresh correction failed: got=%+v want=%+v error=%q", after.Recording.Timing.Settings, want, s.errorMessage)
					}
					assertCursorMeasurementPreservesPhysics(t, before, after)
				})
			}
		}
	}
}

func TestMountedRecordingTimingOtherFieldCannotEraseRefusal(t *testing.T) {
	for _, action := range []string{"page", "direction"} {
		t.Run(action, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			m := mountedTimingControls(t, s)
			typeMountedCursor(t, m, "recorded-timing-level", "bad")
			clickMountedControl(t, m, "recorded-timing-next")
			if s.errorMessage == "" {
				t.Fatal("fixture did not reject threshold")
			}
			typeMountedCursor(t, m, "recorded-timing-end", "450 ms")
			clickMountedControl(t, m, "recorded-timing-start")
			before := s.service.Snapshot()
			if before.Recording.Timing.Settings.End != .45 {
				t.Fatal("fixture did not accept the separate interval field")
			}
			if action == "page" {
				clickMountedControl(t, m, "recorded-timing-next")
			} else {
				changeMountedTimingDirection(t, s, m)
			}
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("another field's success silently reused the rejected threshold")
			}
		})
	}
}

// Invoke the assembled selector callback without uiAction's fatal-on-error
// assertion: a refused choice is an expected outcome in these cases.
func changeMountedTimingDirection(t *testing.T, s *screen, m *mountedControlWindow) {
	t.Helper()
	node := verificationControl(t, s, "recorded-timing-direction")
	s.app.Do(func() { node.Change(ui.ChangeEvent{SelectedID: "rising"}) })
	m.settle()
}

func TestMountedRecordingTimingProbeOwnsPrivateDraft(t *testing.T) {
	for _, edit := range []struct{ id, text string }{
		{"recorded-timing-level", "3"}, {"recorded-timing-second-level", "4.5"},
		{"recorded-timing-start", "1 ms"}, {"recorded-timing-end", "15 ms"},
	} {
		t.Run(edit.id, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			r := s.service.Snapshot().Recording
			if len(r.Scope.Available) < 2 {
				t.Fatal("fixture needs two captured voltage probes")
			}
			cfg := workbench.RecordingTimingParams{ID: r.ID, ProbeID: r.Scope.Available[0].ID, Level: "2.5", SecondLevel: "4", Direction: "both", End: .02}
			agentAction(t, s.service, "recording.timing.set", cfg)
			m := mountedTimingControls(t, s)
			typeMountedCursor(t, m, edit.id, edit.text)
			cfg = workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
			cfg.ProbeID = r.Scope.Available[1].ID
			agentAction(t, s.service, "recording.timing.set", cfg)
			before := s.service.Snapshot()
			m.frame()
			changeMountedTimingDirection(t, s, m)
			after := s.service.Snapshot()
			cfg.Direction = "rising"
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || workbench.RecordingTimingParams(after.Recording.Timing.Settings) != cfg {
				t.Fatalf("old private draft changed the agent-selected probe: got=%+v want=%+v error=%q", after.Recording.Timing.Settings, cfg, s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
			// Fresh input on the new probe must remain usable.
			typeMountedCursor(t, m, edit.id, edit.text)
			other := "recorded-timing-level"
			if edit.id == other {
				other = "recorded-timing-end"
			}
			clickMountedControl(t, m, other)
			switch edit.id {
			case "recorded-timing-level":
				cfg.Level = edit.text
			case "recorded-timing-second-level":
				cfg.SecondLevel = edit.text
			case "recorded-timing-start":
				cfg.Start = .001
			case "recorded-timing-end":
				cfg.End = .015
			}
			if s.errorMessage != "" || workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings) != cfg {
				t.Fatal("fresh input failed on the agent-selected probe", s.errorMessage)
			}
		})
	}
}

func TestMountedRecordingTimingCoupledCorrectionMatchesAgent(t *testing.T) {
	for _, edit := range []struct{ field, rejected, other, corrected string }{
		{"recorded-timing-level", "4 V", "recorded-timing-second-level", "4.5 V"},
		{"recorded-timing-second-level", "2.5 V", "recorded-timing-level", "2 V"},
		{"recorded-timing-start", "400 ms", "recorded-timing-end", "500 ms"},
		{"recorded-timing-end", "100 ms", "recorded-timing-start", "0 s"},
	} {
		for _, action := range []string{"direction", "page"} {
			if action == "page" && (edit.field == "recorded-timing-start" || edit.field == "recorded-timing-end") {
				continue // Short corrected windows can have no next page.
			}
			for _, delivery := range []string{"private", "committed"} {
				t.Run(edit.field+"/"+action+"/"+delivery, func(t *testing.T) {
					s, cleanup := testScreen(t)
					t.Cleanup(cleanup)
					agent := workbench.New()
					t.Cleanup(agent.Close)
					for _, service := range []*workbench.Service{s.service, agent} {
						prepareRecordedTiming(t, service)
						cfg := workbench.RecordingTimingParams(service.Snapshot().Recording.Timing.Settings)
						cfg.SecondLevel = "4 V"
						if edit.field == "recorded-timing-start" {
							cfg.End = .3
						} else if edit.field == "recorded-timing-end" {
							cfg.Start = .2
						}
						agentAction(t, service, "recording.timing.set", cfg)
					}
					m := mountedTimingControls(t, s)
					typeMountedCursor(t, m, edit.field, edit.rejected)
					changeMountedTimingDirection(t, s, m)
					if s.errorMessage == "" {
						t.Fatal("fixture did not reject the dependent entry")
					}
					typeMountedCursor(t, m, edit.other, edit.corrected)
					if delivery == "committed" {
						other := "recorded-timing-start"
						if other == edit.other {
							other = "recorded-timing-end"
						}
						clickMountedControl(t, m, other)
					}
					before := s.service.Snapshot()
					if action == "page" {
						button := verificationControl(t, s, "recorded-timing-next")
						s.app.Do(button.Activate)
						m.settle()
					} else {
						changeMountedTimingDirection(t, s, m)
					}
					after := s.service.Snapshot()
					want := workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
					switch edit.field {
					case "recorded-timing-level":
						want.Level, want.SecondLevel = edit.rejected, edit.corrected
					case "recorded-timing-second-level":
						want.SecondLevel, want.Level = edit.rejected, edit.corrected
					case "recorded-timing-start":
						want.Start, want.End = .4, .5
					case "recorded-timing-end":
						want.Start, want.End = 0, .1
					}
					agentAction(t, agent, "recording.timing.set", want)
					if action == "page" {
						want.Offset = agent.Snapshot().Recording.Timing.NextOffset
					} else {
						want.Direction = "rising"
					}
					agentAction(t, agent, "recording.timing.set", want)
					revisions := uint64(2)
					if delivery == "private" {
						revisions++
					}
					if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+revisions || !reflect.DeepEqual(after.Recording.Timing, agent.Snapshot().Recording.Timing) {
						t.Fatalf("valid coupled entries did not match public atomic settings: got=%+v want=%+v error=%q", after.Recording.Timing.Settings, want, s.errorMessage)
					}
					assertCursorMeasurementPreservesPhysics(t, before, after)
				})
			}
		}
	}
}

func TestMountedRecordingTimingRejectedPairCanCommitTogether(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
	cfg.SecondLevel = "4 V"
	agentAction(t, s.service, "recording.timing.set", cfg)
	m := mountedTimingControls(t, s)
	for _, edit := range []struct{ id, text string }{{"recorded-timing-level", "4 V"}, {"recorded-timing-second-level", "2.5 V"}} {
		typeMountedCursor(t, m, edit.id, edit.text)
		clickMountedControl(t, m, "recorded-timing-start")
		if s.errorMessage == "" {
			t.Fatal("fixture needs two individually rejected values")
		}
	}
	before := s.service.Snapshot()
	changeMountedTimingDirection(t, s, m)
	after := s.service.Snapshot()
	cfg.Level, cfg.SecondLevel, cfg.Direction = "4 V", "2.5 V", "rising"
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || workbench.RecordingTimingParams(after.Recording.Timing.Settings) != cfg {
		t.Fatalf("valid pair did not commit atomically before direction: got=%+v error=%q", after.Recording.Timing.Settings, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedRecordingTimingRevalidationRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent", "menu", "file"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
			cfg.SecondLevel = "4 V"
			agentAction(t, s.service, "recording.timing.set", cfg)
			field := verificationControl(t, s, "recorded-timing-level")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "4 V"}) })
			if s.errorMessage == "" {
				t.Fatal("fixture did not reject the dependent threshold")
			}
			field = verificationControl(t, s, "recorded-timing-second-level")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "4.5 V"}) })
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			node := verificationControl(t, s, "recorded-timing-direction")
			s.app.Do(func() { node.Change(ui.ChangeEvent{SelectedID: "rising"}) })
			if finish == nil || s.pendingRecordedTimingPage == nil {
				t.Fatal("fixture did not defer direction settlement")
			}
			switch scenario {
			case "agent":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
			case "menu":
				s.app.Do(func() { s.menuSerial++ })
			case "file":
				s.app.Do(func() { s.fileBusy = true })
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				for range 4 {
					if s.pendingRecordedTimingPage != nil {
						finish(nil)
					}
				}
			})
			if s.errorMessage == "" || s.pendingRecordedTimingPage != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("revalidation applied private entries after intervening work", s.errorMessage)
			}
		})
	}
}
