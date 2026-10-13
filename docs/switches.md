# SPST switches and bounded circuit networks

Place **SPST switch** from the tray, connect terminals `1` and `2`, then select it and choose **Close switch** or **Open switch**. Voltage, current and power appear in the Inspector; its symbol follows the saved state. Users and external agents share `switch.set {"id":"S1","closed":true}`. Switches default open and support duplication, deletion, history, atomic batches, probes and SVG export.

## Electrical behavior

`elab.spst-resistive.v1` models a closed switch as 1 Ω and an open switch as 1 GΩ. These finite resistances prevent a topology discontinuity; open circuits therefore have leakage. No contact bounce, arcing, heating, ratings or named physical device are modeled. Current is positive from terminal 1 to 2. Probe IDs are `i:S1` and `v:S1`.

Direct actuation during a fresh network transient stages an algebraic solve at the existing time with existing capacitor voltages. Only a successful solve commits the state, electrical revision, right-hand sample and `switch:S1:open|closed` input event. Simulation run ID, pause preference and running clock remain continuous. Both sides of the instantaneous event are retained at the same timestamp. A no-op actuation preserves physics. Undo/redo, an atomic schematic batch or an edit made while results are stale follows ordinary schematic restart behavior. Switch startup states are captured by Verify; Verify does not replay the live actuation history.

Projects and recorded results include the complete switch model and explicit Boolean state. Unsupported/tampered model parameters are refused before project replacement. Earlier schematics remain readable. Reopening starts fresh simulation at zero, using the saved switch and capacitor startup preferences.

## Network scope and numerical limits

`elab.network-mna-be.v1` reuses the bounded MNA/Newton kernel for switched LED/RC/NPN circuits and multi-branch passive networks, including a resistor voltage divider. Sources, resistors, switches, capacitors, generic red LEDs and grounded-emitter NPNs are supported. The existing specialized series LED, exact RC and NPN paths remain available; existing coupled RC/NPN files retain their engine identity.

Bounds are 12 non-ground nets, 3 sources, 4 capacitors, 2 NPNs, 12 resistors, 8 switches and 4 LEDs, within the existing sheet/object limits. The educational model range is ±30 V at network nodes. Source negative terminals and NPN emitters must be grounded. Disconnected pins, floating islands, parallel ideal-source constraints, redundant capacitor startup constraints and capacitors directly across ideal sources are refused explicitly. This is a bounded educational solver, not a general SPICE implementation.

Network DC outputs do not advance time. Capacitors or pulse sources select transient analysis. Capacitor integration uses adaptive backward Euler with full-step/two-half-step comparison (absolute 5e-8 V plus relative 5e-6 for the new network path). These local tolerances do not promise a global error bound. Integration retains the existing limit of 512 attempts and 100 ms per requested output sample; failures publish no partial physical advance. Existing mixed RC/NPN tolerances are unchanged.

## Validation

The regression suite checks closed/open LED operating points, leakage, KCL/power conservation, analytical switched RC charging and off-state hold, continuity of charge/time/run, continuing live ticks, no-op state, snapshot isolation, native/agent parity, divider outputs, Verify and strict project round trips. Native GPU smoke constructs and toggles a switched LED through UI callbacks.

Run `python3 scripts/switch-demo.py --output docs/validation/headless-switch.json` after building. Add `--connect build/switch.endpoint.json` only for an owned development session: the demo replaces that session's schematic. `--artifacts PATH` preserves the switched RC project and SVG.
