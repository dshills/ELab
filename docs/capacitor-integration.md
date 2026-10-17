# Capacitor network integration

The unreleased source uses two-stage SDIRK2 for general capacitor networks. It repairs a reproduced work-budget collapse in a 64-part, four-capacitor NMOS LED circuit. Users and agents keep the same placement, wiring, value, startup, Run/Step and Verify commands. No method-selection control is required. The canonical 0.105 app remains unchanged until separate release qualification.

## Reproduction

The fixture has 55 series 1 kΩ resistors split into four stages of 14/14/14/13 kΩ, four grounded 100 nF capacitors, a pulse driver, a separate 5 V supply, and an NMOS/1 kΩ/red-LED load: **64 simulated parts /60 nodes including ground**. The insulated gate draws zero current, making the filter independently solvable while its drain load remains nonlinear.

The previous backward-Euler path exceeded 100 ms during its first 1 ms output interval. A controlled-clock regression also reaches attempt 513 after approximately 119 µs of physical advance. This demonstrates an attempt collapse without scheduler pressure. The fixture, sample interval, device assumptions, local accuracy scales and work limits were not reduced to make it pass.

## Stages and error control

For `gamma = 1 - 1/sqrt(2)`, accepted capacitor voltages `Vn` and proposed interval `h`:

1. Solve an implicit companion stage of duration `gamma*h` from `Vn`, with sources at `t+gamma*h`, obtaining `Y1`.
2. Form `history = Vn + (1-gamma)/gamma * (Y1-Vn)`.
3. Solve another companion stage of duration `gamma*h` from that history, with sources at `t+h`, obtaining the final stage.

These are the two nonzero stages of the implicit [ARS222 tableau documented by SUNDIALS](https://sundials.readthedocs.io/en/latest/arkode/Butcher_link.html#c.ARKODE_ARS222_ESDIRK_3_1_2). ELab implements its own MNA solves and does not depend on SUNDIALS. The method is second order, stiffly accurate and L-stable. Each stage retains the existing matrix dimension. Those mathematical properties alone do not guarantee monotonicity, global accuracy or a physical device match.

One full interval is compared with two half intervals, and only the refined result can be accepted. The difference is not divided by a Richardson factor. Growth uses `min(2, cbrt(.75/ratio))`; rejection halves the interval. Every proposal undergoes the complete solves and existing error checks. Source corners still split intervals and retain their existing left/right rows. Intermediate stages never publish physical state.

Existing local scales remain: mixed RC/NPN uses 2 µV plus 0.02%; general networks use 50 nV plus 0.0005%; waveform networks use 500 nV plus 0.0005%; op-amp capacitor control uses 50 nV plus 0.000125%. Coupling-node checks remain. These are local acceptance bounds, not global-error promises. Limits remain **512 attempts and 100 ms per requested output sample**. Time checks occur at stage boundaries and do not preempt an ongoing Newton solve. Failure preserves previously accepted physical state.

The optimized single-RC exponential path, algebraic NPN phase paths and inductor Radau IIA path remain separate. No device law, intrinsic gate charge, node/part limit, source model or startup parameter changes. Capacitor-free networks retain their algebraic equations; their generic engine-family identity advances with this implementation.

Capacitor current is the final implicit stage current. Power balance uses instantaneous terminal power `V × I`, as for the inductor path. Fresh results report zero `numerical_damping_w`; internal SDIRK stage history must not be interpreted as a backward-Euler energy difference. Zero in this legacy readout does not imply that the integration method has no numerical dissipation. Import rejects nonzero damping for current methods and negative/nonfinite damping for any supported method. Historical backward-Euler captures retain their original nonnegative readout.

## Captures and older files

| Analysis family | Current source engine | Earlier engine retained on import |
| --- | --- | --- |
| Mixed RC/NPN | `elab.mixed-mna-sdirk2.v1` | `elab.mixed-mna-be.v1` |
| General network | `elab.network-mna-sdirk2.v1` | `elab.network-mna-be.v1` |
| Per-source waveforms | `elab.network-waveforms-mna-sdirk2.v1` | `elab.network-waveforms-mna-be.v1` |

New snapshots and Verify captures expose the current identity. Earlier captures retain their method and samples; decoding/export does not relabel or replay them. A historical identity is admitted only within its matching analysis family, and the final mixed result must name the exact same engine as the capture. Unknown or mismatched engines refuse. Three genuine pre-change engine-family fixtures re-export byte for byte.

Project and recording shapes/versions remain unchanged. A new recording can require a newer reader because older applications do not recognize its engine. Existing supported schematic models and saved startup intent remain unchanged. Importing an older run preserves the live circuit.

## Public capacity workflows

`elab-conformance --capacity-suite` constructs fresh owned sessions using ordinary commands:

- Fifty potentiometers and a pulse source: **51 parts /100 nodes**, with an explicit final wiper tie and four live probes.
- A 63-resistor chain and a pulse source: **64 parts /64 nodes**, with four voltage/current probes.
- The four-stage NMOS filter above: **64 parts /60 nodes**, four probes, 200 ms captures at 1 ms outputs and a half-step capture.

Series/divider laws and an independent 4×4 matrix exponential check the rows. An 80-digit standard-library Decimal eigensystem separately freezes twelve RC reference times. The MOS load line uses observed gate voltage, whose evolution is separately checked against the RC reference. Both pulse corners are retained. Fresh reopen restores graph/probes, restarts physics and re-exports the recording exactly.

Removing the 100-node fixture's wiper tie creates node 101. Adding a resistor to either 64-part fixture creates part 65. These drawings remain editable and savable with `mixed-limit` guidance; Step refuses without changing accepted state. A paused lifecycle remains `paused`, while graph compilation reports `unsupported`. Fresh reopen preserves the drawing and earlier valid capture.

This is representative capacity coverage. [Repeated timing for the three named boundary fixtures](capacity-performance.md) extends this evidence separately. Visible latency, simultaneous maxima, all device combinations, physical hardware, native accessibility and release/distribution remain separate gates. Difficult circuits can still produce bounded failures.

[Qualification receipt](validation/capacitor-capacity-validation.json) and [retained raw evidence](validation/capacitor-capacity/README.md) bind the tested source, candidate, complete numerical reports and timing scope.
