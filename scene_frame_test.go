package main

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestConditionalSceneIsOpaqueScopedAndInvalidatedByEdits(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	response := f.request("GET", "/api/state", nil, "")
	var first map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	var token string
	_ = json.Unmarshal(first["scene_token"], &token)
	if token == "" || strings.Contains(token, f.cookie.Value) {
		t.Fatal("scene token absent or exposes cookie")
	}
	query := url.Values{"scene_token": {token}, "instance": {f.server.instance}}.Encode()
	response = f.request("GET", "/api/state?"+query, nil, "")
	var delta map[string]json.RawMessage
	_ = json.Unmarshal(response.Body.Bytes(), &delta)
	if delta["scene"] != nil || delta["scene_live"] == nil {
		t.Fatal("unchanged scene geometry retransmitted", response.Code)
	}
	var fullScene scene
	_ = json.Unmarshal(first["scene"], &fullScene)
	var live liveScene
	_ = json.Unmarshal(delta["scene_live"], &live)
	expected := sceneFrame(fullScene)
	if !reflect.DeepEqual(live, expected) {
		t.Fatal("delta lost dynamic presentation")
	}
	before := f.state()
	f.command("component.move", workbench.MoveParams{ID: "R1", Position: workbench.Point{X: 800, Y: 700}})
	response = f.request("GET", "/api/state?"+query, nil, "")
	_ = json.Unmarshal(response.Body.Bytes(), &delta)
	if delta["scene"] == nil {
		t.Fatal("geometry edit did not invalidate token")
	}
	if f.state().ElectricalRevision != before.ElectricalRevision {
		t.Fatal("geometry protocol changed physics")
	}
	other := &webFixture{server: f.server, handler: f.handler, t: t}
	response = other.request("GET", "/api/state?"+query, nil, "")
	var otherReply map[string]json.RawMessage
	_ = json.Unmarshal(response.Body.Bytes(), &otherReply)
	if otherReply["scene"] == nil || string(otherReply["scene_token"]) == string(first["scene_token"]) {
		t.Fatal("scene token crossed workspaces")
	}
	restarted := newWebServer(f.server.root)
	defer restarted.close()
	again := &webFixture{server: restarted, handler: restarted.handler(), cookie: f.cookie, t: t}
	response = again.request("GET", "/api/state?"+query, nil, "")
	var fresh map[string]json.RawMessage
	_ = json.Unmarshal(response.Body.Bytes(), &fresh)
	if fresh["scene"] == nil || string(fresh["scene_token"]) == string(first["scene_token"]) {
		t.Fatal("scene token survived host restart")
	}
}
func TestSceneFrameCarriesControlAndDisplayChangesWithoutStaticGeometry(t *testing.T) {
	frame := scene{Parts: []scenePart{{Component: workbench.Component{ID: "S1", Kind: "pushbutton"}, Dynamic: true, Paths: []string{"open"}}, {Component: workbench.Component{ID: "D1", Kind: "seven-segment"}, Dynamic: true, Lit: [7]bool{true}}, {Component: workbench.Component{ID: "R1", Kind: "resistor"}, Paths: []string{"static"}}}}
	first := sceneFrame(frame)
	if len(first.Parts) != 2 {
		t.Fatal("static geometry leaked into live frame")
	}
	frame.Parts[0].Paths = []string{"closed"}
	frame.Parts[1].Lit = [7]bool{false, true}
	next := sceneFrame(frame)
	if next.Parts[0].Paths[0] != "closed" || !next.Parts[1].Lit[1] || next.Parts[1].Lit[0] || first.Parts[0].Paths[0] != "open" {
		t.Fatal("live controls or display state froze")
	}
}

func TestOptionalCommandPresentationKeepsEnvelopeAndRetrySemantics(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	state := f.state()
	params, _ := json.Marshal(workbench.MoveParams{ID: "R1", Position: workbench.Point{X: 800, Y: 700}})
	request := workbench.Request{Version: 1, ID: "projected-move", Key: "projected-move", Command: "component.move", Params: params, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}
	raw, _ := json.Marshal(request)
	r := f.request("POST", "/api/command?presentation=true", raw, "")
	var reply struct {
		workbench.Response
		Scene    *scene `json:"scene"`
		Instance string `json:"instance_id"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Error != nil || reply.Scene == nil || reply.Instance != f.server.instance {
		t.Fatal("current ack missing scene projection", r.Body.String())
	}
	var first map[string]json.RawMessage
	_ = json.Unmarshal(r.Body.Bytes(), &first)
	// A retry must still return the exact retained command result, not relabel it
	// as fresh presentation after another command changed the workspace.
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	retry := f.request("POST", "/api/command?presentation=true", raw, "")
	var again map[string]json.RawMessage
	_ = json.Unmarshal(retry.Body.Bytes(), &again)
	if again["scene"] != nil || string(again["duplicate"]) != "true" || string(first["result"]) != string(again["result"]) {
		t.Fatal("projected retry changed its original result", retry.Body.String())
	}
}
