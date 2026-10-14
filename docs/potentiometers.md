# Linear potentiometer

Place **Potentiometer** from the tray. Connect `1` to the high side and `2` to the low side; use `wiper` for the adjustable output. The Inspector offers total resistance, a 0–100% slider and an exact percentage field. Position 0 moves toward pin 2, and position 1 toward pin 1. Native controls and agents share `potentiometer.wiper.set {"id":"P1","position":0.75}`; the command accepts any finite fraction in 0–1.

An unloaded wiper is valid and measurable without a wire. Two connected pins also support a rheostat; the third pin may be unused or tied to one of the used pins. At least two pins need real terminal connections, and tying all three to one net is diagnosed as a short. These allowances apply to the potentiometer; other device open-pin rules remain intact.

## Model and readings

`elab.linear-potentiometer.v1` is a generic linear three-pin resistive track. Total resistance `R` defaults to 10 kΩ and can be edited from 10 Ω to 100 kΩ. For fraction `w`:

- Pin 1 to wiper: `Rupper = 1 Ω + (R − 2 Ω) × (1 − w)`.
- Wiper to pin 2: `Rlower = 1 Ω + (R − 2 Ω) × w`.

The two segments always sum to the nominal total. Each endpoint retains 1 Ω, so slider endpoints approach the end voltages without introducing a zero-resistance constraint. For an unloaded 5 V / 10 kΩ divider, midpoint is 2.5 V and endpoints are 0.0005 V and 4.9995 V. A 10 kΩ load to ground lowers midpoint to 2 V.

The model has no nonlinear taper, contact noise, wiper bounce, physical part identity, thermal behavior or ratings. The versioned model and its limitations accompany snapshots, projects, recovery and recorded Verify results. Missing/modified parameters or out-of-range/foreign wiper fields are refused during import.

Structured `mixed.potentiometers` results include both segment resistances, each pin voltage relative to ground, all three currents into the track, total resistance, wiper position and dissipated power. Pin currents sum to zero. Wiper current is negative when the track delivers current to a load. Probe IDs include `v:P1` (pin 1 minus pin 2), `v:P1.wiper` (relative to ground), and `i:P1.1`, `i:P1.2`, `i:P1.wiper`.

## Live behavior and ownership

Direct wiper movement during a fresh network transient stages a solve at the existing time with existing capacitor voltages. Success commits the new parameter/electrical revision and both sides of a `wiper:P1:<fraction>` event. Charge, simulated time, run identity and running clock remain continuous. A no-op preserves physics. Failed staging cannot commit the graph or a partial physical update.

Resistance edits, batches and undo/redo follow ordinary schematic-edit behavior. Running edits restart from configured startup. Paused edits mark the previous result stale until explicit restart/resume. A batch creates one named undo entry. Wiper and resistance values participate in dirty state; duplication retains independent wiper storage.

Verify captures the chosen wiper as startup and does not replay live movement history. JSON carries complete model/source data; CSV adds `potentiometer_model` when present. Kept records use canonical model hashes. Reopening starts fresh simulation at zero and honors saved pause preference.

## Scope and evidence

Up to four potentiometers fit within the existing [bounded network](switches.md) device, node and work limits. Dividers, LED loads, capacitor loads and supported NPN/diode/switch networks use the same solver.

Core tests compare loaded/unloaded divider and rheostat values against independent resistor equations, and capacitor charging against the Thevenin RC solution before and after wiper movement. They cover endpoint sums, conservation, running-clock continuity, no-op/range checks, independent snapshots/duplicates, batch/history semantics, strict imports and kept recordings. Native tray construction, slider/exact percentage fields, resistance editing and agent parity are exercised in the native suite and smoke.

Run `python3 scripts/potentiometer-demo.py --output docs/validation/headless-potentiometer.json` after building. Add `--artifacts PATH` for saved projects/SVG/records. `--connect` is only for an owned development desktop: the fixture replaces its schematic.
