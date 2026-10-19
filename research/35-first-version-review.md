# ELab first-version review · 0.145.0 /162

The updated private owner-review alpha is in the [review folder](../build/ELab-0.145.0-162-review), with [ELab.app](../build/ELab-0.145.0-162-review/ELab.app), matching `elab-agent`, three example projects and framework notices. Go, Python, a Vellum checkout and an account are not required. All 22 earlier qualified review folders (0.123.0 /140 through 0.144.0 /161), the original unqualified /147 candidate and canonical 0.105.0 /120 app are preserved. Owner walkthrough remains “Not tried yet.”

Save your work and quit an earlier review before opening this replacement. Version 0.145 repairs MOSFET cutoff voltage convergence: a small current residual can hide a drain-voltage error near a weakly conducting LED load. The native app and external agent now require the existing Newton voltage-correction check in MOSFET networks. Device models, public commands, saved format, native bindings and Vellum SDK `b22478c` are unchanged. The preserved /161 PMOS example retains the earlier 3.218 mV discrepancy.

Clean source `d06251f` produces this app and agent. The full source quality gate passes (313.085 s), as do exact copied native smoke and all four public adapters with repository, SDK, toolchain, cache and outbound IP access denied. Copied LED/RC/flasher parity, value correction, live/recorded/two-capture measurements, pulse Pause/Resume and fresh save/reopen pass. [Qualification receipt](ui-audit/first-version/mosfet-precision-review-checkpoint.json).

The [source proof](../docs/validation/mosfet-cutoff-precision/README.md) retains the old failed result and independent scalar reference. The [copied native/headless precision check](../docs/validation/copied-mosfet-precision-162/README.md) checks every six-probe row of both live and Verify traces, including both pulse-edge sides, and four fresh public reopens. PMOS cutoff voltage agrees with the declared load line within 8.31×10⁻¹⁴ V, using the unchanged strict 1 mV check; no claim of physical-device accuracy follows. The [RC edge check](../docs/validation/copied-rc-edge-162/README.md) also preserves correct scope/cursor/extrema/CSV values at manual input changes. These are public-command integration checks, not new physical typing, pointer, reader or visible-latency acceptance. Historical native evidence retains its original build scope.

## Try these four tasks · 15–20 minutes

1. **Build an LED loop.** Start on the blank sheet. Place a DC source, resistor, red LED and ground. Use 5 V and 1 kΩ. Try typing a coordinate and clicking Place directly. Wire source positive → resistor → LED anode, LED cathode → ground, ground → source negative. Expect about **2.917 mA** with the generic model. Change the resistor to `2k`: about **1.477 mA**. Undo/redo, delete a wire and reconnect it. For value recovery, enter `0` for the resistor and Apply: the circuit should stay unchanged. Replace it with `2.2k` and Apply once: the error should clear and current should be about **1.345 mA**. An incomplete drawing must not present old readings as current.
2. **Watch an RC circuit.** Choose Examples → Load RC example, then Resume. Pause, Restart and Step ten samples: capacitor voltage should be about **3.161 V at 10 ms**. After charging to 10 ms, choose Discharge and Step. Capacitor voltage stays continuous; resistor current immediately changes from about +0.18394 mA to −0.31606 mA at the same 10 ms timestamp. Scope and CSV should retain both sides. Then change a value and inspect the paused/stale indicator. If an external agent restarts while a live cursor or interval edit is unfinished, that old draft should disappear; fresh input should measure the new run. Scope controls scroll independently below the plot; drag the divider to give either area more room.
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
- One native NMOS Pause click was followed by a still-running snapshot before the test host was terminated. A fresh transistor retry and [six single-click Pause/Resume cycles](ui-audit/first-version/native-pause-161/README.md) complete correctly, with stable paused samples and agent parity. The first event remains unexplained; report any Pause action that fails to stop time during the walkthrough.
- The original strict PMOS cutoff comparison failed in /161. Its convergence repair passes the current source and copied native/headless checks; the [historical observation](ui-audit/first-version/native-transistors-161/README.md) remains unchanged. Broader device/topology/physical accuracy is separate.
- Complete VoiceOver, IME/fast typing, visible p95 latency under difficult solves, clean-account installation and the ten-person study remain open. A real Brother queue [completed one LED test job](ui-audit/first-version/physical-print/checkpoint.json); the owner confirmed on October 18 that this page printed clearly and completely. Measured margins, media, scale, other output settings and device cancellation remain unqualified.
- The earlier /160 native invalid-value check retained edit controls in external AX traversal without coordinate fallback; that physical observation retains its original build scope. Complete reader, IME, other inspector states and OS interleavings remain unqualified.
- This is a private unsigned build; notarization and public distribution/native-rights qualification remain open. The versioned review folder is separate from `build/ELab.app` at 0.105.0 /120.

The [delivery plan](34-first-version-plan.md), [updated build receipt](ui-audit/first-version/mosfet-precision-review-checkpoint.json) and [native walkthrough/agent evidence](ui-audit/first-version/README.md) track remaining acceptance work. This handoff continues owner feedback; it does not reopen feature scope.
