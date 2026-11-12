package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/dshills/elab/workbench"
)

//go:embed replay-share.html
var shareTemplateSource string

var shareTemplate = template.Must(template.New("replay").Parse(shareTemplateSource))

type replayPosition struct {
	Index int `json:"index"`
	Sheet int `json:"sheet"`
	Probe int `json:"probe"`
}

type replayShareParams struct {
	Position       replayPosition `json:"position"`
	Note           string         `json:"note"`
	IncludeProject bool           `json:"include_project"`
}

type replaySharePayload struct {
	Capture  replayCapture  `json:"capture"`
	Position replayPosition `json:"position"`
	Project  []byte         `json:"project,omitempty"`
}

const shareBoot = `
replayView(document.getElementById('replay'), share.capture, share.position);
const projectButton = document.getElementById('project');
if (projectButton) {
 projectButton.onclick = () => {
  const data = Uint8Array.from(atob(share.project), byte => byte.charCodeAt(0));
  const url = URL.createObjectURL(new Blob([data], {type: 'application/json'}));
  const link = document.createElement('a');
  link.href = url;
  link.download = 'ELab-captured-circuit.elab';
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 2000);
 };
}
`

func (s *webServer) shareReplay(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var params replayShareParams
	if !readExperimentRequest(w, r, &params) {
		return
	}
	if len(params.Note) > 2400 {
		webError(w, 400, "Keep the note under 2400 bytes.")
		return
	}
	source, err := loadReplay(r, ws, r.URL.Query().Get("id"))
	if err != nil {
		webError(w, 422, err.Error())
		return
	}
	pos := params.Position
	if pos.Index < 0 || pos.Index >= len(source.Rows) || pos.Probe < 0 || pos.Probe >= len(source.Probes) || pos.Sheet < 0 || pos.Sheet >= len(source.Sheets) {
		webError(w, 400, "Choose a sample, measurement and sheet in this recording.")
		return
	}
	payload := replaySharePayload{Capture: source.replayCapture, Position: pos}
	if params.IncludeProject {
		payload.Project, err = workbench.CapturedProjectBytes(*source.metadata)
		if err != nil {
			webError(w, 422, err.Error())
			return
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		webError(w, 500, "Could not encode the replay.")
		return
	}
	viewer, err := assets.ReadFile("static/replay-view.mjs")
	if err != nil {
		webError(w, 500, "The replay viewer is unavailable.")
		return
	}
	// Go's JSON encoder escapes HTML-significant characters in captured text.
	// The only other executable content is the embedded viewer and boot code.
	// HTML parsers normalize CRLF before checking the script hash.
	script := strings.ReplaceAll(string(viewer), "\r\n", "\n")
	script = strings.Replace(script, "export function replayView(", "function replayView(", 1) + "\nconst share = " + string(data) + ";\n" + shareBoot
	hash := sha256.Sum256([]byte(script))
	var html bytes.Buffer
	err = shareTemplate.Execute(&html, struct {
		Script     template.JS
		ScriptHash string
		Note       string
		Project    bool
	}{template.JS(script), base64.StdEncoding.EncodeToString(hash[:]), strings.TrimSpace(params.Note), params.IncludeProject})
	if err != nil {
		webError(w, 500, "Could not create the replay file.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ELab-schematic-replay.html"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html.Bytes())
}
