# ELab first-version review · 0.151.0 /168

The private owner-review alpha is in the [review folder](../build/ELab-0.151.0-168-review), including [ELab.app](../build/ELab-0.151.0-168-review/ELab.app), matching `elab-agent`, three examples and framework notices. No Go, Python, Vellum checkout or account is required. All 27 earlier sealed reviews, original /147, failed /164 and canonical /120 are preserved. Owner walkthrough remains **Not tried yet**.

The [ZIP](../build/ELab-0.151.0-168-review.zip) and [checksum](../build/ELab-0.151.0-168-review.zip.sha256) preserve all 1,208 files/modes through macOS extraction. The extracted agent loads three examples and freshly reopens three saved copies under the denial policy. No new extracted native-window or LaunchServices qualification is claimed. [Evidence](../docs/validation/startup-reset-review-168/README.md).

Save and quit an earlier review before opening this replacement. Version 0.151 repairs Use project startup retaining a stored Verify coil-current override. Shared default settings now clear startup/input/sampling overrides while preserving the accepted duration, so the saved project’s startup supplies the next capture. User/agent revision guards and live physics/history remain intact. Public commands, models, format and SDK `b22478c` are unchanged.

Clean `0f9c984` native/agent/controller binaries pass source quality (213.058 s), copied native smoke (40.001 s), all four adapters (51.073 s) and public acceptance (17.982 s) under source/network/shared-geometry denial. RL/RLC captures start at the +30 mA scenario before reset and the saved −20 mA afterward, with exact native/headless data, independent RL equations, unchanged live state/history and four fresh saved-project reopens preserving both recordings. Existing reference/value/preset/measurement/live/save and the 424-row RC Verify regression pass. [Receipt](ui-audit/first-version/startup-reset-review-checkpoint.json).

The [source proof](../docs/validation/verification-startup-reset/README.md) retains original RL/RLC failures, final-test replay, retained-widget duration/reset input and deferred-agent refusal/fresh reset. Its pending-packaging statement is historical and superseded by /168. Private input remains source-mounted; copied tests exercise public settings projection. No new physical OS reset-input acceptance is claimed. Earlier physical, precision and RC edge/CSV observations keep their original scopes under `Evidence/Historical-167`; /167 stays preserved and affected by the stored-coil scenario.

A later source review reproduced a first-click issue in /168: editing a selected capacitor or inductor startup value and clicking its default button or Restart can commit the value but refuse the requested action as stale. The [source repair](../docs/validation/component-startup-handoff/README.md) passes source/development validation and awaits a clean replacement. This is separate from Verify’s Use project startup repair already packaged here.

## Try these four tasks · 15–20 minutes

1. **Build an LED loop.** Start on the blank sheet. Place a DC source, resistor, red LED and ground. Use 5 V and 1 kΩ. Try typing a coordinate and clicking Place directly. Wire source positive → resistor → LED anode, LED cathode → ground, ground → source negative. Expect about **2.917 mA** with the generic model. Change the resistor to `2k`: about **1.477 mA**. Try leaving unfinished value text and clicking a resistor/capacitor preset once; the chosen preset should apply. Undo/redo, delete a wire and reconnect it. For value recovery, enter `0` for the resistor and Apply: the circuit should stay unchanged. Replace it with `2.2k` and Apply once: the error should clear and current should be about **1.345 mA**. An incomplete drawing must not present old readings as current.
2. **Watch an RC circuit.** Choose Examples → Load RC example, then Resume. Pause, Restart and Step ten samples: capacitor voltage should be about **3.161 V at 10 ms**. After charging to 10 ms, choose Discharge and Step. Capacitor voltage stays continuous; resistor current immediately changes from about +0.18394 mA to −0.31606 mA at the same 10 ms timestamp. Scope and CSV should retain both sides. For a separate captured RC run, select Charge in Verify, type `20 ms` into Duration and click Verify once. With 0 V startup, expect about **4.323 V at 20 ms**, with live time unchanged. Try invalid duration text, then change Sample step: Verify should still refuse capture. Correct duration and click Verify once. Valid unfinished startup voltages should also apply; Use project startup should preserve duration while clearing its overrides. Then change a value and inspect the paused/stale indicator. If an external agent restarts while a live cursor or interval edit is unfinished, that old draft should disappear; fresh input should measure the new run. Scope controls scroll independently below the plot; drag the divider to give either area more room.
3. **Measure the flasher.** Choose the capacitor-timed flasher from Examples. Resume and watch both LEDs. Pause after several cycles; place A/B cursors at visible waveform transitions and read Δt. Select a collector voltage or LED current probe. To check captured measurements, use Verify and view its recorded run. Edit cursor A’s time, then click B or Clear once; the edit and requested action should both complete. For a pulse capture with threshold arrivals, type a threshold and choose Rising, Falling or Both once; the chosen direction and new level should both apply. For multiple pages, type a threshold and click Next once. Invalid text should keep its message; correction should work on the first click. When selecting a different capture, its time fields should show that capture’s values rather than unfinished text from the previous one. Change the capacitor pair from 10 µF to 20 µF and restart/resume: timing should slow. Fit provides an overview; zoom in to inspect a dense drawing.
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
- The original strict PMOS cutoff comparison failed in /161. Its convergence repair passes its recorded source/copied native-headless checks; the engine source is unchanged in /166; the [historical observation](ui-audit/first-version/native-transistors-161/README.md) remains unchanged. Broader device/topology/physical accuracy is separate.
- The owner [deferred ordinary default-startup and clean-account installation qualification](ui-audit/first-version/private-alpha-owner-decisions.json) for the private alpha on October 19. Neither has been tested; both remain broader requirements. Complete VoiceOver, IME/fast typing, visible p95 latency under difficult solves and the ten-person study remain open. A real Brother queue [completed one LED test job](ui-audit/first-version/physical-print/checkpoint.json); the owner confirmed on October 18 that this page printed clearly and completely. Measured margins, media, scale, other output settings and device cancellation remain unqualified.
- The earlier /160 native invalid-value check retained edit controls in external AX traversal without coordinate fallback; that physical observation retains its original build scope. Complete reader, IME, other inspector states and OS interleavings remain unqualified.
- This is a private unsigned build; notarization and public distribution/native-rights qualification remain open. The versioned review folder is separate from `build/ELab.app` at 0.105.0 /120.

The [delivery plan](34-first-version-plan.md), [updated build receipt](ui-audit/first-version/value-preset-review-checkpoint.json) and [native walkthrough/agent evidence](ui-audit/first-version/README.md) track remaining acceptance work. This handoff continues owner feedback; it does not reopen feature scope.
