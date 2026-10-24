//go:build offscreen

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedOutputPage(t *testing.T, s *screen, control string) *mountedControlWindow {
	t.Helper()
	if state := s.service.Snapshot(); !previewVisible(state) && state.InspectorSection != "document" {
		uiAction(t, s, "inspector-section-document", nil)
	}
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column}
		if !previewVisible(state) || control == "preview-refresh" {
			// The assembled view already binds these production callbacks/context.
			assembled := s.view()
			for _, id := range []string{"pdf-margin", control} {
				node, ok := findNode(assembled, id)
				if !ok {
					t.Fatalf("missing output control %s", id)
				}
				node.MinWidth, node.MinHeight = 120, 40
				root.Children = append(root.Children, node)
			}
			root.Focus = ui.FocusRequest{ID: "pdf-margin", Sequence: 1}
		}
		return root
	})
}

func TestMountedOutputPageOneClickPreviewIncludesTypedMargin(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	m := mountedOutputPage(t, s, "print-preview")
	before := s.service.Snapshot()
	if *before.SchematicPage.MarginMM == 17 {
		t.Fatal("fixture must change the margin")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "17" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("typing prematurely changed public settings")
	}
	clickMountedControl(t, m, "print-preview")
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.PrintPreview == nil || *after.PrintPreview.Page.MarginMM != 17 {
		t.Fatalf("one-click preview did not capture typed margin: margin=%g preview=%+v revision=%d→%d error=%s", *after.SchematicPage.MarginMM, after.PrintPreview, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
}

func waitOutputPreview(t *testing.T, s *screen) *workbench.PrintPreview {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := s.service.Snapshot()
		if state.PrintPreview != nil && !previewActive(state.PrintPreview) {
			if state.PrintPreview.Status != "completed" {
				t.Fatal("preview failed", state.PrintPreview)
			}
			return state.PrintPreview
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("preview did not complete")
	return nil
}

func TestMountedOutputPageFitAndRefreshUseTypedMargin(t *testing.T) {
	for _, control := range []string{"pdf-fit", "preview-refresh"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			if control == "preview-refresh" {
				agentAction(t, s.service, "print.preview.start", workbench.PreviewParams{})
				waitOutputPreview(t, s)
			} else {
				page := s.service.Snapshot().SchematicPage
				scale := 2.
				page.Scale = &scale
				agentAction(t, s.service, "export.page.set", page)
			}
			m := mountedOutputPage(t, s, control)
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range "17" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			clickMountedControl(t, m, control)
			after := s.service.Snapshot()
			if s.errorMessage != "" || *after.SchematicPage.MarginMM != 17 {
				t.Fatal("output action refused its own margin commit", control, s.errorMessage)
			}
			if control == "pdf-fit" {
				if after.SchematicPage.Scale != nil || after.SchematicPage.Tile {
					t.Fatal("Fit did not reset scale/tiling")
				}
			} else if after.PrintPreview.ID == before.PrintPreview.ID || *after.PrintPreview.Page.MarginMM != 17 {
				t.Fatal("Refresh did not capture updated margin")
			}
		})
	}
}

func TestMountedOutputPageHeldPreviewRefusesAgentAfterOwnCommit(t *testing.T) {
	for _, control := range []string{"print-preview", "preview-refresh", "pdf-fit"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			if control == "preview-refresh" {
				agentAction(t, s.service, "print.preview.start", workbench.PreviewParams{})
				waitOutputPreview(t, s)
			}
			m := mountedOutputPage(t, s, control)
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range "17" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			button := mountedControlElement(t, m, control)
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			page := s.service.Snapshot().SchematicPage
			margin := 7.
			page.MarginMM = &margin
			agentAction(t, s.service, "export.page.set", page)
			before := s.service.Snapshot()
			m.frame()
			if mountedControlElement(t, m, control).Incarnation != button.Incarnation {
				t.Fatal("fixture did not retain held output button")
			}
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("held output action adopted agent settings", control, s.errorMessage)
			}
			clickMountedControl(t, m, control)
			after := s.service.Snapshot()
			if s.errorMessage != "" || *after.SchematicPage.MarginMM != 7 {
				t.Fatal("fresh output action failed", control, s.errorMessage)
			}
			if control != "pdf-fit" && *after.PrintPreview.Page.MarginMM != 7 {
				t.Fatal("fresh preview lost agent margin")
			}
			assertOutputCircuitPreserved(t, before, after)
		})
	}
}

func TestMountedOutputPagePreviewPixelsMatchPublicAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	agentAction(t, agent, "example.reset", workbench.ReplaceParams{})
	m := mountedOutputPage(t, s, "print-preview")
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "17" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	m.frame()
	clickMountedControl(t, m, "print-preview")
	actual := waitOutputPreview(t, s)
	agentAction(t, agent, "export.page.set", s.service.Snapshot().SchematicPage)
	agentAction(t, agent, "print.preview.start", workbench.PreviewParams{})
	deadline := time.Now().Add(5 * time.Second)
	var expected *workbench.PrintPreview
	for time.Now().Before(deadline) {
		expected = agent.Snapshot().PrintPreview
		if expected != nil && !previewActive(expected) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if expected == nil || expected.Status != "completed" || s.errorMessage != "" || !reflect.DeepEqual(actual.Page, expected.Page) || actual.Width != expected.Width || actual.Height != expected.Height || actual.Bytes != expected.Bytes || actual.SHA256 != expected.SHA256 || actual.PageIndex != expected.PageIndex || actual.PageCount != expected.PageCount || !actual.Current || !expected.Current {
		t.Fatal("mounted preview differs from public agent", actual, expected, s.errorMessage)
	}
	if !bytes.Equal(outputPreviewPNG(t, m.ctx, s.service, actual), outputPreviewPNG(t, m.ctx, agent, expected)) {
		t.Fatal("public preview PNG bytes differ")
	}
	assertOutputCircuitPreserved(t, before, s.service.Snapshot())
	t.Logf("matched preview: %dx%d, %d bytes, SHA-256 %s", actual.Width, actual.Height, actual.Bytes, actual.SHA256)
}

func outputPreviewPNG(t *testing.T, ctx context.Context, service *workbench.Service, preview *workbench.PrintPreview) []byte {
	t.Helper()
	if preview.Bytes <= 0 || preview.Bytes > workbench.MaxProjectBytes {
		t.Fatal("invalid preview size")
	}
	data := make([]byte, 0, preview.Bytes)
	for len(data) < preview.Bytes {
		params, _ := json.Marshal(workbench.PreviewImageParams{ID: preview.ID, Offset: len(data)})
		response := service.Dispatch(ctx, workbench.Request{Version: 1, ID: fmt.Sprintf("output-preview-%d", len(data)), Command: "print.preview.image", Params: params}, "agent.output-proof")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		chunk, ok := response.Result.(workbench.PreviewImageChunk)
		if !ok || chunk.ID != preview.ID || chunk.Offset != len(data) || chunk.Total != preview.Bytes || chunk.SHA256 != preview.SHA256 || len(chunk.PNG) == 0 || chunk.Next != len(data)+len(chunk.PNG) || chunk.Next > preview.Bytes {
			t.Fatal("invalid public preview chunk")
		}
		data = append(data, chunk.PNG...)
	}
	return data
}
