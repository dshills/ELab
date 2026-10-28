package main

import (
	"strconv"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func blockPairs(s string) [][2]string {
	out := [][2]string{}
	for _, word := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		name, target, _ := strings.Cut(strings.TrimSpace(word), "=")
		out = append(out, [2]string{strings.TrimSpace(name), strings.TrimSpace(target)})
	}
	return out
}
func (s *screen) blockLibraryControls(state workbench.Snapshot) []ui.Node {
	if !s.blockFormReady {
		s.blockForm = [7]string{"BLOCK1", "Circuit block", "", "", "", "0", "300"}
		s.blockFormReady = true
	}
	out := []ui.Node{quiet("block-help", "Group ordinary parts behind named pins. Enter member IDs, pins such as in=R1.1, and optional values such as resistance=R1. Copy and paste blocks between drawings.")}
	labels := []string{"New block ID", "Block name", "Member IDs · comma separated", "Pins · name=member.terminal", "Published values · name=member", "Block X", "Block Y"}
	for k, label := range labels {
		node := ui.Node{ID: "block-form-" + strconv.Itoa(k), Kind: ui.TextField, Text: s.blockForm[k], Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) { s.blockForm[k] = e.Text }}
		if k >= 5 {
			v, _ := strconv.ParseFloat(s.blockForm[k], 64)
			node.Kind = ui.NumberField
			node.Text = ""
			node.Number = float32(v)
			node.Numeric = ui.Numeric{Min: -10000, Max: 10000, Step: 10}
			node.Change = func(e ui.ChangeEvent) { s.blockForm[k] = strconv.FormatFloat(float64(e.Number), 'g', -1, 32) }
		}
		out = append(out, quiet("block-form-label-"+strconv.Itoa(k), label), node)
	}
	position := func() workbench.Point {
		x, _ := strconv.ParseFloat(s.blockForm[5], 64)
		y, _ := strconv.ParseFloat(s.blockForm[6], 64)
		return workbench.Point{X: x, Y: y}
	}
	create := blockFormButton("block-create", "Create block", func() {
		ports := []workbench.BlockPort{}
		for _, p := range blockPairs(s.blockForm[3]) {
			ports = append(ports, workbench.BlockPort{Name: p[0], Terminal: p[1]})
		}
		parameters := []workbench.BlockParameter{}
		for _, p := range blockPairs(s.blockForm[4]) {
			parameters = append(parameters, workbench.BlockParameter{Name: p[0], ComponentID: p[1]})
		}
		members := strings.FieldsFunc(s.blockForm[2], func(r rune) bool { return r == ',' || r == ' ' || r == '\n' })
		s.invokeAtCommandRevision("block.create", workbench.BlockCreateParams{ID: s.blockForm[0], Name: s.blockForm[1], Position: position(), Members: members, Ports: ports, Parameters: parameters}, state.CommandRevision, state.Document.Generation)
	})
	out = append(out, create)
	paste := blockFormButton("block-paste", "Paste copied block", func() {
		s.invokeAtCommandRevision("block.paste", workbench.BlockPasteParams{ID: s.blockForm[0], Position: position()}, state.CommandRevision, state.Document.Generation)
	})
	paste.Disabled = !state.BlockClipboardAvailable
	out = append(out, paste)
	for _, p := range state.Components {
		if p.Block != nil {
			out = append(out, s.action("block-select-"+p.ID, p.ID+" · "+p.Name, "component.select", workbench.IDParams{ID: p.ID}))
		}
	}
	return out
}
func (s *screen) blockInspector(state workbench.Snapshot, p workbench.Component) []ui.Node {
	g := workbench.Graph{Components: state.Components, Wires: state.Wires}
	out := []ui.Node{heading("block-heading", p.ID+" · "+p.Name), quiet("block-model", workbench.BlockModelID), quiet("block-members", "Members: "+strings.Join(p.Block.Members, ", "))}
	for _, pin := range p.Block.Ports {
		out = append(out, quiet("block-pin-"+pin.Name, pin.Name+" → "+pin.Terminal))
	}
	for _, parameter := range p.Block.Parameters {
		c, ok := g.Component(parameter.ComponentID)
		if !ok {
			continue
		}
		name := parameter.Name
		out = append(out, quiet("block-value-label-"+name, name+" · "+c.Unit), ui.Node{ID: "block-value-" + name, Kind: ui.TextField, Text: workbench.FormatValue(c.Value, c.Unit), Accessibility: ui.Accessibility{Name: name}, Change: func(e ui.ChangeEvent) {
			s.invokeAtCommandRevision("block.parameter.set", workbench.BlockValueParams{ID: p.ID, Parameter: name, Value: e.Text}, state.CommandRevision, state.Document.Generation)
		}})
	}
	out = append(out, s.action("block-expand", "Show internals", "block.view.set", workbench.BlockViewParams{ID: p.ID, Expanded: true}), s.action("block-collapse", "Hide internals", "block.view.set", workbench.BlockViewParams{ID: p.ID, Expanded: false}), s.action("block-copy", "Copy block", "block.copy", workbench.IDParams{ID: p.ID}), s.action("block-release", "Release into ordinary parts", "block.release", workbench.IDParams{ID: p.ID}), quiet("block-inspection-help", "Show internals to select and edit every ordinary component. Document → Circuit blocks lets you select this block again, or paste the independent copied template into another drawing."))
	return out
}

func blockFormButton(id, text string, action func()) ui.Node {
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: action}, false)
}
