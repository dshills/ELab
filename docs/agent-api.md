# ELab agent protocol, version 1

All semantic commands use `workbench.Service.Dispatch`. Vellum widget callbacks, keyboard commands, canvas gestures, built-in Go clients, headless stdio, and the external socket client share this entry point. An agent does not need screenshots or simulated clicks to operate the implemented circuit slice.

## Request and response

```json
{"version":1,"id":"inspect-1","command":"project.inspect","params":{}}
```

`version`, `id`, `command`, and object `params` are required. IDs are 1–128 bytes. Mutations also require a 1–128-byte `key` and at least one of `expected_revision` or `expected_command_revision`. JSON uses canonical field names; unknown fields and duplicate keys are rejected. One request is bounded to 32 KiB.

Responses include `version`, the request `id`, the session `revision`, and either `result` or structured `error` with `code`/`message`. Successful mutation results are full snapshots. Read `capabilities.list` for descriptions and input/output schemas. `project.inspect` reports components and terminals, wires, exact values/units, selection, viewport, model parameters, diagnostics, undo availability, last actor, and simulation state/results.

Socket responses are bounded to 1 MiB; an oversized result returns `response-too-large` explicitly.

Revision is a session admission version and changes on accepted mutations and solver result publication. `electrical_revision` changes on circuit configuration edits. Live switch/wiper input events preserve it and advance `actuation_revision`; that revision also guards Verify source association. `run_id` identifies calculation attempts. A selection or view change preserves the electrical revision and run. Live transient samples also change the session revision; `command_revision` changes only for admitted commands, allowing clients to distinguish edits from clock updates. The full-state guard requires the latest session revision. The optional command guard instead matches `command_revision`, which changes on every admitted mutation but remains stable during solver publication. It therefore detects intervening actors without clock-only conflicts. If both guards are supplied, both must match. Conflict/retry rules remain explicit. A `conflict` response is an explicit refusal; re-inspect before deciding how to proceed. `capabilities.list.mutation_preconditions` advertises the available guard fields and meanings; older versions only advertise the full-state guard. Native controls use the command guard and do not automatically resubmit after an intervening command conflict.

## Draw an LED circuit from blank

New sessions start blank. Query `parts.list` for defaults, supported kinds, models, and terminal names. Use `component.place` with a kind and position; its mutation snapshot identifies the selected new ID. Optional explicit IDs start with an ASCII letter and contain letters, digits, `_` or `-` (1–64 characters). Auto IDs use V/R/D/GND prefixes and monotonically avoid previously used identities.

Example command parameters, with a fresh observed revision and unique retry key on **each** mutation:

| Command | Parameters |
|---|---|
| `component.place` | `{"kind":"dc-source","id":"Supply","position":{"x":-200,"y":0}}` |
| `component.place` | `{"kind":"resistor","id":"Limit","position":{"x":0,"y":-110}}` |
| `component.place` | `{"kind":"led","id":"Lamp","position":{"x":200,"y":0}}` |
| `component.place` | `{"kind":"ground","id":"Ground","position":{"x":0,"y":110}}` |
| `wire.connect` | `{"from":"Supply.positive","to":"Limit.1"}` |
| `wire.connect` | `{"from":"Limit.2","to":"Lamp.anode"}` |
| `wire.connect` | `{"from":"Lamp.cathode","to":"Ground.reference"}` |
| `wire.connect` | `{"from":"Ground.reference","to":"Supply.negative"}` |
| `simulation.wait` | `{}` (query; no revision/key needed) |

Defaults are 5 V and 1 kΩ. Expect approximately 0.0029167966 A once the loop is complete. `component.set` with `{"id":"Limit","value":"2k"}` changes the live current to approximately 0.0014767235 A. See `scripts/drawing-demo.py` for an executable workflow with revisions and checks.

New, example loading, and close now refuse dirty documents unless `discard_changes:true` is provided. Save first when keeping the previous work.

Positions snap to a 10-unit grid within ±10000; up to 100 drawing objects and 300 wires are admitted; helpers do not count toward the 64-simulated-part envelope. A wire explicitly joins two existing named terminals. Crossings, touching geometry, and nearby dots never create connectivity. Ground terminals share the reference net. Current analysis accepts one source, one resistor, one forward LED, and one or more grounds in a series loop, including LED-before-resistor order and either resistor pin direction.

`component.move` preserves connections; optional `previous_position` refuses a move if another actor already moved that part. `component.rotate` rotates 90° clockwise while preserving pin identities. `object.delete` accepts component or wire IDs. `project.clear` starts blank and can be undone; `example.reset` loads the prepared fixture as one edit. The legacy `circuit.connection.set` alias only addresses R1.2 ↔ D1.anode; use wire IDs and `object.delete` for new drawings.

`drawing.tool.set` exposes select/place/wire/pan, `drawing.position.set` exposes exact placement, `wire.begin` starts the visible terminal gesture, `wire.terminals.set` controls named choices, `editor.value.set` controls the inspector draft, and `drawing.cancel` returns to selection. These UI operations are also available to agents. Agents can directly commit placement, connection, or values without staging the UI. `component.select` accepts parts or wires. Snapshot `drawing`, `nets`, diagnostics, and selection describe the same visible state.

`component.set` accepts values as strings with common resistance suffixes, for example `1k`, `2.2kΩ`, or `1000 ohm`; voltage may include `V`. DC sources accept 0–12 V; resistors accept 10–100000 Ω. All agents use the same validation as the UI.

