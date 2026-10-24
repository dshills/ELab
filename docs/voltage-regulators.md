# Voltage regulators

The Amplifiers / ICs category includes `linear-regulator` and `ldo-regulator`.
Both have `in`, `out` and `gnd` pins. The ordinary value field or
`component.set` edits the output setpoint from 1.2–12 V (defaults 5 V and
3.3 V respectively). `parts.list` and `capabilities.list` expose the same
model and editing domain to agents.

These are generic educational functional models. The
[TI LM340 / LM7805 reference](https://www.ti.com/product/LM340) provides
background on practical linear regulators; ELab's model is not fitted to
that device or any LDO. It has no thermal shutdown, physical ratings, noise,
feedback bandwidth, capacitor stability requirement or reverse protection.
Displayed power is electrical dissipation, not a temperature prediction.

## Electrical contract

The fixed `regulator_model` declares 2 V headroom for the standard part,
0.3 V for the LDO, 0.2 Ω small-signal output resistance and a smooth
100 mA asymptotic pass-current limit. Local voltages are relative to GND.
The target is max(0, min(setpoint, VIN−dropout)). With insufficient headroom
the output follows the available input less headroom and load droop.

For positive headroom and error E=target−VOUT ≥0, forward pass current is
(E/0.2)/sqrt(1+(E/(0.2×0.1))²). Otherwise it is zero. The pass device does
not sink an externally held output or conduct backward. Permanent 1 MΩ
input and 1 GΩ output loads remain. Current flows from input to output;
GND carries the quiescent/bleed currents. Power loss is (VIN−VOUT)×Ipass
plus these two resistive losses. External capacitors preserve voltage and
loads determine discharge. Current limiting and finite load regulation
are observable, rather than ideal fixed output voltage.

The Inspector reports regulating/dropout/current-limited/unpowered/not-sourcing
state, input/output voltage, delivered current and loss. The current-limited
label starts at 90% of the declared asymptotic limit. `v:U1` is output minus
local GND; `i:U1` is delivered output current. All named terminal voltages
and currents are available, with output current positive out of the device.
`mixed.regulators.pin_currents_a` instead uses raw currents entering
in, out, GND. All readings, models and value domains share UI/agent commands.

## Examples

**Regulated supply and load** (`regulator-supply`) charges a 10 µF output
capacitor from 9 V toward 5 V. A 10 Ω input resistor gives the input
capacitor a finite charging path. Closing S1 adds a 100 Ω load and shows
finite output droop. Changing RLOAD to 10 Ω demonstrates current limiting
and output collapse; S1 also contributes its existing 1 Ω closed resistance.
Plot output voltage, regulator current and COUT voltage; inspect power loss.

**Linear versus LDO dropout** (`regulator-dropout`) feeds two 5 V regulators
with a 4–9 V, 1 Hz sine input. With equal 1 kΩ loads, the standard part
leaves regulation below 7 V; the LDO below 5.3 V. Those thresholds are model
choices, not physical device guarantees. The actual regulated voltage is
slightly below the setpoint because load regulation is finite.

Placement, wiring, values, source/load switching, scope, DC sweep, transient
simulation, verification, save/load and exports use existing commands.
Saved projects, recovery and kept captures retain the exact model. Strict
capture validation recomputes terminal currents, voltage, state and loss.
Tests independently invert the load line, check dropout/current limiting,
translated grounds, non-sinking operation, Jacobians, KCL/power, capacitor
charging and load steps, input sweeps, value bounds and project round trips.
