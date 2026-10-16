# Stacked and floating DC supplies

Choose **Examples → Split-rail DC supply**. Two ideal 3 V sources in series share a grounded midpoint. The top rail is +3 V and the bottom rail is −3 V. R1 spans both rails: 6 V across 1 kΩ gives 6 mA and 36 mW. Each source delivers 6 mA and 18 mW. This is an ideal-source experiment, not a physical battery or power-supply rating.

Select V2. Its negative-terminal reading is **−3 V relative to ground**; its component voltage is **+3 V**, meaning positive minus negative. The source's positive terminal is the zero-volt midpoint. Source current stays positive **out of the positive terminal**; positive source power means delivered power. A source can absorb power in another operating point, producing negative delivered power.

Change either source to 5 V, or move the ground connection from the midpoint junction to V2's negative terminal. With V1 at 5 V and V2 at 3 V, bottom-referenced rails become 0, +3 and +8 V. Moving the reference changes node readings without changing the voltage differences and current around the same electrical loop. Geometry edits and electrical rewiring retain their existing history/state rules; moving a ground symbol's drawn position alone does not change its connection.

## Build and inspect through shared commands

Use the ordinary `component.place`, `component.rotate`, `wire.connect` and `component.set` commands to build the example. Agents discover the template as `split-supply` through `examples.list` and load it with `example.load`. The native alias `example.split-supply` routes to that same command and retains Save/Discard/Cancel protection; no public command was added for this template.

`measurement.inspect` can query `v:V2.negative`, `v:V2`, `v:R1`, `i:R1` and `i:V2` without changing selection. Native component/terminal/probe inspection and pinned readings use the same signed values. Static DC remains a single operating point; existing recorded verification can capture it, while live scope plotting remains reserved for transient analyses.

`parts.list` explains DC source terminal references and model assumptions. Detailed `component.place`/`component.set` discovery includes `x-source-reference-rules`. Ordinary source values stay 0–12 V; the existing DC sweep bias override may be signed. All source models, pin names, serialized parameters and file versions remain unchanged. Save/open, recovery, recorded Verify, signed sweeps and SVG/PDF/PNG use the existing paths.

## Supported topology and numerical bounds

Only **DC** sources gain floating terminals in this checkpoint. Pulse, sine and finite-pulse sources still require a directly grounded negative terminal, and NPN emitters remain grounded. Every network needs an explicit common reference and connected required terminals. An electrically isolated source is not made solvable by adding a decorative ground elsewhere.

The existing MNA branch already enforces `Vpositive − Vnegative = source voltage`. Compilation now uses both actual terminals when checking independence of ideal source constraints. Source loops—whether redundant or contradictory—and capacitor/source startup constraint loops refuse rather than inventing currents. A capacitor directly across a source still refuses the existing impulse-model case. A resistor in the return path is supported: direct source-positive to capacitor is not itself a parallel source/capacitor constraint.

The three-source capacity, component/node limits, Newton iteration ceiling and integration attempt/time budgets stay unchanged. [Linear source/passive networks now take full affine corrections](linear-network-solves.md); nonlinear circuits retain their 0.2 V update cap. Accepted node voltages must remain within ±30 V. Review found a converged initial guess could bypass that range check; acceptance now checks it as well as the existing check after updates. A stack within per-source input bounds can still fail the overall range or convergence limits. Such failure remains explicit; no simplified model or guessed state is substituted.

## Validation and remaining work

Independent Ohm/Kirchhoff references check split and stacked rails, currents, differential voltages and power conservation. A shifted-reference LED agrees with the existing independent current-domain series solution. Two shifted RC arrangements—including resistance in the return path—match the analytic 10 ms time constant within the retained 3 mV tolerance on the ordinary 1 ms output grid. A direct 10 ms kernel advance initially hit the existing per-output work limit; the test now uses the real output grid without raising that budget.

Regressions cover floating waveform refusal, source/capacitor constraint loops, the out-of-range converged seed, native blank construction, signed Inspector values, protected example reset, recorded captures/save/reopen and sweeps. The public reference `scripts/floating-dc-demo.py` builds from blank, compares with the template, exports SVG/PNG and captured JSON/CSV, reopens the kept recording, moves the reference, runs a signed sweep and checks RC charge/discharge with time and capacitor voltage preserved at the input switch. Native/headless runs use the same commands and compare their observations and schematic SVG/PNG bytes.

Native/output inspection caught DC-source and vertical-resistor labels crossing their own wire stubs. Those native labels now sit beside their symbols, and direct inspection confirms the new example is clear. General label/crossing layouts, exhaustive mounted input/accessibility, visible latency, clean distribution, participant and physical-printer qualification remain separate open work. This feature does not add real batteries, configurable semiconductor models or floating waveform sources.
