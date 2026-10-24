# NPN and PNP circuit networks

PNP transistors are available under Semiconductors and through global component
search. Both polarities use named base, collector and emitter terminals, with a
polarity-correct schematic arrow and the same placement, wiring, rotation,
selection, plotting, history and file commands. The default PNP symbol has its
emitter above its collector; rotation changes drawing geometry only.

The network solver handles all three transistor terminals. Emitters no longer
need to connect directly to ground. Supported arrangements include emitter
followers, common-emitter stages with emitter resistance, common-base stages,
diode-connected base/collector terminals and complementary NPN/PNP stages.
They can combine with the existing supported network parts, including external
capacitors and inductors. Existing reference, connectivity, ideal-constraint,
convergence and work limits still apply; this is a bounded educational solver.

## Working examples

Choose one of these entries in the Examples menu, or use `example.load` with
the same ID. `examples.list` returns the question, expected behavior, suggested
edits and model limitations for each template.

| ID | Menu example | What to observe |
|---|---|---|
| `pnp` | PNP high-side LED | A 5 V emitter supply, 10 kΩ base resistor, 100 kΩ pull-up and 1 kΩ LED load. High input turns the LED off; low input turns it on. The default square source switches every 50 ms. |
| `emitter-follower` | NPN emitter follower | A 10 Hz sine with 2.5 V offset and 1 V amplitude drives the base through 1 kΩ. The emitter drives 1 kΩ to ground and follows below the base voltage. Compare input, base, emitter and terminal currents. |
| `bipolar-push-pull` | Complementary transistor buffer | A 10 Hz ±2.5 V sine drives an NPN/PNP pair from ±5 V rails. Each base has 1 kΩ resistance; the shared emitters drive 1 kΩ. The NPN sources the positive half-cycle and PNP sinks the negative half-cycle. The unbiased crossover dead band is intentional. |

Templates start clean, protect subsequent unsaved edits, and support Reset
current example. Pause and use Step, the existing scope probes and cursors to
compare exact stored samples. Playback speed defaults to 0.1 simulated seconds
per wall second for these examples.

## Signed measurements and model

`v:Q1` measures collector minus emitter; `v:Q1.base-emitter` measures base minus
emitter. Individual terminal voltages remain relative to ground. `i:Q1.base`
and `i:Q1.collector` are positive into the device, while `i:Q1.emitter` is
positive out of the device. Thus `IE = IB + IC` for both polarities. A conducting
PNP normally has negative VBE, VCE, IB, IC and IE. These are physical reference
directions, not absolute magnitudes. Power is `VBE*IB + VCE*IC`.

The selected transistor inspector shows its region, VBE, VCE, all three signed
terminal currents and dissipated power. Probes support live scope, cursors,
pins, captured verification, CSV and saved project reopening.

Both generic models use transport saturation current 10 fA, forward beta 100,
reverse beta 1 and 27°C. For an NPN, let
`F = IS*expm1(VBE/VT)` and `R = IS*expm1((VBE-VCE)/VT)`:
`IB = F/BF + R/BR`, `IC = F - (1+1/BR)*R`, `IE = IB+IC`.
The PNP uses the opposite voltage and current polarity with the same illustrative
parameter magnitudes. Analytic derivatives stamp base, collector and emitter
into the network equations; no emitter is implicitly treated as ground.

These are educational Ebers–Moll models, not named parts or a physically matched
pair. They omit Early effect, breakdown, heating, junction capacitance, charge
storage, intrinsic switching delay and physical ratings. External capacitors
provide circuit timing. Regions are cutoff, active, saturation and
reverse-active; cutoff uses the existing 10 nA current threshold.

The existing limit of two bipolar transistors is **combined across NPN and
PNP**, within the other [network bounds](circuit-capacity.md). Snapshots expose
`pnp_transistors` and `bipolar_transistors_total` alongside the retained
`npn_transistors` limit key.

## Shared commands and compatibility

No separate electronics command path is introduced. Agents can place the new
part using `component.place {"kind":"pnp","position":{"x":100,"y":0}}`,
connect `Q1.base`, `Q1.collector` and `Q1.emitter` with `wire.connect`, and use
the existing source, component-value, simulation, measurement, example and file
commands. Discover generated IDs through the command result/selection; use
the normal revision and document guards described in [the API](agent-api.md).

Projects remain `elab.schematic` version 4. A PNP schematic requires the exact
`pnp_model` parameters with ID `elab.generic-pnp.v1`. Captures, embedded model
hashes, recovery inspection and DC sweep sources retain that dependency.
Missing, mismatched or altered PNP parameters are refused. Imported transistor
results are checked against their saved terminal voltages even when no
transistor probe was selected.

Existing grounded-emitter LED/pulse circuits retain their specialized engines;
the NPN flasher retains its established integration path. Broader topologies
route to the general DC, transient, waveform or inductor network engine.
General bipolar networks use voltage-correction convergence, including precise
leakage-scale LED currents; emitter capacitor timing uses the established
general-network integration tolerance.

The NPN numerical law and model ID are unchanged. New files describe the wider
scope accurately; readers also accept the exact historical NPN model description
without changing captured bytes or hashes. Older ELab builds cannot read a new
PNP component they do not implement. Historical passive/RC/MOSFET capacity
reports keep their original limit envelope and do not imply PNP qualification.

## Validation

`workbench/bipolar_test.go` checks independent scalar load lines for the PNP
switch, loaded emitter follower, emitter-resistor stage, common-base stage,
diode-connected junction and complementary transfer curve. It checks the full
three-terminal Jacobian against finite differences, including grounded base,
collector and emitter; KCL, power balance, cancellation and combined limits.

An independent scalar RK4 ODE checks emitter capacitor timing for an NPN and
its polarity-reversed PNP circuit. A coupled coil/capacitor test checks polarity
symmetry and conservation while retaining the production attempt limit; its
race-instrumented execution is not wall-clock latency evidence. Existing frozen
engine captures remain byte-exact on import/export.

Desktop tests build all three circuits from blank through actual view actions
and compare the shared agent results. Menu, stepping, paused edits, plotting, unsaved protection, selected readings and component symbols are covered.
Pinned VBE preferences round-trip with both present and removed owners.

Run `python3 scripts/bipolar-demo.py --agent PATH --output REPORT --artifacts DIR`
to exercise a fresh external stdio agent for each example: discovery, live
stepping, independent follower references, signed PNP switching, capture,
save/reopen and SVG/PNG/PDF exports. These references verify the declared
educational equations; they are not hardware measurements or a separate SPICE
execution. Native visual checks are recorded separately with the candidate.

Background references: [Analog Devices emitter followers](https://wiki.analog.com/university/courses/electronics/text/chapter-9),
[complementary output stages](https://ez.analog.com/adiacademy/university-program/a/studentzone-articles/c/amplifiers/SA961/adalm2000-activity-amplifier-output-stages),
and [ngspice BJT model documentation](https://ngspice.sourceforge.io/docs.html).
