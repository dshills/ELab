package main

import (
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestInspectorModelSummaryFollowsSavedTransistorSettings(t *testing.T) {
	for _, kind := range []string{"npn", "pnp"} {
		for _, mode := range []string{"generic", "profile", "thermal", "both"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				p := workbench.Component{ID: "Q1", Kind: kind, Name: "Transistor"}
				if mode == "profile" || mode == "both" {
					cfg := workbench.DefaultTransistorProfile()
					cfg.ForwardBeta = 200
					p.Device = &workbench.DeviceParameters{Bipolar: &cfg}
				}
				if mode == "thermal" || mode == "both" {
					cfg := workbench.DefaultThermalConfig(kind)
					p.Thermal = &cfg
				}
				root := ui.Node{ID: "root", Kind: ui.Column, Children: s.componentInspector(s.service.Snapshot(), workbench.Graph{}, p)}
				n, ok := findNode(root, "npn-model")
				if !ok {
					t.Fatal("missing transistor explanation")
				}
				if mode == "generic" {
					if !strings.Contains(n.Text, "βF 100 / βR 1") {
						t.Fatal("default model explanation lost", n.Text)
					}
					return
				}
				for _, misleading := range []string{"βF 100", "Instantaneous junction response", "No heating"} {
					if strings.Contains(n.Text, misleading) {
						t.Errorf("configured transistor still claims %q: %s", misleading, n.Text)
					}
				}
				if p.Device != nil && !strings.Contains(n.Text, "saved transistor profile") {
					t.Fatal("profile explanation missing", n.Text)
				}
				if p.Thermal != nil && !strings.Contains(n.Text, "temperature settings") {
					t.Fatal("temperature explanation missing", n.Text)
				}
			})
		}
	}
}
