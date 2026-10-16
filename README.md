# ELab

ELab is a hobbyist electronics workbench built with Vellum. The current slice lets you draw LED, RC, NPN and N/P-channel MOSFET switch, and mixed transistor/RC circuits from blank: place components, connect named terminals, move and rotate parts, and edit exact values while simulation updates live. Users and agents invoke the same validated application commands.

Choose **Examples → Split-rail DC supply** to explore [stacked DC sources and signed rail voltages](docs/floating-dc-sources.md). Two 3 V sources produce +3 V and −3 V around their midpoint reference. [Linear source/passive networks](docs/linear-network-solves.md) also solve 24 V stacks without artificial small-step nonconvergence, within the existing ±30 V node limit. [Nonlinear startup](docs/nonlinear-startup.md) also supports 24/30 V generic LED loads through a bounded grounded-constraint retry and rejects undefined floating MOSFET gates at zero supply.

Choose **Examples → MOSFET LED switch** for pulse-driven NMOS/LED simulation, gate/drain/source probes and an external gate-RC experiment. See [MOSFET behavior and model](docs/nmos.md). **Examples → High-side MOSFET LED** adds the complementary [PMOS workflow](docs/pmos.md). **Examples → CMOS inverter** combines both devices with input/output and signed drain-current plots; see [the guided inverter](docs/cmos-inverter.md).

## Run the desktop app

The development layout expects this repository beside the inspected Vellum checkout:

```text
projects/
  ELab/
  Vellum/
```

Use Go 1.25.6 or newer and Vellum's macOS native prerequisites. Vellum must already have its pinned Cogent and typesetting dependencies prepared; on a fresh checkout run `Vellum/drivers/cogent/prepare.sh` following its development guide. ELab's `desktop/go.mod` contains the required four explicit framework/native replacements. The [initial baseline](docs/vellum-baseline.json) preserves the first integration record; the [current qualification](docs/vellum-baseline-current.json) records the SDK commit, dependency hashes and tested release.

From the ELab directory:

```sh
./scripts/build.sh
./build/elab
```

The window starts with a blank schematic and focuses component Search. Pick a DC, sine, finite pulse or ideal square source, resistor, potentiometer, capacitor, red LED, silicon diode, NPN, N/P-channel MOSFET, SPST switch, junction, net label, or ground from the left palette, then click the sheet. Use Wire to click pairs of terminal dots. Select a component to edit its value in the right inspector; inputs such as `2.2k ohm` are accepted. The inspector also offers exact X/Y placement while a part is selected for placement, and named terminal connections while Wire is active.

Connect source positive → resistor → LED anode, LED cathode → ground, and ground → source negative. Completing the loop starts live simulation. Examples → Load LED example loads a ready-made 5 V / 1 kΩ circuit. The 1 kΩ and 2 kΩ presets produce approximately 2.9168 mA and 1.4767 mA.

**Examples → Manual transistor switch** opens a transistor-driven LED with its SPST control already selected. Close/open the switch in the Inspector and inspect base/collector currents. See the [manual transistor experiment](docs/manual-transistor-example.md) and [switch/wiper timing and shared commands](docs/interactive-actuation.md).

Captured runs now offer [time-weighted interval measurements](docs/interval-measurements.md): average, RMS, AC ripple RMS and peak-to-peak for voltage/current, through the same user and agent commands.

**Examples → LED fade** shows stored capacitor charge extending an LED pulse. Single-capacitor LED networks and series-capacitor RC high-pass circuits now use the existing network solver. See [RC/LED and high-pass experiments](docs/rc-led-networks.md).

**Examples → Rectifier / smoothing** opens a sine-driven diode/capacitor supply. Compare input/output voltage and reduce ripple by increasing C1. See the [rectifier experiment](docs/rectifier-example.md).

LED IDs now use [shared rotation-aware label positions](docs/led-labels.md), clearing emission arrows and terminal stubs in the canvas and schematic output.

Select a **Finite pulse** source to enter its frequency, period or high-phase percentage directly. The same [pulse clock command](docs/source-pulse-clock.md) is available to agents and preserves configured voltage/ramp values.

