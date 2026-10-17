# Linear network voltage range and solve behavior

Linear source/passive networks now take full MNA corrections. Two ideal 12 V sources in series can produce **24 V through 1 kΩ: 24 mA and 576 mW**. A third 6 V source reaches the existing **30 V** node limit. Previously the same affine equations were unnecessarily restricted to 0.2 V updates and exhausted the 100-iteration ceiling before reaching 24 V.

Start with **Examples → Split-rail DC supply**, set V1 and V2 to 12 V and reconnect the ground wire from J1 to V2.negative. Source values remain 0–12 V each. R1 then spans the bottom-referenced 24 V stack. Connecting the common reference to V1.positive instead gives a −24 V bottom rail with the same load voltage/current. These are ideal-source/resistor calculations, not physical component ratings.

## Numerical strategy

Resistors, finite-resistance switches, fixed potentiometer segments, ideal capacitors and supported ideal voltage sources produce affine equations at each solve instant. Capacitors retain startup-voltage constraints or implicit companion stamps; the unreleased [SDIRK2 composition](capacitor-integration.md) uses these stamps for each diagonal stage. Source values are fixed for that instant even when their configured waveform changes over time. Reference/connectivity helpers add no device nonlinearity.

The solver applies a full correction to these equations, checks residuals and voltage bounds, and publishes only an accepted result. Even zero excitation now requires a matrix solve to check rank; a zero residual alone does not establish a unique operating point. A private-kernel regression checks a singular floating resistor rather than accepting an arbitrary zero-voltage state.

LED, silicon-diode, NPN, NMOS and PMOS circuits retain nonlinear damping. Unknown future device kinds conservatively retain it too. [Nonlinear startup](nonlinear-startup.md) adds one grounded-constraint fallback after a failed cold attempt; this change does not replace semiconductor models with linear approximations or promise that every circuit inside the voltage bounds converges.

The per-attempt iteration ceiling stays **100**, nonlinear node updates stay capped at **0.2 V**, and accepted nodes stay within **±30 V**. Component/source/node capacities, cancellation, residual tests, integration error control and per-output attempt/time limits stay in place. A 31 V stack now reaches the range check directly and refuses with that explanation instead of consuming damped iterations and reporting generic nonconvergence.

## Shared commands and saved results

Users and agents use the existing placement, wiring, value, pause/resume/restart, measurement, verification, history and file commands. No new command or saved format is required. Detailed `component.place`/`component.set` discovery exposes `x-network-numerics` with the strategy and actual retained bounds. The choice is automatic and depends on the shipped model kinds.

An over-range electrical edit does not publish a valid zero. The prior accepted result remains explicitly stale, the probe association remains available with its numerical reading marked stale, and ordinary undo/restart recovers the previous circuit. Captured recordings keep their stored numbers and model/input identity; fresh simulation recomputes. The physical equations, model parameters and engine identities are unchanged, although convergence and final-digit roundoff can differ between builds.

## Validation

Failing-before regressions cover the 24 V convergence problem and the 31 V diagnostic. Independent Ohm/Kirchhoff references cover 0, 18, 24 and 30 V, negative-reference rails, current and power conservation. A 24 V implicit-stage test matches the independent companion equation `C·v/dt = (24−v)/R`. The original affine-solve repair preserved the then-current method; general capacitor time integration subsequently advances to SDIRK2. Every library part's classification and conservative handling of unknown kinds are checked, as are rank and cancellation.

Native palette/named-wire/value construction produces the same 24 V /24 mA result as an agent. Packaged smoke edits and reconnects the example through its real controls, then waits for that current result to reach mesh preparation. The public reference `scripts/linear-network-demo.py` measures 24 V and 30 V, exports schematic/captured artifacts, refuses 31 V, checks stale labeling and undo recovery, reopens the saved recording and changes the reference to −24 V. The conformance client refreshes revision/document guards and retries at most eight explicit admission conflicts in its owned sessions; it refuses retries across document or electrical changes. Logical workflow evidence excludes these guard queries. The scroll smoke waits for configured overflow geometry before testing wheel movement. Existing nonlinear/electrical and transport regressions remain distinct evidence scopes.

Full mounted input/accessibility, visible latency, clean distribution, participant testing and physical-printer acceptance remain open. No hardware-accuracy or universal convergence claim follows from these checks.
