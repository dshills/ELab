# ELab

ELab is a hobbyist electronics workbench built with Vellum. The current slice lets you draw LED, RC, NPN switch, and mixed RC/NPN circuits from blank: place components, connect named terminals, move and rotate parts, and edit exact values while simulation updates live. Users and agents invoke the same validated application commands.

## Run the desktop app

The development layout expects this repository beside the inspected Vellum checkout:

```text
projects/
  ELab/
  Vellum/
```

Use Go 1.25.6 or newer and Vellum's macOS native prerequisites. Vellum must already have its pinned Cogent and typesetting dependencies prepared; on a fresh checkout run `Vellum/drivers/cogent/prepare.sh` following its development guide. ELab's `desktop/go.mod` contains the required four explicit framework/native replacements. The inspected [baseline](docs/vellum-baseline.json) records the framework commit and dependency manifest hash.

From the ELab directory:

```sh
./scripts/build.sh
./build/elab
```

The window starts with a blank schematic. Pick a DC or pulse source, resistor, capacitor, red LED, or ground from the left palette, then click the sheet. Use Wire to click pairs of terminal dots. Select a component to edit its value in the right inspector; inputs such as `2.2k ohm` are accepted. The inspector also offers exact X/Y placement while a part is selected for placement, and named terminal connections while Wire is active.

Connect source positive → resistor → LED anode, LED cathode → ground, and ground → source negative. Completing the loop starts live simulation. The LED example button loads a ready-made 5 V / 1 kΩ circuit. The 1 kΩ and 2 kΩ presets produce approximately 2.9168 mA and 1.4767 mA.

Drag a selected part to move it with its wires. Pan drags the sheet; scroll zooms; Fit frames the components. Arrow keys move a selected part by one grid step, or pan when no part is selected. Delete removes a selected part and its wires, or a selected wire. Escape cancels placement/wiring. Cmd+Z and Cmd+Shift+Z use shared document history when a focused text control does not consume the shortcut.

For a local double-clickable development bundle, after building:

```sh
python3 ../Vellum/scripts/package-app.py --binary build/elab \
  --output build/ELab.app --name ELab --identifier org.dshills.elab \
  --version 0.11.0 --build-version 25 --platform darwin
```

The packaging script requires a new output path. This creates a private unsigned bundle; public distribution and platform qualification remain future gates.

## Watch an RC circuit charge and discharge

Click **RC** to load the 5 V pulse / 10 kΩ / 1 µF example, or build it from blank. Connect source positive → resistor → capacitor, capacitor return → ground, and source negative → ground. Capacitors accept values such as `1uF`, `1 µF`, `100nF`, or `0.001 F`.

A valid RC circuit automatically advances in simulated time. The docked scope plots capacitor voltage relative to ground and source voltage; meters show signed RC current, capacitor voltage, and stored energy. **Auto** repeats the pulse-source cycle, **Charge** holds the source high, and **Discharge** drives it to zero while preserving capacitor voltage. A DC source stays high in Auto and also supports Charge/Discharge.

Pause freezes time; **Step +1 sample** advances it deliberately. Resume preserves time and energy unless an electrical edit invalidated the run. Changing R, C, wiring, or experiment settings restarts from the configured initial voltage. Restart starts a new run even while paused. Pulse timings, sample interval, initial voltage, and playback speed are editable in the inspector and available to agents. Defaults are 1 ms samples, 50 ms high/low, 0 V initial voltage, and 0.02 simulated seconds per wall second. Playback is paced for observation; it is not a wall-clock real-time claim.

The first transient engine supports one source, one resistor, and one ideal capacitor to ground. It uses the exact exponential RC response and splits at pulse edges. The 10 kΩ / 1 µF example reaches 3.160602794 V after 10 ms. This is a bounded ideal RC engine, with no ESR/leakage or finite pulse rise time. General networks and mixed LED/capacitor circuits remain unsupported. See the [RC implementation and validation record](docs/rc-transient.md).

## Switch an LED with a transistor

