package main

import (
	"net/http"

	"github.com/dshills/elab/workbench"
)

type checkEvidence struct {
	Pass    bool                       `json:"pass"`
	Level   float64                    `json:"level"`
	Before  *workbench.VerificationRow `json:"before,omitempty"`
	After   *workbench.VerificationRow `json:"after,omitempty"`
	Minimum workbench.VerificationRow  `json:"minimum"`
	Maximum workbench.VerificationRow  `json:"maximum"`
}

func evaluateCheck(rows []workbench.VerificationRow, rule string, level float64) checkEvidence {
	evidence := checkEvidence{Level: level, Minimum: rows[0], Maximum: rows[0]}
	var lastNonZero *workbench.VerificationRow
	for i := range rows {
		row := &rows[i]
		value := row.Values[0]
		if value < evidence.Minimum.Values[0] {
			evidence.Minimum = *row
		}
		if value > evidence.Maximum.Values[0] {
			evidence.Maximum = *row
		}
		if rule == "reverses" && value != 0 {
			if lastNonZero != nil && (lastNonZero.Values[0] < 0) != (value < 0) && evidence.After == nil {
				evidence.Before, evidence.After = lastNonZero, row
			}
			lastNonZero = row
		}
		if i > 0 && evidence.After == nil {
			previous := rows[i-1].Values[0]
			if rule == "rises" && previous < level && value >= level || rule == "falls" && previous > level && value <= level {
				evidence.Before, evidence.After = &rows[i-1], row
			}
		}
	}
	evidence.Pass = evidence.After != nil || rule == "max" && evidence.Maximum.Values[0] <= level
	return evidence
}

func (s *webServer) runExperimentCheck(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var params workbench.ExperimentCheck
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
	duration, level, err := params.Quantities()
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	capture, err := workbench.CaptureExperiment(r.Context(), experimentGraph(state), state.TransientSettings, params.Probe, duration)
	if err != nil {
		webError(w, 422, err.Error())
		return
	}
	if capture.Probe.Unit != params.Unit {
		webError(w, 400, "The measurement units have changed. Choose the measurement again.")
		return
	}
	jsonReply(w, 200, map[string]any{"capture": capture, "evidence": evaluateCheck(capture.Rows, params.Rule, level)})
}