Agents can discover [schematic command rules](docs/schematic-command-contracts.md), including snapping, exact move guards, terminal planning and explicit junctions. Malformed move preimages now refuse without changing the circuit.

Search components by name or hobbyist alias, press Return, and choose a matching part with the keyboard. Placement X/Y and named-terminal wiring sit together in the left rail. Successful placement returns focus to search; successful connection returns to the first terminal. See the [keyboard construction workflow](docs/keyboard-construction.md).

Examples now [fit the actual schematic workspace](docs/viewport-fitting.md) and follow scope/window resizing. Pan, zoom or center an object to keep a manual view; click Fit to resume following. Saved files retain their own view. An [adjustable schematic/scope divider](docs/workspace-layout.md) gives the drawing more room while keeping scope controls available.

Drag a selected part to move it with its wires. Pan drags the sheet; scroll zooms; Fit frames bodies, labels, wires and annotations. The Inspector has Previous/Next and Center selection controls; with canvas focus, Option + Left/Right navigates objects and Option + Home/End chooses first/last. See [schematic navigation](docs/schematic-navigation.md). Arrow keys move a selected part by one grid step, or pan when no part is selected. Delete removes a selected part and its wires, or a selected wire. Escape cancels placement/wiring. Cmd+Z and Cmd+Shift+Z use shared document history when a focused text control does not consume the shortcut.

For a local double-clickable development bundle, after building:

```sh
python3 ../Vellum/scripts/package-app.py --binary build/elab \
  --output build/ELab.app --name ELab --identifier org.dshills.elab \
  --version 0.102.0 --build-version 117 --platform darwin
```

On macOS, File, Examples, Theme and Inspector dropdowns use native menus with checked current choices. See [native menu behavior](docs/native-menus.md).

**File → Close window** / **Cmd+W** use shared [unsaved-work close protection](docs/native-close.md), preserving active file operations and pending choices.

**Export PDF** creates a [printable schematic](docs/schematic-pdf.md) with shared A4/Letter page settings. **Export PNG** creates a [schematic image](docs/schematic-image.md) from the same Output page settings. [Print captured page](docs/printing.md) submits the preview to a chosen macOS printer through shared commands, with observable spool jobs and cancellation. Physical-device qualification remains pending.

The packaging script requires a new output path. This creates a private unsigned bundle; public distribution and platform qualification remain future gates.

Choose **Examples → RC high-pass filter** to compare input, positive/negative output pulses and continuous capacitor voltage. See the [high-pass experiment and accuracy checks](docs/rc-high-pass.md).

## Watch an RC circuit charge and discharge

Choose **Examples → Load RC example** to load the 5 V pulse / 10 kΩ / 1 µF example, or build it from blank. Connect source positive → resistor → capacitor, capacitor return → ground, and source negative → ground. Capacitors accept values such as `1uF`, `1 µF`, `100nF`, or `0.001 F`.

A valid RC circuit automatically advances in simulated time. The docked scope plots capacitor voltage relative to ground and source voltage; meters show signed RC current, capacitor voltage, and stored energy. **Auto** repeats the pulse-source cycle, **Charge** holds the source high, and **Discharge** drives it to zero while preserving capacitor voltage. A DC source stays high in Auto and also supports Charge/Discharge.

Pause freezes time; **Step +1 sample** advances it deliberately. Resume preserves time and energy unless an electrical edit invalidated the run. Changing R, C, wiring, or experiment settings restarts from the configured initial voltage. Restart starts a new run even while paused. Pulse timings, sample interval, initial voltage, and playback speed are editable in the inspector and available to agents. Defaults are 1 ms samples, 50 ms high/low, 0 V initial voltage, and 0.02 simulated seconds per wall second. Playback is paced for observation; it is not a wall-clock real-time claim.

