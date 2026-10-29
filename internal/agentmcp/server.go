// Package agentmcp presents the unchanged workbench envelope through local MCP.
// It does not supply mutation guards/keys, retry changes, or own a remote app.
package agentmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/dshills/elab/transport"
	"github.com/dshills/elab/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MaxFrameBytes bounds MCP framing; the enclosed ELab request stays at 32 KiB.
const MaxFrameBytes = 2 * transport.MaxRequestBytes

// Dispatch must send one unchanged envelope and return a bounded JSON response.
type Dispatch func(context.Context, []byte) ([]byte, error)

// Local uses the same strict bounded admission as the existing stdio/socket adapters.
func Local(service *workbench.Service) Dispatch {
	return func(ctx context.Context, line []byte) ([]byte, error) {
		return json.Marshal(transport.DispatchLine(ctx, service, line, "agent.mcp"))
	}
}

// NewServer uses the official SDK for protocol negotiation and stdio lifecycle.
// Closing the MCP connection disposes a headless session in the caller; it must
// never implicitly dispatch app.close to an explicitly connected desktop host.
func NewServer(dispatch Dispatch) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "ELab", Version: "1"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
		Instructions: "Use elab_discover for the actual host catalog and command schemas, then elab_inspect for current state. elab_command accepts the exact ELab version-1 request envelope. Mutations require an observed revision guard and explicit retry key; document generation guards remain available. Never refresh guards or retry a refused/uncertain mutation automatically. Use analysis.list/page for bounded data. This is the same service used by ELab's UI. Headless state lasts for this process; --connect uses only the explicitly configured private host. No network AI provider is used by this server.",
	})
	var serial atomic.Uint64
	slots := make(chan struct{}, 8)
	execute := func(ctx context.Context, line []byte) (*mcp.CallToolResult, error) {
		if len(line) > transport.MaxRequestBytes {
			return failure("request-too-large", "ELab requests are limited to 32 KiB"), nil
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			return failure("adapter-busy", "Eight adapter calls are already in flight; this request was not dispatched."), nil
		}
		raw, err := dispatch(ctx, line)
		if err != nil {
			return failure("transport-error", "Connection failed; the mutation outcome may be unknown. Inspect/reconnect, and use the original retry key if retrying: "+err.Error()), nil
		}
		if len(raw) > transport.MaxResponseBytes {
			return failure("response-too-large", "Use analysis.list/page or another bounded query; a mutation may already have been admitted."), nil
		}
		var header struct {
			Version int                `json:"version"`
			Error   *workbench.Failure `json:"error"`
		}
		if err = json.Unmarshal(raw, &header); err != nil || header.Version != 1 {
			return failure("transport-error", "The host returned an invalid ELab response; do not assume an uncertain mutation failed."), nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}, StructuredContent: json.RawMessage(raw), IsError: header.Error != nil}, nil
	}
	readonly := func(name, description, command string, schema json.RawMessage) {
		server.AddTool(&mcp.Tool{Name: name, Description: description, InputSchema: schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(false)}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.Params.Arguments
			if len(args) == 0 {
				args = json.RawMessage(`{}`)
			}
			if command == "capabilities.list" {
				var q workbench.CapabilityQuery
				if err := workbench.DecodeStrict(args, &q); err != nil {
					return failure("invalid-params", err.Error()), nil
				}
			} else {
				var empty struct{}
				if err := workbench.DecodeStrict(args, &empty); err != nil {
					return failure("invalid-params", err.Error()), nil
				}
			}
			line, err := json.Marshal(workbench.Request{Version: 1, ID: fmt.Sprintf("mcp-query-%d", serial.Add(1)), Command: command, Params: args})
			if err != nil {
				return nil, err
			}
			return execute(ctx, line)
		})
	}
	readonly("elab_discover", "Discover all shared commands and limits; pass command for its detailed input schema.", "capabilities.list", json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","maxLength":128}},"additionalProperties":false}`))
	readonly("elab_inspect", "Inspect the current circuit, selection, jobs, command revision and document generation.", "project.inspect", json.RawMessage(`{"type":"object","additionalProperties":false}`))
	server.AddTool(&mcp.Tool{Name: "elab_command", Description: "Run any shared ELab command with its exact version-1 envelope. Mutations require explicit observed guards and a retry key. Commands may edit circuits, read/write files, export or print. No automatic retry or guard refresh.", InputSchema: json.RawMessage(`{"type":"object","properties":{"version":{"type":"integer","const":1},"id":{"type":"string","minLength":1,"maxLength":128},"command":{"type":"string","minLength":1,"maxLength":128},"params":{"type":"object"},"key":{"type":"string","minLength":1,"maxLength":128},"expected_revision":{"type":"integer","minimum":0},"expected_command_revision":{"type":"integer","minimum":0},"expected_document_generation":{"type":"integer","minimum":0}},"required":["version","id","command","params"],"additionalProperties":false}`)}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Preserve raw JSON, including duplicate/unknown fields, for shared strict
		// validation. Decoding to a map here would silently change that contract.
		return execute(ctx, req.Params.Arguments)
	})
	return server
}

func boolPtr(v bool) *bool { return &v }
func failure(code, message string) *mcp.CallToolResult {
	raw, _ := json.Marshal(workbench.Response{Version: 1, Error: &workbench.Failure{Code: code, Message: message}})
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}, StructuredContent: json.RawMessage(raw), IsError: true}
}
