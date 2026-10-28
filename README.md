# ELab

ELab is a hobbyist electronics workbench built with Vellum. Draw circuits from
blank, connect named terminals, and edit values while simulation updates live.
Explore LEDs and RC timing, discrete transistor amplifiers, logic and counters,
motors, sensors and power supplies. Users and AI agents invoke the same validated
application commands for drawing, simulation, measurement, saving and export.

The [first-version completion checklist](docs/v1-release-checklist.md) records
the earlier private-alpha release scope and acceptance limits. Later authorized
work is described below; the preserved release and its evidence remain separate.

## Current development

The signal dock adds **Scope / Spectrum** tabs and live FFT using selected
scope probes, with Hold and independent recorded analysis. Users and agents share
`scope.view.set` and `spectrum.live.set`. See the [spectrum guide](docs/spectrum-analysis.md)
and [internal review/tests](docs/validation/scope-spectrum-20261028/README.md).
These controls are included in the /282 candidate below.

**Current candidate: [ELab 0.200.4 /282](build/ELab-0.200.4-282-Spectrum/ELab.app)**,
with **87 examples**, **61 component kinds** and **224 shared commands**.
It includes capacity noise-capture/agent-response repairs, bounded
[analysis queries](docs/analysis-pages.md), cached owned compilation and grouped
analysis-job storage behind the same user/agent command service. See the [spectrum checks](docs/validation/scope-spectrum-20261028/README.md),
[prior release-readiness evidence](docs/validation/release-readiness-20261028/README.md)
and [architecture guide](docs/architecture.md). The candidate remains unsigned/ad-hoc;
original native-input, participant, clean-account and public-signing gates stay separate.

The preserved [0.200.3 /281 readiness package](build/ELab-0.200.3-281-Readiness/ELab.app),
[0.200.1 /279 repair package](build/ELab-0.200.1-279-Reliability/ELab.app)
and [0.200.2 /280 tooling candidate](build/ELab-0.200.2-280-Readiness/ELab.app)
remain unchanged. The [project review](research/126-project-code-and-architecture-review.md)
records the original defects and their reproduction.

The preserved [0.200.0 /278 electronics candidate](build/ELab-0.200.0-278-Electronics/ELab.app)
adds five capabilities, with the same validated commands available to users and
AI agents:

- [Reusable circuit blocks](docs/circuit-blocks.md): single-level named pins,
  published values and independent copies across drawings. Members remain ordinary
  parts with the existing simulation limits.
- [Temperature effects and self-heating](docs/temperature.md): resistor,
  silicon-diode and bipolar temperature laws, optional thermal RC and ambient sweeps.
- [Small-signal noise analysis](docs/noise-analysis.md): declared independent white
  noise, component contributions, spectral density and integrated RMS budgets.
- [Photodiode sensing](docs/photodiodes.md): light-controlled photocurrent,
  dark current, capacitance and a transimpedance-amplifier example.
- [SCR/TRIAC control](docs/thyristors.md): gate triggering, actual-current latching,
  holding-current release and low-voltage AC commutation examples.

The /278 matching app/agent package contains **87 examples**, **61 component kinds**
and **220 shared commands**. Search Examples for RC block, self-heating, noise,
photodiode, SCR or TRIAC. Electrical references, user/agent parity, complete
Anthropic Claude Opus 5.5 Prism reviews and exact package checks are recorded in
the [engineering evidence](docs/validation/electronics-blocks-batch/README.md).
Resource, accuracy and simulation work limits remain unchanged. Models are
illustrative; the guides describe supported domains and assumptions.

The preserved [0.199.0 /277 candidate](build/ELab-0.199.0-277-Electronics/ELab.app)
contains [all-discrete class-AB/speaker transients](docs/class-ab-speaker.md),
[passive losses](docs/passive-losses.md), [current sources and electronic loads](docs/current-devices.md),
[JFETs](docs/jfets.md), and [loop-return/step-response analysis](docs/feedback-and-step-response.md).
Its matching app/agent has 78 examples, 58 kinds and 209 commands. Loop injection
targets configured dynamic op-amps; all-discrete physical-wire loop analysis is
not claimed. [Prior checks](docs/validation/electronics-next-five/README.md).