The exact RC engine handles one source, one resistor, and one ideal capacitor to ground. It uses the exponential RC response and splits at pulse edges. The 10 kΩ / 1 µF example reaches 3.160602794 V after 10 ms. This engine has no ESR/leakage or finite pulse rise time. Other supported LED/capacitor and multi-part circuits use the bounded network/mixed engine; inspect diagnostics and `snapshot.limits` for supported topology/capacity. See the [RC implementation](docs/rc-transient.md) and [mixed transient limits](docs/mixed-transient.md).

## Switch an LED with a transistor

Choose **Examples → Load NPN LED example** for an editable low-side LED switch, or add **NPN transistor** from the component tray. Connect its base through a resistor to a DC input, its emitter to ground, and its collector to the LED/resistor load. Use the base-drive buttons to explore cutoff, active operation, and saturation. All edits and live readings use the same commands and structured results for users and agents.

Choose **Examples → Load blinking NPN LED** for a pulse-driven transistor LED with a live clock, voltage/current scope, and Auto / High / Low controls. Select its ideal-square source to set frequency, period or high-phase percentage; the inspector makes the shared-clock scope explicit. Pause, Step, Resume, pulse settings, and all drawing operations use the same commands for agents. See the [shared clock controls](docs/shared-pulse-clock.md). See the [NPN pulse workflow and validation](docs/npn-transient.md).

Choose **Examples → Load capacitor-timed LED flasher** for a DC-powered, capacitor-timed two-NPN oscillator with alternating LEDs. Its coupled solver preserves charge, supports signed capacitor startup conditions, and exposes the same commands and measurements to agents. See the [mixed RC/NPN workflow and validation](docs/mixed-transient.md) and [flasher example spacing](docs/flasher-layout.md). [Capacitor caption clearance](docs/capacitor-labels.md) is shared by the native canvas and exported schematics.

The generic Ebers–Moll junction model remains instantaneous; physical transistor delay, internal charge storage, heating and breakdown remain unsupported. See the [NPN model](docs/npn-switch.md).

## Inspect both sides and measure timing

The right inspector's **Scope probes** selectors choose up to six terminal voltages or branch currents, including both flasher collectors, both LEDs, and each capacitor. Newly selected traces include already retained history without restarting the circuit. Voltage/current axes show signed values in V/mA.

Select a component, terminal dot or wire to see signed live readings and references. Named terminal/net/probe selectors offer the same inspection without a pointer; Plot and Pin sit beside each reading. See [measurement inspection](docs/measurement-inspection.md).

The tray shows the current example’s question, expected template behavior, experiment and limitation. **Reset [example title]** restores that example with unsaved-change protection, while Restart reruns your edited circuit. Saved projects retain example origin. See [guided examples](docs/guided-examples.md).

Known circuit problems highlight their component, terminal or wire. The Inspector gives a next action and an optional **Technical details** disclosure; correcting the circuit removes the marker. Users and agents share the same diagnostic state and disclosure command. See [diagnostic guidance](docs/diagnostic-guidance.md).

Enable **Circuit display** voltage colors, static current arrows or illustrative moving dots in the Inspector. Reduced motion keeps static arrows; electron mode reverses visual cues while numerical amperes stay conventional. Settings are saved and use the same agent command. See [display behavior and limits](docs/circuit-display.md).

Choose **Pin a reading…** in the inspector to keep a voltage or signed current on the sheet. Drag its card or edit X / Y; positions are saved, and each completed drag is one undoable presentation edit. The cards show units, reference direction and stale/unavailable status. See [pinned readings](docs/pinned-readings.md).

Choose **Inspect trace…** below the scope to highlight the trace’s component and terminals without changing the simulation. Agents use `scope.trace.select` and receive the same source associations.

Pause, select cursor **A** or **B**, then click/drag the scope or enter its requested time in ms. Cursors snap to recorded samples and show actual times, Δt, reciprocal Hz, and each trace's A/B/change readings. All selections, cursor gestures, and measurements are exposed through the same agent commands. Choose a fixed interval or follow live history; the inspector shows interval min/max/sample mean/RMS. Export scope CSV or schematic SVG through native file pickers or the same agent commands. Search the component tray, duplicate selected parts, and apply paired capacitor values as one named undoable transaction. See the [scope workflow](docs/scope-probes.md) and [ten-item implementation record](research/09-next-ten-implementation.md).

