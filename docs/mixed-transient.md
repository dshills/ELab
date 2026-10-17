# Capacitor-timed RC/NPN LED flasher

Implemented October 13, 2026. ELab now solves coupled resistor/capacitor/NPN/LED transient networks and includes a self-oscillating two-transistor LED flasher powered by a constant DC source. Timing comes from the circuit's capacitor state; the example has no pulse generator, hidden timer, or precomputed switching schedule.

## Try it

Choose **Flasher** in ELab. The example contains a 5 V DC supply, two grounded-emitter generic NPNs, two LED/1 kΩ collector loads, two 47 kΩ base resistors, and two cross-coupled 10 µF capacitors. Each collector connects through a capacitor to the opposite transistor's base. Both emitters and supply negative connect to ground. The LEDs alternate as capacitor voltages change the transistor bias.

The example explicitly sets **C1's initial voltage to 0.1 V**, with C2 initially at zero. This breaks the otherwise perfect symmetry of the ideal circuit. Initial voltage is not random noise or a private solver adjustment: select a capacitor to inspect/edit it. The same field is saved, undoable, and available to agents. A perfectly symmetric ideal circuit is not guaranteed to self-start.

The 5 ms output sample step and 0.5 simulated seconds per wall second make the approximately 0.49 s simulated period appear as roughly one wall-clock second. Playback is best-effort paced, not a hardware real-time guarantee. Double both timing capacitors to 20 µF to approximately double the period. Edit resistors, capacitance, or capacitor startup voltage directly in the inspector.

**Pause** freezes stored charge and time. **Step** advances a paused valid run. **Resume** continues its current state and run; **Restart** applies the saved initial conditions at time zero. **Power off/on** actuates the example supply while preserving capacitor voltage. In a mixed network with a pulse source, the equivalent controls are Low/High. Auto restores the source's normal behavior. A paused override is queued until Step or Resume. Opening a saved file starts a fresh Auto run; it does not restore live trace/history or overrides.

Electrical/initial/settings edits restart when live, or remain paused with stale readings. Move, rotation, selection and view navigation preserve capacitor memory. Source value changes remain ordinary saved electrical edits; runtime overrides do not dirty the schematic. The Flasher example uses the existing Save/Discard/Cancel protection.

The scope shows the first transistor's VCE (teal), first capacitor's signed voltage (purple), first LED's current on the right mA axis (coral), and the selected stimulus source voltage (amber). Both LEDs illustrate their measured currents, including dim charging currents. The inspector exposes each transistor's bias/state and each capacitor's voltage/energy; selecting an LED, resistor or source shows its branch readings. Full node and device data is available through the shared snapshot. Wire routes avoid component bodies; their geometry does not establish electrical connections.

## Numerical engine and boundary

