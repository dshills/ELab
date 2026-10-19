package conformance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/elab/workbench"
)

type shutdownObservation struct {
	Started  time.Time          `json:"started_at"`
	Finished time.Time          `json:"finished_at"`
	Response workbench.Response `json:"response"`
	Error    string             `json:"error,omitempty"`
}

// closeElectrical retains the final response even when its envelope or cleanup
// fails. A transport success alone does not mean the native host may exit.
func (s *session) closeElectrical() (err error) {
	observation := shutdownObservation{Started: time.Now().UTC()}
	defer func() {
		observation.Finished = time.Now().UTC()
		if err != nil {
			observation.Error = err.Error()
		}
		file, writeErr := os.OpenFile(filepath.Join(s.directory, "shutdown-response.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if writeErr == nil {
			writeErr = errors.Join(json.NewEncoder(file).Encode(observation), file.Close())
		}
		if writeErr != nil {
			err = errors.Join(err, fmt.Errorf("record shutdown response: %w", writeErr))
		}
	}()
	observation.Response, err = s.request("app.close", workbench.ReplaceParams{DiscardChanges: true}, true)
	if err != nil {
		return err
	}
	if failure := observation.Response.Error; failure != nil {
		return fmt.Errorf("app.close: %s: %s", failure.Code, failure.Message)
	}
	data, err := json.Marshal(observation.Response.Result)
	if err != nil {
		return fmt.Errorf("app.close result: %w", err)
	}
	var state workbench.Snapshot
	if err = json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("app.close result: %w", err)
	}
	if !state.ReadyToExit() {
		return fmt.Errorf("app.close did not complete exit readiness: closed=%t shutdown=%+v", state.Closed, state.Shutdown)
	}
	if state.Shutdown.Status != "completed" || !state.Shutdown.WorkersJoined || !state.Shutdown.RecoveryFinalized {
		return fmt.Errorf("app.close cleanup did not complete successfully: %+v", state.Shutdown)
	}
	return nil
}
