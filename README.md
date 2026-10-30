# ELab

**Draw a circuit. Watch it work. Measure what changes.**

ELab is a hobbyist electronics workbench built with [Vellum](https://github.com/dshills/vellum).
Build schematics from scratch, change component values, and explore live voltage,
current and timing. The desktop app and AI agents use the same validated commands
for drawing, simulation, analysis, saving and export.

**Current source: 87 working examples · 61 component kinds · 229 shared commands.**

![ELab showing an RC circuit, capacitor inspector, charge/discharge traces and timing cursors in the Light theme](docs/images/rc-scope.jpg)

*Circuit screenshots illustrate earlier builds; current inspectors group additional settings under Details and Advanced.*

*RC charge and discharge: edit the circuit above, inspect the waveforms below.
Screenshots show an earlier panel layout; the current drawer controls are described below.*

## Experimental Native UI

The `exp-ui` branch rebuilds the workspace from the supplied `exp-design`
references: full-height side rails, a centered toolbar, floating drawing tools,
and a Scope/Spectrum/Sweep dock. Existing user and AI agent commands, simulation
models, local files and exports are retained. This experiment is kept separate
from main. See [design QA](design-qa.md) and the
[validation record](docs/validation/experimental-native-ui/README.md).

## What you can do

- **Draw and edit schematics.** Search components by name or hobbyist alias,
  browse Common or topic categories, preview a snapped component before placing
  it, and connect named terminals. Move or rotate parts with their wires attached.
  Use junctions, net labels, keyboard shortcuts, Undo/Redo and reusable circuit blocks.
- **Simulate as you build.** Watch supported analog and digital circuits evolve;
  pause, restart or step to an exact simulated time. Inspect signed voltage/current,
  transistor operating states, stored energy, motor speed and other device readings.
- **Measure in time and frequency.** Switch between Scope and Spectrum using the
  same selected probes. Inspect timing cursors, live FFT, ranked peak markers and
  A/B frequency/amplitude cursors. Hold a spectrum while simulation continues.
- **Run experiments.** Capture an independent Verify run, compare recordings,
  measure intervals and threshold timing, and explore DC sweeps, AC/Bode response,
  harmonic distortion, small-signal noise, tolerances and feedback response.
- **Keep and share your work.** Save `.elab` projects with supported model settings
  and kept recordings. Export schematics as SVG, PNG or PDF, preview and print
  pages, and export measurements as CSV or JSON. Recovery and unsaved-work
  protection help preserve edits.
- **Work with agents.** Discover commands and schemas, inspect circuit state,
  preview atomic edits, apply guarded changes, and observe jobs/events through
  headless stdio or a private local connection to the running desktop app.
- **Choose your workspace.** Light, Dark, System and Apollo themes, native macOS
  menus, a searchable component browser, docked Parts/Inspector drawers, a
  focused Simulate tool chooser, and an adjustable schematic/instrument split.

See the [drawing workflow](docs/workbench-workflows.md),
[component browser](docs/component-browser.md), [workspace](docs/workspace-layout.md)
and [shared command guide](docs/agent-api.md).

## Drawers and Simulate tools

**Parts** and **Inspector** open independently and move the canvas aside. The
Parts sidebar button sits above the library and moves into the toolbar when
closed. The Inspector button stays at the toolbar’s right edge. Both announce
Open/Close and free their horizontal space for drawing when closed.
Drag an inside divider to resize, or toward the outside edge to snap closed.
With divider focus, Home closes Parts and End closes Inspector. Reopening
retains the preferred width. Layout choices last for the session and do not
mark the circuit dirty or become part of a saved schematic.

The **Simulate** inspector shows one tool at a time: **Live controls**, **Recorded
runs**, **Recorded spectrum**, **DC / AC / noise sweeps**, **Component tolerances**
or **Circuit details**. The chooser and primary Run/Record action stay above the
scrolling form. Essentials appear first; **Advanced** keeps the remaining settings. Switching
tools keeps their accepted settings/results and does not start an analysis or
stop live simulation. The live Scope/Spectrum dock remains available below.

**Inspect** puts value and common actions first, with model/startup/geometry under
**Details**. **Measure** uses one source chooser; the bottom strip follows that
measurement, with full readings and the named circuit overview still available.

Try the [self-guided first-use kit](docs/first-use-test-kit.md) for LED drawing,
RC recording, flasher measurements and saving/exporting. It includes a blank
feedback form; [expected results](docs/first-use-maintainer-key.md) are separate.

Users and agents control the same state through `workspace.details.set`, `workspace.panels.set` and
`workspace.simulation.set`; electrical commands still work with either drawer
closed. See the [workspace reference](docs/workspace-layout.md) for parameters,
keyboard behavior and layout limits.

## Scope and Spectrum

Both tabs stay accessible for every circuit, even with the dock collapsed. DC
and empty circuits explain why no time-based data is available and point to
transient examples or **Simulate → Recorded runs**. Opening an instrument never
changes the circuit or starts a recording.

The **Scope** plots up to six selected voltage/current channels. Place A/B cursors
by clicking, dragging or entering times; inspect timing differences, interval
statistics and recorded experiments without changing the live circuit.

The **Spectrum** tab reuses those probes for live FFT or a separately captured
recording. It offers window/sample controls, a declared fundamental, harmonic
measurements and THD when the sample interval supports them. The newest additions
are **P1–P5 peak markers** and **A/B frequency/amplitude cursors**, with frequency,
peak-amplitude and dB differences. Cursor frequencies snap to actual FFT bins;
controls stay in place as frames arrive.

![ELab in Dark theme showing a clipped amplifier and a held live spectrum with ranked peaks, A/B cursors and frequency/amplitude differences](docs/images/spectrum-cursors.jpg)

*Clipped-amplifier output: the held live FFT shows the fundamental and harmonics.
A/B cursors compare the 10 Hz and 30 Hz bins.*

[Scope/probes](docs/scope-probes.md) · [Spectrum and cursor commands](docs/spectrum-analysis.md) ·
[Recorded measurements](docs/interval-measurements.md) · [AC sweeps](docs/ac-frequency-sweeps.md)

## Electronics library

| Area | Components and capabilities |
| --- | --- |
| Sources and power | DC, sine and pulse sources; current sources, electronic loads, batteries with charge/sag, rectifiers and voltage regulators |
| Passive circuits | Resistors, potentiometers, capacitors and inductors; optional losses; transformers and coupled inductors |
| Semiconductors | LEDs, silicon/Schottky/Zener diodes, NPN/PNP BJTs, N/P MOSFETs, JFETs, SCRs, TRIACs, optocouplers and photodiodes |
| Analog and timing | Powered op-amps, comparators with hysteresis, 555 timers, configurable transistor profiles, temperature effects and self-heating |
| Digital | Basic gates, flip-flops, counters, BCD decoding and seven-segment displays |
| Interaction and sensors | Switches, momentary pushbuttons with bounce, thermistors, light-dependent resistors, temperature/light controls and debounce experiments |
| Motion and loads | DC motors, PWM and H-bridge forward/reverse/brake/coast control, relays with flyback experiments, loudspeaker electrical loads and explicit audio preview |
| Schematic helpers | Grounds, junctions, named nets, pinned live readings and reusable circuit blocks |

Models have explicit supported domains and work limits. The current envelope is
**64 simulated parts, 100 electrical nodes, 100 drawing objects and 300 wires**,
with device-specific bounds, including **16 combined NPN/PNP transistors**.
See [circuit capacity](docs/circuit-capacity.md), [electronics models](docs/hobbyist-devices.md)
and [numerical qualification](docs/numerical-qualification.md).

### From a first LED to a discrete amplifier

Open **Examples** to browse topic categories, search across all 87 circuits, and
see schematic previews, a question to explore, an experiment and model limits.
Opening an example retains Save/Discard/Cancel protection for your current work.

Good starting points:

| Example | Try this |
| --- | --- |
| LED current limiter | Change resistance and compare current and illustrative brightness. |
| RC charge and discharge | Change capacitance, watch the time constant and measure with cursors. |
| Capacitor-timed LED flasher | Compare the two transistor/LED branches and their switching times. |
| Amplifier clipping and harmonic distortion | Reduce input amplitude, then compare the waveform, harmonics and THD. |
| Reversible PWM motor driver | Compare forward/reverse drive, PWM, braking and coasting. |
| All-discrete class-AB amplifier | Inspect a differential pair, current mirror, gain stage and complementary output. |

![ELab in Apollo theme showing the complete eight-transistor all-discrete class-AB amplifier and live NPN transistor readings](docs/images/class-ab-apollo.jpg)

*The class-AB example uses eight BJTs and 27 simulated parts, with no op-amp
component. Apollo gives the workspace its cyan instrument-console appearance.*

[Example browser](docs/example-browser.md) · [Discrete amplifier](docs/discrete-class-ab.md) ·
[Motor driver](docs/hbridge.md) · [Themes](docs/appearance.md)

## Build and run

The desktop target is **macOS on Apple Silicon**. Packaged candidates have been
tested on macOS 26.x and 27.2. The portable Go simulation core and headless agent
do not require Vellum, a window system or a GPU.

For the desktop build, install **Go 1.25.6 or newer**, **Python 3.10 or newer** and
**Xcode Command Line Tools**, then run:

```sh
git clone https://github.com/dshills/ELab.git
cd ELab
./scripts/build.sh
./build/elab
```

The build verifies and extracts the repository's pinned Vellum SDK and its
compatibility patches. A sibling Vellum checkout is not required. If invoking
native Go tools directly, run `python3 scripts/prepare-vellum.py` first.
See [the SDK pin and preparation details](third_party/vellum/README.md).

Start with **Examples → LED current limiter** or **RC charge and discharge**.
To build from blank, select a component from the left browser, place it on the
sheet, then choose **Wire** and click terminal dots. Select a part to edit values
such as `2.2k Ω` or `1 µF` in Inspect.

With the canvas focused: **W** Wire, **V** Select, **H** Pan, **R** Rotate,
**Delete** remove, and **Escape / Cancel** end an unfinished gesture.
**Cmd+Z / Cmd+Shift+Z** undo/redo when a text editor does not consume the shortcut.
**Fit** follows the actual schematic workspace as it resizes.

For a local double-clickable development bundle after building:

```sh
python3 -B scripts/package_app.py --binary build/elab \
  --output build/ELab-local/ELab.app --version 0.0.0 --build-version 1
```

Use a new output path for each bundle. This produces an unsigned/ad-hoc local
build, not a Developer ID signed or notarized installer. Historical local package
paths in validation records are not GitHub downloads. Build from source or use
the qualified alpha described below.

The current **0.203.0 /290 unsigned alpha** is available from
[GitHub Releases](https://github.com/dshills/ELab/releases/tag/v0.203.0-alpha.1).
[Download the macOS Apple Silicon ZIP](https://github.com/dshills/ELab/releases/download/v0.203.0-alpha.1/ELab-0.203.0-macos-arm64.zip),
check [SHA256SUMS](https://github.com/dshills/ELab/releases/download/v0.203.0-alpha.1/SHA256SUMS.txt)
and follow the included **START-HERE.md**. The folder includes the matching agent,
all 87 examples, notices and the self-guided first-use kit. It has no Developer ID
signature or notarization. [Release details and qualification limits](docs/current-release.md).


## Use an AI agent

Standard MCP support is included in local 0.201.0 /285 and newer builds.
Run the matching agent for your app (or build the current source) with
`--mcp-config`, then use the printed configuration. It starts a local headless
circuit with full command discovery, inspection and guarded command dispatch.
Add `--connect` when generating
the configuration for an explicitly started desktop endpoint.
[Setup, examples and lifecycle](docs/agent-setup.md).

The raw JSON protocol remains available. Run a headless session:

```sh
go run ./cmd/elab-agent
```

Send one JSON request per line; receive one JSON response per line. For example:

```json
{"version":1,"id":"catalog","command":"capabilities.list","params":{}}
{"version":1,"id":"inspect","command":"project.inspect","params":{}}
```

Agents can discover detailed input schemas, component/model limits, command
preconditions and asynchronous job outcomes. Mutations carry an observed
revision and a retry key; document guards protect against stale work. Drawing,
measurement and analysis use the same service as the desktop controls.

To let an external agent work with a desktop session:

```sh
./build/elab --agent-endpoint "$PWD/build/session.endpoint.json"
```

In another terminal:

```sh
./build/elab-agent --connect build/session.endpoint.json
```

Use a new descriptor path. The descriptor is a private local credential; keep it
out of shared files and logs. Normal shutdown removes it. Persistent headless
socket sessions are also supported through `elab-agent --listen`.

[Agent protocol](docs/agent-api.md) · [Command discovery](docs/capability-details.md) ·
[Events](docs/events.md) · [User/agent conformance](docs/agent-conformance.md)

## Development and checks

```sh
# Portable core, protocol and tooling checks
ELAB_QUALITY_PROFILE=portable ./scripts/check.sh

# Complete macOS checks, including native and rendered UI suites
./scripts/check.sh

# Build and run the native drawing/simulation smoke harness
./scripts/build.sh
./build/elab --smoke --smoke-timeout 2m
```

The quality runner checks formatting/imports, Staticcheck, Python tooling,
Go vet and race tests. Electrical tests cover reference values, conservation,
transients and guarded failures; integration tests cover shared commands,
persistence, retries, background work and UI/agent parity.

[Drawer review and test evidence](docs/validation/drawers-20261029/README.md)
records the latest workspace correction. [Spectrum measurement evidence](docs/validation/spectrum-markers-20261028/README.md)
records the peak/cursor implementation. Native OS-input, clean-account install,
participant usability and signing/notarization gates remain separately recorded;
passing software tests do not close them.

| Code | Responsibility |
| --- | --- |
| `workbench/` | Circuit graphs, electrical models/solvers, shared command service and analysis jobs |
| `desktop/` | Vellum macOS workspace, schematic/scope/spectrum rendering and command bindings |
| `transport/` and `cmd/elab-agent/` | Headless JSON protocol and private local desktop connections |
| `cmd/elab-conformance/` | Cross-adapter workflow/electrical checks |
| `scripts/` | Builds, quality gates, examples, packaging and qualification tools |
| `docs/` and `research/` | User/API guides, engineering evidence and product research |

[Architecture](docs/architecture.md) · [Release checklist](docs/v1-release-checklist.md) ·
[Product requirements](research/07-product-requirements.md)

ELab is an actively developed educational simulator. Component models are
illustrative rather than manufacturer-qualified hardware models. Each device
and analysis guide describes its assumptions and limits. Audio playback is
explicit; opening or simulating a circuit stays silent.