`simulation.pause` stops automatic calculation. Electrical edits while paused remain paused and mark the previous result stale. `simulation.resume` calculates the current circuit. `simulation.restart` recalculates; it preserves the paused condition when already paused. `simulation.wait` completes immediately for paused/incomplete/error/closed states, so inspect the returned status before relying on a result. The LED engine provides a DC operating point. The separate ideal RC engine provides the live transient described below.

## NPN LED switch workflow

`parts.list` includes `npn` with named `base`, `collector`, and `emitter` pins. Place it through `component.place`; connect wires and edit source/resistor values through the existing commands. `example.npn` loads the editable 5 V supply / separate 5 V drive / 10 kΩ base / 1 kΩ LED load example, with `discard_changes:true` required if the current document is dirty.

Wire input → base resistor → base; emitter and source negatives → ground; supply → load resistor / forward LED → collector. One shared DC source is also supported, and the LED is optional for a resistor-only load. Either resistor orientation and LED-before-resistor order work. Pulse base drive is now supported as described below; the single-switch engines retain their topology limits. Capacitor networks with up to two NPNs use the mixed engine described below.

The UI drive buttons call `component.set` on the discovered input source with `0 V`, `0.8 V`, or `5 V`. These are undoable, saved electrical edits. Find the source ID in a valid `snapshot.npn.input_id`, or derive it from the graph while a first solve is pending. `simulation.wait` waits for DC calculation. Pause/edit/Resume follows the same stale-reading rules as LED DC.

`snapshot.npn` contains engine/model identity, device IDs, `base_current_a`, `collector_current_a`, `emitter_current_a`, `vbe_v`, `vce_v`, `state`, `explanation`, `nodes_v`, powers, KCL/KVL residuals, run/electrical revision, and stale status. It is separate from LED `result` and RC `transient`. Engine `elab.npn-switch-dc.v1` uses generic model `elab.generic-npn.v1` (IS 1e−14 A, BF 100, BR 1, 27°C); no device heating, breakdown, Early effect, capacitance, or switching delay. See [NPN workflow and validation](npn-switch.md) and `scripts/npn-demo.py`.

## NPN pulse workflow

Use `pulse-source` for a separate NPN base drive and a `dc-source` collector supply, or load `example.npn-pulse` (Blink). Use `simulation.transient.set` for the complete pulse/sample/playback settings and `simulation.input.set` with `auto`, `high`, or `low`. RC retains `auto`, `charge`, and `discharge`. Pause/Step/Resume share the existing transient clock semantics. An override while paused is queued; changing source amplitude is a saved/undoable electrical edit.

`snapshot.transient.engine` is `elab.npn-switch-quasistatic.v1`. Time samples expose drive `input_v`, VCE `voltage_v`, collector/LED `current_a`, and NPN state/base readings. `snapshot.npn` carries the matching full operating point. Pulses switch the memoryless generic model instantaneously; there is no physical device delay or charge storage. Files preserve the existing model and transient settings, then reopen at time zero in Auto. See [NPN transient contract, data fields, limits and validation](npn-transient.md) and `scripts/npn-pulse-demo.py`.

## Mixed RC/NPN workflow

`example.flasher` loads the DC-powered two-transistor, two-capacitor LED oscillator. `capacitor.initial.set` sets a saved, undoable startup voltage (pin 1 minus pin 2, ±12 V), or value `default` restores the experiment default. Drawing, values, pulse settings, pause/step/resume/restart and file commands are shared. Mixed Auto/High/Low controls actuate `transient.source_id` while preserving capacitor state; the DC example labels them Power on/off.

Snapshot `mixed` exposes all node/device/capacitor/branch readings and conservation residuals; `transient` contains its primary rolling voltage/current/capacitor trace. The unreleased engine `elab.mixed-mna-sdirk2.v1` couples ideal capacitors to the generic nonlinear junction model with [adaptive SDIRK2](capacitor-integration.md); imported backward-Euler captures retain their original identity. Limits, signed quantities, initialization, transactional failures and complete data semantics are in [mixed transient documentation](mixed-transient.md). Repeat the workflow with `scripts/mixed-demo.py`.

## RC transient workflow

`parts.list` includes `capacitor` (terminals `1` / `2`, default 1 µF, `ideal-capacitor.v1`) and `pulse-source` (positive / negative, default 5 V, `ideal-square.v1`). Both use the existing placement, wire, value, move, rotate, and delete commands. Capacitance accepts p/n/u/µ/μ/m prefixes and F, within 1 pF–1 F. Pulse amplitude uses the source's 0–12 V value; its low level is 0 V.

Connect one source through one resistor to the capacitor output, connect the other capacitor pin to ground, and ground source negative. Either resistor or capacitor pin order is supported. Measurement voltage is the output node relative to ground; positive current flows through the resistor into that output. Other supported LED/RC and multi-capacitor circuits use the bounded network/mixed engine; inspect diagnostics and `snapshot.limits` for topology/capacity support.

| Command | Parameters | Behavior |
|---|---|---|
| `example.rc` | `{"discard_changes":true}` | Load editable 5 V pulse / 10 kΩ / 1 µF example; omit discard when the current document is clean. |
| `simulation.transient.set` | `{"step_s":0.001,"high_s":0.05,"low_s":0.05,"initial_voltage_v":0,"seconds_per_wall_second":0.02}` | Set the complete experiment configuration; undoable, saved, and restarts an active electrical run. |
| `simulation.input.set` | `{"mode":"charge"}` | Hold source at its configured high value, preserving capacitor voltage. `discharge` drives 0 V; `auto` follows pulse phase, or holds a DC source high. |
| `simulation.pause` | `{}` | Freeze simulated time and retain the current sample. |
| `simulation.step` | `{"duration_s":0.01}` | Advance a valid paused RC run by 10 ms; preserve paused status. |
| `simulation.resume` | `{}` | Continue the same valid run, or restart stale electrical state from configured initial voltage. |
| `simulation.restart` | `{}` | Start a fresh time-zero run with the configured initial voltage and current input mode; remain paused if already paused. |

