# ELab first-version review · 0.125.0 /142

This is the updated owner-review alpha. Open [ELab.app](../build/ELab-0.125.0-142-review/ELab.app) in the [review folder](../build/ELab-0.125.0-142-review). The folder includes the matching `elab-agent`, three example projects and framework notices. Go, Python, a Vellum checkout and an account are not required to use the app. The canonical app and both earlier review folders (0.123.0 /140 and 0.124.0 /141) are preserved.

Before opening the replacement, save your work and quit the earlier review app when ready. Its open window was left untouched. Version 0.125 fixes accumulated live-clock rounding that could stop a valid pulsed RC/MOSFET circuit near 1.1 seconds. It includes the earlier routing-cache, retained-value-error and print-label repairs. Models, project format, commands and the pinned Vellum SDK are unchanged.

The reference workflows and source quality gate pass. The copied app and agent also pass native smoke and four-adapter public conformance with repository/SDK/toolchain/cache reads and outbound IP denied on the recorded Mac. This does not declare the full PRD MVP complete.

The exact copied replacement passes native smoke, all four public adapters, LED/RC/flasher graph and numerical parity, and the repaired pulse case through 3.41 seconds with a nonzero Pause/Resume and fresh save/reopen. Earlier actual native interaction through 40.9 seconds used the identical repaired source in a private development build. These checks do not establish visible-latency percentiles. [Replacement receipt](ui-audit/first-version/corrected-review-checkpoint.json).

**Newly found concurrency limitation in this package:** a retained Pause/Resume/Restart button from before an agent replaces the document can act on the replacement experiment. The source repair passes stale-context regressions, the full quality gate and fresh native focus checks, but is not yet included here. [Repair and replacement status](../docs/validation/rendered-simulation-buttons/checkpoint.json).

## Try these four tasks · 15–20 minutes

1. **Build an LED loop.** Start on the blank sheet. Place a DC source, resistor, red LED and ground. Use 5 V and 1 kΩ. Wire source positive → resistor → LED anode, LED cathode → ground, ground → source negative. Expect about **2.917 mA** with the generic model. Change the resistor to `2k`: about **1.477 mA**. Undo/redo, delete a wire and reconnect it. An incomplete drawing must not present old readings as current.
2. **Watch an RC circuit.** Choose Examples → Load RC example, then Resume. Pause, Restart and Step ten samples: capacitor voltage should be about **3.161 V at 10 ms**. Try Charge/Discharge, change a value, and inspect the paused/stale indicator. Scope controls scroll independently below the plot; drag the divider to give either area more room.
3. **Measure the flasher.** Choose the capacitor-timed flasher from Examples. Resume and watch both LEDs. Pause after several cycles; place A/B cursors at visible waveform transitions and read Δt. Select a collector voltage or LED current probe. Change the capacitor pair from 10 µF to 20 µF and restart/resume: timing should slow. Fit provides an overview; zoom in to inspect a dense drawing.
4. **Keep and share it.** Save As into your own project folder, close/reopen it, and export SVG or PDF plus scope CSV. Try Cancel on an unsaved-change prompt. Open Print preview; physical-printer qualification remains pending. The included Examples files are copies for exploration; Save As keeps their starting versions available.

Note the task, action and visible result when something fails. A screenshot and the saved `.elab` file are useful. Mark anything that felt confusing even if it eventually worked.

## External agents

From the review folder, `./elab-agent` accepts one JSON request per line and replies on stdout. Start with:

```json
{"version":1,"id":"catalog","command":"capabilities.list","params":{}}
{"version":1,"id":"place-help","command":"capabilities.list","params":{"command":"component.place"}}
{"version":1,"id":"state","command":"project.inspect","params":{}}
```

Commands use the same validated state as the UI. Mutations require a unique retry `key` and the current `expected_command_revision` (or the documented revision guard); use `project.inspect` to read current state and document identity. `simulation.wait` waits for accepted results after operations such as loading a project. Command discovery describes input schemas and errors.

To control the visible app, close the current review window, then launch a new instance from the review folder with a fresh private endpoint:

```sh
ELAB_REVIEW_SESSION=$(mktemp -d "${TMPDIR:-/tmp}/elab-review.XXXXXX")
./ELab.app/Contents/MacOS/ELab --agent-endpoint "$ELAB_REVIEW_SESSION/endpoint.json"
```

In another terminal, run `./elab-agent --connect /absolute/path/to/endpoint.json`. Use the endpoint path from the first terminal. Do not start a separate headless agent when the intent is to edit the open window.

## Review boundary and known limits

- Tested on this Apple Silicon Mac with macOS 26.6.2. The enforced minimum is 1040×720 in the mounted UI's units; the observed native frame is **1053×762 pixels**. App windows at 1280×800 and 1440×900 were inspected earlier. OS display modes, other hardware and monitor transitions remain unqualified.
- A dense 64-part/100-node stress circuit reaches the existing 100 ms integration work limit before producing a sample; the committed graph and physical state are preserved. Supported size does not guarantee convergence.
- Educational generic device models have the limits shown in the app. This is not general SPICE compatibility or PCB layout software.
- Complete VoiceOver, IME/fast typing, visible p95 latency under difficult solves, clean-account installation and the ten-person study remain open. A real Brother queue [completed one LED test job](ui-audit/first-version/physical-print/checkpoint.json); the owner confirmed on October 18 that this page printed clearly and completely. Measured margins, media, scale, other output settings and device cancellation remain unqualified.
- Some visible inspector controls can be omitted from external accessibility snapshots. The native Cocoa selector check passes, but complete external-reader acceptance remains open.
- This is a private unsigned build; notarization and public distribution/native-rights qualification remain open. The versioned review folder is separate from `build/ELab.app` at 0.105.0 /120.

The [delivery plan](34-first-version-plan.md), [updated build receipt](ui-audit/first-version/corrected-review-checkpoint.json) and [native walkthrough/agent evidence](ui-audit/first-version/README.md) track remaining acceptance work. This handoff continues owner feedback; it does not reopen feature scope.
