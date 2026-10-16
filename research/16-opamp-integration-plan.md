# Powered op-amp integration

October 16, 2026. In-progress P1 parts expansion after the qualified 0.105 app and demo-session tooling checkpoint. **Source feature implemented; packaged application not yet released.** The constitutive law, analytic Jacobian, independent references, MNA/result integration, compiler reference checks, shared placement/wiring, metadata persistence, probes and native/export symbol are implemented. Guided examples and scoped mounted visual testing are now implemented; unresolved mounted editing/accessibility, broader adapter/performance and application qualification remain required. The canonical app remains 0.105.0.

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

## Integration requirements and current scope

1. **Implemented in source:** a two-op-amp ceiling within existing total part/node/source budgets, five named pins, classification and intentional pin joins. Follower feedback and equal power rails compile; an unloaded output is allowed. Reference checks use the input-resistor pair and output-to-V− bleed separately; V+ requires an external path. Ideal MOS gates cannot supply references in op-amp networks. Existing no-op-amp compilation retains its prior paths.
2. **Implemented in source:** the five-current/5×5 stamp in DC, RC and coupled inductor stages, result ownership, signed per-pin/output/differential/supply probes, unique wire injections and terminal power accounting. Actual cold/warm feedback, source/sink/off/reversed, RC/RLC, two-stage and cancellation checks run under existing numerical bounds. Broader topology/performance qualification remains required.
3. **Implemented in source:** exact model metadata in project/recovery, Verify, imports/exports and sweep sources; canonical dependency hashes and CSV provenance. Missing/altered models, invalid/missing captured device IDs, inconsistent readings and missing ground refuse. Project reopen, dirty recovery restore and saved sweep/Verify checks pass focused tests. Unused optional model fields preserve no-op-amp serialized shapes/hashes; broad adapter/legacy qualification remains required before packaging.
4. **Implemented in source:** shared triangle/sign/power-lead paths, named terminal geometry, rotation, native inspector/readings and four plot controls. Blank native palette/wiring/source-value actions match agent commands and accepted results in a focused native test. SVG tests cover all rotations, pin titles, supply labels and exact model metadata. Guided follower/inverting examples and actual native screenshots now exist. Mounted text-field/focus/accessibility defects and full print/export/native visual qualification remain open.
5. **Release acceptance remains open:** complete guided blank/example workflows, model and result corruption cases across adapters, save/reopen/recovery/Verify/sweeps, bounded performance and native/headless visual/input parity. Preserve source-pinned packaging and the prior app bundle until qualification passes. Source tests alone do not establish a released application feature.

## Foundation evidence

`scripts/opamp-reference.py` independently evaluates the chosen equations with 90-digit Decimal arithmetic and freezes 14 terminal-voltage cases plus six scalar feedback-circuit roots. It does not import Go/device code or a vendor model. Go tests compare all terminal currents/targets, check analytic derivatives by perturbation, exercise 1,000 deterministic voltage vectors for charge/power conservation and bounds, and check common-voltage translation. The amplifier harness eliminates its passive feedback node and solves output KCL independently; it is not the application's network solver.

Initial scalar results include a ±5 V follower at **0.999989811 V** for 1 V input, a gain-10 noninverting stage at **0.999893270 V** for 0.1 V input, and an inverting stage at **−0.999969336 V** for 0.5 V input. A nominal gain-10 request into 100 Ω is current constrained to **1.901253222 V**. Application integration, physical-device accuracy and the remaining product acceptance gates are unproven.

## Actual network integration

`mixed_opamp.go` maps the five terminal currents and every analytic derivative onto physical MNA rows, summing intentional pin joins such as follower feedback. Results own their fixed model and signed terminal currents; output current is explicitly delivered current. Power is the sum of terminal voltage/current products relative to V−, preserving common-voltage translation without an invented ground path. Supply states are `powered`, `unpowered`, or `reversed-supply`; the last two retain only the declared input resistor and output bleed. `cloneMixed` copies the result slice. Existing circuits omit `opamps` from serialized results. Reflected output discovery includes the optional result shape. Subsequent source integration now registers `opamp` in the shared library and compiler, while the packaged app still predates it.

Actual feedback tests exposed Newton oscillation after a 10 mV perturbation: saturation erases the useful differential slope, and a legacy 0.2 V node proposal jumps across the narrow transition. A per-op-amp proposal limiter now allows a saturated input to approach zero in one update, while limiting outward movement and zero-crossing overshoot to 0.1 normalized tanh-input units. It runs across every Radau stage as well as ordinary MNA. This affects guesses only: the original constitutive equations, 100-iteration attempt limit, node bound, residual tolerances and integration budgets remain. No-op-amp paths keep their prior arithmetic. Op-amp networks also require a converged Newton voltage correction before acceptance, preventing a small 1 GΩ bleed-current residual from hiding a large unloaded output-voltage error.

