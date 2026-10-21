# Try ELab 0.178.0 /195

The [separate review app](../build/ELab-0.178.0-195-candidate/ELab.app) is prepared for owner testing. It includes the public Vellum update, prior input fixes and the repaired blank-inspector transition. The existing /189 owner app and projects are preserved. The isolated engineering test window is closed; this guide does not assume an app is currently open. Save work before switching binaries, and use a separate project copy. /195 is not a promoted release.

Try the four tasks below when convenient. Start with the ordinary workflow; you do not need to run the detailed engineering regressions. Report a blocked action or a confusing control even if you find a way around it.

## 1. Draw and edit an LED circuit

Start a blank schematic. Place a 5 V DC source, 1 kΩ resistor, red LED and ground. Connect source positive → resistor → LED anode, then LED cathode → ground → source negative. You can click terminal dots or choose the named terminals in the wiring controls.

The generic LED model should report about **2.917 mA**. Change the resistor to `2k`; current should fall to about **1.477 mA** automatically. Try Undo and Redo, then remove and reconnect a wire. An incomplete circuit must not show an old reading as a current result.

## 2. Watch an RC circuit charge and discharge

Load the RC example from Examples. Pause, Restart, then Step ten samples. For the default 10 kΩ /1 µF circuit, capacitor voltage should be about **3.161 V at 10 ms**. Resume and watch the voltage change, then Pause again.

Try Charge and Discharge. Capacitor voltage should remain continuous when the input changes. Use the voltage/current probes and scope cursors to inspect the trace. Notice whether it is clear which probe, time and live or recorded result you are viewing.

## 3. Inspect the transistor flasher

Load the capacitor-timed LED flasher from Examples. Resume and watch both LEDs, then Pause after several cycles. Select collector-voltage or LED-current probes on both sides. Use A/B cursors to measure the time between matching transitions; Δt should agree with the plotted waveform.

Try Verify to create a recorded run and inspect it separately from the live run. Change both timing capacitors from 10 µF to 20 µF, restart and resume: the flashing should slow. Report any measurement whose target, units or meaning are unclear. Advanced timing-field recovery has known /189 issues listed below.

## 4. Save and share your work

Save As into your own project folder, reopen your copy and check the components, values and wiring. Export a schematic as SVG or PDF, and export scope data as CSV. Check that the exported drawing is complete and readable. Try Cancel on an unsaved-change prompt and confirm your edits remain.

The included Examples files are editable copies; Save As preserves their starting versions. One LED page has already been confirmed by you as printed clearly and completely; another print is optional.

## Known limits of this exact version

- The four complete physical workflows and full minimum/normal-size layout pass remain open. Native /195 checks cover the repaired inspector transition, LED editing/Undo/Redo, preservation decisions, Save/Open and SVG export. LED/RC/flasher graph and numerical comparisons also pass through the native endpoint and matching agent; those reconstructions use public commands.
- Representative recording/timing, Verify/reset and sweep correction paths passed with actual native input on /194. /195 carries the same controls and passes the full mounted source suite; additional native field branches remain on the checklist. Report a refusal or surprising target with the exact input and action.
- Generic educational models are not general SPICE compatibility or PCB design. Complete accessibility/IME, broader environments/performance, the hobbyist study and public distribution remain open. The private app has a local ad-hoc signature and is not notarized. You deferred clean-account/default-startup qualification for this private alpha. The tested host is macOS 26.6.2 arm64.


For useful feedback, include the task, exact action or text, expected result and visible result. A saved `.elab` file and screenshot can help reproduce a failure. No task acceptance has been recorded yet.

## External agents

The /195 candidate folder contains the matching `elab-agent`. It accepts one JSON request per line on stdin and replies on stdout. These discovery calls are read-only:

```json
{"version":1,"id":"catalog","command":"capabilities.list","params":{}}
{"version":1,"id":"place-help","command":"capabilities.list","params":{"command":"component.place"}}
{"version":1,"id":"state","command":"project.inspect","params":{}}
```

A standalone agent has its own headless project. To control a visible app, a future session must be launched with `--agent-endpoint /absolute/path/to/new-descriptor.json`; connect the matching agent with `--connect /absolute/path/to/new-descriptor.json`. Keep any existing owner work saved before choosing which app to run. Public command discovery documents mutation revision guards and retry keys; the same validated commands back the UI.

The [delivery plan](34-first-version-plan.md), [remaining checklist](66-first-version-completion-checklist.md) and [candidate receipt](ui-audit/first-version/inspector-candidate-195.json) track /195. The [native/package evidence](../docs/validation/inspector-candidate-195/README.md) records exact scope and hashes. The previous [October 20 history](35-first-version-review-history-2026-10-20.md) and [verbatim /189 guide](35-first-version-review-history-2026-10-21.md) remain preserved. No owner acceptance of the four tasks has been recorded.
