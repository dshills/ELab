# Vellum adoption for ELab

**Date:** October 13, 2026  
**Decision:** Use Vellum for ELab's desktop UI, as requested by the product owner.  
**Status:** Framework selected; initial source review recorded below. The first LED application slice is now implemented; see the implementation update and linked validation record below.

Build an independent Go application using Vellum's public APIs and native Cogent driver. Keep the hobbyist draw–observe–adjust–measure–recover–save workflow and complete agent operability defined in the [PRD](07-product-requirements.md). Vellum supplies UI infrastructure; ELab owns the circuit editor, electrical model, solver integration, and semantic application contract.

## Inspected baseline

The local checkout at `/Users/dshills/Development/projects/Vellum` was inspected at commit `9a34670e989517ab0bb776eeb3e0638d5a4c4959`, with no changes reported by `git status --short`. Its release metadata identifies `v0.1.0-alpha.1`, module `github.com/dshills/vellum`, and MIT licensing for Vellum's own code. Its root module declares Go `1.25.6`. Native dependencies live in the separate `github.com/dshills/vellum/drivers/cogent` module.

The framework describes itself as experimental. macOS is experimental, Windows and Linux X11 are unqualified, and native Wayland is unsupported in the inspected release metadata. The third-party distribution audit remains pending. These are source/documentation observations, not results of running Vellum or ELab in this review. Pin an immutable SDK and its prepared dependency manifest before the first implementation baseline; the alpha version label alone is insufficient to identify these inspected bytes.

## Responsibilities and fit

| ELab area | Available Vellum foundation | ELab responsibility |
|---|---|---|
| Application shell | `app`, `ui`, stable widget IDs, controlled Go state, native controls | Searchable parts, contextual inspector, simulation status, menus, keyboard behavior, and simple defaults |
| Workspace | `workbench` docking and separate layout state | One circuit-centered workspace with an optional scope; keep document ownership independent of panels |
| Schematic | `canvas` meshes, polyline strokes, rectangle index; native `cogent.MeshCanvas` | Symbols, text labels, snapping, terminal targeting, wires/junctions, selection, placement previews, and electrical connectivity |
| Scope | Immutable `canvas.Waveform` with min/max envelope queries | Time/unit mapping, bounded rolling samples, four traces, cursors, exact measurements, discontinuities, and export metadata |
| Commands | Stable IDs, availability checks, menus/shortcuts through `command.Registry` | Typed semantic commands/queries and schemas shared with all agents; route direct widget changes through these operations too |
| Background work | `tasks` with cancellation, bounded capacity, owner-thread result commits | Solver adapter, run IDs, electrical revision checks, coalesced edits, simulation clock, errors, and progress |
| Undo and files | `history`, `document`, typed codecs, saved-revision tracking | Atomic circuit transactions, actor attribution, self-contained project format, recovery, result snapshots, and nonvisual file decisions |
| Agent connection | Opt-in `automation` with typed actions/queries, revision checks, local macOS socket | Complete capability coverage, schemas, idempotency, jobs, events, batches, structured decisions, and headless sessions |
| Accessibility | Semantic controls and macOS accessibility infrastructure | Navigable parts/terminals/nets and text readings beyond canvas pixels; actual keyboard and screen-reader acceptance |

These facilities are useful starting points, not completed ELab features. In particular, the drawing contract does not supply circuit connectivity or text shaping, and its rectangle index is not precise wire/terminal hit testing. Native pointer routing must prove that wire drawing, selection, component dragging, panning, and zooming coexist without conflicting gestures.

## Shared application core

Use one application service layer for the desktop UI, an in-process agent client, and the external agent adapter. It owns validation, canonical units, stable IDs, revisions, undo, jobs, files, and the live simulation state machine. Vellum command handlers and controlled-widget callbacks translate user intent into this layer; neither may mutate circuit data through a separate path.