All entries are mutations with revision/key admission. For a deterministic reference, load the example, Pause, Restart, then Step 0.01 s: the capacitor voltage is 3.160602794142788 V. `simulation.wait` waits for DC computation or reads ready RC state; it does not wait for a future RC time. Use paused Step to obtain exact requested sample times. A queued input while paused does not change time/voltage until Step or Resume applies it.

Snapshot `transient` includes engine `elab.ideal-rc-exact.v1`, run/electrical revision, source/resistor/capacitor IDs, output net, τ in seconds, settings, input/pending mode, current sample, retained samples, events, and stale state. Every sample has `time_s`, `input_v`, `voltage_v`, signed `current_a`, and `energy_j`. The live ring retains at most 512 samples including exact pulse-edge points, plus at most 64 events. Its retained interval is the first sample time through the current sample time. No display decimation changes these samples.

Sample steps are 1 µs–1 s. Each pulse duration is at least the configured step and at most 10 s. Initial voltage is 0–12 V. Playback speed is 0.001–1 simulated s/wall s, at most 1000 samples/wall s. Step duration must be positive, advance floating-point time, and be at most 1 s / 1000 sample steps. Real-time playback is best-effort pacing; exact paused stepping is independent of wall time.

Electrical edits invalidate the prior trace and start a new run from configured initial voltage when live; paused edits stay paused/stale until Resume or Restart. Geometry and view changes preserve time and energy. Input actuation is runtime state and does not dirty the project. Save preserves graph/configuration, not waveform history or live input override; reopening starts a new Auto run from saved initial voltage. Legacy `snapshot.model` remains LED library metadata; RC identity is in component model IDs and `transient.engine`.

`view.fit` now accepts optional `width_px` / `height_px` (64–100000); defaults remain 650 × 400. The UI supplies actual sheet dimensions. See `scripts/rc-demo.py` and [RC validation](rc-transient.md).

## Save and load schematics

Discover the current versioned catalog through `capabilities.list`. `project.save` and `project.load` are mutations and require the observed revision and a retry key, including operations that write a file without changing the graph.

| Command | Parameters | Behavior |
|---|---|---|
| `project.save` | `{"path":"/absolute/path/My LED.elab"}` | Create a new file and make it current. Existing destinations require explicit overwrite. |
| `project.save` | `{}` | Save to the current path; refuse if its contents changed outside ELab. |
| `project.save` | `{"path":"/absolute/path/My LED.elab","overwrite":true}` | Explicitly replace an existing or externally changed regular file. |
| `project.load` | `{"path":"/absolute/path/My LED.elab"}` | Validate and restore graph/view, mark clean, and resume live simulation. Refuse unsaved changes. |
| `project.load` | `{"path":"/absolute/path/My LED.elab","discard_changes":true}` | Explicitly replace unsaved work after file validation succeeds. |

Relative paths resolve against the process's working directory; absolute paths are recommended, especially for native app bundles. Parent folders must exist. The file format is `elab.schematic` version 4 (versions 1/2/3 still load; v1 uses default transient settings), bounded to 8 MiB (legacy files retain 256 KiB), 100 drawing objects and 300 wires. Simulation separately supports at most 64 simulated components and 100 electrical nodes including ground, subject to the per-family/topology bounds in `snapshot.limits`. Capacitors can additionally carry a signed per-component `initial_voltage_v` override. It carries built-in model identities, LED parameters, NPN parameters when an NPN is present, and transient settings; unknown models/versions, malformed metadata, invalid values, missing pins, duplicate objects, and noncanonical JSON fields are rejected. Symlinks and special files are refused.

Snapshots expose `document.path`, `document.dirty`, `document.format_version`, and a session-only `document.generation` changed by document replacement. Dirty tracks graph content and transient settings against the saved baseline. Pan/zoom/selection/pause alone do not mark the graph dirty; Save captures the view at that moment. Solver results, session revisions, selection, tool drafts, retry cache, and history are not persisted. A loaded circuit is recalculated; no file-supplied result is trusted.

`project.clear`, `example.reset`, `example.rc`, `example.npn`, `example.npn-pulse`, `example.flasher`, and `app.close` accept optional `discard_changes` and default to protecting dirty schematics. File operations add `unsaved-changes`, `file-exists`, `file-conflict`, `file-error`, and `invalid-document` error codes. Failed reads/writes preserve the current drawing, history, save point, and file association. An admitted background open may publish operation/revision bookkeeping even when validation fails. Save is not an undo edit; graph changes on Load are undoable, with the loaded path still current after undo. New detaches the file association. Retry keys prevent a successful save/load from executing twice; their normal retention and process-lifetime limits still apply.

Desktop Open/Save/Save As and native picker adapters invoke these same semantic commands. An external or built-in agent provides the path directly and needs no picker. The desktop Save / Discard / Cancel dialog maps to saving first or explicitly supplying `discard_changes:true`.

## Retries, history, and errors

Keep the same mutation key after a lost response. A repeated key with the same command and semantically identical JSON parameters returns the original response with `duplicate:true`, even if its original revision is now stale. That response describes the original accepted state; query current state separately. Reusing a key for different intent returns `key-reused`.

