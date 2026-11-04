package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshills/elab/internal/agentmcp"
	"github.com/dshills/elab/internal/webagent"
	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const workspaceAgentFile = "agent.endpoint.json"

func runWorkspaceMCP(ctx context.Context, descriptor string) error {
	client, err := webagent.New(descriptor)
	if err != nil {
		return err
	}
	defer client.Close()
	go func() { <-ctx.Done(); _ = os.Stdin.Close() }()
	err = agentmcp.NewServer(client.Send).Run(ctx, &mcp.StdioTransport{MaxLineLength: agentmcp.MaxFrameBytes})
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func (ws *workspace) agentStatus() map[string]any {
	return map[string]any{"enabled": ws.agentToken != "", "last_command": ws.agentLast}
}

// Called with files held; tokens are never included in browser responses.
func (ws *workspace) retireAgent() error {
	credential := ws.agentToken
	ws.agentToken, ws.agentLast = "", ""
	path := filepath.Join(ws.directory, workspaceAgentFile)
	data, err := readBounded(path, 16384)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var endpoint webagent.Endpoint
	if workbench.DecodeStrict(data, &endpoint) != nil || endpoint.Token != credential {
		return fmt.Errorf("agent descriptor changed; refusing to remove it")
	}
	if err = os.Remove(path); err != nil {
		return err
	}
	return nil
}

func (s *webServer) agentAccess(w http.ResponseWriter, r *http.Request, ws *workspace) {
	ws.files.Lock()
	defer ws.files.Unlock()
	if r.Method == http.MethodPost {
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") || decodeJSON(w, r, &input) != nil || input.Enabled == nil {
			webError(w, 400, "An explicit enabled boolean is required.")
			return
		}
		if *input.Enabled && ws.agentToken == "" {
			if s.host == nil {
				webError(w, 503, "The workspace host is not ready.")
				return
			}
			endpoint := webagent.Endpoint{Version: 1, Kind: webagent.Kind, URL: s.host.info.URL,
				Instance: s.instance, Workspace: filepath.Base(ws.directory), Token: token()}
			if err := endpoint.Validate(); err != nil {
				webError(w, 500, err.Error())
				return
			}
			path := filepath.Join(ws.directory, workspaceAgentFile)
			if old, err := readBounded(path, 16384); err == nil {
				var previous webagent.Endpoint
				if workbench.DecodeStrict(old, &previous) != nil || previous.Kind != webagent.Kind || previous.Workspace != endpoint.Workspace {
					webError(w, 409, "The descriptor path contains an unrelated file.")
					return
				}
			} else if !os.IsNotExist(err) {
				webError(w, 409, "The private descriptor path is unavailable.")
				return
			}
			if err := writeJSON(path, endpoint); err != nil {
				webError(w, 500, err.Error())
				return
			}
			ws.agentToken, ws.agentLast = endpoint.Token, ""
		} else if !*input.Enabled && ws.agentToken != "" {
			if err := ws.retireAgent(); err != nil {
				webError(w, 409, "Agent access is disabled, but descriptor cleanup failed: "+err.Error())
				return
			}
		}
	} else if r.Method != http.MethodGet {
		webError(w, 405, "GET or POST required")
		return
	}
	status := ws.agentStatus()
	if ws.agentToken != "" {
		executable, err := os.Executable()
		if err != nil {
			webError(w, 500, "Could not locate the installed MCP bridge.")
			return
		}
		status["configuration"] = map[string]any{"mcpServers": map[string]any{"elab": map[string]any{
			"command": executable, "args": []string{"--mcp-workspace", filepath.Join(ws.directory, workspaceAgentFile)},
		}}}
	}
	jsonReply(w, 200, status)
}

func (s *webServer) agentCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		webError(w, 405, "POST JSON required")
		return
	}
	id := r.Header.Get("X-ELab-Workspace")
	if r.Header.Get("X-ELab-Instance") != s.instance || !managedID.MatchString(id) {
		webError(w, 404, "Workspace connection unavailable")
		return
	}
	s.mu.Lock()
	ws := s.sessions[id]
	if ws != nil {
		ws.touched = time.Now()
	}
	s.mu.Unlock()
	if ws == nil {
		webError(w, 404, "Workspace connection unavailable")
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, transport.MaxRequestBytes))
	var request workbench.Request
	if err != nil || workbench.DecodeStrict(data, &request) != nil {
		webError(w, 400, "One bounded, strictly valid ELab envelope is required.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	authorized := func() bool {
		return ws.agentToken != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-ELab-Agent-Token")), []byte(ws.agentToken)) == 1
	}
	response := s.dispatchWorkspace(ctx, ws, request, "agent.web", authorized)
	encoded, err := json.Marshal(response)
	if err != nil || len(encoded) > transport.MaxResponseBytes {
		jsonReply(w, 200, workbench.Response{Version: 1, ID: request.ID, Error: &workbench.Failure{
			Code: "response-too-large", Message: "Use analysis.list/page for bounded results; a mutation may already have been admitted.",
		}})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(encoded)
}
