# Connect an AI agent to ELab

ELab supports standard local MCP (`elab-agent --mcp`). It uses the same service,
validation, commands, measurements and revision guards as the web host.
There is no AI provider key or cloud connection to configure in ELab. Your MCP
client supplies its own model and tool-authorization settings.

## Headless: one configuration

Build a durable agent executable from this checkout:

```sh
go build -o build/elab-agent ./cmd/elab-agent
./build/elab-agent --mcp-config
```

Add the printed `elab` entry to a client that accepts `mcpServers` configuration. It
contains the absolute executable path and `args: ["--mcp"]`; paths with spaces
are ordinary JSON strings. Clients using another configuration format use those
same command/arguments. Generate this from the executable's final location, not
from `go run` or a folder you will remove. ELab prints the configuration and does
not edit any client settings. Restart/reconnect the client after configuring it.

Each headless server process starts one blank circuit. It keeps that state across
tool calls. Closing its MCP connection ends the owned process; saving a project
uses the ordinary project commands. Persistence is opt-in through the existing
`--recovery-directory` and `--appearance-file` flags.

## Private headless socket hosts

Start a shared headless session using an explicit **new** private descriptor path:

```sh
./build/elab-agent --listen /absolute/path/session.endpoint.json
./build/elab-agent --mcp-config --connect /absolute/path/session.endpoint.json
```

Use the second command's configuration in the client. The descriptor is a local
credential; the configuration contains its path, never its token. A missing or
failed connection never falls back to a new session. After the host restarts,
restart/reconnect the MCP bridge. Closing the bridge does not close its host.
Configure persistence on the host, not a forwarding client.

The normal raw JSON-line mode and private socket mode remain available. Browser
workspaces are separate HTTP sessions: `--connect` does not attach to the web app,
and the web launcher's private `.host.json` control file is not an agent endpoint.
The former desktop UI's `--agent-endpoint` instructions apply only to the
[archived native source](native-archive.md).

## Three tools, the complete command contract

| Tool | Use |
| --- | --- |
| `elab_discover {}` | Read the actual host's full command catalog, limits and guard metadata. |
| `elab_discover {"command":"component.set"}` | Get one command's detailed schema and completion/error guidance. |
| `elab_inspect {}` | Read circuit state, current command revision, document generation and jobs. |
| `elab_command {…}` | Dispatch an unchanged version-1 ELab request envelope. All shared commands remain available. |

Results include the same JSON response as structured data and a text fallback.
Service refusals set MCP `isError: true` while retaining ELab's error/help fields.
Unknown or duplicate fields are not normalized away. The generic command tool
can perform circuit edits, file operations, exports and printing according to
the existing command contract; it is conservatively described as mutating.

For an edit, first inspect the current state and discover its command schema.
Then supply an explicit new retry key and the guards from that observation:

```json
{
  "version": 1,
  "id": "set-limit-1",
  "command": "component.set",
  "params": {"id":"R1","value":"2k"},
  "expected_command_revision": 12,
  "expected_document_generation": 3,
  "key": "my-explicit-edit-identity"
}
```

The numbers are illustrative, not reusable defaults. The bridge never supplies
missing guards/keys or refreshes/retries a refused edit. On conflict, inspect and
make a new decision. On a transport failure, the outcome may be unknown: reconnect
and inspect, then retain the original key if retrying that same request. Retry
responses may describe the original result and must not be mistaken for the
latest state. Use a fresh inspect to observe current state.

ELab envelopes retain their 32 KiB request and 8 MiB response bounds. MCP framing
has a separate 64 KiB input bound; it does not enlarge command parameters. At most
eight adapter dispatches run concurrently; excess calls are explicitly refused
before dispatch. `analysis.list` and `analysis.page` provide bounded reads of
recordings, sweeps, spectra and tolerance results; discover their row limits and
source/document guards rather than requesting every point in a large snapshot.

## Reproduce a complete agent experiment

These demos create only their own blank sessions; neither accepts an arbitrary
live endpoint or opens an owner project:

```sh
python3 -B scripts/mcp-demo.py --agent /path/to/elab-agent --artifacts build/mcp-demo
python3 -B scripts/mcp-demo.py --agent /path/to/elab-agent --socket --artifacts build/mcp-demo-socket
```

Output directories must be new. The generated config is exercised from another
working directory. The demo places/wires an LED circuit, checks its current,
records RC responses at 10 kΩ and 20 kΩ, reads a five-row page and compares the
10 ms voltages against the RC equations. Each planned edit inspects first; a
refusal is reported rather than silently resubmitted. Owned bridges/hosts close
normally. These are deterministic protocol/electrical checks, not a claim that a
particular model completes the task without help.

The server uses the pinned [official Go MCP SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0)
for negotiation, framing, cancellation and connection lifecycle. Tests exercise
the 2025-11-25 handshake and 2026-07-28 protocol through the reference SDK client;
the demo uses the former. Protocol behavior follows the official
[lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)
and [tool-result](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)
contracts. Pinned agent dependency notices are retained in `third_party/agent-mcp`;
web packages carry the notices for their own linked dependencies.
