# Comparator with hysteresis

The Amplifiers / ICs category includes `comparator`, a powered five-terminal
functional device: `noninverting`, `inverting`, `out`, `vcc`, `gnd`. Connect
both inputs and power pins; an unloaded output is allowed. Its numeric value
is the full **differential hysteresis width H**, editable from 1 mV to 1 V
(default 20 mV) through the normal value field or `component.set`.

The output becomes high when V+input ≥ V−input + H/2, becomes low when
V+input ≤ V−input − H/2, and retains its state between those thresholds.
Startup chooses low inside the band; actual inputs outside the band settle
it immediately. The [TI comparator/hysteresis reference](https://www.ti.com/tool/CIRCUIT060073)
explains why two thresholds avoid multiple transitions near a comparison level.
ELab's model is generic and is not fitted to the reference circuit or a
physical comparator's pinout or specifications.

**Comparator threshold detector** (`comparator-threshold`) applies a 2.5 V
reference and a 2.5 V ±0.4 V, 1 Hz sine input. With H=200 mV it rises at 2.6 V
and falls at 2.4 V; an LED shows the output. **Analog signal to digit counter**
(`comparator-counter`) uses the same cleaned output to clock a decade counter
and seven-segment display. A slow sine produces one rising clock per cycle.
SRESET is a momentary reset. Change source, reference or hysteresis using the
same controls as for existing sources and numeric components.

The fixed `control_model` defines 2–12 V powered operation, 1 GΩ input paths
to local GND, 1 MΩ supply load and 50 Ω output paths to the selected rail.
Below 2 V, including reversed supply, memory clears and active output paths
turn off; a 1 GΩ output bleed remains. Above 12 V is refused. This is a
push-pull output; no external pull-up is required by this model. Output droop
and supply current are electrical results, not ideal logic-only decoration.
There is no modeled input protection/common-mode restriction, physical delay,
current limit, offset, metastability, heat, noise or hardware rating.

The existing control hybrid engine localizes analog guard changes, settles
connected sequential devices at the same event time, and preserves external
capacitor voltage/coil current. Instantaneous contradictory feedback is
refused rather than converted into invented propagation delay. DC sweeps are
refused because hysteresis requires a state history.

## Commands, probes and saved results

Ordinary placement, wiring, rotation, value editing, input/wiper controls,
simulation, capture, saving and export commands apply. No comparator-specific
editing API is required. The native value field and agents share the exact
hysteresis domain via `capabilities.list` and `parts.list`.

`mixed.controls` publishes stored high/powered state, output voltage, delivered
output current, power and five pin currents ordered noninverting, inverting,
out, VCC, GND. Raw pin currents enter the device; output current probes report
delivered current. `v:U1` is output minus local GND; named terminal voltages
use the circuit reference. `v:U1.input-difference` is noninverting minus
inverting. The Inspector and plot controls expose those same readings.

Projects/captures retain the fixed model and actual hysteresis value. Restart
and reopen establish a fresh state. Captured validation checks stored guard
consistency and terminal currents/voltages/power; it does not reconstruct an
unknown prior hysteresis trajectory. Tests cover translated grounds, supply
limits, threshold retention, Jacobians, conservation, independently calculated
sine crossing times, analog-clock counting, capture validation, value domains,
project reopen and native/agent parity.
