# Pulse-driven NPN LED simulation

Implemented October 13, 2026. The later [mixed RC/NPN engine](mixed-transient.md) adds external capacitor dynamics and a DC-powered autonomous flasher; this record describes the earlier pulse-only slice. ELab now runs a grounded-emitter NPN LED switch from a separate pulse-source base drive, advances simulated time live, and displays blinking LED brightness with synchronized voltage/current traces. Users and agents invoke the same commands.

## Workflow

Choose **Blink** for an editable example: 5 V DC collector supply, 5 V pulse base drive, 10 kΩ base resistor, 1 kΩ LED load, generic red LED, grounded-emitter NPN, and common ground. The example has 50 ms high and low phases and playback at 0.1 simulated seconds per wall second, giving roughly half a wall-clock second on and half off. Playback is best-effort paced, not a hardware real-time guarantee.

To draw from blank, use **Pulse source** for the base-drive branch in the [NPN circuit](npn-switch.md), and **DC source** for the collector supply. The source value sets the high amplitude; low is 0 V. Set pulse high/low durations, sample step, and playback speed in the Pulse experiment inspector. These settings and source-value edits are saved and undoable.

The live scope shows base-drive voltage in amber and VCE in teal on the left voltage axis, with collector/LED current in coral on the right mA axis. Readouts and LED brightness follow the latest simulated sample. Exact discontinuities are retained as left/right samples at the same edge time, even when an edge falls between sample times.

**Pause** freezes time. **Step +1 sample** advances a paused valid run. **Resume** continues the same run/time. **Restart** returns to time zero, including while paused, and preserves the current input override. **Auto**, **High**, and **Low** select scheduled pulse operation or a constant base input. A paused override is queued until Step or Resume; overrides are runtime state and do not mark the circuit dirty. Returning to Auto uses the pulse phase at the current run time.

Electrical/settings edits start a new run when live or mark both the trace and NPN reading stale while paused. Geometry, selection, and view changes preserve time. Broken/unsupported circuits stop advancement and retain stale historical readings. Loading a file starts a fresh Auto run at time zero; an example also restores its default configuration. Unsaved-work protection applies to Blink just like the other examples.

## Model boundary

Engine `elab.npn-switch-quasistatic.v1` evaluates the existing reciprocal Ebers–Moll NPN and generic LED model at each pulse level and switches between those states on a real simulated clock. The phase results retain the existing current/voltage/power-conservation checks. Both phase operating points are calculated once per electrical run; time stepping and live publication remain bounded and do not perform a nonlinear solve for every sample.

This is a **memoryless, instantaneous transistor response**. It models scheduled input changes and resulting waveforms, but does not model transistor capacitance, charge storage, turn-on/off delay, or finite pulse rise/fall time. A DC collector supply and separate pulse base source are supported; pulsing the collector supply, shared pulse supply/base drive, multiple transistors, and mixed capacitor/transistor networks remain unsupported. The collector load may be resistive or include the supported forward LED.

The fixed model `elab.generic-npn.v1` and its assumptions are unchanged, preserving previous saved NPN files. The per-phase result `snapshot.npn.engine` remains the DC operating-point engine; `snapshot.transient.engine` identifies the time-based experiment. `initial_voltage_v` is retained in the common saved settings for RC compatibility and has no effect on this memoryless NPN experiment; the NPN inspector does not show that capacitor-only setting.

## Shared command and data contract

| Command | NPN pulse behavior |
|---|---|
| `example.npn-pulse` | Load the Blink example; use `discard_changes:true` only when intentionally replacing dirty work. |
| `component.place` / `wire.connect` | Draw the same circuit with a `pulse-source` in the base branch. |
| `component.set` | Set pulse amplitude or resistor values; electrical edit/restart semantics. |
| `simulation.transient.set` | Set the complete saved sampling/pulse/playback configuration; undoable. |
| `simulation.input.set` | `{"mode":"auto"}`, `{"mode":"high"}`, or `{"mode":"low"}`; RC retains auto/charge/discharge. |
| `simulation.pause` / `simulation.resume` | Freeze or continue the run. |
| `simulation.step` | Advance a valid paused run by `duration_s`, bounded to 1 second / 1000 sample steps. |
| `simulation.restart` | Reset to time zero while keeping paused/running state and current input mode. |
| `simulation.wait` | Read the ready transient state; it does not wait for future simulated time. |
| `project.save` / `project.load` | Preserve graph/model/pulse settings and recalculate a new Auto run on opening. |

All mutations retain revision preconditions, retry-key deduplication, actor attribution, and shared history. The complete 37-command catalog remains available over headless stdio and the private desktop socket. RC commands and results remain compatible.

For the NPN transient, `transient.source_id` is the base-drive source; `supply_id`, `base_resistor_id`, `resistor_id`, `transistor_id`, and optional `led_id` identify the other parts. `output_net` is the collector. Samples use `time_s`, `input_v`, `voltage_v` (VCE), and `current_a` (collector/LED current), with NPN `state`, `base_current_a`, and `vbe_v`. The last two omit exact zero values; treat absent values as zero. Legacy RC `energy_j`, `time_constant_s`, and `capacitor_id` have zero/empty values here and are not capacitor measurements. Live `snapshot.npn` carries the complete current phase result with matching run/electrical revision.

The trace retains at most 512 samples including edge points and at most 64 events. Event modes include restart, pulse-high, pulse-low, and manual override modes. Autonomous clock updates change `revision` but not `command_revision`; a client must follow normal conflict/retry rules during a live run. Stale traces remain labeled. Save excludes time, trace/history, and runtime input overrides.

## Validation

- Pulse high current is **2.860330491 mA**, VCE approximately **57.533808 mV**, matching the existing NPN DC phase reference. Pulse low has model leakage approximately **2 × 10−14 A** and cutoff state.
- Tests verify exact left/right edge samples, including a 31 ms high / 47 ms low pulse with a 7 ms sample grid; sampling does not move the 31 ms and 78 ms transitions.
- Autonomous clock tests observe both cutoff and saturation without commands advancing time. Pause freezes the sample; queued overrides, Step, and Resume preserve the run. Settings/amplitude changes, presentation edits, broken wires, trace limits, detached snapshots, and saved configuration are covered.
- Actual UI callbacks build the pulse circuit from blank and match direct agent stepping/input results. The test client assigns a separate retry key to each intended repeated step/restart.
- Root and desktop race tests and vet pass. Native smoke exercises existing LED/RC/NPN DC flows plus Blink, its falling edge, a manual high override, and rendered voltage/current scope pixels.
- Repeatable headless stdio and packaged-desktop socket workflows construct from blank, verify transitions and live advancement, preserve time across geometry edits, and reopen a saved pulse circuit in a fresh process.

Evidence: [final validation summary](validation/npn-pulse-validation.json), [headless pulse workflow](validation/headless-npn-pulse.json), [desktop pulse workflow](validation/desktop-npn-pulse.json), and [portable pulse schematic](validation/npn-pulse.elab). Run `python3 scripts/npn-pulse-demo.py --output docs/validation/headless-npn-pulse.json`; add `--connect /absolute/path/to/endpoint.json` for a desktop session. The script intentionally replaces that session's drawing, so use a development session or save existing work first.

Physical transistor switching/storage timing and general mixed transient networks remain future work. This slice does not claim independent SPICE or hardware validation of those effects.