The preserved [0.198.0 /276 candidate](build/ELab-0.198.0-276-Analysis/ELab.app)
contains semiconductor AC/Bode sweeps, FFT/distortion, configurable transistor
profiles, seeded tolerances and electrical loudspeaker loads, with 67 examples,
54 kinds and 203 commands. Its original fast class-AB/speaker transient limitation
is repaired in /277; the /276 bytes and historical evidence remain preserved.
[Prior checks](docs/validation/electronics-five/README.md).

The preserved [0.197.0 /275 candidate](build/ELab-0.197.0-275-Discrete/ELab.app)
expanded combined NPN/PNP capacity to **16** and added an **eight-transistor,
all-discrete class-AB amplifier**. Search Examples for “class AB”, or use
`example.load {"id":"discrete-class-ab"}`. That package includes its matching
agent and **63** example projects, with 53 component kinds and
194 shared commands. Other capacities, electrical models and numerical
accuracy/work guards remain unchanged. [Amplifier guide](docs/discrete-class-ab.md),
[electrical and package checks](docs/validation/discrete-class-ab/README.md).

The earlier workbench update adds snapped placement ghosts, pending wire previews,
nearest unambiguous terminal targeting, canvas shortcuts, component explanations,
explicit live plotting, experiment comparisons and shared edit proposals.
See the [workflow guide](docs/workbench-workflows.md) and
[engineering checks](docs/validation/workbench-improvements/README.md).
The [previous /274 runtime candidate](build/ELab-0.196.1-274-Vellum/ELab.app)
and qualified /270 download below are preserved. Existing native-input,
participant and separate-account acceptance limits remain open.

The UI runtime uses Vellum v1.0.0 public source `4cf0176` with ELab’s
existing input/ownership compatibility fixes and the new
[application icon](docs/application-icon.md).
[Runtime upgrade checks](docs/validation/vellum-v1-runtime/README.md).

## Downloadable release

**Preserved release: ELab 0.195.9 /270** packages 53 component kinds and the 62
examples available at that release, with its matching agent. [Open app](build/releases/ELab-0.195.9-macos-arm64/ELab.app),
[download ZIP](build/releases/ELab-0.195.9-macos-arm64.zip), or
[verify SHA-256](build/releases/ELab-0.195.9-macos-arm64.zip.sha256).

This quality update keeps live scope buttons, time/status text and footer readings
in stable positions as simulation values change. It also qualifies shared-agent
LED/RC/counter workflows, a 30-minute run, exact archive extraction and all 62
reopened examples. [Completion evidence and limits](docs/validation/v1-acceptance/README.md).

Four native input checks remain unqualified: Save-panel editing, controlled
divider rates and precise native pre-agent parser origin. Their investigation
and tool limits are recorded separately from passing software/runtime tests.

The **Examples** browser has Start here, topic categories, global word search,
actual schematic previews and short experiments. Browsing preserves the live
circuit; opening uses the existing Save / Discard / Cancel protection. Users and
agents share the same browser commands. See the [browser guide](docs/example-browser.md).
This round also fixes scroll-related numeric help and verifies keyboard drawing,
editing, measurement and save workflows. It adds no components or electrical models
and leaves simulation capacities and precision unchanged.

Requires Apple Silicon and macOS 26.0 or later; tested on macOS 26.6.2. This is
the owner-selected unsigned channel, with an ad-hoc integrity seal and no
Developer ID or notarization. A downloaded copy can be blocked by Gatekeeper;
see [installation/use instructions](build/releases/ELab-0.195.9-macos-arm64/README.md).
The [ordered work and qualification](research/116-reliability-release-and-examples-plan.md)
records native, agent, race, rendered, archive and offline checks. Owner walkthrough,
separate-account installation and full PRD acceptance retain their recorded deferrals.
The package is prepared locally; nothing has been publicly uploaded or pushed.

