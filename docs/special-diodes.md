# Zener and Schottky diodes

The component browser includes **Zener diode** (`zener`) and **Schottky diode**
(`schottky`) under Semiconductors and All. Search also finds “voltage clamp”,
“shunt regulator” and “low voltage rectifier”. Each has a distinct cathode symbol
on the canvas, in the browser and in schematic exports. Both terminals retain
the ordinary `anode` / `cathode` names and rotation behavior.

## Try the examples

| Examples menu | Agent example ID | Experiment |
| --- | --- | --- |
| Zener voltage regulator | `zener` | An 8–12 V, 10 Hz input feeds a 470 Ω ballast, reverse-biased 5.1 V Zener and 4.7 kΩ load. Output stays near 5.2 V while diode current changes. Change D1 to 3.3 V, or increase R1 to see regulation fail. |
| Schottky rectifier | `schottky` | Identical 100 Ω / 10 µF / 10 kΩ half-wave branches share a 1 V peak, 10 Hz input. C1 behind the Schottky charges higher than C2 behind the silicon diode. Both discharge between peaks. |

Pause, step or restart through the existing simulation controls. Select either
diode to inspect voltage, signed current and absorbed power, and use its Plot
voltage / Plot current actions. The Zener inspector labels its numeric field
**Breakdown voltage · V** and shows the 1–24 V bounds and knee convention.
The Schottky uses fixed generic parameters and has no editable voltage drop.

## Same commands for people and agents

No diode-specific command path is introduced. `parts.list` exposes model IDs,
terminal names, aliases, limitations and the Zener value domain. `examples.list`
describes both experiments. The UI uses the same protected `example.load`,
`component.place`, `component.set`, `component.rotate`, `wire.connect`,
`scope.probes.set`, simulation, undo/redo and project/export commands.

```json
{"command":"example.load","params":{"id":"zener"}}
{"command":"component.set","params":{"id":"D1","value":"3.3 V"}}
{"command":"scope.probes.set","params":{"ids":["v:RL","v:D1","i:D1"]}}
```

These are command/parameter fragments; use the normal protocol envelope,
mutation guards and retry keys documented in [the agent API](agent-api.md).
Agents can draw either circuit from blank with `component.place` kinds `zener`
or `schottky` and ordinary named-terminal wires. User and agent value edits
share unit parsing, bounds, dirty protection, paused/stale behavior and history.
The Zener defaults to 5.1 V; values such as `3300 mV` are accepted. Out-of-range
or malformed input preserves the graph, accepted result and history.

## Electrical models

The signed voltage is `V = Vanode − Vcathode`; positive current enters the
anode and leaves the cathode. Absorbed power is `V * I`. Reverse breakdown
therefore has negative voltage and current and positive power.

Schottky model **`elab.generic-schottky-diode.v1`** solves the implicit law

```
I = Is * expm1((V − I*Rs) / (n*Vt))
Is = 2e−7 A, n = 1.05, Rs = 2 Ω, T = 27°C
Vt = k*(273.15 + T)/q
```

It uses a monotone junction/load-line solve with its analytic differential
conductance. Reverse saturation leakage approaches −0.2 µA; it is not a perfect
open circuit. There is no Schottky breakdown model.

Zener model **`elab.generic-zener-diode.v1`** combines the existing silicon
junction (`Is = 1 nA`, `n = 1.7`, `Rs = 1 Ω`, 27°C) with a continuous reverse
knee. For the component's editable `Vz`:

```
I(V) = Isilicon(V) + min(0, (V + Vz)/10 Ω)
```

Its reverse dynamic resistance is 10 Ω, including reverse series loss. The
Jacobian adds 0.1 S below `−Vz`. At exactly the knee the current is continuous
and the blocking-side derivative is used. `Vz` is the zero-breakdown-current
knee, **not a datasheet voltage specified at a test current**. A load carrying
10 mA of breakdown current consequently develops about `Vz + 0.1 V`.
Forward conduction retains the original silicon law. Neither the existing
silicon model nor the MOSFET body-diode parameters have changed.

These are illustrative educational models, with no physical part identity,
ratings, tolerance, heating, temperature coefficient, noise, soft breakdown
knee, capacitance, charge storage or reverse-recovery timing. External capacitors
and inductors still contribute circuit dynamics. The lower forward-drop versus
reverse-leakage tradeoff and the reverse-knee/dynamic-resistance behavior follow
the concepts in [Nexperia's Diode Application Handbook](https://www.nexperia.com/dam/jcr:05f224d4-6db3-4459-95fb-bbdf8a2c3cc3/Nexperia_document_book_DiodeApplicationHandbook_2022.pdf).
The [onsemi BAT54 datasheet](https://www.onsemi.com/download/data-sheet/pdf/bat54s-d.pdf)
provides physical-device context for low-voltage Schottky behavior; ELab's
parameters are **not fitted to, or a substitute for, BAT54**.

## Persistence, recordings and limits

Projects remain `.elab` version 4. Optional `zener_model` and `schottky_model`
envelopes carry exact versioned parameters in snapshots, saved projects,
inspections, recovery, verification captures and DC-sweep source documents.
They are required when their respective parts are present; unsupported or
missing models refuse import. Captured model hashes and CSV model columns use
the captured parameters. Copies returned to callers own their model data.

`mixed.diodes` holds all three diode kinds; the graph's kind and model identify
each result. Recording validation checks complete, duplicate-free per-kind
inventories and voltage/current/power agreement with the saved terminal state,
even for diodes not selected as probes. Reopening computes fresh live physics;
kept recordings remain historical data.

The existing **eight-diode total budget** is shared by silicon, Schottky and
Zener devices. The legacy `limits.silicon_diodes` JSON key is retained and now
represents that combined budget; diagnostics spell out the three kinds.
No part/node/source/storage limits were increased. Older ELab versions cannot
open the new component kinds; existing schematics retain their model behavior.

## Verification

`workbench/special_diodes_test.go` checks independent inverse load lines,
forward/reverse operation, signed power and conservation, monotonicity and
finite-difference conductance, knee continuity, line/load regulation and loss
of regulation, DC sweeps, capacitor clamp/release against piecewise exponential
solutions, inductor reverse-clamp decay/release against a piecewise exponential, and both examples against closed-form/RK4 references. Tests also
cover unit edits, invalid edits, undo/redo, detached model copies, model-bearing
save/reopen, model tampering, captured-result tampering, mixed provenance and
the combined eight-part bound. Pinned readings exercise both new kinds through
save/reopen and missing-owner handling.

`desktop/special_diodes_test.go` draws circuits through native palette/value/
named-wire controls and compares them with ordinary agent commands, including
Zener edits, signed readings, plot actions, example loading, live steps and
unsaved-work protection. Existing browser tests require a unique nonempty
symbol for every catalog entry.

Run `python3 scripts/special-diodes-demo.py --agent PATH --output REPORT
--artifacts NEW_DIRECTORY` for public stdio discovery, capture, live stepping,
independent Zener references, Schottky comparison/leakage, editing, save/reopen
and SVG/PNG/PDF export. The equations are independent references; these checks
are not measured hardware or a separate SPICE execution. Native visual checks
and broader regression results are recorded separately in
[the validation ledger](validation/special-diodes/README.md).