Keep the circuit model and solver adapter independent of Cogent, native widgets, and GPU resources. A headless entry point must create/open a project, edit it, simulate it, measure results, and save/export using the same services without opening a window. UI-specific state such as selection, viewport, and scope configuration remains structured and agent-accessible even when no native widget is mounted. Persist/export schematic geometry directly so SVG creation does not depend on GPU readback.

Serialize model mutations through a single owner. In interactive mode this must respect Vellum's application ownership contract. Workers receive immutable snapshots and return results tagged with project, electrical revision, and run identity. Admit a result only when it still belongs to the active request. Treat presentation changes separately from electrical changes, as required by the PRD.

Vellum's document session requires a registered window. Therefore, use a window-independent ELab persistence service underneath any UI session integration. Do not make the native file chooser or window lifecycle a prerequisite for headless saving and opening. Retain equivalent validation, atomic-write behavior, and saved-revision semantics in both modes.

## Live simulation integration

Committed electrical edits validate and start or restart simulation according to the PRD; unfinished gestures remain previews. A slider gesture coalesces obsolete solve requests and creates one undo action. Switch actuation is an input event at an accepted solver step, preserving stored capacitor state as specified.

Run numerical work outside the UI owner. Publish bounded batches of samples and status rather than requesting a UI redraw on every numerical timestep. Rendering cadence, numerical timestep, and simulation speed are separate controls. A slow solver must leave edit, cancel, undo, and save responsive.

Keep waveform values and timestamps as authoritative data. Vellum's envelope reduction is a display aid; cursor values, interval extrema, and CSV output use the underlying recorded samples with explicit sampling/interpolation semantics. Its immutable waveform construction is O(N), so benchmark a bounded/chunked publication strategy rather than rebuilding an unlimited trace every frame. Preserve stale markers and run boundaries during worker replacement.

## Agent interface gaps to close

Vellum's existing automation is a candidate macOS transport, not a complete implementation of AGENT-01 through AGENT-06. The inspected contract has one request/response per connection, a frozen registry of at most 256 bindings, 32 KiB request and 64 KiB response limits, eight combined client/owner-work credits, and deadlines of at most one minute. Actions use revision preconditions. Request IDs correlate responses but do **not** deduplicate mutations. These limits should shape the adapter, not reduce ELab's agent requirements.

| Gap | Required ELab behavior |
|---|---|
| Capability discovery | Publish a versioned, paginated catalog with input/output schemas, units, defaults, preconditions, errors, and examples. Register semantic operation types, not a separate binding for every circuit object |
| Reliable retries | Add idempotency keys with bounded retention and documented expiry. Resolve an already accepted retry before applying a stale-revision check to it; reject reuse with different arguments. After an uncertain outcome, support inspection of the original operation |
| Long operations | Return a job ID promptly; expose progress, completion, cancellation, input revision, and artifacts through separate bounded queries. A socket timeout or disconnect is not proof of rollback |
| Change events | Supply sequenced events with cursor-based retrieval or another supported subscription mechanism. Bound retention and provide snapshot resynchronization when the cursor expires |
| Large results | Page waveform/schema/state queries and return explicit artifact references for exports. Never silently truncate data to fit transport limits |
| Concurrent state | Separate relevant document/view/run identities. Map the transport revision check to an admission revision while preserving domain-specific preconditions; test live progress alongside edits so continuous updates do not starve agent work |
| Transactions | Validate a batch against its final state, commit once, restart simulation once, and record one attributed undo action. A transport audit ring is not the application transaction history |
| Diagnostics and decisions | Return stable domain error/decision DTOs with affected object IDs and recovery choices. Do not depend on callback error text, which the Vellum transport deliberately does not transmit |
| Lifecycle and headless use | Document authorized launch/attach/discovery/shutdown and run without native windows. Prove owner-loop draining and file operations in the headless entry point |

Vellum's workbench example exposes only a small demonstration allowlist; copying that list would not meet ELab's requirements. Every user function, including settings, view controls, recovery, and export, must enter the ELab capability catalog. UI, built-in, and external clients run the same conformance suite.