The [0.195.8 /269 collection-origin release](build/releases/ELab-0.195.8-macos-arm64.zip),
[0.195.4 /265 modal/export release](build/releases/ELab-0.195.4-macos-arm64.zip),
[0.195.3 /264 examples release](build/releases/ELab-0.195.3-macos-arm64.zip),
[0.194.5 /260 reliability release](build/releases/ELab-0.194.5-macos-arm64.zip)
and [0.181.4 /205 qualified alpha](build/releases/ELab-0.181.4-macos-arm64.zip)
remain byte-for-byte preserved, alongside earlier candidates and owner projects.

The document-editor UI has a component library, central canvas, Inspect / Simulate /
Document navigation, live readings and Light, Dark, System and Apollo themes.
See the [component browser](docs/component-browser.md),
[workspace layout](docs/workspace-layout.md) and [electronics model guide](docs/hobbyist-devices.md).
Audio playback is explicit; opening or simulating a circuit stays silent.

## Build and run from source

Use Go 1.25.6 or newer, Python 3.10 or newer, and Vellum's macOS native prerequisites. Normal builds verify and extract the [pinned SDK from public Vellum source](third_party/vellum/README.md), including prepared Cogent/typesetting sources and explicit compatibility fixes. The public Vellum v1.0.0 source `4cf0176` and derived compatibility source `19c9ebc` are pinned with their patch/archive hashes in `third_party/vellum/sdk.lock.json`. A sibling Vellum checkout is not required. ELab's `desktop/go.mod` contains the four explicit local replacements into the verified build cache. Run `python3 scripts/prepare-vellum.py` before invoking native Go tools directly; `build.sh`, `lint.sh` and `check.sh` prepare it automatically.

The SDK lock distinguishes the public repository commit from the compatibility-patched source and unreleased SDK artifact. The [initial baseline](docs/vellum-baseline.json) and [packaged application qualification](docs/vellum-baseline-current.json) retain their historical release identities; changing the source dependency does not promote an application bundle.

From the ELab directory:

```sh
./scripts/build.sh
./build/elab
```

The window starts with a blank schematic, focuses component Search, and shows Common. Choose a category or All components, or type a name/alias and press Return to search the entire catalog. Pick a DC, sine, finite pulse or ideal square source, resistor, potentiometer, capacitor, inductor, red LED, silicon diode, NPN, PNP, N/P-channel MOSFET, SPST switch, junction, net label, or ground from the left palette, then click the sheet. Use Wire to click pairs of terminal dots. Select a component to edit its value in the right inspector; inputs such as `2.2k ohm` are accepted. The inspector also offers exact X/Y placement while a part is selected for placement, and named terminal connections while Wire is active.

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
python3 -B scripts/package_app.py --binary build/elab \
  --output build/ELab-local/ELab.app --version 0.0.0 --build-version 1
