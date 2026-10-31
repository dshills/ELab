# H-bridge motor driver

`hbridge` is a generic single bridge with terminals, in result-array order:
`in1`, `in2`, `enable`, `out1`, `out2`, `vm`, `gnd`. Connect the motor between
OUT1 and OUT2, VM to its supply and GND to the control reference. The part is in
**Motors**, searchable by H bridge, motor driver, forward, reverse, brake, coast
or PWM. Its automatic reference prefix is U.

| ENABLE | IN1 | IN2 | Output command |
| --- | --- | --- | --- |
| Low | Either | Either | Coast: both active output stages disabled |
| High | High | Low | Forward: OUT1 to VM, OUT2 to GND |
| High | Low | High | Reverse: OUT1 to GND, OUT2 to VM |
| High | Low | Low | Brake low: both outputs to GND |
| High | High | High | Brake high: both outputs to VM |

Low is at or below 0.8 V and high at or above 2 V, relative to the device GND.
These are fixed input thresholds, independent of VM. Intermediate control
voltages are reported explicitly. ENABLE can be driven by a finite pulse source
or 555 output for drive/coast PWM. PWM on a direction input can alternate drive
and brake. Braking produces opposing motor torque; it does not force RPM to zero
instantaneously. Coasting retains a path for winding current through the clamps.

## Working example

**Reversible PWM motor driver** (`hbridge-motor`) connects a 12 V supply, default
brushed DC motor and U1. S1 and S2 connect the actual direction pins to VM;
external 10 kΩ pulldowns define open inputs. SEN connects a separate 5 V,
100 Hz finite pulse source to ENABLE. Default duty is 50%, with 200 µs edges.
The example samples at 50 µs with slow playback so switching current is visible.

- S1 closed / S2 open: forward; swap them for reverse.
- Both direction switches open: brake low. Both closed: brake high.
- Open SEN: coast, regardless of the direction switches.
- Scope **High** holds ENABLE high for a clean brake/coast comparison;
  **Low** holds it low, and **Auto** restores PWM. These use the ordinary
  `simulation.input.set` command on the primary PWM source.
- Select PWM and edit high/low phase lengths to change duty. These source
  configuration edits use the usual startup/restart rules.
- Select MTR1 to see signed RPM, winding current and back EMF. Compare brake and
  coast from the same initial speed. Select U1 for mode, input levels, output
  voltage, signed VM current, dissipation and four clamp currents.

## Shared commands and observations

No hidden mode override or second control API exists. Native input and agents
use ordinary `component.place`, `wire.connect`, `switch.set`, `source.pulse.set`,
`simulation.step`, `scope.probes.set`, `simulation.verify` and project commands.
Agents discover the template with `examples.list` and load it with
`example.load {"id":"hbridge-motor"}`. For example, on that circuit:

```json
{"command":"switch.set","params":{"id":"SEN","closed":false}}
```

Real requests use the standard envelope, document/revision guards and retry key.
A paused switch change queues until the next accepted step; accepted motor speed,
armature current and time remain unchanged until that step. The existing input
handoff preserves stored electrical/mechanical state at the switching instant.

`mixed.hbridges` contains model identity, commanded mode, input classifications,
VM and differential output voltages, seven signed currents **into** the driver,
four clamp currents and dissipation. Clamp array order is OUT1→VM, GND→OUT1,
OUT2→VM, GND→OUT2; all clamp values are nonnegative. A negative VM current returns energy to the
supply. `v:U1` measures OUT1 minus OUT2; each terminal has voltage and signed
current probes, such as `v:U1.enable`, `i:U1.vm` and `i:U1.out1`. Output current
probe signs also enter the driver. This convention is retained in wire-current
cues, scope CSV, recordings and saved projects.

## Explicit model

`elab.generic-hbridge.v1` uses four controlled passive legs: 0.25 Ω fully on,
1 GΩ off. Each control has a 1 MΩ pulldown; VM has a 100 kΩ supply load. A smooth
supply enable is zero at/below 2 V and reaches one at/above 3 V. Intermediate
inputs use a smooth conductance transition whose high and low legs never conduct
actively together. This is an algebraic transition, with no simulated temporal
dead time or gate energy.

Four always-present passive flyback clamps conduct from each output to VM and
from GND to each output. Their piecewise-linear law is zero below 0.65 V and
`I = (V - 0.65 V) / 0.25 Ω` above it. They remain present when disabled or
unpowered. They can return energy to the supply. An ideal voltage source absorbs
this energy; a battery or storage network must be modeled explicitly to study
its voltage response. Every branch contributes balanced KCL, analytical control
and voltage derivatives, and nonnegative dissipation.

This functional IC is one simulated part and adds no hidden nodes or storage.
Existing part/node/source/storage and integration work limits apply unchanged.
AC frequency analysis refuses this switching device rather than presenting a
misleading linear frequency response. There is no fitted IC identity, current
limit, thermal shutdown, rated voltage/current, propagation delay, switching
loss or physical gate-drive model.

The enable/output arrangement follows the general approach illustrated by
[TI's L293 driver reference](https://www.ti.com/lit/ds/symlink/l293.pdf).
[TI's DRV8833 reference](https://www.ti.com/lit/ds/symlink/drv8833.pdf) explains
winding-current recirculation and fast/slow decay; its two-input truth table and
hardware protections differ from ELab's model. Neither datasheet defines ELab's
numerical model or establishes physical component suitability.


## Four-MOSFET bridge

Open **Four-MOSFET H-bridge** from Power & relays. The direction pulse drives one
PMOS/NMOS pair directly and a powered logic inverter drives the other pair.
The 100 Ω load connects between their drains. In the first 20 ms the left
output is low and the right output is high. The next 20 ms reverse the polarity.
The scope shows the direction input, differential load voltage and load current.

With the 5 V supply, expect about −4.725 V and −47.25 mA in the first direction
and +4.725 V and +47.25 mA in the other. Each conducting MOSFET drops about
137 mV. Pause and Restart, then step across 20 ms to see the change. A recording
longer than 40 ms includes both directions and the return. Keep and save the
recording to reopen it beside the same circuit. The models omit gate charge,
switching losses and dead time.