Choose **NPN** for an editable low-side LED switch, or add **NPN transistor** from the component tray. Connect its base through a resistor to a DC input, its emitter to ground, and its collector to the LED/resistor load. Use the base-drive buttons to explore cutoff, active operation, and saturation. All edits and live readings use the same commands and structured results for users and agents.

Choose **Blink** for a pulse-driven transistor LED with a live clock, voltage/current scope, and Auto / High / Low controls. Pause, Step, Resume, pulse settings, and all drawing operations use the same commands for agents. See the [NPN pulse workflow and validation](docs/npn-transient.md).

Choose **Flasher** for a DC-powered, capacitor-timed two-NPN oscillator with alternating LEDs. Its coupled solver preserves charge, supports signed capacitor startup conditions, and exposes the same commands and measurements to agents. See the [mixed RC/NPN workflow and validation](docs/mixed-transient.md).

The generic Ebers–Moll junction model remains instantaneous; physical transistor delay, internal charge storage, heating and breakdown remain unsupported. See the [NPN model](docs/npn-switch.md).

## Inspect both sides and measure timing

The right inspector's **Scope probes** selectors choose up to six terminal voltages or branch currents, including both flasher collectors, both LEDs, and each capacitor. Newly selected traces include already retained history without restarting the circuit. Voltage/current axes show signed values in V/mA.

Pause, select cursor **A** or **B**, then click/drag the scope or enter its requested time in ms. Cursors snap to recorded samples and show actual times, Δt, reciprocal Hz, and each trace's A/B/change readings. All selections, cursor gestures, and measurements are exposed through the same agent commands. Choose a fixed interval or follow live history; the inspector shows interval min/max/sample mean/RMS. Export scope CSV or schematic SVG through native file pickers or the same agent commands. Search the component tray, duplicate selected parts, and apply paired capacitor values as one named undoable transaction. See the [scope workflow](docs/scope-probes.md) and [ten-item implementation record](research/09-next-ten-implementation.md).

## Verify a captured experiment

The right inspector's **Verify from startup** panel runs the current circuit with explicit duration, sample step and startup conditions while preserving the live circuit/time. It returns an operation with progress and cancellation; completed readings are labeled as recorded. Export captured input/models/settings and samples as JSON or CSV. Users and agents share all settings, commands and results. See the [Verify workflow and limits](docs/verification.md).

## Keep your schematics

Use **Save** (Cmd+S), **Save As** (Cmd+Shift+S), and **Open** (Cmd+O). Native file pickers choose `.elab` files. Schematics keep component IDs, positions/rotations, exact values, wires, the saved view, versioned model parameters, capacitor startup overrides, and transient experiment settings. Version 3 files also capture selected probes. Existing version 1/2 files still load; v1 uses default RC settings. Opening validates the whole file before replacing the drawing and recalculates live readings; incomplete circuits can also be saved.

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

The current engines support a series LED DC circuit, an ideal first-order RC transient, and a grounded-emitter NPN switch with DC or pulse base drive, plus bounded coupled RC/NPN transient networks with up to two NPNs and four capacitors. Other topologies report incomplete, unsupported, or error states. General transient networks, physical transistor timing, scope cursors/captures, automatic recovery, atomic command batches, and export remain upcoming MVP work. DC results remain stable until input changes. RC results advance in genuine simulated time with a bounded live trace.

- `workbench/`: portable circuit model, LED/RC/NPN solvers, shared semantic commands, revisions, history, and snapshots.
- `transport/`: bounded JSON-line stdio and private local socket adapters.
- `cmd/elab-agent/`: headless session and reference external client.
- `desktop/`: independent native module using Vellum, controlled UI, command bindings, and schematic rendering.
- `research/`: evidence, [PRD](research/07-product-requirements.md), and [Vellum integration decision](research/08-vellum-integration.md).

Vellum's own code is MIT licensed; Cogent and other native dependencies retain their upstream licenses. No framework source is vendored here. Application licensing has not been selected.

Autosave and crash recovery are available in the Inspector. Snapshots preserve dirty edits separately from saved files; restoring opens a new copy for Save As. See [recovery commands and limits](docs/recovery.md). Native startup enables autosave by default; headless sessions opt in with `--recovery-directory` or the shared configuration command.