```

The wrapper verifies the separately pinned Vellum packager, includes ELab’s
application icon and applies an ad-hoc integrity seal. It requires a new output
path and preserves existing bundles. This command creates a local development
bundle; use the candidate linked above for the qualified /276 executable and
matching agent. See [packaging and icon details](docs/application-icon.md).

On macOS, File, Examples, Theme and Inspector dropdowns use native menus with checked current choices. See [native menu behavior](docs/native-menus.md).

**File → Close window** / **Cmd+W** use shared [unsaved-work close protection](docs/native-close.md), preserving active file operations and pending choices.

**Export PDF** creates a [printable schematic](docs/schematic-pdf.md) with shared A4/Letter page settings. **Export PNG** creates a [schematic image](docs/schematic-image.md) from the same Output page settings. [Print captured page](docs/printing.md) submits the preview to a chosen macOS printer through shared commands, with observable spool jobs and cancellation. The owner confirmed that one LED page printed clearly and completely; measured margins/media/scale, other settings and device paths remain unqualified. [Print evidence](research/ui-audit/first-version/physical-print/checkpoint.json).

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

Choose **PNP high-side LED**, **NPN emitter follower**, or **Complementary transistor buffer** from Examples to explore both bipolar polarities and emitters above/below ground. Signed VBE/VCE, base/collector/emitter currents, live transient simulation, saved captures and the same user/agent commands are supported. [Models, examples and electrical checks](docs/bipolar-transistors.md).

## Explore an all-discrete class-AB amplifier

Open **Examples**, search for **class AB**, and choose **All-discrete class-AB
amplifier** under **Amplifiers & filters**. The eight-BJT circuit includes a
differential input, current mirror, tail sink, voltage-gain stage and biased
complementary output with feedback. It uses ±9 V rails and a 100 mV peak / 10 Hz
input to produce roughly 1.1 V peak into 100 Ω. The combined NPN/PNP capacity
is 16, including embedded optocoupler phototransistors.

The six default traces show input/output, Q6/Q7 collector currents, tail current
and compensation-capacitor voltage. Compare the two output currents through a
zero crossing, or change RF to explore feedback gain. The capacitor starts at
an explicit 7.55 V biased condition; generic models omit physical thermal,
breakdown and internal charge behavior. [Circuit walkthrough, probes and model
limits](docs/discrete-class-ab.md), [test evidence](docs/validation/discrete-class-ab/README.md).

Agents load the same template with `example.load {"id":"discrete-class-ab"}`
and use existing drawing, value, scope, simulation, history and file commands.
The reproducible workflow verifies captures, rebuilds the circuit from blank,
and exports a recorded project plus SVG/PNG:

```sh
python3 scripts/discrete-class-ab-demo.py \
  --agent build/elab-agent --output build/my-class-ab-experiment
```

Choose a new output directory for each run. By default this uses an independent
headless session; it does not change an open desktop document.

## Explore diodes and timers

Choose **Zener voltage regulator** or **Schottky rectifier** from Examples to explore reverse breakdown and lower forward loss. Both parts support drawing, live simulation, signed probes, saving/exports and the same agent commands. [Models, examples and electrical checks](docs/special-diodes.md).

Choose **555 LED flasher**, **Adjustable 555 oscillator**, or **555 one-shot timer** to watch capacitor charge control a timer latch and LED. The eight-pin component supports live simulation, external control/reset, shared user/agent commands, captures and exports. [555 model and examples](docs/timer555.md).

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

The [formative-study packet](research/first-use-study/README.md) prepares the PRD’s ten-person hobbyist validation with task cards, blank records and checked fixtures. It is unrun and does not replace owner, accessibility, performance or release acceptance.

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke --smoke-timeout 2m
```

The checks require Python 3 and first run pinned project-local Staticcheck and goimports for both modules, then the Python demo-safety tests and Go vet/race tests. The portable core retains standard-library-only runtime dependencies; lint tools live in their own module. The checks include electrical reference/conservation, invalid values, stale solve rejection, pause/edit/resume, undo/recovery, revision conflicts, retry deduplication, local socket access, and UI/agent equivalence. The native smoke verifies GPU schematic pixels, LED value editing, RC scope stepping, and NPN DC/pulse switching, the autonomous RC/NPN flasher, and selectable probes, timing cursors, and scope pixels through actual UI callbacks. Socket tests need permission to bind a local Unix socket; they cannot run inside a sandbox that forbids binding.

Both desktop race suites use a 30-minute test-runner timeout. The offscreen run
includes the complete desktop suite and all mounted-control tests. This deadline
does not change simulation integration, accuracy or work limits.

Connected standalone demos require a fresh owned blank document; they refuse existing work without closing its host. General agent clients remain unrestricted. See [demo session admission and validation](docs/demo-sessions.md).

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

