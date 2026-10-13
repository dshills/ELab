# Generic silicon diode

Choose **Silicon diode** from Components, then connect its named `anode` and `cathode` pins. Its plain diode symbol is distinct from the red LED and has no brightness indicator. The Inspector displays the immutable model defaults and signed voltage/current/power. Current flows positive from anode to cathode; reverse leakage is negative and remains visible at small magnitudes. `component.place`, `component.rotate`, wire commands, history, duplication and probes are the same semantic commands used by native controls and agents.

## Model

`elab.generic-silicon-diode.v1` uses:

| Parameter | Default |
|---|---|
| Saturation current `is_a` | 1 nA |
| Emission coefficient `n` | 1.7 |
| Series resistance `rs_ohm` | 1 Ω |
| Temperature `temperature_c` | 27 °C |
| Pin map | `anode`, `cathode` |

The current relation is `I = Is × expm1((V − I × Rs)/(n × Vt))`, with `Vt = kT/q`. Forward current is solved by bounded monotone bisection. Reverse bias solves junction voltage to avoid loss of precision as current approaches `−Is`; the analytic derivative feeds the existing MNA/Newton kernel. Series resistance is part of the model even without an external resistor.

These are inspectable educational parameters, with no named physical part, rated safe current, reverse breakdown, junction capacitance, stored charge, thermal dynamics or temperature adjustment. Reverse leakage is modeled; breakdown and reverse recovery are not. The existing generic red LED model is unchanged.

## Networks and measurement

The [bounded network solver](switches.md) accepts up to eight silicon diodes, within the existing 12 non-ground net and sheet limits. Diodes force the general network path and can combine with resistors, switches, capacitors, pulse sources, LEDs and grounded-emitter NPNs. Existing ground, source, constraint-loop, convergence and ±30 V node limits remain explicit.

`i:D1`, `v:D1`, and individual pin voltage probes are available in live and recorded scopes. Verify captures the full graph/model and its diode branch results. JSON, kept project recordings and recovery include complete model parameters and canonical dependency hashes. Missing or modified parameters are refused before document replacement. CSV recordings add a `diode_model` column when the captured circuit has a diode; switch circuits similarly add `switch_model`. Consumers should use header names. JSON remains the complete portable source record.

Agent discovery still uses `schema_version:2`; repeated recorded metadata now resolves through `urn:elab:verification-metadata:1` in the local registry. No network lookup is needed. Schema references in the full registry and command catalog are checked for resolution, and the existing 64 KiB catalog regression bound remains intact.

## Evidence

Core fixtures independently solve the inverse Shockley series relation for DC and integrate the scalar nonlinear RC equation with RK4. Tests cover forward/reverse conduction, analytic derivative agreement, monotonicity, current/power conservation, diode-isolated capacitor hold, finite internal series resistance, Verify, strict model validation and portable recordings. Native controls construct the same diode circuit as an agent and agree on results. Tiny current readouts use significant digits.

After building, run `python3 scripts/diode-demo.py --output docs/validation/headless-diode.json`. The demo checks an automatic pulse edge, reverse-isolated charge retention, live/Verify equivalence, record import, CSV attribution and fresh-process project reopening. `--artifacts PATH` preserves fixtures. Use `--connect` only for an owned development desktop because the demo replaces its schematic.
