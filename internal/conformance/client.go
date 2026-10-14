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
	revision  uint64
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
	if mutation {
		rev := s.revision
		r.ExpectedCommandRevision = &rev
		r.Key = r.ID
	}
	response, err := s.client.Dispatch(s.ctx, r)
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
	if response.Version != r.Version || response.ID != r.ID {
		return fmt.Errorf("%s: response envelope identity differs", command)
	}
	if response.Error != nil {
		return fmt.Errorf("%s: %s: %s", command, response.Error.Code, response.Error.Message)
	}
	encoded, err := json.Marshal(response.Result)
	if err != nil {
		return err
	}
	// Query job metadata contains the original captured revision, not the current
	// command revision. Only complete snapshots refresh admission preconditions.
	var state struct {
		Components json.RawMessage `json:"components"`
		Document   json.RawMessage `json:"document"`
		Revision   *uint64         `json:"command_revision"`
	}
	if json.Unmarshal(encoded, &state) == nil && state.Components != nil && state.Document != nil && state.Revision != nil {
		s.revision = *state.Revision
	}
	if target != nil {
		if err := json.Unmarshal(encoded, target); err != nil {
			return fmt.Errorf("%s result: %w", command, err)
		}
	}
	return nil
}
func (s *session) state(command string, p any, mutation bool) (workbench.Snapshot, error) {
	var state workbench.Snapshot
	err := s.ask(command, p, mutation, &state)
	return state, err
}
