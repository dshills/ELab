package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/dshills/elab/workbench"
)

type compareChangeParams struct {
	Part     string  `json:"part"`
	Value    string  `json:"value"`
	Probe    string  `json:"probe"`
	Duration float64 `json:"duration_s"`
}

func readExperimentRequest(w http.ResponseWriter, r *http.Request, out any) bool {
	if r.Method != http.MethodPost {
		webError(w, 405, "POST required")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil || decoder.Decode(new(any)) != io.EOF {
		webError(w, 400, "Invalid experiment request")
		return false
	}
	return true
}

func observedExperiment(w http.ResponseWriter, r *http.Request, ws *workspace) (workbench.Snapshot, bool) {
	revision, generation, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return workbench.Snapshot{}, false
	}
	state := ws.service.Snapshot()
	if state.CommandRevision != revision || state.Document.Generation != generation {
		webError(w, 409, "The circuit changed. Reopen the experiment to use the current circuit.")
		return workbench.Snapshot{}, false
	}
	return state, true
}

func experimentGraph(state workbench.Snapshot) workbench.Graph {
	return workbench.Graph{Components: state.Components, Wires: state.Wires, Sheets: state.Sheets}
}

func (s *webServer) compareChange(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var params compareChangeParams
	if !readExperimentRequest(w, r, &params) {
		return
	}
	if !ws.experimentMu.TryLock() {
		webError(w, 409, "Wait for the current experiment to finish.")
		return
	}
	defer ws.experimentMu.Unlock()
	state, ok := observedExperiment(w, r, ws)
	if !ok {
		return
	}
	graph := experimentGraph(state)
	part, ok := graph.Component(params.Part)
	if !ok || part.Unit == "" || part.Block != nil {
		webError(w, 400, "Choose a part with an editable numeric value.")
		return
	}
	next, err := workbench.SetValue(part, params.Value)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	if part.Value == next.Value {
		webError(w, 400, "Enter a different value to compare.")
		return
	}
	before, err := workbench.CaptureExperiment(r.Context(), graph, state.TransientSettings, params.Probe, params.Duration)
	if err != nil {
		webError(w, 422, "Original circuit: "+err.Error())
		return
	}
	for i := range graph.Components {
		if graph.Components[i].ID == next.ID {
			graph.Components[i] = next
			break
		}
	}
	after, err := workbench.CaptureExperiment(r.Context(), graph, state.TransientSettings, params.Probe, params.Duration)
	if err != nil {
		webError(w, 422, "Trial circuit: "+err.Error())
		return
	}
	jsonReply(w, 200, map[string]any{
		"probe": before.Probe, "before": before.Rows, "after": after.Rows,
		"original": workbench.FormatValue(part.Value, part.Unit), "trial": workbench.FormatValue(next.Value, part.Unit),
	})
}
