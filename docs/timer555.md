# 555 timer

**555 timer** (`timer555`) is an eight-pin component in Semiconductors and All.
Search for “555”, “timer”, “oscillator”, “astable” or “one-shot”. Its functional
symbol keeps timing pins together, numbers each pin, and rotates the pin text
with the body. This is a generic educational CMOS timer, not a fitted TLC555 or
NE555 device or a PCB footprint.

## Examples

| Menu / agent ID | Circuit | What to watch |
| --- | --- | --- |
| 555 LED flasher / `555-flasher` | 5 V, RA 10 kΩ, RB 47 kΩ, C1 10 µF, LED through 1 kΩ | C1 charges to about 3.33 V, output goes low and the discharge pin drains C1 to about 1.67 V. Output then goes high again. The first charge starts at zero and takes longer. |
| Adjustable 555 oscillator / `555-oscillator` | The same circuit with a 100 kΩ pot wired as a rheostat instead of RB | Select P1 and move the wiper while running or paused. Position 1 is fastest; 0 is slowest. Capacitor voltage and the existing latch state survive the adjustment. |
| 555 one-shot timer / `555-one-shot` | 47 kΩ / 10 µF timing network and a brief low trigger, repeating every 2 s | Output initially goes high for about 0.516 s, then the discharge pin drains C1. Plot trigger, C1 voltage, output voltage and LED current. Holding trigger low keeps the output high; Reset overrides it. |

All three select U1 and configure relevant scope traces. CC is a 10 nF control
bypass, initialized to its nominal 3.333 V to make startup of the curated
examples predictable; C1 starts discharged. A circuit drawn from blank can leave
CC at its ordinary zero-voltage default and will settle into operation.

Select U1 to see the latch state, output voltage and delivered current,
comparator levels, supply voltage and power. Plot actions include output,
trigger, threshold and control voltage, and delivered output current. Every pin
also has ordinary voltage and current probes. Scope cursors and captured timing
measurements remain available. Timing is produced by the external circuit, so
ordinary resistor/capacitor edits change it; there is no hidden frequency knob.

## Pins and commands

| Package pin | Named terminal | Function |
| --- | --- | --- |
| 1 | `gnd` | Local timer reference and current return |
| 2 | `trigger` | Below half the control voltage sets the latch |
| 3 | `out` | Resistive drive toward VCC when high, GND when low |
| 4 | `reset` | Below 0.7 V relative to GND forces output low |
| 5 | `control` | Comparator reference; nominally two-thirds of supply |
| 6 | `threshold` | Above control resets the latch when trigger is not low |
| 7 | `discharge` | Low-resistance path to GND when output is low |
| 8 | `vcc` | Positive supply |

Place with `component.place` kind `timer555`, then connect named terminals with
`wire.connect`. Reset, trigger, threshold, GND and VCC require explicit wiring.
Control may remain open because the internal divider references it; output and
discharge may be unloaded. Intentional joins such as trigger/threshold or
reset/VCC are allowed; shorted supply pins refuse.

`examples.list` discovers the three IDs; `example.load` loads them through the
same protected path as the native menu. `component.set`,
`potentiometer.wiper.set`, source and switch commands, `scope.probes.set`,
`simulation.pause`, `simulation.step`, `simulation.restart`, captures, history,
project files and schematic exports use their existing user/agent paths.

```json
{"command":"example.load","params":{"id":"555-flasher"}}
{"command":"scope.probes.set","params":{"ids":["v:C1","v:U1","i:D1"]}}
{"command":"component.set","params":{"id":"C1","value":"22uF"}}
```

These are command/parameter fragments; use the normal protocol envelope,
mutation guards and retry keys from [the agent API](agent-api.md).
`v:U1` is OUT minus the timer's own GND. `i:U1` and `i:U1.out` are delivered
output current; other pin currents enter the device. `mixed.timers555` exposes
`high`, `powered`, state, trip levels, supply/output voltages, absorbed power and
`pin_currents_a` in package-pin order 1–8. Discovery gives that array an exact
length of eight. The inspector marks prior readings stale after paused edits
until an ordinary restart/resume recomputes them.

## Functional model

Model **`elab.generic-cmos-555.v1`** implements a latch with this priority:

1. Supply below 2 V clears the latch and disables output/discharge drive.
2. Reset below 0.7 V forces low.
3. Trigger below half control sets high, including when threshold is high.
4. Otherwise, threshold above control forces low.
5. Otherwise, hold the previous state.

