# Grounded ideal constraints in nonlinear startup

A 24 V series-source LED circuit now starts successfully without replacing its nonlinear LED equation. With a 1 kΩ resistor and the generic red LED, current is about **21.794 mA**, matching the independent inverse-Shockley load line. A 30 V load gives about **27.775 mA** within the existing node limit. Previously, these cold starts spent the 100-iteration Newton attempt ramping a fixed supply rail in 0.2 V steps and reported nonconvergence.

To try 24 V, load **Examples → Split-rail DC supply**, set both sources to 12 V and move the ground connection from J1 to V2.negative. Delete the R1.2-to-V2.negative wire, place a red LED, then connect R1.2 to its anode and its cathode to V2.negative. All placement, wiring, exact value, measurement and simulation commands are shared with agents. This is an ideal-source/generic-device calculation, not a rating or hardware-accuracy claim.

## Starting points and bounds

The ordinary Newton starting point runs first. If a **cold nonlinear** attempt fails, one additional attempt initializes only voltages connected to ground through ideal voltage-source constraints. At time zero, configured capacitor startup voltages also form ideal constraints. Propagation uses actual terminal polarity and the same source stimulus/time side as the MNA equations. Finite branches do not become ideal connections. Unanchored ideal-source groups receive no invented absolute reference; unknown node and branch-current guesses remain zero.

The resulting vector is only a starting guess. The full existing device equations, analytic Jacobians, KCL residuals, voltage bounds and acceptance checks still decide whether a solution exists. No gmin, substitute resistor/device model, artificial gate leakage or arbitrary accepted branch currents are introduced.

Normal successful starts retain their existing solution basin. An accepted warm vector does not use this extra retry; ordinary transient continuation and existing NPN alternate seeds remain. This grounded-constraint fallback adds **one additional Newton attempt**.
The later larger-BJT fallback below has its own bounded attempt sequence. Each attempt remains limited to **100 iterations**, with **0.2 V nonlinear node corrections** and accepted nodes within **±30 V**. This increases the bounded cold-start work envelope; it does not increase the per-attempt iteration ceiling. Context cancellation and the existing integration attempt/time budgets remain. Detailed `component.place`/`component.set` discovery publishes this distinction under `x-network-numerics`.

A reference-shifted network or another nonlinear operating point can still fail to converge. Initializing grounded ideal rails is not a universal convergence method. Device parameter/model/engine identities and saved formats are unchanged; fresh calculation may differ in convergence or final-bit roundoff. Stored recording numbers remain stored evidence, rather than being silently recalculated.

## Larger discrete bipolar networks

The [sixteen-device bipolar expansion](discrete-class-ab.md) adds a cold-start
fallback for networks containing **more than two explicit BJTs**. It runs only
after ordinary and grounded-constraint starts fail, before the existing
per-transistor alternate seeds. It is limited to sources, resistors,
silicon/Schottky/Zener diodes, LEDs, NPN/PNP devices and connectivity helpers.
Networks containing capacitors, inductors or other device families do not take
this source-stepping path.

Eight private solves scale the instantaneous authored source voltages from
one-eighth to full value. The first uses a grounded ideal-constraint seed;
each following solve uses the preceding accepted vector. A final solve checks
the original full-source network with ordinary residual, uniqueness and voltage
guards. A failed stage stops this ramp. No intermediate voltage, current,
source setting or graph is published or persisted, and context cancellation
remains active throughout. Each Newton attempt keeps the same 100-iteration
ceiling and 0.2 V correction bound; integration guards are unchanged.

The regression is a cold −100 mV input operating point of the eight-transistor
amplifier with C1 removed. Positive/negative operating points, independent tail
and 16-device load-line references, graph/source immutability, cancellation and
exclusion of reactive startup are tested. The default transient example retains
its explicit 7.55 V capacitor condition instead of using this algebraic fallback.
[Electrical tests](../workbench/class_ab_test.go),
[qualified source and runtime evidence](validation/discrete-class-ab/README.md).

## Rejecting undefined zero states

Every Newton attempt now solves its matrix before accepting an initial zero residual. This matters at zero excitation: two insulated MOSFET gates joined only through a resistor have no DC reference. An arbitrary gate voltage can satisfy zero branch current, so zero residual alone does not establish a unique operating point. The previous executable accepted that graph as fresh zero. The current solver refuses its singular matrix; explicitly grounding the gate network yields a valid zero reading.

This extends the linear rank check from 0.93 to nonlinear starts and warm guesses. When the initial residual is already acceptable, a rank-only solve leaves that guess unchanged and avoids rebuilding the nonlinear equations. Graph compilation and its independent ideal-source/capacitor-loop checks still apply. It does not convert a numerical singularity into a fabricated source/object diagnosis.

## Validation scope

Independent inverse-Shockley load lines qualify 24 V and 30 V LED current, with KCL/power conservation and 31 V refusal. Focused tests check grounded-only propagation, explicit capacitor startup, absence of capacitor voltage constraints after time zero, actual pulse-edge stimulus side, unchanged ordinary PMOS startup/continuation and floating-gate refusal. The pulse-corner regression measures input current on both sides; for the mixed base-capacitor fixture its jump matches 5 V / RB while capacitor voltage remains continuous. LED current continuity is not mistaken for a missing input edge. Native palette/named-wire/value construction agrees with agents; packaged smoke inserts the LED into the 24 V circuit through real controls and waits for current mesh preparation.

The public reference `scripts/constraint-start-demo.py` builds the floating-gate fault, repairs it, measures both high-rail LED loads, checks stale failure/undo recovery, captures/reopens the saved 30 V result, and checks an explicit 1.25 V capacitor startup followed by charging. Existing electrical/reference and transport workflows are separate regression evidence. Full native input/accessibility, visible-latency, distribution, participant and physical-printer acceptance remain open.

The 0.94 direct 24 V PNG review found its LED ID crowding the outward arrows. [Shared LED label placement](led-labels.md) repairs that issue in 0.95. General neighboring-object clearance/readability remains a rendering follow-up; matched export bytes do not establish complete visual acceptance.