The current engines support series LED DC, exact first-order RC, NPN/PNP transistor networks, coupled RC/NPN and bounded passive/switched circuit networks. Unsupported topologies report explicit diagnostics. Scope probes/cursors, recorded verification, recovery, atomic batches and portable exports are implemented. Participant usability, accessibility and distribution qualification remain upcoming work. DC results remain stable until input changes; transient results advance in simulated time with a bounded live trace.

- `workbench/`: portable circuit model, LED/RC/NPN solvers, shared semantic commands, revisions, history, and snapshots.
- `transport/`: bounded JSON-line stdio and private local socket adapters.
- `cmd/elab-agent/`: headless session and reference external client.
- `desktop/`: independent native module using Vellum, controlled UI, command bindings, and schematic rendering.
- `research/`: evidence, [PRD](research/07-product-requirements.md), and [Vellum integration decision](research/08-vellum-integration.md).

Vellum's own code is MIT licensed; Cogent and other native dependencies retain their upstream licenses. The private SDK archive retains its license inventory, notices and private-build marker. Application licensing has not been selected.

Autosave and crash recovery are available in the Inspector. Snapshots preserve dirty edits separately from saved files; restoring opens a new copy for Save As. See [recovery commands and limits](docs/recovery.md). Native startup enables autosave by default; headless sessions opt in with `--recovery-directory` or the shared configuration command.

Open exported recorded JSON from the Verify inspector to inspect/play its captured samples in a separate Recorded scope. Live state remains independent. [Recorded viewer commands and limits](docs/recording.md) describe exact edge stepping, probes and timing measurements.

The unreleased [recording consistency repair](docs/recording-consistency.md) rejects contradictory final MOSFET/storage readings and probe values through the same user/agent import and project-load paths, preserving existing work on failure.

[Recorded viewer controls](docs/recording-viewer-controls.md) document exact pulse-edge row navigation, playback, windows, captured probes/cursors and kept results through the same user/agent commands. Filtered discovery describes eleven more controls; windows that collapse after boundary normalization now refuse without changing work.

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

The tested envelope supports 64 simulated parts, 100 electrical nodes including ground and 16 combined NPN/PNP devices, within 100 drawing objects and 300 wires. [Capacity definitions, per-device bounds and validation](docs/circuit-capacity.md).

Fresh network wiper and running input changes now return background work immediately; the Inspector exposes status and cancellation. Agents inspect `input_work` or wait before dependent commands. See [input work](docs/input-work.md).

Project Open reads and validates files in the background with shared phase/status and exact-ID cancellation. Agents can use immediate `project.load.start` or the existing waiting `project.load`. See [project work](docs/project-work.md).

Save now captures the committed project and writes in the background; later edits stay unsaved. Users and agents share preparation cancellation, an explicit publication boundary and bounded project-job inspection. See [captured saves](docs/project-save.md).

Schematic SVG, live scope CSV and recorded JSON/CSV exports now encode/write a captured source in the background. Native controls and agents share job inspection and preparation cancellation. See [captured exports](docs/project-export.md).

Shared export workers protect the saved schematic even when another path reaches it through a directory alias or same-file link. Filtered discovery documents six export/page controls, including complete page overrides, queued versus waiting completion and atomic file outcomes.

Recorded-result import now validates in the background, preserves live editing and later viewer choices, and rechecks kept-record capacity before attachment. Users and agents share jobs/provenance/cancellation. See [record import](docs/project-import.md).

Manual recovery refresh and checkpoint now run as shared background jobs, preserving selection, current edits and newer recovery lists. See [recovery work](docs/project-recovery-work.md).

Recovery Restore validates files in the background and commits only if the original document still matches. Native controls and agents share cancellation, unsaved protection and new-copy provenance. See [recovery Restore](docs/project-restore.md).

The reference [agent conformance harness](docs/agent-conformance.md) builds LED, RC and NPN circuits from empty sessions through in-process, stdio and private sockets, including native app processes. It checks fault repair, measurements, exports, close/reopen and equivalent project/result state. The standard build produces `build/elab-conformance`.

