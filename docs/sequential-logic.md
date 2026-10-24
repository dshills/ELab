# Flip-flops and counters

The **Logic** category includes D, JK and T flip-flops, a 4-bit binary counter,
and a decade counter. These are powered schematic functions, not models of a
particular packaged IC. Search by name, register, divider, digital or CMOS;
searching counter finds the two counters.

## Wiring and examples

| Kind / example ID | Inputs | Outputs | Behavior on a rising clock edge |
|---|---|---|---|
| `d-flipflop` | `d`, `clk`, `reset` | `q`, `nq` | Q takes D |
| `jk-flipflop` | `j`, `k`, `clk`, `reset` | `q`, `nq` | 00 holds, 10 sets, 01 clears, 11 toggles |
| `t-flipflop` | `t`, `clk`, `reset` | `q`, `nq` | T high toggles; T low holds |
| `binary-counter` | `clk`, `reset`, `enable` | `q0`–`q3` | Enabled count increases modulo 16 |
| `decade-counter` | `clk`, `reset`, `enable` | `q0`–`q3` | Enabled count increases modulo 10 |

Every device also has `vcc` and `gnd`. Connect both power pins and every input;
outputs may be unloaded. A VCC/GND short is refused. `q0` is the least
significant bit; the decade counter represents 0–9 as four binary bits. Counter
bits update together. **Reset is active high and asynchronous**, with priority
over clock and data. There is no preset input. Add an explicit pull-down or
pull-up for a switched input; a crossing wire alone is not a junction.

Five Examples menu entries use the same IDs as the parts above. Each provides
a 5 V supply, a finite-ramp clock, a reset switch, resistor-limited output LEDs,
and scope traces. D has an independent data waveform. JK starts with J=K=high;
T starts with T=high, so these demonstrate divide-by-two behavior. Counters
start enabled and provide a separate enable switch. Open the selected switch
in the Inspector to hold the count; close Reset to clear immediately. During
live simulation these controls preserve elapsed time. Paused startup edits
take effect on Restart or Resume through the existing simulation controls.

Examples use a 200 ms clock period, 20 ms ramps and a 2 ms output sample step.
The first recognized rising edge is at 14 ms (70% of the ramp). The Inspector
shows the selected device's numeric state, clock/reset levels, local supply,
primary output and power. Plot actions include all named terminal voltages and
output current. Use scope cursors or capture an experiment to inspect edges.

## Electrical and timing contract

The fixed model is `elab.generic-sequential-cmos.v1`; sequential networks use
analysis `Sequential logic transient` and engine `elab.sequential-hybrid.v1`.
The [TI D flip-flop reference](https://www.ti.com/product/CD4013B),
[BCD counter reference](https://www.ti.com/product/CD4518B), and
[binary counter reference](https://www.ti.com/lit/gpn/CD4520B) provide background.
ELab's thresholds, resistance and startup policy are educational choices,
not fitted TI parameters or pinouts. In particular, these counters use one
positive-edge clock with a level enable; they do not implement the
interchangeable clock/enable edge behavior of those counter packages.

For local supply S = VCC − GND, low is at or below 0.3 S and high at or above
0.7 S. Clock and reset retain their last recognized level between thresholds.
An initially intermediate reset is refused because no safe baseline exists.
An initially intermediate clock can establish its first recognized level
without creating an edge. D/J/K/T/enable must be valid low or high when sampled
on an actual rising edge; an intermediate sampled input refuses that advance
without publishing partial state. Active reset overrides sampled inputs.

The supported powered supply is 2–12 V. A lower or reversed supply clears
memory and disables active outputs; above 12 V is refused. Startup, Restart,
and restored power establish a clock baseline and a zero state. A clock that
starts high, including a clock derived from a settling 555 or another device,
does not invent a rising edge. Loading a saved project starts fresh rather
than resuming the old live counter state; saved experiment captures retain
their original measured rows.

Inputs have 1 GΩ resistance to local GND. A powered output connects through
50 Ω to the rail selected by its stored bit; unpowered outputs have 1 GΩ to
GND. A 1 MΩ supply load remains at all supply states. Outputs therefore droop
under load: use an LED resistor. Conductive branches conserve current and
absorb nonnegative power. There is no modeled output current limit, hardware
rating, input protection, intrinsic capacitance, propagation delay, setup/hold
time, metastability, temperature or noise. Do not use this functional model to
qualify real hardware timing or electrical safety.

Analog integration freezes memory until a clock/reset/power guard changes.
The engine localizes that event to 1 ns or eight floating-point ULPs, whichever
is larger, and preserves capacitor voltage and inductor current across the
zero-time state update. All sequential devices sample the same pre-update
solution, so a common-clock pipeline reads the preceding Q values regardless
of component order. Clocks derived from changing outputs settle through
bounded zero-time iterations. An instantaneous switching loop is refused.
This localization tolerance is numerical, not a physical device delay.

The existing 64 simulated-part, 100-node, source/storage/device limits and
512-attempt / 100 ms integration guards remain. No separate sequential-device
count limit is added. DC sweep refuses circuits with sequential history.
Previously supported circuits retain their existing engine paths.

## Shared user and agent commands

No separate sequential editing API is needed. Native controls dispatch the
same commands as an external client:

- Discover with `parts.list`, `examples.list` and `capabilities.list`.
- Build with `component.place`, `component.rotate`, `wire.connect` and
  `edit.batch`; these five parts have no editable numeric value.
- Load an example through `example.load {"id":"binary-counter"}`.
- Operate with `switch.set`, `simulation.pause`, `simulation.resume`,
  `simulation.restart` and `simulation.step`.
- Inspect through `project.inspect`, named probes, `scope.probes.set`, cursors
  and `simulation.verify`; save/reopen/capture/export through existing commands.

Snapshots include `sequential_model` and `mixed.sequential`. Each result
includes model, stored value, powered/clock/reset memory, numeric state string
(or `unpowered`), observed clock/reset levels, supply/output voltages, complement
voltage for flip-flops, four bit voltages for counters, power and 15 pin currents.
The fixed current order is:

`d, j, k, t, clk, reset, enable, q, nq, q0, q1, q2, q3, vcc, gnd`.

Unused entries are zero. Raw currents enter the device. Output current probes
report **delivered** current; input/supply current probes enter the device.
The primary aliases `v:U1` and `i:U1` refer to Q for a flip-flop and Q0 for a
counter. Primary voltage is relative to local GND; named terminal voltage
probes use the circuit's common reference. All named output currents are
available individually. Command schemas expose the exact 15-element array.

Projects, recovery, captured JSON/CSV and SVG metadata preserve the optional
`sequential_model`, required for graphs containing these parts. Model hashes
include its parameters. Captured import validates required memory fields,
state ranges, complete unique device identities, guard consistency and
voltages/currents/power against the captured terminal voltages and stored
memory. It does not authenticate a capture or replay all prior clock history.
Earlier supported documents and recordings remain readable.

## Verification

`workbench/sequential_test.go` checks truth tables, reset priority, enable/hold,
wraparound, startup/power recovery, threshold hysteresis, sampled input errors,
all pin Jacobians, KCL/power, edge localization, common-clock pipelines in both
component orders, feedback, RC/inductor continuity, a 555-derived clock, strict
capture validation, persistence/recovery ownership and atomic failure paths.
Native tests construct a T circuit through shared commands, load all five
examples and check rotated symbols/hit areas. Export tests check rotated SVG
and closed-switch raster continuity. `scripts/sequential-demo.py` independently
checks expected edge sequences, all counter states, live controls, captures,
saved project reopen and SVG/PNG/PDF output through the public stdio transport.