## Verify a captured experiment

The right inspector's **Verify from startup** panel runs the current circuit with explicit duration, sample step and startup conditions while preserving the live circuit/time. It returns an operation with progress and cancellation; completed readings are labeled as recorded. Export captured input/models/settings and samples as JSON or CSV. Users and agents share all settings, commands and results. See the [Verify workflow and limits](docs/verification.md).

## Keep your schematics

Use **Save** (Cmd+S), **Save As** (Cmd+Shift+S), and **Open** (Cmd+O). Native file pickers choose `.elab` files. Schematics keep component IDs, positions/rotations, exact values, wires, the saved view, versioned model parameters, capacitor startup overrides, and transient experiment settings. Version 4 files include kept recorded results, dependency hashes and the current paused/viewer preferences. Version 3 adds selected probes; existing version 1/2/3 files still load; v1 uses default RC settings. Opening validates the whole file before replacing the drawing and recalculates live readings; incomplete circuits can also be saved.

The footer names the current file and marks unsaved changes. New, Open, loading the example, and closing protect unsaved work with Save / Discard / Cancel. Saving replaces the destination atomically and detects changes made outside ELab. Use Save As to keep both versions. View navigation alone does not mark the circuit dirty; an explicit Save captures the current view.

Agents call `project.save` and `project.load` through the same service, with normal revision/retry preconditions. See the [persistence format and behavior](docs/persistence.md).

## Run with an agent

Headless operation requires only the portable Go module, with no Vellum/native/GPU dependency:

```sh
go run ./cmd/elab-agent
```

Send one JSON object per line; receive one JSON response per line. Start by asking for `capabilities.list` or `project.inspect`. Mutations require the observed revision and a retry key. See the [agent protocol and examples](docs/agent-api.md).

To attach an external agent to the desktop app:

```sh
./build/elab --agent-endpoint "$PWD/build/session.endpoint.json"
```

In another terminal:

```sh
./build/elab-agent --connect build/session.endpoint.json
```

The endpoint descriptor is a private local credential. Keep it out of shared logs. The application removes it on ordinary shutdown and refuses to overwrite an existing descriptor. Use a new path after an abnormal exit. For a persistent headless socket session, use `./build/elab-agent --listen build/headless.endpoint.json` and the same reference client.

Use an absolute endpoint path when launching the `.app` bundle executable; the native bundle runtime may change the working directory.