The unreleased source engine `elab.mixed-mna-sdirk2.v1` assembles modified nodal equations for resistors, ideal sources, ideal capacitors, the existing generic red LED model, and reciprocal Ebers–Moll NPN junctions. Capacitor initial voltage constraints establish a consistent time-zero solution. This follows the general distinction between capacitor initial conditions and a steady-state operating point described in the [official ngspice manual](https://ngspice.sourceforge.io/docs/ngspice-manual.pdf); this implementation is an independent bounded solver. Earlier backward-Euler captures retain their recorded identity and samples.

Each implicit capacitor companion stage uses `C (Vnew − Vhistory) / Δt`, coupled to nonlinear devices by a damped Newton solve. The unreleased [two-stage integration method](capacitor-integration.md) compares one full step with two half steps under the unchanged `2 µV + 0.02%` voltage scale. Alternate nonlinear starting guesses remain available; accepted states still satisfy the same circuit equations and no switching time is imposed by those guesses.

Output sampling and integration substeps are distinct. Pulse edges split integration intervals; source values are held on each side of an edge. Capacitor voltage remains continuous while algebraic node/current values are recalculated for the new source. An output sample and an entire manual Step are staged before publication, so a failed solve cannot commit partial capacitor memory, time, events or queued input.

There are at most 100 Newton iterations per attempt, 512 adaptive attempts and a 100 ms wall-time budget per output sample. Live mixed playback advances at most 16 samples per tick and caps catch-up. A singular/inconsistent/stiff circuit or exhausted budget produces an explicit diagnostic; live failures stop advancement and mark the last readings stale. A failed manual Step leaves its prior state intact. These bounds are not comprehensive performance qualification.

The NPN remains an educational, memoryless junction model with no physical part calibration, breakdown, heating, Early effect, junction capacitance or intrinsic switching/storage delay. The external capacitors now provide genuine dynamic state. Capacitors are ideal and unpolarized, with no ESR/leakage. LED reverse current is neglected. Node solutions are bounded to ±30 V; source and component value limits remain unchanged.

The mixed engine supports **100 nodes including ground and 64 simulated parts**, with **3 sources, 4 capacitors, 2 grounded-emitter NPNs, 64 resistors and 4 LEDs**; other supported network parts retain their [published bounds](circuit-capacity.md). All source negatives must be grounded. At least one capacitor and NPN are required to select this engine. Open pins, disconnected islands, shorts, unknown models and excess complexity are diagnosed. Direct ideal-source/capacitor connections and redundant source/capacitor constraint loops are refused: impulse or indeterminate startup-current cases are not modeled. Existing simpler LED, RC and pulse-NPN engines remain available.

## Shared commands and persistence

`example.flasher` loads the editable autonomous example. `capacitor.initial.set` takes `{"id":"C1","value":"0.1 V"}` or `{"id":"C1","value":"default"}`. A signed override must be within ±12 V and denotes **pin 1 minus pin 2**. Default uses the experiment's common initial setting; legacy RC output-to-ground initialization remains compatible. The capacitor-specific override honors pin polarity in either engine. These edits use standard revision/retry admission, history, dirty state and actor attribution.

Construction uses `component.place`, `wire.connect`, move/rotate/delete and `component.set`. `simulation.transient.set`, pause/resume/restart/step and `simulation.input.set` remain shared controls. Mixed input modes are Auto/High/Low; the target is `transient.source_id`, chosen as the first pulse source when present, otherwise the first DC source. Other sources retain their normal behavior; pulse sources share the existing experiment durations. No GUI-only actuation is introduced.

Snapshot `mixed` contains engine/run/electrical revision, time, all non-ground and ground net voltages, transistor bias/currents/power/state and model identity, capacitor voltage/current/energy, LED/resistor branches, and source readings. Resistor/capacitor current is positive from pin 1 to pin 2; LED current from anode to cathode. Source current is **delivered out of its positive terminal**, so positive source power denotes delivered power. The legacy single-switch `npn` result is absent for a mixed run.

`mixed.kcl_residual_a` is the maximum node-current residual. Power balance uses instantaneous capacitor terminal power `voltage × current`; `power_residual_w` is the remaining balance error. Fresh SDIRK2 and Radau results report zero `numerical_damping_w`: an internal stage history is not an accepted-state energy difference. Historical backward-Euler captures retain their original nonnegative damping readout. Readings at source events/time zero use constrained capacitor currents.

The common `transient` carries the rolling primary trace, events, mode/pending mode, stale state, run/time identity and settings. Here `voltage_v` is the primary transistor's VCE, `current_a` is the primary LED current (or collector current without LEDs), `capacitor_voltage_v` is the primary capacitor's signed voltage, and `energy_j` is its stored energy. Omitted zero optional fields mean zero. `time_constant_s` is zero because the nonlinear network has no single RC time constant. Up to 512 samples and 64 events are retained; all other current device readings are in `mixed`.

Version 2 `.elab` files store optional component `initial_voltage_v` plus graph, model metadata and experiment settings. Existing version 1 and version 2 files still load. The new reader rejects initial-condition metadata on non-capacitors or out-of-range values. Older builds explicitly reject the new field or report an unsupported mixed topology. Files do not contain solver vectors, device results, waveform history or credentials.

## Validation

- Sustained two-LED alternation under DC power, including actual negative base excursions and capacitor charging currents.
- With a 5 ms output grid, the 10 µF example period measures about **0.485 s** and 20 µF about **0.97 s**. The repeatable API workflow reads every 10 ms and obtains approximately **0.49 / 0.98 s**; the difference reflects measurement-grid resolution.
- Tests check KCL, instantaneous power balance, timestep continuity, explicit initial conditions, bounded traces, live clock/pause/resume, geometry preservation, stale electrical edits, initial-condition undo and save/reopen.
- A pulse-driven base capacitor retains its voltage and LED current at the falling input edge, then discharges. At low input bias, where NPN loading is negligible, the response agrees with the analytical RC reference within the tested numerical tolerance.
- Failed-Step tests verify no partial state/queued-input commit; invalid saved initial conditions and redundant ideal constraints are rejected.
- Actual UI callbacks build from blank and match direct agent results. Wire-routing tests check the example's component bodies. Native smoke verifies the full earlier LED/RC/NPN workflows plus Flasher and rendered capacitor/current scope pixels.
- Root/desktop race tests and vet pass. Headless stdio and a packaged desktop agent execute the repeatable construction, capacitance scaling, source override, live resume, initial-condition editing and fresh-process persistence workflow.

Evidence: [headless results and traces](validation/headless-mixed.json), [desktop results and traces](validation/desktop-mixed.json), [portable schematic](validation/rc-npn-flasher.elab), and [final validation](validation/mixed-validation.json). Run `python3 scripts/mixed-demo.py --output docs/validation/headless-mixed.json`; use `--connect /absolute/endpoint.json` for a development desktop session. The script replaces that session's drawing; save existing work first.

The flasher has conservation, analytical limiting-case and timing-scaling validation. It has not been independently characterized in hardware or rerun in an external SPICE engine. Full general-netlist support, intrinsic transistor timing, probes/cursors/captures/export, recovery and platform/performance/accessibility qualification remain future work.
