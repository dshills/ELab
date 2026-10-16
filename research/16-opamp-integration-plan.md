# Powered op-amp integration

October 16, 2026. In-progress P1 parts expansion after the qualified 0.105 app and demo-session tooling checkpoint. **Not yet a palette component or a released application feature.** The constitutive law, analytic Jacobian, independent references and internal MNA/result integration are implemented. Public graph compilation, editor, probes, captures and native qualification remain required.

## Model and scope

Use five named terminals: `noninverting`, `inverting`, `out`, `vplus`, `vminus`. All internal terminal currents are positive into the device. Public output-current presentation should explicitly identify delivered current as the negative of the output terminal current. Supply currents remain signed into their respective pins. The input pair and power pins remain visible and wireable through the same placement/connection commands users and agents already share.

A macromodel can represent useful amplifier behavior through simplified controlled elements while retaining explicit limits. Analog Devices' [AN48](https://www.analog.com/media/en/technical-documentation/application-notes/an48.pdf) discusses that tradeoff and the importance of knowing which behavior is modeled. Its [LT1013/LT1014 model note](https://www.analog.com/en/resources/design-notes/lt1013-lt1014-opamp-spice-macromodel.html) also treats supply current and output behavior as important model characteristics. TI's [open-loop gain discussion](https://e2e.ti.com/blogs_/archives/b/thesignal/posts/offset-voltage-and-open-loop-gain-they-re-cousins) explains why finite gain leaves a differential input error and changes near the rails. These sources motivate the categories; **the following equations and constants are ELab engineering choices**, not copied vendor models or claimed device specifications.

Candidate `elab.generic-opamp.v1` parameters:

| Parameter | Chosen value |
| --- | --- |
| Center-region open-loop gain | 100,000 V/V |
| Differential input resistance | 10 MΩ |
| Active output resistance scale | 50 Ω |
| Active output-current scale | 20 mA |
| Quiescent-current scale | 1 mA |
| Supply soft-enable scale | 0.1 V |
| Always-present output-to-V− bleed | 1 GΩ |

There is no intrinsic bandwidth/pole, slew rate, offset/bias mismatch, input common-mode range, protection diode, noise, thermal/rating or physical-device identity. External capacitors/inductors supply circuit timing. This memoryless model must not be presented as evidence of a physical amplifier's stability, speed or capacitive-load behavior. Fixed generic parameters follow the existing curated-device approach; manufacturer model import remains a separate requirement.

## Constitutive equations

Let `d = V(noninverting) − V(inverting)`, `s = V(vplus) − V(vminus)`, `A` be gain, and `b = 0.1 V`. For `s > 0`:

- `e = s² / (s² + b²)` enables the active stage and quiescent current smoothly.
- `target = (V+ + V−)/2 + (s/2) tanh(2 A d / s)` stays between the modeled rails and is centered at their midpoint for zero differential input.
- `u = (Vout − target)/(Rout Ilimit)`.
- `Iactive = e Ilimit u / sqrt(1 + u²)` is positive into the output pin. The rational limiter keeps a usable derivative over the existing node-voltage envelope.

For `s ≤ 0`, `e = 0` and `Iactive = 0`. Input resistance and the explicitly declared bleed remain; they are part of this device model, not solver-created grounding paths. Reversed supply will require an explicit explanatory state/warning in the integrated result/UI. This does not model physical reverse-supply protection.

At every supply state:

- `Iinput+ = d/Rinput`, `Iinput− = −Iinput+`.
- `Ibleed = (Vout − V−)/Rbleed`.
- `Iout = Iactive + Ibleed`.
- `I+ = Iq e + max(−Iactive, 0)`.
- `I− = −Iout − I+`.

Thus all five terminal currents sum to zero. The active stage draws from V+ when sourcing and returns to V− when sinking; the bleed returns to V− and quiescent current flows V+ → V−. Terminal power accounts for the input resistance, bleed, quiescent consumption and output-stage loss. Common translation of every terminal voltage must preserve currents and power. The supply-current handover uses the central generalized derivative at zero active current; the complete analytic stamp is checked by independent perturbations.

The target derivative uses an exponential expression for `sech²`, retaining a derivative after floating-point `tanh` rounds to ±1. Infinite normalized target arguments are handled without `∞ × 0`. All inputs to the network stamp must meet the existing finite ±30 V node bounds; no solver work/accuracy or capacity limits are widened to accommodate the device.

## Integration still required

1. Add a bounded op-amp count (initial target two, within existing total part/node/source budgets), part/pin definitions, supported graph classification and intentional pin joins. A follower must be able to join `out` to `inverting`; shorted/equal power rails use the declared inactive model. Preserve rank diagnostics for genuinely unreferenced common modes.
2. The five-current/5×5 stamp, result ownership and terminal-power accounting now run in the shared nonlinear MNA path, including RC and coupled inductor stages. Add public signed per-pin/output/differential/supply probes and operating-state explanations. Broaden the actual-network validation below to the compiler's supported topology envelope; scalar references alone do not establish network convergence.
3. Persist the exact model in projects/recovery/Verify/sweeps and captured dependency hashes. Reject missing/tampered models and unsupported capture IDs. Existing no-op-amp file shapes/model hashes must remain compatible; old readers should refuse the new component rather than silently drop it.
4. Provide a shared triangle symbol with clearly labeled signal/power pins, rotation/routing/printing/export, inspector assumptions, plot controls and guided follower/inverting examples. Use the existing guarded user/agent commands. Refresh/review native source inventory only when native source changes.
5. Qualify blank construction, dual/single/shifted supplies, resistor feedback/load/current limiting, power-off/reversed behavior, mixed storage, save/reopen/recovery, explicit Verify/sweep captures, native/headless parity, actual rendered/keyboard controls and bounded backend performance. Preserve source-pinned packaging and the prior app bundle until qualification passes.

## Foundation evidence

`scripts/opamp-reference.py` independently evaluates the chosen equations with 90-digit Decimal arithmetic and freezes 14 terminal-voltage cases plus six scalar feedback-circuit roots. It does not import Go/device code or a vendor model. Go tests compare all terminal currents/targets, check analytic derivatives by perturbation, exercise 1,000 deterministic voltage vectors for charge/power conservation and bounds, and check common-voltage translation. The amplifier harness eliminates its passive feedback node and solves output KCL independently; it is not the application's network solver.

Initial scalar results include a ±5 V follower at **0.999989811 V** for 1 V input, a gain-10 noninverting stage at **0.999893270 V** for 0.1 V input, and an inverting stage at **−0.999969336 V** for 0.5 V input. A nominal gain-10 request into 100 Ω is current constrained to **1.901253222 V**. Application integration, physical-device accuracy and the remaining product acceptance gates are unproven.

## Actual network integration

`mixed_opamp.go` maps the five terminal currents and every analytic derivative onto physical MNA rows, summing intentional pin joins such as follower feedback. Results own their fixed model and signed terminal currents; output current is explicitly delivered current. Power is the sum of terminal voltage/current products relative to V−, preserving common-voltage translation without an invented ground path. Supply states are `powered`, `unpowered`, or `reversed-supply`; the last two retain only the declared input resistor and output bleed. `cloneMixed` copies the result slice. Existing circuits omit `opamps` from serialized results. Reflected output discovery includes the optional result shape, but `opamp` is still absent from the placeable library and unsupported by public graph compilation.

Actual feedback tests exposed Newton oscillation after a 10 mV perturbation: saturation erases the useful differential slope, and a legacy 0.2 V node proposal jumps across the narrow transition. A per-op-amp proposal limiter now allows a saturated input to approach zero in one update, while limiting outward movement and zero-crossing overshoot to 0.1 normalized tanh-input units. It runs across every Radau stage as well as ordinary MNA. This affects guesses only: the original constitutive equations, 100-iteration attempt limit, node bound, residual tolerances and integration budgets remain. No-op-amp paths keep their prior arithmetic. Op-amp networks also require a converged Newton voltage correction before acceptance, preventing a small 1 GΩ bleed-current residual from hiding a large unloaded output-voltage error.

The six frozen scalar amplifier references pass actual DC and transient-startup network solves, each from cold and perturbed warm guesses. Further checks cover a merged follower Jacobian by perturbation, source/sink sign and power balance, zero/reversed supplies, unloaded equal rails, two cascaded buffers, a floating tied-input common mode, cancellation and owned result copies. A 10 kΩ/1 µF buffered input charges over 50 output samples against its exponential envelope. A buffer driving a damped RLC load runs through the shared simulation kernel and Radau integrator; 20 outputs agree with a half-step run and approach the passive divider's DC limit. These fixtures deliberately construct internal compiled maps; they do not prove palette, file, project/recovery, probe/capture, command or native UI support.

Before public registration, graph validation must treat the input-resistance connection and the output/supply conductive group separately for reference-path checks. Controlled derivatives must not manufacture a common-mode reference. One exploratory entirely ungrounded internal fixture demonstrated that floating-point Gaussian elimination with the existing absolute pivot threshold can accept a spurious tiny pivot; public compilation currently rejects that disconnected island, but a five-pin device's indiscriminate all-pin adjacency would be unsafe. Add explicit physical reference-path validation for the new component and unreferenced input-pair tests before enabling it.

The next implementation slice should establish that compiler/reference contract and model-bearing persistence before exposing placement. Extend the existing model dependencies in `persistence.go`, recovery, verification/import/export and sweep captures; preserve no-op-amp hashes and reject omitted or tampered op-amp models. Then connect `scope.go`, `wire_currents.go` and `mixedSample` to all five signed injections, including delivered-output sign. Library aliases, geometry, desktop drawing/inspector and SVG output need the same five named pins. Required public acceptance still includes blank construction, input and supply edits, pulse-driven RC charge/discharge, two-device networks, captured Verify/sweeps, reopen/recovery, rendered/keyboard controls and native/headless parity. None of these are established by the internal fixtures above.
