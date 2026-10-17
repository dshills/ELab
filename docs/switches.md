# SPST switches and bounded circuit networks

Place **SPST switch** from the tray, connect terminals `1` and `2`, then select it and choose **Close switch** or **Open switch**. Voltage, current and power appear in the Inspector; its live symbol follows the applied state. The Inspector distinguishes saved desired state, applied state and a pending change; SVG export depicts the saved desired startup. Users and external agents share `switch.set {"id":"S1","closed":true}`. Switches default open and support duplication, deletion, history, atomic batches, probes and SVG export.

## Electrical behavior

`elab.spst-resistive.v1` models a closed switch as 1 Ω and an open switch as 1 GΩ. These finite resistances prevent a topology discontinuity; open circuits therefore have leakage. No contact bounce, arcing, heating, ratings or named physical device are modeled. Current is positive from terminal 1 to 2. Probe IDs are `i:S1` and `v:S1`.

Direct actuation during a fresh network transient queues the desired state. While paused, readings, time, capacitor charge and input events remain unchanged. The next accepted output sample advances the previously applied circuit, then stages an algebraic solve at that sample time with the preserved capacitor voltages. Only successful completion publishes the advance and the entire pending switch group. Both sides of each switch event share a timestamp. Repeated toggles coalesce to the final desired Boolean; returning to the applied state cancels the pending change.

`switch_states` reports `desired_closed`, optional `applied_closed` and `pending`. A missing applied state means no fresh result is available. Live switch and wiper commands preserve `electrical_revision` and `run_id`; `actuation_revision` advances separately and participates in Verify source provenance. Switch desired values remain saved, undoable document state. Restart consumes the desired startup states at time zero. DC, stale/incomplete edits, undo/redo and atomic schematic batches follow ordinary configuration/restart behavior. Verify captures desired startup states and does not replay live actuation history.

Projects and recorded results include the complete switch model and explicit Boolean state. Unsupported/tampered model parameters are refused before project replacement. Earlier schematics remain readable. Reopening starts fresh simulation at zero, using the saved switch and capacitor startup preferences.

## Network scope and numerical limits

The unreleased `elab.network-mna-sdirk2.v1` reuses the bounded MNA/Newton kernel for switched LED/RC/NPN circuits and multi-branch passive networks, including a resistor voltage divider. Sources, resistors, switches, capacitors, generic silicon diodes, generic red LEDs and grounded-emitter NPNs are supported. The existing specialized series LED, exact RC and NPN paths remain available; existing coupled RC/NPN captures retain their recorded engine identity.

Bounds are 64 simulated parts and 100 electrical nodes including ground, with at most 3 sources, 4 capacitors, 2 NPNs, 64 resistors, 8 switches, 8 diodes, 64 potentiometers and 4 LEDs. Drawing bounds are 100 objects including helpers and 300 wires. See [capacity definitions and fixtures](circuit-capacity.md). The educational model range is ±30 V at network nodes. Waveform-source negative terminals and NPN emitters must be grounded; DC sources may float in a ground-referenced network. Disconnected pins, floating islands, parallel ideal-source constraints, redundant capacitor startup constraints and capacitors directly across ideal sources are refused explicitly. This is a bounded educational solver, not a general SPICE implementation.

Network DC outputs do not advance time. Capacitors or pulse sources select transient analysis. Capacitor integration uses [adaptive SDIRK2](capacitor-integration.md) with full-step/two-half-step comparison (absolute 5e-8 V plus relative 5e-6 for the new network path). These local tolerances do not promise a global error bound. Integration retains the existing limit of 512 attempts and 100 ms per requested output sample; failures publish no partial physical advance. Existing mixed RC/NPN tolerances are unchanged. Non-waveform coupling capacitors additionally check their two non-ground node voltages; see [high-pass accuracy](rc-high-pass.md).

## Validation

The regression suite checks closed/open LED operating points, leakage, KCL/power conservation, analytical switched RC charging and off-state hold, continuity of charge/time/run, continuing live ticks, no-op state, snapshot isolation, native/agent parity, divider outputs, Verify and strict project round trips. Native GPU smoke constructs and toggles a switched LED through UI callbacks.

Run `python3 scripts/switch-demo.py --output docs/validation/headless-switch.json` after building. Add `--connect build/switch.endpoint.json` only for an owned development session: the demo replaces that session's schematic. `--artifacts PATH` preserves the switched RC project and SVG.

Detailed [switch/wiper contracts](interactive-actuation.md) now describe state-dependent completion, pending-state inspection and startup/batch semantics through the same discovery used by agents and inspector guidance.
