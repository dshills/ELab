# ELab first-version review · 0.143.0 /160

**Known measurement blocker:** at a manual RC Charge → Discharge edge, scope and CSV repeat the pre-edge current although the solver current is correct. [Reproduction and scope](ui-audit/first-version/native-rc-160/README.md). [Source repair and full quality pass](../docs/validation/rc-input-frames/README.md); clean replacement qualification is pending. Do not promote this affected review.


This updated private owner-review alpha is in the [review folder](../build/ELab-0.143.0-160-review), with [ELab.app](../build/ELab-0.143.0-160-review/ELab.app), matching `elab-agent`, three example projects and framework notices. Go, Python, a Vellum checkout and an account are not required. All 20 earlier qualified review folders (0.123.0 /140 through 0.142.0 /159), the original unqualified /147 candidate and canonical 0.105.0 /120 app are preserved. Owner walkthrough remains “Not tried yet.”

Before opening this replacement, save your work and quit an earlier review when ready. Version 0.143 gives the inspector stable named groups so value controls remain discoverable through an invalid edit and correction. Existing control IDs, order, commands and revision/document guards stay the same. Vellum remains pinned to `b22478c`; models and saved format are unchanged. The earlier coordinate/text/run-ownership repairs remain included.

Clean source `4403097` produces this app and agent. The complete source quality gate passes; the exact copied native smoke and all four public adapters pass with repository, SDK, toolchain, cache and outbound IP access denied on the recorded Mac. Copied LED/RC/flasher graph and numerical parity, invalid-value correction, live/recorded/two-capture measurements, pulse Pause/Resume and fresh save/reopen pass. [Replacement receipt](ui-audit/first-version/inspector-recovery-review-checkpoint.json). This is not full PRD MVP completion.

An [actual /160 native check](../docs/validation/native-inspector-recovery-160/README.md) also keeps the field and Apply target available after `0` Ω is refused. AX-identified correction to `2.2k` and one Apply produce fresh **1.345 mA**, matching the exact packaged agent, with unchanged accepted state during refusal/private drafting and matching saved-file reopening. This uses a publicly seeded LED fixture in an isolated custom bundle with the byte-identical executable; it is not a new blank construction or whole-bundle reader test. The [source investigation](../docs/validation/inspector-accessibility/README.md) retains its development scope and does not attribute the earlier external omission to a proven framework defect. General VoiceOver/IME/fast typing, visible-p95 and the owner walkthrough remain open.

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
- The previously observed invalid-value path now retains edit controls in actual external AX traversal and needs no coordinate fallback. Complete reader, IME, other inspector states and OS interleavings remain unqualified.
- This is a private unsigned build; notarization and public distribution/native-rights qualification remain open. The versioned review folder is separate from `build/ELab.app` at 0.105.0 /120.

The [delivery plan](34-first-version-plan.md), [updated build receipt](ui-audit/first-version/inspector-recovery-review-checkpoint.json) and [native walkthrough/agent evidence](ui-audit/first-version/README.md) track remaining acceptance work. This handoff continues owner feedback; it does not reopen feature scope.
