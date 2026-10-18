# ELab first-version review · 0.141.0 /158

This is the updated owner-review alpha. Open [ELab.app](../build/ELab-0.141.0-158-review/ELab.app) in the [review folder](../build/ELab-0.141.0-158-review). It includes the matching `elab-agent`, three example projects and framework notices. Go, Python, a Vellum checkout and an account are not required. The canonical app, eighteen earlier qualified review folders (0.123.0 /140 through 0.140.0 /157) and the original unqualified /147 candidate are preserved. Owner walkthrough remains pending (October 18: “Not tried yet”).

Before opening the replacement, save your work and quit an earlier review app when ready. Version 0.141 includes a native text handoff repair: fallback keys stay ordered until Cocoa receives matching text and selection, preventing the reproduced lost/truncated/appended input. Three old-source failing Cocoa cases, driver ordering/validation guards and six native cursor entries plus an interval edit pass in the source development harness. SDK `274f5cc` changes eight source/test members, with prepared dependencies/bootstrap unchanged. Earlier live-run/capture input ownership, one-click cursor, invalid-value, schematic pointer/key, output-page, unsaved-choice, placement, slider and clock repairs are included. Public IDs/commands, models and project format remain unchanged.

The reference workflows and source quality gate pass. The copied app and agent also pass native smoke and four-adapter public conformance with repository/SDK/toolchain/cache reads and outbound IP denied on the recorded Mac. This does not declare the full PRD MVP complete.

The exact copied replacement passes live measurement reset, stale-revision refusal and fresh-edit native/headless parity, LED/RC/flasher graph and numerical parity, and the live pulse case through at least **3.4 seconds**, nonzero Pause/Resume and fresh save/reopen. Native and headless commands agree on saved recorded-cursor measurements and fresh reopening. Two distinct 10 kΩ/20 kΩ RC captures also retain their metadata and all 21 rows each across selection and fresh cursor/seek/window edits. A stale capture ID is refused without mutation; a fresh 6 ms edit uses the selected new capture. Source-pinned Cocoa/FIFO regressions and a development native replay separately establish the text handoff repair; the six automated entries and interval are not general human typing/IME acceptance. Existing source-mounted tests prove that live cursor and interval drafts survive same-run repaint/sampling, reset across run replacement, and accept new input. Existing recorded cursor/seek/window controls retain capture ownership. Exact copied native smoke covers existing controls/rendering. Raw source evidence under Evidence retains its original repository-relative references. This does not establish exhaustive private input, physical OS/IME/reader or visible-p95 acceptance. [Replacement receipt](ui-audit/first-version/native-text-review-checkpoint.json).

## Try these four tasks · 15–20 minutes

1. **Build an LED loop.** Start on the blank sheet. Place a DC source, resistor, red LED and ground. Use 5 V and 1 kΩ. Try typing a coordinate and clicking Place directly. Wire source positive → resistor → LED anode, LED cathode → ground, ground → source negative. Expect about **2.917 mA** with the generic model. Change the resistor to `2k`: about **1.477 mA**. Undo/redo, delete a wire and reconnect it. For value recovery, enter `0` for the resistor and Apply: the circuit should stay unchanged. Replace it with `2.2k` and Apply once: the error should clear and current should be about **1.345 mA**. An incomplete drawing must not present old readings as current.
2. **Watch an RC circuit.** Choose Examples → Load RC example, then Resume. Pause, Restart and Step ten samples: capacitor voltage should be about **3.161 V at 10 ms**. Try Charge/Discharge, change a value, and inspect the paused/stale indicator. If an external agent restarts while a live cursor or interval edit is unfinished, that old draft should disappear; fresh input should measure the new run. Scope controls scroll independently below the plot; drag the divider to give either area more room.
3. **Measure the flasher.** Choose the capacitor-timed flasher from Examples. Resume and watch both LEDs. Pause after several cycles; place A/B cursors at visible waveform transitions and read Δt. Select a collector voltage or LED current probe. To check captured measurements, use Verify and view its recorded run. Edit cursor A’s time, then click B or Clear once; the edit and requested action should both complete. When selecting a different capture, its time fields should show that capture’s values rather than unfinished text from the previous one. Change the capacitor pair from 10 µF to 20 µF and restart/resume: timing should slow. Fit provides an overview; zoom in to inspect a dense drawing.
4. **Keep and share it.** Save As into your own project folder, close/reopen it, and export SVG or PDF plus scope CSV. Try Cancel on an unsaved-change prompt. Try typing a margin then clicking Print preview or Refresh directly; Fit should keep the margin while clearing explicit scale. The owner confirmed one LED page printed clearly and completely. Other print settings remain unqualified. The included Examples files are copies for exploration; Save As keeps their starting versions available.

Note the task, action and visible result when something fails. A screenshot and the saved `.elab` file are useful. Mark anything that felt confusing even if it eventually worked. If typing changes, drops or appends characters, record the intended text, visible text and what happened on Enter.

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

The [delivery plan](34-first-version-plan.md), [updated build receipt](ui-audit/first-version/native-text-review-checkpoint.json) and [native walkthrough/agent evidence](ui-audit/first-version/README.md) track remaining acceptance work. This handoff continues owner feedback; it does not reopen feature scope.
