package main

import (
	"fmt"
	"path/filepath"

	"github.com/dshills/elab/workbench"
)

type liveScenePart struct {
	ID    string   `json:"id"`
	Paths []string `json:"paths"`
	Lit   [7]bool  `json:"lit"`
}
type liveScene struct {
	Markers []workbench.CurrentMarker `json:"markers"`
	Parts   []liveScenePart           `json:"parts"`
}

func sceneFrame(frame scene) liveScene {
	out := liveScene{Markers: frame.Markers, Parts: []liveScenePart{}}
	for _, p := range frame.Parts {
		if p.Dynamic {
			out.Parts = append(out.Parts, liveScenePart{p.ID, p.Paths, p.Lit})
		}
	}
	return out
}
func (s *webServer) sceneToken(ws *workspace, state workbench.Snapshot) string {
	// Opaque and scoped to host, workspace and document/command admission. Never
	// expose the HttpOnly workspace-cookie value as a client-side cache token.
	return digest([]byte(fmt.Sprintf("%s:%s:%d:%d", s.instance, filepath.Base(ws.directory), state.Document.Generation, state.CommandRevision)))
}

// Optional browser projection preserves the original command envelope/result.
// It is attached only while the acknowledged snapshot is still current; retries
// and intervening publications keep the normal response and fresh-read path.
type browserCommandReply struct {
	workbench.Response
	Workspace  workspaceInfo  `json:"workspace"`
	Agent      map[string]any `json:"agent"`
	Scene      scene          `json:"scene"`
	Instance   string         `json:"instance_id"`
	SceneToken string         `json:"scene_token"`
}