## Check the implementation

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
```

The checks first run pinned project-local Staticcheck and goimports for both modules, then Go vet and race tests. The portable core retains standard-library-only runtime dependencies; lint tools live in their own module. The checks include electrical reference/conservation, invalid values, stale solve rejection, pause/edit/resume, undo/recovery, revision conflicts, retry deduplication, local socket access, and UI/agent equivalence. The native smoke verifies GPU schematic pixels, LED value editing, RC scope stepping, and NPN DC/pulse switching, the autonomous RC/NPN flasher, and selectable probes, timing cursors, and scope pixels through actual UI callbacks. Socket tests need permission to bind a local Unix socket; they cannot run inside a sandbox that forbids binding.

Run `python3 scripts/drawing-demo.py` for a headless workflow or add `--connect build/session.endpoint.json` to exercise a running desktop session. The [persistence implementation record](docs/persistence.md) describes validation and remaining MVP work.

Run `python3 scripts/persistence-demo.py --output docs/validation/headless-persistence.json` to verify save/reopen in a new process, file conflicts, and unsaved-change protection. It uses temporary files.

Run `python3 scripts/rc-demo.py --output docs/validation/headless-rc.json` to build from blank, check charge/discharge references, and save/reopen an RC circuit. Add `--connect` to exercise a desktop session.

Run `python3 scripts/npn-demo.py --output docs/validation/headless-npn.json` to build and switch an NPN LED circuit from blank, verify conservation, and save/reopen it. Add `--connect` for a desktop session.

Run `python3 scripts/npn-pulse-demo.py --output docs/validation/headless-npn-pulse.json` to build a pulse-driven NPN LED, verify exact edges and live controls, and save/reopen its settings. Add `--connect` for a development desktop session.

Run `python3 scripts/mixed-demo.py --output docs/validation/headless-mixed.json` to construct an autonomous flasher, verify capacitor-controlled timing and conservation, and save/reopen its initial conditions. Add `--connect` for a development desktop session.

Run `python3 scripts/probes-demo.py --output docs/validation/headless-probes.json` to select both flasher branches and measure a period with the shared scope cursors. Add `--connect` for an owned development desktop session.

Run `python3 scripts/workflow-demo.py --output docs/validation/headless-workflow.json` to verify the next ten workflow improvements. Add `--artifacts PATH` to keep a probed flasher, CSV interval, and SVG schematic.

Run `python3 scripts/verification-demo.py --output docs/validation/headless-verification.json` to exercise captured startup, analytical references, paging, exports, mixed parity, and cancellation.

## Scope and structure

The current engines support series LED DC, exact first-order RC, grounded-emitter NPN, coupled RC/NPN and bounded passive/switched circuit networks. Unsupported topologies report explicit diagnostics. Scope probes/cursors, recorded verification, recovery, atomic batches and portable exports are implemented. Participant usability, accessibility and distribution qualification remain upcoming work. DC results remain stable until input changes; transient results advance in simulated time with a bounded live trace.

- `workbench/`: portable circuit model, LED/RC/NPN solvers, shared semantic commands, revisions, history, and snapshots.
- `transport/`: bounded JSON-line stdio and private local socket adapters.
- `cmd/elab-agent/`: headless session and reference external client.
- `desktop/`: independent native module using Vellum, controlled UI, command bindings, and schematic rendering.
- `research/`: evidence, [PRD](research/07-product-requirements.md), and [Vellum integration decision](research/08-vellum-integration.md).

Vellum's own code is MIT licensed; Cogent and other native dependencies retain their upstream licenses. No framework source is vendored here. Application licensing has not been selected.

Autosave and crash recovery are available in the Inspector. Snapshots preserve dirty edits separately from saved files; restoring opens a new copy for Save As. See [recovery commands and limits](docs/recovery.md). Native startup enables autosave by default; headless sessions opt in with `--recovery-directory` or the shared configuration command.

Open exported recorded JSON from the Verify inspector to inspect/play its captured samples in a separate Recorded scope. Live state remains independent. [Recorded viewer commands and limits](docs/recording.md) describe exact edge stepping, probes and timing measurements.

Projects now save kept recorded results with their complete inputs/data, and reopen with fresh live startup alongside the recorded viewer. Completion/import marks those assets unsaved; Include/Exclude controls are shared with agents. [Project format, ownership and bounds](docs/project-format.md).

Junctions and net labels now support explicit branched/named connectivity. Select a wire to insert a junction, or review crossing routes in the X/Y controls. Matching label names join attached nets; crossings remain separate. [Shared primitive commands and rules](docs/schematic-primitives.md).

SPST switches queue live changes until the next accepted simulation step, with desired/applied/pending states and preserved capacitor charge. Undo/redo expose actor ownership and refuse stale targets or unacknowledged cross-actor reversals. The bounded network solver also supports resistor dividers. [Shared switch commands, model parameters and network limits](docs/switches.md).

The silicon diode adds nonlinear forward conduction and signed reverse leakage to bounded networks. [Model parameters, shared workflow and evidence](docs/diodes.md).

Linear potentiometers provide editable total resistance, bounded slider/exact wiper controls, divider/rheostat simulation and charge-preserving live adjustments. [Shared commands, pin map, model and evidence](docs/potentiometers.md).

Sine and finite pulse sources have independent amplitude/levels, offset/phase and timing controls, with shared sampling-fit guidance. Existing ideal-square examples retain their behavior. [Waveform commands, solver limits and evidence](docs/waveforms.md).

The [PRD conformance audit](research/11-prd-implementation-audit.md) records every P0 requirement and remaining acceptance gaps. [Audit repairs](docs/prd-hardening.md) add hobbyist search terms, model assumptions for every part, explicit unavailable probes, and robust duplication/cursor handling.

Live general-network samples compute outside the shared service lock, with guarded publication and tracked clock shutdown. [Responsiveness evidence and remaining synchronous paths](docs/live-responsiveness.md).

Manual Step uses a cancellable background worker and publishes the whole requested advance atomically. [Progress, cancellation, compatibility and validation](docs/manual-step-work.md).

Network and NPN pulse initialization computes in the background. Paused Restart refreshes all supported DC/transient paths. [Initialization contract and evidence](docs/initialization.md).

The tested envelope supports 64 simulated parts and 100 electrical nodes including ground, within 100 drawing objects and 300 wires. [Capacity definitions, per-device bounds and validation](docs/circuit-capacity.md).

Fresh network wiper and running input changes now return background work immediately; the Inspector exposes status and cancellation. Agents inspect `input_work` or wait before dependent commands. See [input work](docs/input-work.md).

Project Open reads and validates files in the background with shared phase/status and exact-ID cancellation. Agents can use immediate `project.load.start` or the existing waiting `project.load`. See [project work](docs/project-work.md).

Save now captures the committed project and writes in the background; later edits stay unsaved. Users and agents share preparation cancellation, an explicit publication boundary and bounded project-job inspection. See [captured saves](docs/project-save.md).

Schematic SVG, live scope CSV and recorded JSON/CSV exports now encode/write a captured source in the background. Native controls and agents share job inspection and preparation cancellation. See [captured exports](docs/project-export.md).

Recorded-result import now validates in the background, preserves live editing and later viewer choices, and rechecks kept-record capacity before attachment. Users and agents share jobs/provenance/cancellation. See [record import](docs/project-import.md).

Manual recovery refresh and checkpoint now run as shared background jobs, preserving selection, current edits and newer recovery lists. See [recovery work](docs/project-recovery-work.md).

Recovery Restore validates files in the background and commits only if the original document still matches. Native controls and agents share cancellation, unsaved protection and new-copy provenance. See [recovery Restore](docs/project-restore.md).

The reference [agent conformance harness](docs/agent-conformance.md) builds LED, RC and NPN circuits from empty sessions through in-process, stdio and private sockets, including native app processes. It checks fault repair, measurements, exports, close/reopen and equivalent project/result state. The standard build produces `build/elab-conformance`.

Use `capabilities.list {"command":"source.pulse.set"}` for [detailed electrical parameter discovery](docs/capability-details.md), including normalized time bounds, coupled limits and examples. The ordinary catalog query remains available.

The [native source inventory](research/13-native-command-inventory.md) checks literal routes and tracks dynamic binding gaps. Rendered actions can use the shared [document-context guard](docs/document-context-guard.md) to refuse stale edits after background document replacement.

[Committed native fields](docs/rendered-fields.md) retain their rendered command/document context, so old waveform or selector callbacks cannot overwrite newer agent work.

Time fields now accept values such as `10 ms` and `250 us` through the same native/agent commands. Numerical seconds, strict saved data and existing simulation bounds remain compatible. See [time entry](docs/time-entry.md).

## Appearance and agent observation

The header offers Light, Dark and System [themes saved on this device](docs/appearance.md), and File/Examples/Theme open native macOS menus. Agents use `appearance.theme.set` for the same preference. [Coalesced observation](docs/events.md) lets a separate `events.poll` connection wait for state/job changes without blocking commands. [UI audit screenshots](research/ui-audit/README.md) record the visible repairs and remaining limits.

Use **Hide scope** in the scope toolbar to give the schematic more room. **Show scope** restores the live or recorded plot with its measurements intact. Agents use the same [scope visibility command](docs/scope-visibility.md).

For frozen independent LED/RC/NPN references and flasher output-step refinement, run the [electrical qualification suite](docs/numerical-qualification.md) with `elab-conformance --electrical-suite`.

For repeated fresh/warm backend edit/result and bounded Verify timings, run the [performance qualification suite](docs/performance-qualification.md) with `elab-conformance --performance-suite --repetitions 20`. Visible UI acknowledgment is measured separately.

Transient failures now expose [distinct integration-limit causes and recovery](docs/integration-limits.md) through shared native/agent guidance. The original 512-attempt and 100 ms limits remain unchanged.

Waveform sources now show [shared schematic/export annotations](docs/source-annotations.md) for sine parameters, pulse timing and applied/pending/stale input state. Fit includes the annotation block.

[Captured DC sweeps](docs/dc-sweeps.md) plot source bias, resistance or wiper position without changing live physics. **DC sweep → Open sweep JSON…** reopens shared curves, including on a blank sheet. Users and agents share import, trace/cursor, keep, export and project-save commands. [Schematic PDF/PNG output](docs/schematic-pdf.md), [print preview](docs/print-preview.md) and [local printer submission](docs/printing.md) are available; physical printer qualification remains pending. [Captured-run comparisons](docs/recording-comparison.md) now include optional dashed reference traces on shared axes, without alignment or extrapolation.

Agents can inspect [sweep command contracts](docs/capability-details.md#sweep-command-contracts) for normalized parameter bounds, state/error conditions and completion behavior. Native **Clear point** and `sweep.cursor.set {"index":null}` share the same clearing behavior; omitting the index remains invalid.

[Recovery setup](docs/recovery.md#background-recovery-setup) now prepares and scans in the background. Users and agents share immediate admission, cancellation and an explicit installed-configuration outcome; the previous configuration stays active until the new one is ready.

[Recovery dismissal](docs/recovery.md#background-recovery-dismissal) now validates and removes a confirmed snapshot in the background, with shared cancellation and an explicit removal outcome. Changed files and other active sessions remain protected; live circuits and saved schematics stay unchanged.

DC sweeps support linear and positive logarithmic spacing, including descending ranges. Select **DC sweep**, choose a target and range, then choose the spacing before running. Captured curves preserve spacing through JSON/project saves and imports; cursors select exact measured points. See the [resistance experiment](docs/dc-sweeps.md#logarithmic-resistance-experiment).

Choose **Show recorded → Compare captures** to compare two saved experiments at a requested time. The Inspector and agents share exact saved-sample readings, B − A and optional whole-run statistics, with both actual sample times shown. See [captured comparisons](docs/recording-comparison.md).

Captured runs now offer **Threshold timing**: choose a saved probe, level, direction and interval to inspect exact threshold-arrival brackets and the first same-direction spacing bounds. Users and agents share the same measurement and result pages. See [timing workflow](docs/recording-timing.md).

Component values accept engineering voltage units such as **500 mV**, with visible ranges/coupled waveform limits and the same discoverable domains for agents. Nonzero inputs that underflow to zero are refused. See [component value contract](docs/component-values.md).

The Inspector now publishes exact editable value text. Reapplying an untouched value preserves its stored bits, live state, clean-save status and undo/redo history; common values use compact forms such as **1 µF** and **1 kΩ**. See [unchanged value handling](docs/component-values.md#reapplying-an-unchanged-value).

Captured recordings also support [two-threshold rise/fall timing](docs/recording-timing.md#two-threshold-rise-and-fall-timing): for a 5 V RC charge, use 500 mV and 4.5 V to inspect 10–90% duration bounds through the same commands as an agent.

Live controls expose a [shared input-mode inventory and state/completion contracts](docs/live-simulation-controls.md). A 1 µs sample step now works at 0.001 playback, and exact 1,000-sample manual advances no longer refuse from quotient rounding; larger values still respect the original limits.

The waiting Step alias now handles [unit text during initialization](docs/live-simulation-controls.md#waiting-for-initialization) consistently with numeric seconds, rejects already-stale context promptly, and preserves original retry/guard behavior.

Scopes now have [discoverable control contracts and bounded narrow-window clipping](docs/scope-probes.md#discoverable-control-contracts). Empty intervals report no data, CSV agrees with plotted rows, and captured/reference views preserve exact saved timestamps.
