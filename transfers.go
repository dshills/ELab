package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/elab/workbench"
)

func awaitFile(ctx context.Context, ws *workspace, id string) error {
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		work := ws.service.Snapshot().ProjectWork
		if work != nil && work.ID == id {
			switch work.Status {
			case "completed":
				return nil
			case "failed", "canceled", "superseded":
				if work.Error != nil {
					return fmt.Errorf("%s", work.Error.Message)
				}
				return fmt.Errorf("file operation %s", work.Status)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("file operation timed out")
		case <-tick.C:
		}
	}
}
func (s *webServer) transfer(w http.ResponseWriter, r *http.Request, ws *workspace, rev, gen uint64, kind string) bool {
	supported := map[string]string{"pdf": "schematic-pdf", "png": "schematic-png", "recording-json": "verification-json", "recording-csv": "verification-csv", "sweep-json": "sweep-json", "sweep-csv": "sweep-csv", "spectrum-json": "spectrum-json", "spectrum-csv": "spectrum-csv", "tolerance-json": "tolerance-json", "tolerance-csv": "tolerance-csv", "audio-wav": "audio-wav"}
	native, ok := supported[kind]
	if !ok {
		return false
	}
	extension := map[string]string{"pdf": "pdf", "png": "png", "recording-json": "json", "recording-csv": "csv", "sweep-json": "json", "sweep-csv": "csv", "spectrum-json": "json", "spectrum-csv": "csv", "tolerance-json": "json", "tolerance-csv": "csv", "audio-wav": "wav"}[kind]
	path := filepath.Join(ws.directory, "export-"+token()+"."+extension)
	defer os.Remove(path)
	state := ws.service.Snapshot()
	id := r.URL.Query().Get("id")
	if id == "" {
		switch native {
		case "verification-json", "verification-csv":
			id = state.SelectedOperation
		case "sweep-json", "sweep-csv":
			if state.DCSweep != nil {
				id = state.DCSweep.Selected
			}
		case "spectrum-json", "spectrum-csv":
			if state.Spectrum != nil && state.Spectrum.Result != nil {
				id = state.Spectrum.Result.ID
			}
		case "tolerance-json", "tolerance-csv":
			if state.Tolerance != nil {
				id = state.Tolerance.ID
			}
		case "audio-wav":
			if state.Audio != nil {
				id = state.Audio.ID
			}
		}
	}
	response := fileCommand(r.Context(), ws, "export.start", workbench.ExportStartParams{Kind: native, ID: id, Path: path, Overwrite: true}, rev, gen)
	if response.Error != nil {
		jsonReply(w, 409, response)
		return true
	}
	work := ws.service.Snapshot().ProjectWork
	if work == nil {
		webError(w, 500, "Missing export operation")
		return true
	}
	if err := awaitFile(r.Context(), ws, work.ID); err != nil {
		webError(w, 409, err.Error())
		return true
	}
	data, err := readBounded(path, 64<<20)
	if err != nil {
		webError(w, 500, err.Error())
		return true
	}
	mime := map[string]string{"pdf": "application/pdf", "png": "image/png", "json": "application/json", "csv": "text/csv", "wav": "audio/wav"}[extension]
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ELab-%s.%s"`, kind, extension))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
	return true
}
func (s *webServer) importResult(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != "POST" {
		webError(w, 405, "POST required")
		return
	}
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	kind := r.URL.Query().Get("kind")
	command := "verification.import"
	if kind == "sweep" {
		command = "sweep.import"
	} else if kind != "recording" {
		webError(w, 400, "Choose recording or sweep")
		return
	}
	ws.files.Lock()
	defer ws.files.Unlock()
	r.Body = http.MaxBytesReader(w, r.Body, workbench.MaxProjectBytes)
	var raw json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if decoder.Decode(&raw) != nil || decoder.Decode(new(any)) != io.EOF {
		webError(w, 400, "Choose a native exported JSON result")
		return
	}
	path := filepath.Join(ws.directory, "import-"+token()+".json")
	defer os.Remove(path)
	if err = atomicWrite(path, raw); err != nil {
		webError(w, 500, err.Error())
		return
	}
	response := fileCommand(r.Context(), ws, command, workbench.RecordingImportParams{Path: path}, rev, gen)
	if response.Error == nil {
		_ = ws.persist()
	}
	jsonReply(w, 200, response)
}