Use `capabilities.list {"command":"source.pulse.set"}` for [detailed electrical parameter discovery](docs/capability-details.md), including normalized time bounds, coupled limits and examples. The ordinary catalog query remains available.

The [native source inventory](research/13-native-command-inventory.md) checks literal routes and tracks dynamic binding gaps. Rendered actions can use the shared [document-context guard](docs/document-context-guard.md) to refuse stale edits after background document replacement.

[Committed native fields](docs/rendered-fields.md) retain their rendered command/document context, so old waveform or selector callbacks cannot overwrite newer agent work.

Time fields now accept values such as `10 ms` and `250 us` through the same native/agent commands. Numerical seconds, strict saved data and existing simulation bounds remain compatible. See [time entry](docs/time-entry.md).

## Appearance and agent observation

The header offers Light, Dark, System and Apollo [themes saved on this device](docs/appearance.md), and File/Examples/Theme open native macOS menus. Apollo uses near-black blue panels, cyan outlines and compact monospace labels inspired by the owner's instrument-console reference. Agents use `appearance.theme.set` for the same preference. [Coalesced observation](docs/events.md) lets a separate `events.poll` connection wait for state/job changes without blocking commands. [UI audit screenshots](research/ui-audit/README.md) record the visible repairs and remaining limits.

Use **Hide scope** in the scope toolbar to give the schematic more room. **Show scope** restores the live or recorded plot with its measurements intact. Agents use the same [scope visibility command](docs/scope-visibility.md).

For frozen independent LED/RC/NPN references and flasher output-step refinement, run the [electrical qualification suite](docs/numerical-qualification.md) with `elab-conformance --electrical-suite`.

For repeated fresh/warm backend edit/result and bounded Verify timings, run the [performance qualification suite](docs/performance-qualification.md) with `elab-conformance --performance-suite --repetitions 20`. Visible UI acknowledgment is measured separately.

Transient failures now expose [distinct integration-limit causes and recovery](docs/integration-limits.md) through shared native/agent guidance. The original 512-attempt and 100 ms limits remain unchanged.

Waveform sources now show [shared schematic/export annotations](docs/source-annotations.md) for sine parameters, pulse timing and applied/pending/stale input state. Fit includes the annotation block.

[Captured DC sweeps](docs/dc-sweeps.md) plot source bias, resistance or wiper position without changing live physics. **DC sweep → Open sweep JSON…** reopens shared curves, including on a blank sheet. Users and agents share import, trace/cursor, keep, export and project-save commands. [Schematic PDF/PNG output](docs/schematic-pdf.md), [print preview](docs/print-preview.md) and [local printer submission](docs/printing.md) are available; one LED page has [owner-confirmed readability and completeness](research/ui-audit/first-version/physical-print/checkpoint.json), while other printer settings/device paths remain unqualified. [Captured-run comparisons](docs/recording-comparison.md) now include optional dashed reference traces on shared axes, without alignment or extrapolation.

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

Recovery snapshots now distinguish [active, inactive and unverified owners](docs/recovery-ownership.md). An unreadable ownership record preserves another session's copy and blocks dismissal, while restoration as a new copy remains available through the same user/agent commands.

[Repeated recovery timing](docs/recovery-timing.md) passes 240 paused/live native/headless measurements on the recorded workstation: maximum 2.544 s with ordinary two-second autosave, plus 20 complete crash restores. That timing checkpoint used unchanged runtime binaries.

[Automatic storage errors and retry results](docs/recovery-failure.md) now remain observable while you edit. Restoring available storage saves a complete recovery copy automatically, while later dirty edits, manual saves and newer recovery work keep their own state.

You can [disable autosave even while its current recovery folder is unavailable](docs/recovery-disable.md). The last error/list stay visible, and restoring storage leaves autosave off until you enable it again. Users and agents use the same commands.

