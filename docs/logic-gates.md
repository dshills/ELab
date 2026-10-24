# Basic logic gates

ELab has seven powered, single-function gates in the **Logic** category:
NOT (`logic-not`), AND, OR, NAND, NOR, XOR and XNOR (the other kinds use the
same `logic-` prefix). Search finds names, CMOS, boolean and digital; searching
“inverter” finds NOT. These are schematic functions, not packaged multi-gate ICs.

Each gate has `a`, `out`, `vcc`, and `gnd` pins. Two-input gates also have `b`.
Connect both power pins and every input; an unloaded output is allowed. Input
joins, tied inputs and cascading are allowed. A VCC/GND short is refused.
Use explicit pull-downs or pull-ups with switches; wires crossing on the sheet
do not make electrical connections. IEC functional rectangles show `1`, `&`,
`>=1`, or `=1`; an output bubble marks inversion. All geometry, pins and pin
text rotate together, and the component browser uses the same symbol renderer.

## Live examples

- **Logic inverter LED** (`logic-inverter`): finite 0–5 V pulses turn D1 on
  when A is low. The scope shows A, output and LED current.
- **AND gate with switches** (`logic-switches`): SA and SB each have an explicit
  pull-down. Try all four combinations; D1 lights only with both closed. This
  circuit is DC: while paused, restart through the shared command or Resume to
  recompute edited startup states. Running switch edits update automatically.
- **Logic half adder** (`logic-half-adder`): two pulse sources exercise all four
  input combinations. U1 XOR produces sum/D1; U2 AND produces carry/D2.

The selected gate's inspector reports A/B/output levels, actual output voltage,
delivered current, local supply and absorbed power. Plot buttons add inputs,
output and output current to the existing scope; named terminal probes,
cursors, saved captures, schematic export and printing work as for other parts.

## Electrical contract

The fixed `elab.generic-cmos-logic.v1` model is a continuous combinational
macromodel. Logic functions agree with the conventional truth tables; the
[TI NAND reference](https://www.ti.com/product/CD4011B) and
[TI XOR/XNOR reference](https://www.ti.com/product/CD4070B) provide background.
ELab's numeric parameters are its own educational choices, not fitted TI data.

For positive local supply span S = VCC − GND, an input is low at or below
0.3 S and high at or above 0.7 S. Between those levels it is **intermediate**.
The interpolation q = 3x² − 2x³ uses x = (Vin/S − 0.3)/0.4, clamped to [0,1].
It is a numerical analog interpolation, not a guarantee of digital operation.
The corresponding truth polynomial combines the input weights: AND ab,
OR a+b−ab, XOR a(1−b)+(1−a)b; NAND/NOR/XNOR complement those, and NOT is 1−a.

For resulting weight q, output-to-VCC conductance is e q / 50 Ω, and
output-to-GND conductance is e (1−q) / 50 Ω, where e = S²/(S²+0.1² V²).
Both active paths are disabled for zero/reversed supply. Every input has a
1 GΩ resistor to local ground; OUT has a permanent 1 GΩ bleed to local ground.
A 1 MΩ supply load remains at all supply states. An inverter has no hidden B
terminal/current. All conductive branches conserve current and absorb
nonnegative power; intermediate inputs can produce static supply current.

The output is load dependent. Add a resistor for an LED. There is no modeled
output current limit or physical rating. The model has no intrinsic storage,
propagation delay, hysteresis, input protection, temperature or noise. An
external capacitor/inductor supplies circuit memory using the existing
SDIRK2/Radau engines and work bounds; DC and memoryless networks remain
algebraic. Feedback can settle at an intermediate analog equilibrium. It is
not a digital latch or ring-oscillator timing model.

For explicit stateful devices, use the separate [flip-flops and counters](sequential-logic.md).

The existing 64 simulated-part, 100 electrical-node and device/source limits
still apply; no extra gate-specific capacity field or engine is introduced.
Previously supported graphs take the original numerical paths.

## Users and agents

Native drawing, menus, switch controls and plot actions dispatch the same
commands as external clients. No separate logic-only editing API is needed.
Discover kinds with `parts.list`, categories in `project.inspect` (`part_categories`),
and examples with `examples.list`; load one using `example.load` and its ID.
Use `component.place`, `wire.connect`, `component.rotate`, `switch.set`,
`simulation.pause`/`resume`/`restart`/`step`, `scope.probes.set`,
`simulation.verify`, and the existing project/export commands.

`project.inspect` publishes `logic_model` and `mixed.logic_gates`. Per-device
results include input/output level strings, local output/supply voltages,
absorbed power and five pin currents ordered A, B, OUT, VCC, GND (B is zero
for NOT). These raw currents enter the device. Probes `i:U1` and `i:U1.out`
report **delivered** output current; the other terminal currents enter it.
`v:U1` is OUT minus local GND; individual terminal voltage probes use the
circuit's explicit common reference. Command schemas expose the fixed model
and exact five-current array.

Projects, recovery inspections, verification JSON/CSV and schematic SVG
metadata preserve the optional `logic_model`. Schematics containing gates
require the exact supported model. Saved captures validate gate identities,
complete pin arrays, levels, voltages, currents and power against the captured
terminal voltages, without replaying the circuit. Model hashes include the
logic parameters. Existing documents and historical captures remain readable.

## Verification

`workbench/logic_test.go` checks all truth rows at 3.3/5/12 V with translated
local grounds, finite load droop, analytic Jacobians, passive power/current
balance, zero/reversed/vanishing supply, intermediate levels, cascading,
independent output RC response, timer-to-gate integration, all examples,
strict captures, project recovery and probe signs. Native-control tests draw
from blank, exercise the same commands as an agent, load every example and
check symbol rotation. `scripts/logic-demo.py` replays the public stdio route
against a selected agent binary and saves example exports.
