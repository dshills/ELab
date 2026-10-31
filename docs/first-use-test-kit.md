# ELab self-guided first-use test

Allow about 25 minutes. Intended for a hobbyist who has never used ELab. You
need an Apple Silicon Mac and the extracted alpha folder. No account, AI agent,
hardware or internet connection is needed for these tasks. This kit collects
feedback; its existence does not mean anyone has passed the tasks.

## Prepare

Follow `START-HERE.md` beside the app. Keep the whole extracted folder together.
Use a new folder named `ELab practice` for every file in this test. Do not use a
personal project. If an existing document is open, save it before starting.
In the macOS **File** menu, choose **New blank** for the first task. If something is confusing,
write it down before trying to work around it. Stop a task after five minutes
without progress; incomplete tasks are useful feedback, not a failure by you.

Please record the app version/build, macOS version, window size and theme.
First try each task using only the goal. The expandable help below is optional;
record when you use it. The separate maintainer key contains expected values.

## 1. Make an LED light

Build a circuit from scratch using a DC source, resistor, red LED and ground.
Set the source to 5 V and the resistor to 1k. Connect a complete loop. Find the
LED current, then change the resistor to 2k and observe what happens. Use Undo
once. Explain in your own words whether the circuit is running or paused.

<details><summary>Help if needed</summary>

If Parts is closed, use the sidebar button at the left end of the toolbar.
Search for each component, select it and click the canvas to place it. Use Select to move parts. Use Wire and click two terminal
dots for each connection: source + → resistor → LED anode; LED cathode → ground;
ground → source −. Cancel or Escape abandons an unfinished drawing operation.
Select a component, open Inspect, type its Value and choose Apply. Complete
circuits simulate automatically. Select the LED and read its current in Inspect.
Under Measure, Plot adds a scope trace when transient data is available. Pin places a live reading on the schematic.
The sidebar button at the right end of the toolbar opens or closes Inspector.

</details>

Record: time, where you looked for parts/values/current, current at 1k and 2k,
what Undo did, and anything that moved unexpectedly or was hard to read.

## 2. Watch a capacitor charge

Open the RC charge/discharge example. Pause and restart it, advance to 10 ms,
and find the capacitor voltage. Change to discharge, advance another 10 ms and
find the new voltage. Record a separate short run without changing live time.
Find where to change the recorded sample step, then return to the basic form.

<details><summary>Help if needed</summary>

Use Examples to search for RC charge/discharge. A replacement prompt protects
modified work; save your practice circuit or explicitly discard it. In the Scope dock use the Source row to choose Charge or Discharge.
Simulate → Live controls contains the time-step settings. Restart uses the saved startup
condition; stepping while paused makes time easier to inspect. Choose Recorded
runs, enter a duration of 20 ms, then Record from startup. Advanced contains sample
step and startup overrides. Opening Advanced does not start a recording. Choose
Inspect and select C1 (or its terminal) for voltage/current readings.

</details>

Record: the two voltages, whether you found Live controls/Recorded runs/Advanced,
and whether you understood the distinction between live and recorded data.

## 3. Measure a flasher

Open the capacitor-timed LED flasher example. Observe alternating LEDs and plot
both collector voltages or both LED currents. Pause when enough trace exists.
Use two scope cursors to estimate a period, then double both timing capacitors
and see whether the flashing slows down. Try Scope and Spectrum and say what
you think each view tells you. Exact spectral analysis is not required here.

<details><summary>Help if needed</summary>

Use Examples search for the capacitor-timed flasher. Select Q1/Q2 and Plot their
collector voltages, or choose the available probes in Simulate > Live controls.
Open the instrument dock if closed. In its **Scope** tab, expand **Instrument
controls & cursors**. Place cursors A and B on matching edges of one waveform.
Δt is the interval. Measure after several cycles to avoid startup effects.
Set C1 and C2 to twice their original values through
Inspect. The Spectrum tab uses the same live probes. More settings stay available
in the relevant tool's Advanced section. Generic device models are educational.

</details>

Record: which probes you chose, your estimated period before/after, what was
confusing about cursor placement, and whether controls stayed stationary.

## 4. Keep and share your work

Save the flasher into your practice folder, close or replace the document, and
reopen it. Export an SVG and a PDF. Find print preview, choose Letter or A4, and
check that the drawing fits on one page. Do not send a print job for this test.

<details><summary>Help if needed</summary>

Use File > Save As and File > Open. The toolbar export button offers SVG and PDF.
The Document inspector contains
export/preview controls. Fit to page is the easiest single-page output setting;
explicit scale may require multiple pages. Preview the output before exporting.
Keep all test files in your practice folder. Opening a clean example should show
its example title and should not prompt to save merely because it was opened.

</details>

Record: file names/formats, whether the reopened circuit matched, preview page
count, and whether save/export felt like separate understandable operations.

## Feedback form — copy for each person

Do not include private circuit files or personal information. Share only what you
choose; ELab does not upload this form. Keep each person's original wording.

```text
Participant code (optional):
ELab version/build / macOS / window size / theme:
Prior electronics experience / prior schematic app experience:

Task 1: completed / partial / blocked
Minutes / first point where help was needed / help used / wrong turns:
Current at 1k / 2k / Undo observation:
What I expected / what happened:

Task 2: completed / partial / blocked
Minutes / first point where help was needed / help used / wrong turns:
Charge voltage / discharge voltage:
Live vs recorded explanation / Advanced discovery:

Task 3: completed / partial / blocked
Minutes / first point where help was needed / help used / wrong turns:
Probes / period before / period after:
Scope vs Spectrum explanation / moving controls:

Task 4: completed / partial / blocked
Minutes / first point where help was needed / help used / wrong turns:
Saved/reopened / exports / page count:

Three most confusing moments:
One thing that felt easy:
One change I would make first:
Confidence that I understood the results (1–5), and why:
```