Basic powered NOT, AND, OR, NAND, NOR, XOR and XNOR gates are in the Logic
category. Try the inverter LED, switch-controlled AND and half-adder examples;
see [logic gates](docs/logic-gates.md) for wiring, commands and model bounds.

D, JK and T flip-flops plus binary and decade counters share the Logic browser,
live simulation and agent commands. Five examples show sampling, divide-by-two,
enable/hold and asynchronous reset. See [sequential logic](docs/sequential-logic.md)
for pins, startup behavior, electrical limits and verification.

The [seven-segment display and BCD decoder](docs/digital-display.md) add a
555-driven decimal counter example, individual simulated segment currents and
shared native/agent controls.

[Momentary buttons](docs/momentary-buttons.md) support press/release, automatic
clicks, contact bounce, manual decimal counting and an RC debounce comparison.

[Comparators with hysteresis](docs/comparators.md) add threshold detection and
an analog sine signal driving a clean digital counter clock.

A [SPDT relay](docs/relays.md) adds a resistive/inductive coil, pickup/dropout
hysteresis and isolated contacts, with pushbutton and pulse-driven transistor
examples that show flyback current. [Linear and LDO regulators](docs/voltage-regulators.md)
add finite load regulation, dropout and current limiting; examples compare
headroom and capacitor charging/load steps. All eight new components use the
ordinary placement, wiring, values, probes, simulation and project commands.

[NTC thermistors and light-dependent resistors](docs/resistive-sensors.md)
add temperature/light sliders and exact environment entry, live resistance
readings, an automatic night-light and a relay thermostat. Native controls and
agents share `sensor.temperature.set` / `sensor.light.set`; fresh transient
changes preserve capacitor voltage, coil current and accepted time.

[Batteries with voltage sag and stored charge](docs/batteries.md) add internal
resistance, capacity and startup-charge controls, live charge/OCV/current/energy
readings, and LED-fade / regulator-dropout examples. Native controls and agents
share `battery.configure`; discharge uses the existing adaptive storage solver.

[Brushed DC motors](docs/dc-motors.md) add armature RL/back EMF, shaft inertia,
RPM and live viscous load. A 555/MOSFET PWM example includes flyback protection
and an explicit snubber. Native controls and agents share motor configuration
and staged load commands; failed/canceled work preserves current and speed.

[AC frequency sweeps](docs/ac-frequency-sweeps.md) add normalized small-signal
gain/phase Bode curves, exact sampled-point inspection and half-power estimates,
with RC, RLC and op-amp filter examples. Native controls and agents share the
existing sweep commands, captured-data export and saved-project workflow.

[Optocouplers](docs/optocouplers.md) add separate LED/phototransistor ports,
CTR and smooth output saturation, live port readings and an isolated LED-switch
example. Inspector and agents share `optocoupler.configure`; each port has its
own voltage/current probes and power accounting.

[Transformers and coupled inductors](docs/transformers.md) add turns ratio,
mutual flux, dot polarity, winding resistance and shared magnetic energy.
Examples compare loaded winding polarity and build a 50 Hz transformer / full
bridge / smoothing capacitor / regulated 3.3 V supply with explicit snubber and
reference parts. Inspector and agents share `transformer.configure`.

[H-bridge motor drivers](docs/hbridge.md) add forward/reverse, active braking,
coasting and enable-pin PWM with passive flyback paths. The reversible motor
example uses ordinary switches and a finite pulse source; users and agents share
placement, wiring, switch, source, probe and project commands. Live readouts show
mode, differential output voltage, signed supply current and driver dissipation.

Maintainers can use the [quality and release tooling](docs/release-tooling.md)
for repeatable local gates and unsigned candidate qualification. The
[architecture guide](docs/architecture.md) explains the shared command boundary,
compilation ownership and analysis-job store. Agents can inspect retained data
through bounded [analysis pages](docs/analysis-pages.md).
