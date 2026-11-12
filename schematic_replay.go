package main

import (
	"fmt"
	"net/http"

	"github.com/dshills/elab/workbench"
)

type replaySheet struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Scene scene  `json:"scene"`
}

type replayCapture struct {
	ID     string                      `json:"id"`
	Probes []workbench.Probe           `json:"probes"`
	Rows   []workbench.VerificationRow `json:"rows"`
	Sheets []replaySheet               `json:"sheets"`
}

type replaySource struct {
	replayCapture
	metadata *workbench.VerificationMetadata
}

func loadReplay(r *http.Request, ws *workspace, id string) (replaySource, error) {
	query := func(command string, params any) workbench.Response {
		return fileCommand(r.Context(), ws, command, params, 0, 0)
	}
	// Keep metadata and every exact sample page on the same retained recording.
	ws.files.Lock()
	locked := true
	defer func() {
		if locked {
			ws.files.Unlock()
		}
	}()
	revision, generation, err := guards(r)
	if err != nil {
		return replaySource{}, err
	}
	state := ws.service.Snapshot()
	if state.CommandRevision != revision || state.Document.Generation != generation {
		return replaySource{}, fmt.Errorf("the circuit changed; choose the recording again")
	}
	inspect := query("operation.inspect", workbench.IDParams{ID: id})
	if inspect.Error != nil {
		return replaySource{}, fmt.Errorf("%s", inspect.Error.Message)
	}
	view, ok := inspect.Result.(workbench.OperationView)
	if !ok || view.Operation.Status != "completed" || view.Metadata == nil {
		return replaySource{}, fmt.Errorf("choose a completed recording")
	}
	result := replayCapture{ID: id, Probes: view.Metadata.Probes}
	for offset := 0; offset < view.Metadata.SampleCount; {
		if err := r.Context().Err(); err != nil {
			return replaySource{}, err
		}
		response := query("operation.samples", workbench.VerificationPageParams{ID: id, Offset: offset, Limit: 200})
		page, ok := response.Result.(workbench.VerificationPage)
		if !ok || page.NextOffset <= offset || page.Total != view.Metadata.SampleCount {
			return replaySource{}, fmt.Errorf("the recording changed or is unavailable")
		}
		result.Rows = append(result.Rows, page.Rows...)
		offset = page.NextOffset
	}
	if len(result.Rows) == 0 || len(result.Probes) == 0 {
		return replaySource{}, fmt.Errorf("the recording contains no measurements")
	}
	ws.files.Unlock()
	locked = false
	graph := view.Metadata.Graph
	state = workbench.Snapshot{Components: graph.Components, Wires: graph.Wires, Sheets: graph.Sheets}
	result.Sheets = append(result.Sheets, replaySheet{Name: "Main", Scene: buildSceneGeometry(state).drawing})
	for _, sheet := range graph.Sheets {
		state.ActiveSheet = sheet.ID
		result.Sheets = append(result.Sheets, replaySheet{sheet.ID, sheet.Name, buildSceneGeometry(state).drawing})
	}
	return replaySource{result, view.Metadata}, nil
}

func (s *webServer) schematicReplay(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodGet {
		webError(w, 405, "GET required")
		return
	}
	result, err := loadReplay(r, ws, r.URL.Query().Get("id"))
	if err != nil {
		webError(w, 422, err.Error())
		return
	}
	jsonReply(w, 200, result.replayCapture)
}