The nominal one-third/two-thirds comparator levels, control terminal, reset
priority, discharge behavior and standard eight-pin numbering follow the
functional behavior described in [TI's TLC555 reference](https://www.ti.com/lit/ds/symlink/tlc555.pdf).
The numerical electrical choices below are ELab parameters, not a fit to TI's
curves, guaranteed limits or frequency specifications.

Three 100 kΩ divider segments are represented by 100 kΩ from VCC to control
and 200 kΩ from control to GND. Trigger, threshold and reset each have a 1 GΩ
path to local GND. A 50 kΩ supply resistor draws quiescent current. Enabled
output drive is 50 Ω to its selected rail; low-state discharge is 10 Ω to GND.
Disabled output and off discharge use 1 GΩ paths. Each resistive branch has
explicit terminal currents and power, including supply and ground returns.
Power/current conservation therefore includes the timer and its load.

The supported supply span is 2–12 V. Above 12 V refuses the staged solve;
reversed or smaller supplies disable drive. Initial/restarted simulations begin
with a cleared latch, then settle the actual powered inputs. A dead-band startup
therefore starts low; a discharged astable timing node sets high. Inputs outside
normal physical common-mode limits have no protection-diode behavior.

No internal charge, propagation delay, temperature effects, noise, tolerances,
output-current rating, short-circuit protection, physical device identity or
maximum operating frequency is modeled. In particular, finite output resistance
is not a physical current-limit model. External capacitors and inductors supply
storage. This model cannot qualify a real timer's switching speed or ratings.

## Switching and numerical bounds

The new **`elab.timer555-hybrid.v1`** engine freezes latch memory during analog
trials. Capacitor networks retain SDIRK2 integration; networks with inductors
retain Radau IIA. Accepted analog half/full substeps check latch guards. Known
waveform frequencies bound scans; memoryless input networks also scan at most
1 ms and check midpoints. A detected first change is bracketed by repeated
ordinary error-controlled trials to 1 ns or floating-point resolution, whichever
is larger. This bracket bound is separate from analog integration error.

At the event, a zero-time solve changes output/discharge resistance while
preserving capacitor voltage and coil current. Distinct left/right edge frames
are retained in live and captured traces, allowing timing measurements between
scope output samples. Identical bookkeeping frames at coincident timer/source/
output boundaries are suppressed only for this new engine. Existing engines
retain their frame ordering and electrical behavior.

Latch memories live in immutable network plans. Trial, canceled or failed work
cannot mutate accepted memory, time or charge. Live wiper/switch/input work
inherits the accepted memory; schematic edits and Restart deliberately create
fresh startup state. Instantaneous inconsistent feedback refuses after a bounded
settling attempt rather than inventing a propagation delay.

The existing 512-attempt / 100 ms per-output integration bounds, node envelope,
source/storage limits and captured-frame bounds remain in force. Very fast or
stiff circuits may need a smaller output step. The established conservative LC
ceiling also applies to timer circuits with inductors, including control-bypass
capacitors; a small bypass can require a shorter sample interval. Two timers
are supported within the existing total component/node bounds. DC sweeps refuse
555 circuits because they cannot preserve latch history; use transient captures.

## Files and provenance

Projects remain `.elab` version 4. Snapshots, project files, recovery inspections,
verification metadata and shared model-bearing document structures carry optional
`timer555_model` parameters. A timer graph requires that exact model envelope;
unsupported/missing parameters refuse import. Kept captures include final latch
state, all pin currents, model hashes and CSV model identity. Imported results
must agree with the saved terminal voltages, allowed latch state and model.
Missing fields, duplicate devices and incomplete eight-current arrays refuse.

Saved schematics preserve configured capacitor startup and wiper intent, then
compute a fresh live run on reopening. Historical captures retain their recorded
latch state and edge samples; they never replace live physics. SVG/PNG/PDF
exports include the numbered symbol and model provenance. Old readers cannot
open the new component kind; old supported files remain readable.

## Verification

Independent RC load-line reductions include declared input/discharge leakage
and finite discharge resistance. They check repeated astable phases and the
one-shot width, external control voltage, LED state and timing at multiple
output steps. Tests also check truth-table priority, all terminal Jacobian
columns, terminal current/power balance, separate timer memories, live wiper
and switch preservation, cancellation/work-budget rollback, and coil-current
continuity at output edges. The references are analytic educational equations,
not measured hardware or a separate SPICE execution.

Native-control tests construct the timer circuit from blank and compare shared
agent results, then exercise all three menu examples, stepping, inspection,
plotting and dirty-work protection. Geometry tests cover rotated pin captions
and body hit testing. Files/captures and pinned probes round-trip through the
same public paths; strict import and schema tests cover the new model/results.

Run `python3 scripts/timer555-demo.py --agent PATH --output REPORT --artifacts
NEW_DIRECTORY` for public stdio discovery, independent timing checks, retained
edge captures, live adjustment, project/capture reopening and schematic exports.
See [the validation record](validation/timer555/README.md) for checks, package
provenance, exported drawings and native observations.
