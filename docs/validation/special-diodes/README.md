# Zener and Schottky validation

See [the model and workflow contract](../../special-diodes.md).

The new electrical tests use fixed-parameter inverse load lines, closed-form
regulator line/load equations, piecewise capacitor charge/release exponentials,
a piecewise inductor reverse-clamp decay, and a separate RK4 rectifier ODE.
Maximum errors in the pinned fixtures: Schottky/silicon rectifier 21.6 µV,
capacitor clamp charge 37.1 µV and release 30.7 µV, coil decay 92.1 nA.
Current and power signs, KCL, power balance, Jacobians, knee continuity,
regulation loss, strict model/result imports, detached copies, value history,
combined diode capacity and kept-record round trips are covered. The final
focused electrical/pinned-probe checks pass under the race detector.

The complete root race suite passes, including conformance, transport/socket
and workbench tests. Lint and root/desktop vet pass. The final example layout
uses separate ground symbols to avoid overlapping ground and output wires;
the subsequent focused electrical/pinned and UI/agent checks pass.

The native-control/agent parity tests construct both diode circuits from blank,
edit Zener voltage, inspect signed results and plot actions, and exercise both
example menus, live stepping and unsaved replacement protection. The command
inventory retains 166 public commands, 404 source sites and 134 literal routes;
two native menu aliases target the existing `example.load` command.

The external stdio replay checks public discovery, both examples, captures,
Zener reference equations, Schottky peak comparison/reverse leakage, live steps,
value editing, kept-record save/reopen and SVG/PNG/PDF exports. The retained
exported PNGs were visually inspected for complete symbols and readable labels.
These are educational equation references, not measured devices or an
independent SPICE execution. Source and callback tests are separate from native
pointer/keyboard acceptance. Packaged and native observations are appended at
completion; the preserved /205 release archive is not replaced.


## Completion

Clean-source **0.184.0 /234**, from `d498adad`, passes strict ad hoc signature
verification. Its packaged agent passes both example workflows, references,
recording round trips, edits and schematic exports. The full desktop offscreen
race suite passes (454.278 s). The full root race suite passes, including
workbench (94.305 s) and conformance (121.896 s); the later example-ground-only
layout adjustment and added coil reference pass the final focused race checks.
Lint, root/desktop vet and native command-inventory verification pass.

Native /234 shows the Apollo Zener regulator with the 8–12 V input and nearly
steady output. Pointer Pause, keyboard entry of 3.3 V, Apply and Restart produce
a fresh 3.43243 V output at 10 V input. The edit remains stale until the existing
restart/resume action, as specified. Loading Schottky from that edited fixture
shows the ordinary unsaved dialog; only the owned synthetic edit was discarded.
The Schottky view shows separate ground returns, distinct diode symbols, and
C1 above C2 on the live scope. It remains open, unmodified and running.

CUA initially rejected stale control snapshots during live updates. A fresh
state read immediately followed by the pointer action succeeded. This is
recorded as a tool observation, not an established mouse defect or a latency
measurement. Feature observations do not replace broader distribution gates.
The preserved /205 archive hash is unchanged; no public release was promoted.
