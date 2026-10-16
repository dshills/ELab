// Package conformance runs release reference workflows through public commands.
package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dshills/elab/workbench"
)

// Client owns one newly created application session. Close releases that session.
// Implementations must not attach to or close pre-existing user sessions.
type Client interface {
	Dispatch(context.Context, workbench.Request) (workbench.Response, error)
	Close() error
}

// Factory creates an empty owned session, using its directory for process logs.
type Factory func(context.Context, string) (Client, error)

type localClient struct{ service *workbench.Service }

// InProcess adapts the public dispatcher without installing fixtures or state.
func InProcess(_ context.Context, _ string) (Client, error) {
	return &localClient{workbench.New()}, nil
}
func (c *localClient) Dispatch(ctx context.Context, r workbench.Request) (workbench.Response, error) {
	return c.service.Dispatch(ctx, r, "conformance.in-process"), nil
}
func (c *localClient) Close() error { c.service.Close(); return nil }

// Call records public intent with artifact paths normalized for comparison.
type Call struct {
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params"`
}

type session struct {
	client    Client
	ctx       context.Context
	sequence  int
	commands  []string
	calls     []Call
	directory string
}

func (s *session) ask(command string, p any, mutation bool, target any) error {
	s.sequence++
	if p == nil {
		p = struct{}{}
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	r := workbench.Request{Version: workbench.ProtocolVersion, ID: fmt.Sprintf("conformance-%d", s.sequence), Command: command, Params: raw}
	response, err := s.dispatch(r, mutation)
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}
	s.commands = append(s.commands, command)
	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return err
	}
	if path, ok := normalized["path"].(string); ok && strings.HasPrefix(path, s.directory+string(filepath.Separator)) {
		normalized["path"] = "<artifacts>/" + strings.TrimPrefix(path, s.directory+string(filepath.Separator))
	}
	callParams, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	s.calls = append(s.calls, Call{command, callParams})
	if response.Error != nil {
		return fmt.Errorf("%s: %s: %s", command, response.Error.Code, response.Error.Message)
	}
	encoded, err := json.Marshal(response.Result)
	if err != nil {
		return err
	}
	if target != nil {
		if err := json.Unmarshal(encoded, target); err != nil {
			return fmt.Errorf("%s result: %w", command, err)
		}
	}
	return nil
}

func (s *session) dispatch(r workbench.Request, mutation bool) (workbench.Response, error) {
	dispatch := func(request workbench.Request) (workbench.Response, error) {
		response, err := s.client.Dispatch(s.ctx, request)
		if err == nil && (response.Version != request.Version || response.ID != request.ID) {
			err = fmt.Errorf("response envelope identity differs")
		}
		return response, err
	}
	if !mutation {
		return dispatch(r)
	}
	var original workbench.Snapshot
	for attempt := 0; attempt < 8; attempt++ {
		// Owned native sessions also report layout through shared commands.
		// Refresh admission guards without adding presentation-dependent calls
		// to the compared logical workflow. Retry only explicit conflict refusals.
		query := workbench.Request{Version: r.Version, ID: fmt.Sprintf("%s-guard-%d", r.ID, attempt), Command: "project.inspect", Params: json.RawMessage(`{}`)}
		response, err := dispatch(query)
		if err != nil || response.Error != nil {
			return response, err
		}
		encoded, err := json.Marshal(response.Result)
		if err != nil {
			return response, err
		}
		var state workbench.Snapshot
		if err := json.Unmarshal(encoded, &state); err != nil {
			return response, err
		}
		if attempt == 0 {
			original = state
		} else if state.Document.Generation != original.Document.Generation || state.ElectricalRevision != original.ElectricalRevision {
			return workbench.Response{}, fmt.Errorf("document or electrical state changed during admission retry")
		}
		request := r
		request.ID = fmt.Sprintf("%s-attempt-%d", r.ID, attempt)
		request.Key = request.ID
		request.ExpectedCommandRevision = &state.CommandRevision
		request.ExpectedDocumentGeneration = &state.Document.Generation
		response, err = dispatch(request)
		if err != nil || response.Error == nil || response.Error.Code != "conflict" {
			return response, err
		}
	}
	return workbench.Response{}, fmt.Errorf("admission conflicts exceeded eight attempts")
}
func (s *session) state(command string, p any, mutation bool) (workbench.Snapshot, error) {
	var state workbench.Snapshot
	err := s.ask(command, p, mutation, &state)
	return state, err
}
