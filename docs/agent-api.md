# ELab agent protocol, version 1

All semantic commands use `workbench.Service.Dispatch`. Vellum widget callbacks, keyboard commands, canvas gestures, built-in Go clients, headless stdio, and the external socket client share this entry point. An agent does not need screenshots or simulated clicks to operate the implemented circuit slice.

## Request and response

```json
{"version":1,"id":"inspect-1","command":"project.inspect","params":{}}
```

`version`, `id`, `command`, and object `params` are required. IDs are 1–128 bytes. Mutations also require `expected_revision` and a 1–128-byte `key`. JSON uses canonical field names; unknown fields and duplicate keys are rejected. One request is bounded to 32 KiB.

Responses include `version`, the request `id`, the session `revision`, and either `result` or structured `error` with `code`/`message`. Successful mutation results are full snapshots. Read `capabilities.list` for descriptions and input/output schemas. `project.inspect` reports components and terminals, wires, exact values/units, selection, viewport, model parameters, diagnostics, undo availability, last actor, and simulation state/results.

Socket responses are bounded to 1 MiB; an oversized result returns `response-too-large` explicitly.

Revision is a session admission version and changes on accepted mutations and solver result publication. `electrical_revision` changes only on electrical edits. `run_id` identifies calculation attempts. A selection or view change preserves the electrical revision and run. Live transient samples also change the session revision; `command_revision` changes only for admitted commands, allowing clients to distinguish edits from clock updates. Transient mutations still require the latest session revision and the same explicit conflict/retry rules. A `conflict` response is an explicit refusal; re-inspect before deciding how to proceed.

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

Positions snap to a 10-unit grid within ±10000; up to 50 components and 150 wires are admitted. A wire explicitly joins two existing named terminals. Crossings, touching geometry, and nearby dots never create connectivity. Ground terminals share the reference net. Current analysis accepts one source, one resistor, one forward LED, and one or more grounds in a series loop, including LED-before-resistor order and either resistor pin direction.

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

Snapshot `mixed` exposes all node/device/capacitor/branch readings and conservation residuals; `transient` contains its primary rolling voltage/current/capacitor trace. Engine `elab.mixed-mna-be.v1` couples ideal capacitors to the generic nonlinear junction model with adaptive backward Euler. Limits, signed quantities, initialization, transactional failures and complete data semantics are in [mixed transient documentation](mixed-transient.md). Repeat the workflow with `scripts/mixed-demo.py`.

## RC transient workflow

`parts.list` includes `capacitor` (terminals `1` / `2`, default 1 µF, `ideal-capacitor.v1`) and `pulse-source` (positive / negative, default 5 V, `ideal-square.v1`). Both use the existing placement, wire, value, move, rotate, and delete commands. Capacitance accepts p/n/u/µ/μ/m prefixes and F, within 1 pF–1 F. Pulse amplitude uses the source's 0–12 V value; its low level is 0 V.

Connect one source through one resistor to the capacitor output, connect the other capacitor pin to ground, and ground source negative. Either resistor or capacitor pin order is supported. Measurement voltage is the output node relative to ground; positive current flows through the resistor into that output. Multiple/mixed storage devices or LED/RC networks are retained with an unsupported diagnostic.

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

The catalog now has 35 commands. `project.save` and `project.load` are mutations and require the observed revision and a retry key, including operations that write a file without changing the graph.

| Command | Parameters | Behavior |
|---|---|---|
| `project.save` | `{"path":"/absolute/path/My LED.elab"}` | Create a new file and make it current. Existing destinations require explicit overwrite. |
| `project.save` | `{}` | Save to the current path; refuse if its contents changed outside ELab. |
| `project.save` | `{"path":"/absolute/path/My LED.elab","overwrite":true}` | Explicitly replace an existing or externally changed regular file. |
| `project.load` | `{"path":"/absolute/path/My LED.elab"}` | Validate and restore graph/view, mark clean, and resume live simulation. Refuse unsaved changes. |
| `project.load` | `{"path":"/absolute/path/My LED.elab","discard_changes":true}` | Explicitly replace unsaved work after file validation succeeds. |

Relative paths resolve against the process's working directory; absolute paths are recommended, especially for native app bundles. Parent folders must exist. The file format is `elab.schematic` version 4 (versions 1/2/3 still load; v1 uses default transient settings), bounded to 8 MiB (legacy files retain 256 KiB) and the existing 50-component/150-wire limits. Capacitors can additionally carry a signed per-component `initial_voltage_v` override. It carries built-in model identities, LED parameters, NPN parameters when an NPN is present, and transient settings; unknown models/versions, malformed metadata, invalid values, missing pins, duplicate objects, and noncanonical JSON fields are rejected. Symlinks and special files are refused.

Snapshots expose `document.path`, `document.dirty`, `document.format_version`, and a session-only `document.generation` changed by document replacement. Dirty tracks graph content and transient settings against the saved baseline. Pan/zoom/selection/pause alone do not mark the graph dirty; Save captures the view at that moment. Solver results, session revisions, selection, tool drafts, retry cache, and history are not persisted. A loaded circuit is recalculated; no file-supplied result is trusted.

