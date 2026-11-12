package main

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/dshills/elab/workbench"
)

type sceneGeometry struct {
	drawing scene
	routes  map[string][]workbench.Point
}

func (g sceneGeometry) live(state workbench.Snapshot) liveScene {
	frame := liveScene{Markers: workbench.MakeDisplayFrameFromPaths(g.routes, state.Display, 0).Markers, Parts: []liveScenePart{}}
	for _, part := range g.drawing.Parts {
		if !part.Dynamic {
			continue
		}
		live := liveScenePart{ID: part.ID, Paths: part.Paths}
		if state.Mixed != nil && !state.Mixed.Stale {
			for _, control := range state.Mixed.Controls {
				if control.ID == part.ID {
					live.Paths = pathLines(workbench.ControlSymbolPaths(part.Kind, control.Memory))
					break
				}
			}
			for _, display := range state.Mixed.Displays {
				if display.ID == part.ID {
					live.Lit = display.Lit
					break
				}
			}
		}
		frame.Parts = append(frame.Parts, live)
	}
	return frame
}

func (g sceneGeometry) present(frame liveScene) scene {
	out := g.drawing
	out.Markers = frame.Markers
	if len(frame.Parts) > 0 {
		out.Parts = slices.Clone(out.Parts)
		index := 0
		for i := range out.Parts {
			if out.Parts[i].Dynamic {
				out.Parts[i].Paths = frame.Parts[index].Paths
				out.Parts[i].Lit = frame.Parts[index].Lit
				index++
			}
		}
	}
	return out
}

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
	return digest([]byte(fmt.Sprintf("%s:%s:%d:%d:%d:%d:%s", s.instance, filepath.Base(ws.directory), state.Document.Generation, state.CommandRevision, state.ElectricalRevision, state.ActuationRevision, state.ActiveSheet)))
}

// Access requires ws.files. A replaced cache leaves all earlier response data
// immutable so encoders can keep using it after that lock is released.
func (ws *workspace) geometry(state workbench.Snapshot, token string) sceneGeometry {
	if token != ws.sceneCacheToken {
		ws.sceneCache = buildSceneGeometry(state)
		ws.sceneCacheToken = token
	}
	return ws.sceneCache
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
