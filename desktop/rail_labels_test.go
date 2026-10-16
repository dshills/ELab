package main

import (
	"reflect"
	"testing"

	"github.com/dshills/vellum/ui"
)

func TestRailLabelContainmentPreservesIDsTextFocusAndRepeatedLayout(t *testing.T) {
	activated := 0
	original := quiet("mixed-X", "50 Ω & literal <text>\nSigned −2 mA")
	original.Accessibility.Name = "Literal reading"
	// Component-derived IDs may share a suffix; bounds containers must not
	// collide with another legitimate reading's ID.
	nodes := []ui.Node{original, quiet("mixed-X-bounds", "Second reading"), {ID: "edit", Kind: ui.Button, Text: "Edit", Activate: func() { activated++ }}}
	beforeFocus := ui.FocusOrder(ui.Node{ID: "root", Kind: ui.Column, Children: nodes})
	for range 2 {
		constrainRail(nodes, 248)
		root := ui.Node{ID: "root", Kind: ui.Column, Children: nodes}
		if err := ui.Validate(root); err != nil {
			t.Fatalf("containment collides with a reading or repeated layout: %v", err)
		}
		reading, ok := findNode(root, original.ID)
		if !ok || reading.Kind != ui.Label || reading.Text != original.Text || reading.Accessibility != original.Accessibility || reading.Style.Foreground != original.Style.Foreground {
			t.Fatal("containment changed literal text, label identity or semantics")
		}
		if !reflect.DeepEqual(beforeFocus, ui.FocusOrder(root)) {
			t.Fatal("passive label containment changed control focus order")
		}
		button, _ := findNode(root, "edit")
		button.Activate()
	}
	if activated != 2 {
		t.Fatal("layout replaced the command callback")
	}
}