Gate A should evaluate reuse of the existing local socket before adding another transport. MCP or another agent adapter may wrap the same application contract later; no particular agent vendor or protocol is selected here. Keep endpoint access explicit and pass credentials only to authorized clients. No account or remote model connection is needed to use the local API.

## Dependency and packaging approach

Consume Vellum as an independent application, keeping ELab code outside the framework checkout. The inspected starter specifies four local replacements: Vellum, the Cogent driver, prepared Cogent source, and prepared go-text/typesetting source. Go dependency replacements do not propagate from dependency modules. Using unpatched upstream dependencies is not an equivalent integration.

For initial development, the adjacent Vellum checkout can supply these paths. Record its commit plus the prepared dependency/patch manifest, and avoid treating machine-specific absolute paths as a release strategy. For reproducible builds, consume a pinned relocatable private SDK with its integrity manifest. Preserve required Vellum/template and dependency notices when copying or distributing source.

Vellum's starter generator requires a new destination and refuses an existing directory. ELab already contains research, so any later bootstrap should generate into a separate staging directory and deliberately integrate the application files. The research directory remains part of the project.

First qualify a macOS development build, then an application bundle on a clean supported installation. Framework documentation and successful compilation alone cannot establish input, accessibility, GPU behavior, numerical correctness, or redistribution readiness. Windows/Linux remain future target decisions with their own runtime acceptance.

## First implementation gate

1. Pin the SDK and prepared dependencies, establish the independent application and headless entry points, and record reproducible build commands.
2. Implement the shared typed command/query catalog and a minimal circuit model before wiring controls to mutations. Demonstrate one edit and undo through the UI, in-process client, and external client with equivalent state.
3. Build the LED circuit from blank in the Vellum canvas. Prove placement, readable labels, wire/terminal targeting, selection, inspector entry, pan/zoom, keyboard alternatives, and an accessible object/terminal representation.
4. Connect the chosen solver through cancellable workers. Complete LED, RC, and NPN cases plus broken circuits, stale-result rejection, pause/restart, and rapid value changes. Measure against the PRD targets; solver selection remains open.
5. Save/reopen/export through both visible and headless clients. Test explicit paths and structured decisions, retries after lost responses, conflicts, failed batches, cancellation, and disconnect/resynchronization.
6. Record canvas/input, numerical, latency, accessibility, dependency, and packaging gaps. Resolve them through application work or explicit framework changes while retaining the chosen Vellum direction and full product contract.

This gate produces the first working ELab slice. The current document records the selection and proposed integration; it does not claim that the gate has passed.

## Local source references

**Implementation update, October 13, 2026:** The [first LED implementation slice](../docs/implementation.md) now supplies the independent Vellum app, live series-LED DC calculation, shared UI/agent dispatcher, headless client, and local socket access. The initial framework inspection above remains historical evidence; the implementation record describes subsequent runtime validation and the work still needed to finish Gate A.

Paths below refer to the inspected Vellum checkout and can change with subsequent development. The commit above identifies the reviewed baseline.

- [Framework overview](../../Vellum/README.md), [release metadata](../../Vellum/release/version.json), and [root module](../../Vellum/go.mod).
- [Application tutorial](../../Vellum/docs/tutorials/building-an-app.md), [SDK contract](../../Vellum/specs/sdk.md), and [independent example dependencies](../../Vellum/examples/hello/go.mod).
- [Drawing contract](../../Vellum/specs/drawing.md), [command implementation](../../Vellum/command/command.go), [worker implementation](../../Vellum/tasks/tasks.go), and [document session implementation](../../Vellum/document/session.go).
- [Automation contract](../../Vellum/specs/automation.md) and [server implementation](../../Vellum/automation/server.go).
- [Platform qualification](../../Vellum/specs/platform-support.md) and [release contract](../../Vellum/specs/release.md).
