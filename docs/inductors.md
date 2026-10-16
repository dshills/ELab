# Ideal inductors — working 0.104 candidate

Inductor support is in the working source and private test bundles. It is **not yet in the qualified `build/ELab.app`**, which remains 0.103.0 /118. The [integration plan](../research/14-inductor-integration-plan.md) records numerical choices and remaining release gates.

Search **Inductor**, **coil** or **choke**, then place and wire pins **1** and **2**. Inductance accepts H, mH and µH text, from **1 µH through 100 H**, with a **10 mH** default. Positive voltage is pin 1 minus pin 2; positive current enters pin 1 and leaves pin 2. Stored energy is `L × I² / 2`. Up to four inductors share the existing network, voltage and solver limits.

The selected inspector offers inductance, signed **Startup current** within **±1 A**, reset to zero, and signed live current/voltage/energy. Commit a startup field with Return or focus change. While paused, an electrical edit marks the previous reading stale; choose Restart to apply new startup intent. A fresh run uses configured startup current, default zero. Moving or rotating the coil preserves electrical state. Live source/switch events preserve accepted coil current and capacitor voltage while recomputing the algebraic state.

The bottom summary names the first coil and shows its signed current. Its voltage card names the first capacitor when present, otherwise the first coil. The energy card sums **all L and C stored energy**. Selection does not retarget these named network summaries; select a part or add probes to inspect a different branch.

## Three starting experiments

- **Examples → Coil buildup / decay**: 5 V square input, 100 Ω and 1 H. The 10 ms time constant makes buildup and decay visible. A 50 ms high phase followed by 50 ms low repeats the experiment.
- **Examples → RLC ringing**: 5 V DC, 50 Ω, 100 mH and 10 µF. Compare 50 Ω underdamping, 200 Ω critical damping and 500 Ω overdamping. The default output sample is 100 µs. Plot capacitor voltage with signed coil current and voltage.
- **Examples → Coil / flyback diode**: a pulse-driven generic NMOS switches a 1 H/100 Ω load with an explicit flyback diode. At turn-off, the diode provides a return path and clamps the drain above the supply. Pause near 50 ms to inspect current continuity and signed coil voltage.

These use the ordinary protected example loader and editable parts. Removing a current-return path can demand a voltage beyond the ±30 V educational envelope or an inconsistent ideal topology. Such a solve refuses and retains the last accepted state; it does not invent a leakage path or silently erase stored current.

## Shared user and agent commands

Native controls invoke the same validated commands available externally. These are separate illustrative actions; do not place an ID already present in a loaded example.

```json
{"command":"example.load","params":{"id":"rlc"}}
{"command":"component.place","params":{"kind":"inductor","id":"L1","value":"100 mH","position":{"x":200,"y":0}}}
{"command":"wire.connect","params":{"from":"L1.2","to":"GND1.reference"}}
{"command":"component.set","params":{"id":"L1","value":"1 H"}}
{"command":"inductor.initial.set","params":{"id":"L1","value":"-20 mA"}}
{"command":"inductor.initial.set","params":{"id":"L1","value":"default"}}
{"command":"scope.probes.set","params":{"ids":["i:L1","v:L1","v:L1.1","v:L1.2"]}}
```

Mutations retain the usual revision/document guards, retry keys, history and replacement protection. `parts.list` exposes units, pins, limits, aliases and model identity. `capabilities.list` exposes startup and numerical contracts. Fresh `project.inspect` results include `mixed.inductors` with `voltage_v`, `current_a`, `energy_j` and model `ideal-inductor.v1`.

Independent Verify accepts `inductor_initials: [{"id":"L1","current_a":0.02}]`. An explicit scenario value wins over the component's saved startup intent, then zero is the default. Verify does not alter accepted live state. Steady DC instead treats a coil as zero voltage and solves its branch current; transient startup overrides do not apply to a DC sweep.

Projects save startup intent, not a trusted live solver memory. Reopening starts a fresh paused experiment. Captured recordings retain engine/model dependencies and validate coil IDs, signed values and derived energy on import. SVG/PDF/PNG and native drawing share the coil symbol geometry. The existing preview/printing path consumes the same schematic output; physical printer qualification remains open.

The reviewed PDF/PNG/preview workflow covers the three examples and four coil rotations on single-page Letter landscape. Routing avoids ordinary component-caption masks as well as symbol bodies so a label does not hide the tested rotated wire segments. Waveform annotations depend on experiment settings and are outside this graph-only routing bound. This is bounded automatic routing, not a guarantee that every possible crowded arrangement has a clear route. [Before/after images and exact qualification scope](../research/ui-audit/inductor/README.md).

## Numerical model and limits

`ideal-inductor.v1` omits winding resistance, saturation, mutual coupling, parasitic capacitance, heating, ratings and named physical devices. Add explicit resistance for the intended experiment. The component bounds are software limits, not safe hardware ratings.

L networks use **`elab.rlc-mna-radau5.v1`**, a three-stage fifth-order Radau IIA implementation with coupled device/KCL/storage equations. Adaptive full/two-half-step comparisons check coil current, capacitor voltage and all node voltages; an LC ceiling also limits steps. Existing circuits without L keep their original engines. Existing 512-attempt/100 ms per-output budgets remain; every stage iteration checks the shared deadline. Failed, canceled or superseded output work cannot publish partial time or storage memory.

Instantaneous KCL/power residuals verify circuit equations, not time-integration accuracy. `numerical_damping_w` retains its legacy backward-Euler compensation meaning and is zero for this engine; this does not mean Radau has no numerical dissipation. Independent RL/RLC/LC, LED-coil and flyback references, plus mixed RC/NPN/coil refinement and a separate reduced reference, qualify selected numerical cases rather than every allowed topology or physical device.

The reproducible public workflow constructs an RL circuit from blank, exercises startup/probes/Verify/save/fresh-process reopen, exports all three examples and runs the mixed RC/NPN/coil regression:

```sh
python3 scripts/inductor-demo.py --output build/inductor-report.json --artifacts build/inductor-assets
```

Use fresh output paths. Add `--connect /absolute/path/to/owned-endpoint.json` for a fresh owned native session; it closes that app afterward unless `--keep-open` is supplied. Current development evidence and release limits remain in the [progress record](../research/10-continuation-progress.md).

The conformance runner also accepts `--inductor-suite`. It builds RL through public commands, checks 22 analytic/startup/reopen values, compares seven complete accepted network observations, exercises all three examples and the mixed RC/NPN/coil period, and rejects missing/changed cross-adapter evidence. Supply `--native-binary` to include the fourth adapter; the ordinary in-process, stdio and socket adapters always run. This supplements the original three reference workflows and their optional independent electrical/performance suites. It does not by itself qualify coil performance or physical devices.
