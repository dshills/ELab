# Try ELab 0.172.0 /189

The [private alpha app](../build/ELab-0.172.0-189-alpha/ELab.app) is the version currently open for your review. You confirmed that its canvas, parts list and toolbar are visible. Keep using that window; no restart or replacement is needed for this walkthrough. Save your work before choosing another example.

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

- After an invalid index, time, Verify or sweep entry, correcting back to its previous accepted value can leave an action blocked. Timing thresholds and intervals have related recovery problems. Report the entered text and action; do not assume a refused action used the new value.
- A private unfinished timing entry can target an agent-selected different probe. Avoid simultaneous agent changes to timing probes while editing those fields in /189. These defects have source and headless fixes in the separate **/193 candidate**, whose native acceptance is pending. It has not replaced your app.
- Earlier automated window captures were blank for an unexplained reason. Your visible initial window is confirmed; report a blank window if you actually encounter one.
- Generic educational models are not general SPICE compatibility or PCB design. Complete accessibility/IME, other environments, broader performance, the hobbyist study and public distribution remain open. The private app has a local ad-hoc signature; it is not notarized. You deferred clean-account/default-startup qualification for this private alpha.

For useful feedback, include the task, exact action or text, expected result and visible result. A saved `.elab` file and screenshot can help reproduce a failure. No task acceptance has been recorded yet.

## External agents

The alpha folder contains the matching `elab-agent`. It accepts one JSON request per line on stdin and replies on stdout. These discovery calls are read-only:

```json
{"version":1,"id":"catalog","command":"capabilities.list","params":{}}
{"version":1,"id":"place-help","command":"capabilities.list","params":{"command":"component.place"}}
{"version":1,"id":"state","command":"project.inspect","params":{}}
```

A standalone agent has its own headless project. To control a visible app, a future session must be launched with `--agent-endpoint /absolute/path/to/new-descriptor.json`; connect the matching agent with `--connect /absolute/path/to/new-descriptor.json`. Do not restart the current owner window just for this walkthrough. Public command discovery documents mutation revision guards and retry keys; the same validated commands back the UI.

The [delivery plan](34-first-version-plan.md) tracks qualification and the [candidate receipt](ui-audit/first-version/timing-candidate-193.json) identifies /193. The [previous walkthrough and engineering history](35-first-version-review-history-2026-10-20.md) are preserved verbatim, including historical /188 and early /189 status statements. Use this page for the current /189 owner tasks.