Deduplication retains the latest 256 successful mutation keys per session and transport actor. It survives disconnect/reconnect to the same process, but not process restart or switching from a socket actor to stdio. Expired keys have no retry guarantee. Admission failures are not retained. Accepted background work retains its retry identity even if numerical work later fails or is canceled. Placement, connection, exact values, movement, rotation, deletion, clear, and example loading share history retaining the latest 100 schematic edits. Deleting a component and its incident wires is atomic; dragging commits one move at release. Geometry changes preserve the electrical revision and solver run. IDs remain reserved after deletion/undo/clear for the life of the session. Concurrent changes require a fresh revision before undo/redo.

Common error codes are `invalid-request`, `invalid-json`, `invalid-params`, `unknown-command`, `conflict`, `key-reused`, `unavailable`, `closed`, `canceled`, `timeout`, and `request-too-large`. Invalid commands leave the previous circuit intact. `simulation.wait` can time out without rolling back other work.

## Local socket

Launch with `--agent-endpoint` in the desktop app or `--listen` in the headless process. The descriptor contains `version`, `socket`, and a random bearer `token`. Its permissions are 0600; the socket is 0600 inside an owned 0700 temporary directory. There is no TCP listener or remote provider connection.

Each connection sends the token plus newline, then one JSON request plus newline; it receives one JSON response plus newline and closes. The reference `elab-agent --connect DESCRIPTOR` client implements this framing. Keep the write side open until the response arrives. The server admits at most eight clients, bounds the request size, and sets a ten-second connection deadline. The only file operations exposed are bounded schematic reads and atomic writes; no arbitrary code execution is exposed.

Transport failure after a mutation is not evidence that it was undone. Retry with its original key or inspect state. Credentials grant access to this ELab session; they are not isolation from other software running as the same OS user. Session changes from socket agents appear as `agent.socket`, headless stdio changes as `agent.stdio`, and interactive changes as `user`.

## Remaining MVP contract

The current catalog covers the implemented application slice. Recovery, export, atomic batches and observable Verify/numerical/file jobs are implemented in the documented slice. Complete action/preference coverage, semantic schema/bounds discovery, event subscriptions, capability/schema migration and future editing/analysis conformance remain open. The PRD's complete user/agent parity remains the release requirement as those capabilities ship.

## Selectable probes and scope cursors

Use `snapshot.scope.available` to discover stable terminal/branch probe IDs and their voltage/current signs. `scope.probes.set` chooses up to six unique IDs without restarting simulation; chosen trace arrays backfill the retained 512-sample history. `scope.cursor.select`, `scope.cursor.set`, and `scope.cursor.clear` are the same commands used by native cursor buttons, number fields, and plot gestures. Cursor times snap to the nearest recorded sample; equal-distance/edge ties choose the latest right-hand state.

The scope snapshot exposes actual/requested A/B times, validity, signed `delta_s`, optional `interval_hz = 1/abs(delta_s)`, and per-probe A/B/delta in SI units. Stale or expired cursors do not report measurements. Probe changes do not dirty the schematic or change solver state; explicit Save now captures the selection in v3 files. Cursors/windows stay session-only. See [scope probes and cursors](scope-probes.md) and `scripts/probes-demo.py`.

## Additional hobbyist workflow commands

`parts.search.set {query}` updates the tray and `matching_parts`; `component.duplicate {id,position?}` creates an unconnected fresh-ID copy. `edit.batch {name,edits:[{command,params}]}` stages 1–100 schematic edits and commits as one named undo. Allowed edits: place/move/rotate/set/duplicate, capacitor.initial.set, object.delete, wire.connect. Failed batches reserve no IDs and publish no partial graph. File/simulation/history/nested batch commands refuse. Next action names are `undo_name` / `redo_name`.

`scope.window.set {mode:"fixed",start_s,end_s}` restricts plotting and statistics to an inclusive fresh retained interval; `{mode:"live"}` follows rolling history. `scope.statistics` reports count, min, max, sample_mean and sample_rms in each trace's SI unit. Expired windows have `window_valid:false` and empty statistics/traces. Statistics do not include startup outside the interval or claim time-weighted integration.

`export.scope.csv {path,overwrite?}` and `export.schematic.svg {path,overwrite?}` atomically write artifacts and return `last_export` metadata. CSV requires fresh selected window samples; SVG renders the whole drawing with shared routes even without a successful solve/window. Existing paths require overwrite, links refuse, and the current schematic path cannot be replaced by an export. Exports preserve project/solver state. Version-3 save/load restores selected probes; version 1/2 still load. See the [implementation and validation](../research/09-next-ten-implementation.md) and `scripts/workflow-demo.py`.

## Pending file actions and document identity

Save, Load, and both exports accept optional `expected_document_generation` in addition to ordinary revision/retry admission. Capture it from `snapshot.document.generation` when a file choice begins. A replacement project causes `conflict` before any read/write/publication; clock ticks and edits within the same document do not change the generation. Native file pickers preserve this guard through unsaved-change decisions. Omit it for ordinary immediate agent commands that already use the current revision.

## Verify operations and schema registry

`capabilities.list` now includes `schema_version:2` and `schemas`. Resolve output-schema references such as `urn:elab:snapshot:1` from that local registry; no network fetching is needed. Query output schemas may contain nested references to `urn:elab:verification-metadata:1`; recursively resolve all references from the same registry.