`project.clear`, `example.reset`, `example.rc`, `example.npn`, `example.npn-pulse`, `example.flasher`, and `app.close` accept optional `discard_changes` and default to protecting dirty schematics. File operations add `unsaved-changes`, `file-exists`, `file-conflict`, `file-error`, and `invalid-document` error codes. Failed reads/writes preserve the current drawing, history, save point, and file association. Save is not an undo edit; graph changes on Load are undoable, with the loaded path still current after undo. New detaches the file association. Retry keys prevent a successful save/load from executing twice; their normal retention and process-lifetime limits still apply.

Desktop Open/Save/Save As and native picker adapters invoke these same semantic commands. An external or built-in agent provides the path directly and needs no picker. The desktop Save / Discard / Cancel dialog maps to saving first or explicitly supplying `discard_changes:true`.

## Retries, history, and errors

Keep the same mutation key after a lost response. A repeated key with the same command and semantically identical JSON parameters returns the original response with `duplicate:true`, even if its original revision is now stale. That response describes the original accepted state; query current state separately. Reusing a key for different intent returns `key-reused`.

Deduplication retains the latest 256 successful mutation keys per session and transport actor. It survives disconnect/reconnect to the same process, but not process restart or switching from a socket actor to stdio. Expired keys have no retry guarantee. Failed requests are not retained. Placement, connection, exact values, movement, rotation, deletion, clear, and example loading share history retaining the latest 100 schematic edits. Deleting a component and its incident wires is atomic; dragging commits one move at release. Geometry changes preserve the electrical revision and solver run. IDs remain reserved after deletion/undo/clear for the life of the session. Concurrent changes require a fresh revision before undo/redo.

Common error codes are `invalid-request`, `invalid-json`, `invalid-params`, `unknown-command`, `conflict`, `key-reused`, `unavailable`, `closed`, `canceled`, `timeout`, and `request-too-large`. Invalid commands leave the previous circuit intact. `simulation.wait` can time out without rolling back other work.

## Local socket

Launch with `--agent-endpoint` in the desktop app or `--listen` in the headless process. The descriptor contains `version`, `socket`, and a random bearer `token`. Its permissions are 0600; the socket is 0600 inside an owned 0700 temporary directory. There is no TCP listener or remote provider connection.

Each connection sends the token plus newline, then one JSON request plus newline; it receives one JSON response plus newline and closes. The reference `elab-agent --connect DESCRIPTOR` client implements this framing. Keep the write side open until the response arrives. The server admits at most eight clients, bounds the request size, and sets a ten-second connection deadline. The only file operations exposed are bounded schematic reads and atomic writes; no arbitrary code execution is exposed.

Transport failure after a mutation is not evidence that it was undone. Retry with its original key or inspect state. Credentials grant access to this ELab session; they are not isolation from other software running as the same OS user. Session changes from socket agents appear as `agent.socket`, headless stdio changes as `agent.stdio`, and interactive changes as `user`.

## Remaining MVP contract

The current catalog covers the implemented application slice. Automatic recovery, export, atomic batches, richer jobs/events, capability/schema version migration, atomic multi-edit transactions, and all future editing/analysis functions still need implementation and conformance tests. The PRD's complete user/agent parity remains the release requirement as those capabilities ship.

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

Recorded JSON import, independent playback, index/time seeking, captured-probe selection, interval statistics and timing cursors are exposed through `verification.import` and `recording.*`. Full exact rows remain available through `operation.samples`; only plotted points are reduced. See [recorded-result import and replay](recording.md).

Project save/open now embeds kept completed recordings. `recording.keep.set {id,keep}` controls inclusion and protects kept assets from eviction; `document` exposes separate schematic/recording dirty flags and recording count. `recovery.samples` pages embedded recovery data. See [project format and lifecycle](project-format.md).

`junction` and `net-label` are single-node tray kinds. `wire.junction.insert` splits/joins explicitly named routes atomically; `component.place` / `.set` use `value` for a net label name. Net inspection includes sorted label names, and `v:<id>.node` probes read their actual nets. See [schematic primitive rules](schematic-primitives.md).

## Curated switch and diode devices

[SPST model and live actuation](switches.md) use `component.place` and `switch.set`. [Silicon diode defaults and probes](diodes.md) use `component.place` with `kind:"diode"`, named `anode`/`cathode` wires and ordinary scope/Verify commands. Snapshots, saved projects and records carry optional `switch_model` and `diode_model` parameters; network branch results include `mixed.switches` and `mixed.diodes`. Discovery is authoritative for the complete schemas.

`potentiometer.wiper.set {"id":"P1","position":0.75}` controls the three-pin linear track. Use `component.set` for its total resistance, and ordinary wire/probe commands for `1`, `2`, `wiper`. [Potentiometer defaults, current directions and transient semantics](potentiometers.md). Snapshots/records include `potentiometer_model` and `mixed.potentiometers`.

Per-source `sine-source` and `finite-pulse-source` configuration uses `source.sine.set`, `source.pulse.set` and `simulation.sampling.fit`. [Complete fields, waveform definitions, primary-source behavior and limits](waveforms.md). Projects/snapshots/records include `waveform_model`; per-component `sine`/`pulse` parameters are required for their respective kinds. The original `pulse-source` remains compatible.
