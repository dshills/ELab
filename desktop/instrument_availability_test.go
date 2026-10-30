package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestInstrumentTabsAvailableWithoutTimeData(t *testing.T) {
	for _, circuit := range []string{"new", "led", "npn"} {
		t.Run(circuit, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			if circuit == "new" {
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{})
			} else {
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: circuit})
			}
			agentAction(t, s.service, "simulation.pause", struct{}{})
			before := s.service.Snapshot()
			if before.Transient != nil {
				t.Fatal("fixture unexpectedly has time-based data")
			}
			for _, theme := range []string{"light", "dark", "system", "apollo"} {
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				for _, view := range []string{"scope", "spectrum"} {
					uiAction(t, s, "scope-view-"+view, nil)
					root := s.view()
					if err := ui.Validate(root); err != nil {
						t.Fatal(err)
					}
					for _, id := range []string{"scope-view-scope", "scope-view-spectrum", "instrument-empty-title", "instrument-empty-help"} {
						n, ok := findNode(root, id)
						if !ok || n.Disabled {
							t.Fatalf("%s/%s: unavailable %s", theme, view, id)
						}
					}
					title, _ := findNode(root, "instrument-empty-title")
					help, _ := findNode(root, "instrument-empty-help")
					if title.Text != "No time-based data" || !strings.Contains(help.Text, "DC") || !strings.Contains(help.Text, "transient") || !strings.Contains(help.Text, "Recorded runs") {
						t.Fatal("missing explanation and recovery path", title.Text, help.Text)
					}
					if _, ok := findNode(root, "rc-scope"); ok {
						t.Fatal("DC circuit mounted a misleading live scope plot")
					}
					if view == "spectrum" {
						summary, _ := findNode(root, "spectrum-dock-summary")
						if summary.Text != "Live spectrum · no time-based data" {
							t.Fatal("empty spectrum incorrectly claims acquisition is running", summary.Text)
						}
					}
				}
			}
			after := s.service.Snapshot()
			if after.Transient != nil || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Document, after.Document) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				t.Fatal("instrument navigation changed circuit or simulation")
			}
		})
	}
}

func TestCollapsedInstrumentKeepsBothTabsAndSharedVisibility(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "simulation.pause", struct{}{})
	uiAction(t, s, "scope-visibility", nil)
	for _, view := range []string{"spectrum", "scope"} {
		uiAction(t, s, "scope-view-"+view, nil)
		state := s.service.Snapshot()
		if state.ScopeVisible || state.ScopeView != view {
			t.Fatal("tab changed visibility or failed to select shared view")
		}
		for _, id := range []string{"scope-view-scope", "scope-view-spectrum", "scope-expand"} {
			if _, ok := findNode(s.view(), id); !ok {
				t.Fatal("collapsed dock lost navigation", id)
			}
		}
	}
	uiAction(t, s, "scope-expand", nil)
	if !s.service.Snapshot().ScopeVisible {
		t.Fatal("shared visibility command did not expand empty dock")
	}
	agentAction(t, s.service, "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: false})
	if _, ok := findNode(s.view(), "scope-expand"); !ok {
		t.Fatal("agent hide removed dock navigation")
	}
}

func TestRecordedInstrumentsRemainUsableWithoutLiveCircuit(t *testing.T) {
	source := workbench.New()
	defer source.Close()
	prepareSpectrumDock(t, source)
	duration, step := .1, .001
	agentAction(t, source, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}})
	operation := completedVerification(t, source, source.Snapshot().LastOperation)
	path := filepath.Join(t.TempDir(), "recording.json")
	agentAction(t, source, "verification.export", workbench.VerificationExportParams{ID: operation.Operation.ID, Path: path, Format: "json"})
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{})
	agentAction(t, s.service, "verification.import", workbench.RecordingImportParams{Path: path})
	before := s.service.Snapshot()
	if _, ok := findNode(s.view(), "recorded-scope-title"); !ok {
		t.Fatal("empty live circuit hid a valid recorded scope")
	}
	if _, ok := findNode(s.view(), "instrument-empty-title"); ok {
		t.Fatal("valid recorded scope was called empty")
	}
	uiAction(t, s, "scope-view-spectrum", nil)
	uiAction(t, s, "spectrum-source-recorded", nil)
	uiAction(t, s, "spectrum-dock-analyze", nil)
	if s.service.Snapshot().Spectrum.Result == nil {
		t.Fatal("recorded spectrum could not analyze independently of live circuit")
	}
	if _, ok := findNode(s.view(), "instrument-empty-title"); ok {
		t.Fatal("valid recorded spectrum was called empty")
	}
	after := s.service.Snapshot()
	if after.Transient != nil || len(after.Components) != 0 || !reflect.DeepEqual(before.Document, after.Document) || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID {
		t.Fatal("recorded instrument navigation changed live circuit")
	}
	uiAction(t, s, "scope-view-scope", nil)
	if _, ok := findNode(s.view(), "recorded-scope-title"); !ok {
		t.Fatal("returning to Scope lost the recording")
	}
}