`simulation.verify` accepts a bounded startup scenario and returns a Snapshot's `last_operation` immediately. Inspect status/captured input/results with `operation.inspect`, page completed SI samples with `operation.samples`, and cancel/select through `operation.cancel` / `operation.select`. `verification.settings.set` changes session defaults without changing live state. `verification.export` saves the completed captured record as JSON or CSV. Recorded data is separate from current live readings. See [Verify command/lifecycle/validation details](verification.md).

Recovery is available through seven `recovery.*` commands and `snapshot.recovery`. Native startup configures it through the shared dispatcher; headless startup is opt-in using `--recovery-directory` or `recovery.configure`. Restores require a freshly listed file SHA-256 and create a dirty unassociated copy. See [autosave and recovery](recovery.md) for ownership, integrity, bounds and command parameters.

Recorded JSON import, independent playback, index/time seeking, captured-probe selection, interval statistics and timing cursors are exposed through `verification.import` and `recording.*`. Full exact rows remain available through `operation.samples`; only plotted points are reduced. See [recorded-result import and replay](recording.md). `recording.timing` and `recording.timing.set` expose saved threshold brackets, high/low pulse widths and first-cycle duty bounds through the same measurement used by native readouts. See [sampled pulse timing](recording-timing.md#pulse-widths-and-duty-cycle).

Project save/open now embeds kept completed recordings. `recording.keep.set {id,keep}` controls inclusion and protects kept assets from eviction; `document` exposes separate schematic/recording dirty flags and recording count. `recovery.samples` pages embedded recovery data. See [project format and lifecycle](project-format.md).

`junction` and `net-label` are single-node tray kinds. `wire.junction.insert` splits/joins explicitly named routes atomically; `component.place` / `.set` use `value` for a net label name. Net inspection includes sorted label names, and `v:<id>.node` probes read their actual nets. See [schematic primitive rules](schematic-primitives.md).

## Curated switch and diode devices

[SPST model and live actuation](switches.md) use `component.place` and `switch.set`. [Silicon diode defaults and probes](diodes.md) use `component.place` with `kind:"diode"`, named `anode`/`cathode` wires and ordinary scope/Verify commands. Snapshots, saved projects and records carry optional `switch_model` and `diode_model` parameters; network branch results include `mixed.switches` and `mixed.diodes`. Discovery is authoritative for the complete schemas.

`potentiometer.wiper.set {"id":"P1","position":0.75}` controls the three-pin linear track. Use `component.set` for its total resistance, and ordinary wire/probe commands for `1`, `2`, `wiper`. [Potentiometer defaults, current directions and transient semantics](potentiometers.md). Snapshots/records include `potentiometer_model` and `mixed.potentiometers`.

Per-source `sine-source` and `finite-pulse-source` configuration uses `source.sine.set`, `source.pulse.set` and `simulation.sampling.fit`. [Complete fields, waveform definitions, primary-source behavior and limits](waveforms.md). Projects/snapshots/records include `waveform_model`; per-component `sine`/`pulse` parameters are required for their respective kinds. The original `pulse-source` remains compatible.

`parts.list` exposes aliases and model limitations. `scope.unavailable` identifies removed-source requests, and `scope.probe.remove {"id":"v:C1"}` removes one without changing the circuit or other requested probes. [Hardening semantics and evidence](prd-hardening.md).


## Guarded history and live switch inputs

`undo_entry` / `redo_entry` expose the tip's name, actor and original `command_revision`. `edit.undo` / `edit.redo` accept optional `expected_entry_revision` and `allow_other_actor`. Capture the tip revision before planning a reversal. A changed tip returns `history-conflict`; another actor's tip also refuses unless explicitly acknowledged with `allow_other_actor:true`. The ordinary session revision precondition still applies. Identity and original actor survive undo/redo. Native buttons name the actor, acknowledge that displayed actor explicitly and guard the rendered tip.

`switch.set` queues desired state during a fresh transient. Inspect `switch_states` for desired/applied/pending, then Step or Resume to apply at the next accepted sample. Paused toggles leave physics/events unchanged; repeated toggles coalesce. Live inputs advance `actuation_revision` while retaining the circuit's electrical revision and run. Wiper events are background algebraic calculations at the committed time and preserve pending switch state; await `input_work.status: completed` before depending on the new parameter. Saving captures desired startup, not pending electrical memory. See [switch behavior](switches.md).


Live network sample work computes from independent captured physical data outside the service lock. Accepted samples publish with a new session revision; obsolete work/failures never replace an intervening run or input state. Navigation, save, presentation undo and pause remain callable during this work. Repeated Resume on an already running simulation preserves the existing clock. Network/NPN pulse startup and manual Step use shared background workers; see [the measured envelope](live-responsiveness.md).


## Background manual Step

`simulation.step.start {duration_s}` immediately returns a snapshot with queued `step_work`. Inspect the latest job ID/status, actor, originating run/electrical/actuation/command revisions, requested duration, start time and staged progress through `project.inspect`. `simulation.wait` waits for DC work or the current manual step for at most five seconds, then returns current state; inspect `step_work.error` on failed work. A wait timeout does not cancel the admitted operation.

`simulation.work.cancel {id}` targets the exact active ID. At most one manual step is active; another start returns `busy`. A failed/canceled/superseded step commits no partial time, capacitor memory, input event or queued-mode consumption. Pan, save and presentation undo remain available. Electrical edits, switch/wiper/input changes, Restart, Resume or project replacement supersede the captured work; Pause and Close cancel it. Source changes cannot be overwritten by a late numerical result or failure.

`simulation.step {duration_s}` remains a wait-for-terminal-result compatibility command backed by the same worker. Its successful response describes the completed advance; accepted numerical failure/cancellation returns structured error plus terminal snapshot. Its retry key preserves that terminal outcome. Canceling the request's wait or disconnecting does not cancel the admitted operation: re-inspect or retry the original key. Immediate-start retries instead preserve the original queued admission snapshot; query current progress separately. IDs are never reused in a process. Both forms retain the one-second/1,000-output-sample bound and a 30-second job wall budget, plus existing kernel work limits. Native Step uses `simulation.step.start` and offers the same exact-ID Cancel command. See [manual-step contract](manual-step-work.md).


## Background initialization

Supported general-network and NPN pulse edits/restarts return `status:"updating"` immediately. `simulation.wait` observes completion; input controls and immediate Step require fresh initialized data. The waiting Step compatibility command can await an initializer, but admission preconditions still apply after publication; prefer the advertised command guard to avoid publication-only conflicts. Malformed requests do not wait behind startup.

Pause cancels a pending initializer without publishing its partial result. Restart while paused explicitly computes fresh DC/transient startup and remains paused at time zero. Resume on an already running/updating session preserves the existing worker. Presentation edits survive publication; electrical edits, project replacement, undo and close prevent an obsolete result/failure from replacing the current run. Startup uses a five-second wall budget, with context checks inside network Newton/seed iteration. A current timeout produces a diagnostic rather than staying Updating. See [initialization semantics](initialization.md).


`snapshot.limits` publishes drawing, wire, physical-part, electrical-node and per-device/trace bounds; `simulated_component_count` excludes ground, junctions and labels. The Inspector shows the same counts. Supported simulation is bounded to 64 physical parts and 100 nodes including ground. Excess simulation complexity produces `mixed-limit` while preserving drawing/edit/save access within the drawing bounds. See [capacity evidence](circuit-capacity.md).

`scope.trace.select {"id":"i:D1"}` focuses a selected current live probe’s source. An empty ID clears it. `snapshot.trace_focus` contains `probe_id`, `object_ids`, `terminal_refs`, `net_ids`, `unavailable`, and optional `reason`. Association uses the current graph and descriptor references; it does not change editing selection, history, physical state or dirty flags. Missing source and recorded view suppress current-schematic highlights with an explicit reason. Hiding/removing a focused trace or replacing the project clears focus. This is session inspection state, not a persisted preference. See [source inspection](scope-probes.md).

`measurement.pin.add {"id":"v:C1","position":{"x":350,"y":250}}` pins a current probe from `scope.available`; it need not be plotted. `measurement.pin.move` takes the same ID/position plus optional `previous_position` to refuse intervening gesture changes. `measurement.pin.remove {"id":"v:C1"}` removes it, including unavailable sources. Up to eight pins use snapped ±10000-unit positions. These are undoable persisted presentation edits: physics/run/electrical revision stay unchanged, while `document.preferences_dirty` contributes to the document dirty guard. `snapshot.pinned_readings` exposes definitions, descriptors, SI values when present, time/run/revision, live/stale/unavailable status and shared display/reference text. Recorded scope view leaves these explicitly live measurements unchanged. [Pinned-reading contract and validation](pinned-readings.md).

`measurement.select {"kind":"terminal","id":"C1.1"}` and the named native selectors use the same source selection. Kinds are `object`, `terminal`, `net`, or `probe`; empty kind/ID returns to edit selection. Selection preserves physical state/history/dirty flags. Net selection captures a stable terminal anchor and follows its current group after rename/split; removing the anchor reports unavailable. `measurement.inspect {}` reads the current context; an optional `target` queries a current source without mutating selection. The result matches `snapshot.measurement_inspection`, including accepted SI readings, sign/reference text, result kind, run/time/revision and exact current source associations. Wire inspection reports voltage without inferring ideal-wire current. Native DC pin controls are available in the general Inspector. See [inspection contract and validation](measurement-inspection.md).

`display.preferences.set` sets the complete persisted voltage/current presentation object; `display.inspect {"phase":0.35}` returns read-only wire values, palette and deterministic marker geometry. `snapshot.display` carries accepted result identity/time, signed conventional amperes, known/unknown distinctions and effective motion. Native and agent frames share the same builder. Unique wire cut currents derive from accepted device injections; ideal-wire cycles receive no guessed currents. Electron mode reverses visuals only, and reduced motion/pause/stale/failure stop moving dots. [Display contract and validation](circuit-display.md).

`diagnostics.details.set {"visible":true}` changes session disclosure only, preserving physical/editing/history/document state and unfinished wire gestures. `snapshot.diagnostic_presentation` carries actionable summaries, technical details and current known object/terminal/wire associations, including sampling and manual-step failures. Applicable rejected responses additionally expose `help` describing related request intent, without changing global state or guessing a solver cause. Unknown references receive no marker. [Diagnostic contract and evidence](diagnostic-guidance.md).

Failed captured Verify operations include `failure_help` in `operation.inspect` and snapshot operation summaries, with a shared explanation/action/technical reason. Original error identity and captured metadata remain available; no partial samples are published. `source_unchanged:false` identifies an earlier captured experiment without guessing a cause on the current schematic. Resolve `urn:elab:verification-operation:1` and `urn:elab:diagnostic-help:1` through the local discovery registry. The same technical disclosure controls its native presentation. See [Verify failure guidance](verification.md).

`examples.list {}` discovers immutable guidance for led/rc/npn/switch-npn/blink/flasher/rectifier. `example.load` selects a template and starts an editable running copy; `example.current.reset` restores the associated template, preserving pause and using optional `expected_example_id` / `expected_document_generation` guards plus unsaved-change protection. `snapshot.example.info` is guidance; `modified` compares the actual graph/experiment to the template. Optional version-4 `example_id` preserves origin across save/recovery/reopen without trusting file physics. Named legacy example commands share the same definitions. [Guide/reset contract](guided-examples.md).

Fresh network wiper and running input changes admit an atomic background calculation; queued network input on Resume does too. `snapshot.input_work` exposes ID, actor, requested mode/wiper and terminal status. `simulation.work.cancel` accepts the exact current input ID; `simulation.wait` waits for input work as well as startup/manual Step. Successful retries retain their original admission response. Await `completed` before dependent commands. [Input-work contract](input-work.md).

`project.load.start` admits background reading/validation; `project.load` waits for the same job. `project_work` exposes phase, bytes, actor, source revision/generation and successful result generation/run. Cancel with `project.work.cancel {id}`. `simulation.wait` waits for project work and subsequent startup. Edits/new file association/replacement supersede an open; prior discard permission does not discard later changes. Admitted file failures preserve project/physics/history while retaining job metadata and consume their retry key. [Project-work contract](project-work.md).

`project.save.start` captures committed project data and returns background admission; `project.save` waits for the same work. `project.work.inspect {id}` reads eight retained jobs. Use current `project.inspect.command_revision` for mutations, rather than a captured job revision. Preparation is cancelable; `cancelable:false` during publishing/cleanup refuses cancellation with `operation-committing`. Later edits remain dirty; replaced/closed documents are not associated with prior captured files. Completion includes artifact SHA-256 and captured generation. New/example/Close also accept optional `expected_document_generation`, and native pending continuations retain it. [Save contract](project-save.md).

`export.start {kind,path,overwrite?,id?,expected_document_generation?}` admits captured schematic-svg/scope-csv/verification-json/verification-csv output; recorded kinds require `id`. Existing export commands wait for the same worker. Shared project-work inspection/cancellation/wait apply, including the publication boundary. `last_export.sha256` and `document_generation` identify exact captured artifacts; later editing/stepping/viewer changes never enter them. [Export contract](project-export.md).

`verification.import.start` admits private recorded-file reading/validation; the existing import waits for the same work. Shared project job inspection/cancellation/wait apply. Completion exposes `result_operation_id`, `source_sha256`, and `viewer_selected`; later viewer commands are preserved and capacity is rechecked before complete kept attachment. Import changes no live physical/graph/history state. [Import contract](project-import.md).

`recovery.work.start {kind:scan|checkpoint}` admits shared recovery jobs; existing scan/checkpoint wait for them. Completion reports recovery capsule ID/time and `recovery_catalog_applied`, preserving newer catalogs and selection. Checkpoint serialization is cancelable; publication/retirement is fenced. Captures preserve explicit saved files and later editing. [Recovery work](project-recovery-work.md).

The [reference conformance harness](agent-conformance.md) runs all three PRD circuit workflows through the in-process API, external stdio and headless/native private sockets, including introduced faults, exports, close and fresh-session reopening. It compares exact command parameters, persisted semantic state and accepted numerical readings. This is workflow evidence; complete capability coverage remains a separate requirement.

Time-valued command inputs also accept bounded decimal/scientific text with `s`, `ms`, `us`/`µs`/`μs`, `ns` or `ps`; bare text is seconds. Discovery resolves these through local `urn:elab:time:1`. Numeric SI outputs/files remain strict, and normal bounds/retry/revision rules apply. See [time input fields and validation](time-entry.md).

## Appearance and observation

[Appearance](appearance.md) uses `appearance.theme.set` with `light`, `dark`, or `system`. [Coalesced observation](events.md) uses `events.poll` to wait for state/job changes without holding the editor lock; use a separate connection for commands during a waiting poll. Both are advertised by discovery. `appearance.configure` selects or disables an independent preference file; `appearance.wait` awaits loading/saving, and snapshot `appearance` exposes outcomes.

`scope.visibility.set {"visible":false}` collapses the live/recorded dock while acquisition/playback continue. Snapshot `scope_visible` reports the session-only preference; measurement configuration and export remain available. See [the scope presentation contract](scope-visibility.md).

macOS menu and selector presentation uses AppKit while retaining original shared command callbacks. Native menus capture the descriptor’s command revision and carry it through shared-command admission, so an agent edit during menu tracking cannot apply a stale selection. Dismissal does not mutate state. See [native menu behavior and limits](native-menus.md).

`capabilities.list` accepts an optional `command` filter for one expanded input schema. Five electrical/example commands additionally advertise bounds, normalized-SI relations and runnable parameter examples; {} retains full discovery. [Detailed discovery contract](capability-details.md). Stateful/error/native-action coverage remains incomplete.

Mutations also accept optional top-level `expected_document_generation` from the observed snapshot; it supplements the required revision guard and is checked atomically. Successful retry lookup still precedes all guard checks. [Document context and rendered actions](document-context-guard.md).


### Observable protected shutdown

`app.close.start` admits the same protected close as `app.close`, returning immediately with `snapshot.shutdown`. The native Close action uses start; `app.close` waits for the job's final outcome. Normal dirty/default, revision/key and document-generation guards apply. `closed:true` freezes mutations immediately; it does not mean resource cleanup has finished. `shutdown.exit_ready` distinguishes that state.

`app.close.inspect {id}` returns owned retained progress/error/outcome without IO. Cleanup failure remains failed and holds automatic native/headless-socket exit; `app.close.acknowledge {id}` is the only mutation allowed after close, accepts an exact failed job, preserves its error and sets exit readiness. Use the waiting command when a final outcome response is needed: successful immediate shutdown may exit the host before a later query. Caller disconnect stops only the wait; a retry of the original waiting key joins the same job. No cancellation is possible after admission. See [shutdown contract](shutdown.md).

On graceful exit, the socket server stops accepting connections and lets already admitted replies finish before retiring their connections. Existing six-second dispatch and ten-second connection deadlines still bound this drain. Exit readiness must not truncate the final waiting `app.close` response. Forced endpoint cleanup before domain exit readiness still cancels work and closes idle/active connections promptly; a completed socket write is not a client acknowledgment or a guarantee against an external process kill.

`simulation.pulse.timing.set {"id":"VIN","frequency_hz":"20 Hz","duty_percent":"25 %"}` sets the shared ideal-square clock. Supply either `frequency_hz` quantity text or `period_s` numeric seconds/time text, plus percent text. All current `pulse-source` components share these experiment phases; finite pulses remain independent. `snapshot.pulse_timings` includes configured values, editable text, `source_kind` and `shared`. The command preserves sampling/playback and refuses phases shorter than the current sample step. It supports guards, retries, one named undo and atomic `edit.batch`. See [full contract and limits](shared-pulse-clock.md).

Waveform parameter commands accept numeric SI or decimal/scientific unit text: `source.sine.set` supports V/mV/uV/µV/μV amplitude and offset, Hz/kHz/mHz frequency, and deg/° phase; `source.pulse.set` supports the same voltage syntax for high/low levels alongside existing time units. Every nested waveform field remains required and coupled model bounds still apply. Input-only normalization shares numeric/quantity retry intent; saved/output data remains numeric SI. [Exact waveform entry](waveforms.md#exact-parameter-entry).

Tiled schematic output uses `export.page.set {"tile":true,"scale":1,"overlap_mm":5}` (plus ordinary paper/orientation/margin choices). PDF/printing include all pages; PNG requires `page_index` for multi-page layouts. `print.preview.page.set {"id":"preview-1","index":1}` renders another page of the same immutable capture under a new preview ID. `export.schematic.png {"preview_id":"preview-2","path":"..."}` saves the exact captured page bytes. See [bounds, capture and completion rules](tiled-output.md).

The eight core project commands expose [detailed state/file/work contracts](project-command-contracts.md) through filtered `capabilities.list`. They describe path/null semantics, captured saves and later dirty edits, Open supersession, retained IDs, cooperative deadlines, noncancelable publication and consumed retry keys after admitted failure. Default discovery and the public command set are unchanged.

`source_sampling` exposes configured per-source raw output targets separately from supported recommendations. A finite-ramp target below the 1 µs floor remains `unresolved`/`minimum_limited` after fitting, with `can_fit:false` at the floor and retained source warning/diagnostic guidance. Constant waveforms are omitted; saved settings recompute advice. The existing `simulation.sampling.fit` command and numerical/model/file limits remain. [Source-sampling contract](source-sampling.md).

Recovery candidate/inspection `owner_status` is the typed enum `active|inactive|unknown`. Legacy `active_session` stays true for unknown foreign ownership so older clients retain the protection. Such a capsule can be inspected/restored as a copy; foreign dismissal refuses `unavailable` before removal. Status is observed at scan/read time and rechecked at commit. [Ownership rules](recovery-ownership.md).

Automatic recovery failure/retry is visible through `snapshot.recovery.error`, candidate metadata and the existing recovery publication topic. Ordinary edits no longer suppress these observations. A newer recovery/save/configuration publication or Close fences stale results; manual Save establishes that fence before unlocked cleanup, including when a later edit prevents cleanup's own catalog update. A published capsule describes its captured document and never marks later edits saved. No manual Scan/Checkpoint is required after restoring available storage. [Failure/retry rules](recovery-failure.md).

Both `recovery.configure` aliases publish `x-existing-store-disable` semantics: `enabled:false` for the current canonical directory updates scheduling/interval without accessing that configured folder. Completion reports `recovery_configuration_applied:true` and `recovery_catalog_applied:false`; last-known error/catalog/selection remain. A captured write may finish but cannot replace the disabled state. New/different stores, enable, bounds, cancellation and admission guards retain their normal behavior. [Disable/explicit re-enable rules](recovery-disable.md).

Eleven [recorded-viewer and kept-membership controls](recording-viewer-controls.md) expose filtered state contracts. They distinguish exact row stepping from nearest-time/rightmost-edge seeking, whole-capture playback from plot windows, captured probe subsets from live probes, and session controls from saved viewing/membership preferences. Near-boundary windows normalize before positive-width validation; collapsed transient intervals refuse atomically. This checkpoint supplied 109 scoped contracts among 162 command IDs, leaving 53 without this annotation; annotation coverage alone does not prove exhaustive behavior or native parity.

The two [complete waveform setters](waveforms.md#complete-waveform-editing-contracts) add `per-source-waveform` state contracts, bringing current coverage to 111 of 162 IDs, with 51 without one. They replace all required nested fields, accept configured sources on incomplete drawings, report missing/wrong-kind targets as `invalid-params`, and preserve paused accepted data as stale. Sampling Fit and finite/ideal-square timing convenience commands remain separate explicit actions. Common admission guards, numeric SI output and atomic edit-batch behavior are unchanged.