The six frozen scalar amplifier references pass actual DC and transient-startup network solves, each from cold and perturbed warm guesses. Further checks cover a merged follower Jacobian by perturbation, source/sink sign and power balance, zero/reversed supplies, unloaded equal rails, two cascaded buffers, a floating tied-input common mode, cancellation and owned result copies. A 10 kΩ/1 µF buffered input charges over 50 output samples against its exponential envelope. A buffer driving a damped RLC load runs through the shared simulation kernel and Radau integrator; 20 outputs agree with a half-step run and approach the passive divider's DC limit. These foundation fixtures deliberately construct internal compiled maps. Later source workflow tests separately exercise wired graph compilation, public placement/wiring, probes, model-bearing project/capture/recovery/sweep paths and native action parity; mounted visual/input qualification is still open.

The compiler now separates the input-resistance pair from the output-to-V− bleed, with an external path required for V+. Controlled derivatives do not create reference edges, and MOS gate control edges are excluded in op-amp networks. One earlier entirely ungrounded internal fixture demonstrated that floating-point Gaussian elimination with the existing absolute pivot threshold can accept a spurious tiny pivot. Actual wired graphs now refuse that disconnected group, separately floating input pairs, tied floating inputs, and positive-rail/input groups attached only to ideal N/P-channel MOS gates. This is explicit graph validation, not a weakened matrix-rank or solver tolerance rule.

Guided follower/inverting examples and the scoped native review are now recorded below; full native/app qualification remains open. The shared library/model/probe contracts are described in [the op-amp workflow](../docs/opamps.md). Keep the existing 0.105 app and prior bundle until source-pinned build, public/native adapter parity, bounded performance, rendered geometry/controls and required regression qualification pass. Further scope includes physical reference-path explanations for diagnostic users, full terminal plot interactions, printed output and mixed storage/device cases; no physical-device accuracy, speed or stability is claimed.

## Guided sine examples and scoped mounted review

The next source slice adds `opamp-follower` and `opamp-inverting` through existing `example.load`, with protected native menu aliases. One-cycle 101-row captures validate input sine, gain/sign, load/feedback output current and independent Decimal peak roots. Native callback tests compare menu load, four plot actions, electrical edits, stepping and protected reload against agents. Saved origin/probes/capture and reset protection pass core tests. The ordinary output/control command surface remains unchanged.

A 10 Ω follower load exposed a warm overload convergence defect: an outward differential proposal limited to a fixed few microvolts cannot reach the required saturated input error within the existing iteration bound. The limiter now permits half the existing outward magnitude plus the transition interval, preserving bounded overshoot when approaching/crossing zero. Both current signs and zero recovery pass without increasing numerical/work limits.

Actual mounted screenshots prompted moving live readings and plot actions above model explanations, compacting the four plots into two rows, and separating example feedback/ground/supply geometry. Menu, pause/collapse, selection and plot interactions pass this scoped review. **Physical resistor editing did not change its visible/shared draft; missing inspector AX nodes, unreliable focus progression, a persistent splitter tooltip and small default-fit labels remain open.** See [the numbered audit](ui-audit/opamp-guides/README.md). These findings prevent full input/accessibility qualification.

An isolated native preview and a fresh headless process also match every sample in both guided captures, preserve live state during Verify, save/reopen one recording and export CSV/SVG through real public transports. This does not cover all adapters or the supported topology/performance envelope. The next electronics-focused qualification should extend actual adapter/performance and mixed-storage/device cases, with release packaging still blocked on unresolved native input/SDK and regression checks. The canonical 0.105 app remains unchanged.

## Public mixed-driver qualification

The optional `elab-conformance --opamp-suite` now runs four named public-command experiments through all four adapters. It retains 101-row sine captures and 105-row mixed captures, including the grid and both pulse-edge observations. RC input memory meets the unchanged max(1 mV, 0.1%) analytic envelope. Output checks use scalar bisection of the declared follower law and actual resistor boundary conditions; NPN base voltage is captured, so this is not an independent NPN oracle. The loaded NPN buffer's expected rail-headroom loss is retained rather than avoided. Two cascaded op-amps and the MOSFET LED driver exercise the declared two-device ceiling in an RC circuit.

This suite found an actual startup precision issue: current-domain LED bisection could leave roughly 20 fA leakage with enough relative error to oscillate the strict Newton voltage correction by nanovolts. Op-amp networks now use the existing junction-voltage evaluation, with the same constitutive equation and work limits. A focused inversion check spans 1e−22 A to 3 mA and confirms that no-op-amp legacy arithmetic is unchanged. The shared evaluator now lives in `mixed_led.go`.

Mixed 5 V discharge also exceeded the analytic RC envelope under the inherited local error path. Op-amp capacitor circuits now use the established coupling-node scales (5e−8 V absolute /1.25e−6 relative) across analysis families. No-op-amp paths, physical laws and iteration/voltage/work limits remain. The conformance comparator recomputes all 931 checks per adapter from retained rows/peaks and rejects malformed coverage, hashes, calls and numerical differences. Graphs, rows, observations and exact per-adapter calls are retained in [losslessly shared evidence](../docs/validation/opamp-adapter-conformance.json).

A narrow [mounted-editor recheck](ui-audit/opamp-mixed/README.md) still finds the value-field/focus/AX and persistent-tooltip issues on the newer local SDK. Those remain release blockers. Actual repeated performance, broader storage/device/corruption and full native/GPU/legacy/source-pinned packaging qualification remain required; no canonical app promotion is implied.
