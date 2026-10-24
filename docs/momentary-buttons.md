# Momentary buttons and contact bounce

`pushbutton` and `bouncing-pushbutton` are normally open two-terminal contacts
in the Switches category. Their resistance is 1 Ω closed /1 GΩ open. The clean
kind changes once; the bouncing kind reverses four times, 1 ms apart, after
both press and release. This deterministic teaching pattern is not measured
switch behavior. [TI's debounce application brief](https://www.ti.com/document-viewer/lit/html/scea094)
provides background on multiple triggers and filtering/hysteresis.

Select a button in the Inspector and use **Click button · 50 ms**, or **Press
and hold** followed by **Release button**. Click releases automatically after
50 ms of **simulation time**. Paused press/release settles contacts at the
accepted time; Step or Resume advances bounce and click duration. Operations
preserve capacitor voltage, coil current, run, dirtiness and undo history.
Restart and project reopen release all buttons; runtime actions are not saved
as permanently pressed startup states.

The **Pushbutton decimal counter** example (`button-counter`) clocks the
seven-segment counter with SCOUNT. A clean press counts once; holding does not
repeat. SRESET asynchronously clears it. **Contact bounce and RC debounce**
(`button-debounce`) compares raw U2 with filtered U4. A complete click creates
five raw counts (three on press, two on release), but one filtered count.
D1/D2 show the low bits; inspect U2/U4 for numeric values. Two inverter stages
isolate the raw contact node from the filter, preventing capacitor backfeed.
The 1 kΩ /10 µF filter and counter's 30/70% clock hysteresis reject the short
transitions. Wait at least 250 ms simulated time between teaching clicks.

## Shared commands and state

- `button.set {"id":"SCOUNT","pressed":true}` holds; false releases.
- `button.click {"id":"SCOUNT"}` schedules one 50 ms click on a released,
  settled button. Clicking an already pressed/bouncing button is refused.
- Both admit background `input_work`. Await `simulation.wait` or inspect its
  matching ID/command, `requested_button` and `button_click` before stepping.
- Native controls use those same commands. Existing revision, document,
  retry and cancellation guards apply. Runtime actuation is outside `edit.batch`.

These controls use `elab.generic-controls.v1`, analysis `Controlled devices
transient`, and engine `elab.controls-hybrid.v1`. `control_model` is preserved
in snapshots, projects, recovery, capture metadata, hashes and exports.
`mixed.controls` includes kind, model, contact/press/event memory, voltage,
current, power and a five-current array. Buttons use indices 0/1 for pins 1/2;
remaining entries are zero. Raw currents enter pins; `i:SCOUNT` is pin 1 to 2.

Timed contact changes have explicit timestamps. Analog clock/reset guards are
still localized with the existing tolerance. Edge searches for the new control
engine reuse accepted lower integration states rather than repeatedly solving
from the original interval start. All analog solves retain the same accuracy,
512-attempt and 100 ms work guards. Capacitor voltage stays continuous across
contact transitions. Tests compare switched-RC trajectories with independent
piecewise exponentials at fine and coarse output intervals.

Verify starts an independent released circuit; it does not replay prior manual
click history. Live scope CSV retains the observed contacts/currents. Captures
validate required memory, timestamps, contact consistency and terminal currents;
this is consistency checking, not authenticated interaction history. Buttons
have no arcing, wear, calibrated physical bounce, heat or hardware ratings.
