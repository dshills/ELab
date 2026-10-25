# Brushed DC motor and PWM control

`dc-motor` has `positive` and `negative` electrical pins. Positive armature
current enters `positive`; positive speed/torque use that polarity. The Motor
browser category and global search include motor, RPM, fan, PWM and back EMF.
The inspector shows live RPM, armature current, back EMF, torque, winding/rotor
energy and losses. A blocked-rotor reference current is terminal voltage / R;
it is a DC reference, not the instantaneous current during a PWM pulse.

The generic reciprocal model is:

- V = R I + L dI/dt + Ke ω.
- J dω/dt = Ke I − (friction + load) ω, with Ke = Kt in SI units.
- RPM = ω × 60 / (2π).
- Winding heat = R I²; drag loss = (friction + load) ω².
- Stored magnetic/rotational energy = L I²/2 + J ω²/2.

The [Portescap PWM reference](https://www.portescap.com/en/newsroom/whitepapers/2022/03/controlling-brushed-dc-motors-using-pwm)
describes winding RL/back-EMF behavior. ELab's chosen parameters are illustrative,
not fitted to a Portescap motor. No brushes, commutation ripple, gearbox, heating,
ratings or intrinsic parasitic capacitance are modeled. A viscous load opposes
motion in either direction and can absorb shaft energy; it is not a constant
external driving torque. Reverse drive and regenerative current are represented.

## Shared commands

`motor.configure` accepts an ID and a complete `parameters` object:

```
{"id":"MTR1","parameters":{"locked_rotor":false,"armature_resistance_ohm":8,"armature_inductance_h":0.01,"ke_v_s_per_rad":0.02,"inertia_kg_m2":0.000001,"friction_n_m_s_per_rad":0.00001,"load_n_m_s_per_rad":0}}
```

Domains: R 0.1–100 Ω; L 10 µH–1 H; Ke 0.001–1 V·s/rad;
J 0.1–1000 µkg·m²; friction 0.1–10000 µN·m·s/rad; load
0–10000 µN·m·s/rad. Configuration is a saved, undoable startup edit;
running circuits restart and paused readings wait for Restart/Resume.
A new run starts at rest with zero armature current. Locking the rotor is a
fixed zero-speed mechanical constraint with reaction torque, not an artificial
very-large friction or deletion of momentum inside a running trajectory.

`motor.load.set {"id":"MTR1","load_n_m_s_per_rad":0.0001}` changes only
viscous load. The logarithmic slider and exact load field call this same
command as agents. A fresh transient stages `input_work.requested_motor_load`
at accepted time, preserving shaft speed, winding current, other capacitor
voltages and battery charge. Only successful publication saves the load and
one undo entry; failures, cancellation and supersession retain accepted state.
Returning to committed load cancels its pending request. Stale/incomplete/DC
and batch edits configure startup. Wait with `simulation.wait` before dependent
work, or cancel the exact work ID with `simulation.work.cancel`.

Dynamics use the existing coupled Radau integration and atomic output publication.
Angular error is checked in physical rad/s as well as the internal scaled
storage coordinates; no simulation work/accuracy guard is increased. Private
primitive parts are not exposed as authored objects/results/nets. One motor
reserves an internal source equation and winding slot; a free rotor also uses
mechanical capacitor/resistor slots. Existing total part/node/resource limits
remain in force. `mixed.motors`, `v:MTR1`, `i:MTR1`, terminal inspection,
project settings, captures and exports use the original motor identity.

## Examples and checks

**DC motor startup and load** (`motor-dc`) shows inductive current rise followed
by back EMF as speed rises. With the default 12 V, 8 Ω motor, blocked-rotor DC
current approaches 1.5 A; free speed approaches about 4,775 RPM. Increase load
live to slow it; use locked startup to compare blocked current.

**555 PWM motor controller** (`motor-pwm`) drives an N-channel MOSFET from a
555/potentiometer oscillator. P1 changes timing/duty; MTR1 load changes live.
D1 carries flyback current. RSN/CSN form an explicit 100 Ω /1 µF snubber, damping
the switching transient in a model with no hidden parasitic capacitance.
Compare timer output, motor voltage/current and flyback current with RPM.
The PWM example uses 50 µs output samples and 0.005× playback so switching
intervals stay short enough for live work; the DC example uses 100 µs samples.
The PWM electrical regression covers 40 ms at each of three wiper positions,
including both extreme duty settings, with the same accuracy and work limits.

Schematic exports and printing include every saved motor parameter and the
rotor mode. Long parts descriptions wrap inside their column, and narrow
schematics use a single column. This also keeps winding and battery details
readable without crossing neighboring entries.

Tests check analytic steady speed, load lines, locked RL current, power/energy,
private-state ownership, live continuity and capture integrity. The event locator
reuses an accepted unchanged lower bracket instead of re-integrating its prefix
at every bisection; every trial retains the original estimator and work budget.
