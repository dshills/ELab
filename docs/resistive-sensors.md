# Thermistors and light-dependent resistors

Passives now includes `thermistor` (NTC) and `ldr` (photoresistor), each with
pins `1` and `2`. The ordinary value editor / `component.set` sets reference
resistance from 100 Ω–100 kΩ, default 10 kΩ. Canvas and exported captions use
R25 for the thermistor and R10 for the LDR to distinguish reference resistance
from the live resistance reported by the Inspector.

## Models and environment

The generic NTC uses R(T)=R25×exp[3950×(1/(T+273.15)−1/298.15)]. Temperature
is in °C, −40 through 125, default 25. The beta form is described by
[Vishay's thermistor selection reference](https://www.vishay.com/doc/?33001=).
ELab's constant beta and resistance are model choices, not a fitted Vishay part.

The LDR uses R(E)=R10×(E/10)^−0.7, clipped to 100 Ω–1 MΩ. Zero light is
explicitly 1 MΩ; illumination is 0–10,000 lux, default 100 lux. Manufacturer
[photocell data](https://www.advancedphotonix.com/storage/ds-nsl-a603-series.pdf)
illustrates resistance/light and gamma specifications; ELab's fixed exponent
and clipping bounds are generic model choices, not that manufacturer's rating.

Both models are resistors at the prescribed environment: I=(V1−V2)/R,
P=VI≥0. Sensor power does not change its temperature or illumination.
Uniform environment changes are instantaneous. Self-heating, thermal/optical
response delays, spectral dependence, tolerance, ageing and physical ratings
are omitted. Sensors consume existing resistor slots and total-part/node/work
limits; no simulation guard or accuracy tolerance is increased.

## Shared controls and commands

Selecting a sensor shows a temperature or light slider, an exact numeric field,
and accepted environment, resistance, voltage, current and power. Temperature
uses a linear scale with half-degree drag steps; light uses log(1+lux), with
darkness at zero, so dim-light
thresholds are accessible. Slider and numeric controls invoke the same commands
as external agents:

- `sensor.temperature.set {"id":"RT1","temperature_c":20}`
- `sensor.light.set {"id":"LDR1","illuminance_lux":10}`

Mutations use the existing revision/retry guards. `capabilities.list` exposes
units, bounds, target kind, completion and cancellation semantics. Numeric
fields do not quantize precise saved values merely by recommitting an unchanged
float32 display value.

A fresh paused/running transient stages `input_work` at accepted time. Until
publication, live resistance/readings use the committed environment while the
controls identify a pending request. Wait with `simulation.wait` before dependent
work. Success commits one undo entry and the environment while preserving time,
run, capacitor voltage and coil current; both sides of the event are retained.
Failure, cancellation and supersession retain committed graph/history/physics.
Selecting the committed value cancels pending input for that sensor. Incomplete,
stale and DC circuits use ordinary startup editing; `edit.batch` configures
startup environments. Save, restart, reopen and independent Verify use committed
environments. A thermostat changes its relay after coil dynamics cross pickup
or dropout, rather than resetting the coil when the slider moves.

`mixed.sensors` reports model, environment value/unit, actual resistance, signed
voltage/current and power. `v:RT1` / `i:RT1` and corresponding LDR probes are
available along with both named terminal voltages. Positive current enters pin
1. Sensor environments, `sensor_model`, captures and model hashes survive project
save/load, recovery inspection, kept recordings and schematic/scope exports.
Old schematics with no sensors remain compatible.

## Examples

**Automatic night-light** (`night-light`) compares an LDR/10 kΩ divider with
a potentiometer reference. At the midpoint setting, low light turns D1 on;
100 lux turns it off. The comparator's 200 mV differential hysteresis keeps its
previous state near 10 lux. Select LDR1 to change light; select P1 to move the
threshold. Plot the divider, comparator output and LED current.

**Thermistor relay thermostat** (`thermostat`) uses the same divider/reference
arrangement, an NPN relay driver and flyback diode. It starts at 20 °C with the
heater-indicator LED on after pickup. At 30 °C it switches off; at 25 °C it
retains its previous state. P1 adjusts the setpoint; this is an open-loop sensor
experiment, not a room/heater thermal model. Select RT1 to move temperature.
Plot sensor voltage, comparator output, coil current and diode/indicator current.

Electrical tests use high-precision reference resistance tables, independent
Ohm/Kirchhoff and switched-RC equations, comparator hysteresis, coil continuity,
conservation, failed/canceled staging, undo, strict captured-data validation,
project round trips and native/agent parity. The reproducible public transport
replay is `scripts/sensors-demo.py`.
