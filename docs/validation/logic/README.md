# Basic logic-gate validation

[Electrical and agent contract](../../logic-gates.md).

Seven functions are tested over all input truth rows at 3.3, 5 and 12 V with
local ground translated by −3, 0 and +2 V. Loaded output voltages are compared
with an independent resistor-divider reduction. Analytic terminal Jacobians
are checked by central differences, including intermediate inputs, reverse
supply and the zero-supply limit. Currents sum to zero; conductive branches
absorb nonnegative power. Vanishing rails retain finite stamps.

Tests also check gate cascading, output RC response against its exponential,
a 555 driving an inverter across retained timer events, required-pin wiring,
all three examples, live switch behavior, strict capture identity/level/model/
pin-array validation, kept recordings, project save/reopen and recovery,
owned snapshots, and probe signs. SVG markup is parsed at every gate rotation.

Native-control tests build from blank through the palette and named-wire
controls, compare the resulting graph and electrical state with external
commands, exercise switching, load each example and step transient examples.
The scope and inspector expose gate-specific readings and plot actions.
Symbol paths and pin text rotate together; every browser part has distinct
artwork. Focused race checks pass, including the existing timer tests after the
final electrical changes. Source inventory remains 166 public commands, 404
sites, 134 literal routes and 82 production desktop sources; new example
aliases dispatch `example.load`.

The stdio replay passes discovery, all three examples, four manual AND rows,
inverter transient truth, all four half-adder input combinations, power/KCL
checks, captures, kept-record save/reopen and SVG/PNG/PDF export. Export
inspection caught an unescaped ampersand in the IEC AND mark; it is escaped
and covered by a per-gate/rotation XML regression. Explicit branch junctions
separate input and power buses in the half-adder; captions clear the supply
feed. These are equation and workflow checks, not measured hardware timing
or independent SPICE validation.

The first broad run found two old catalog expectations (nonlinear-kind list
and eight-category count), the fixed export markup issue, and a timer/coil
wall-clock budget overrun under concurrent race/build load. Catalog tests now
include the new gates/category. Production work limits are unchanged. Final
broad checks and packaged/native observations are recorded below.

Native inspection of the first /237 build found the examples' 5 ms output
interval triggered the existing four-samples-per-ramp guidance on 10 ms input
edges. The final defaults use 2 ms; a regression checks that these curated
examples meet their own waveform-sampling guidance. Focused root and native
race checks pass after the adjustment. This changes output sampling only, not
the gate equations or solver work budget.

The native AND check also showed its first ground symbol crowding the SB
caption. The first pull-down ground is moved 100 drawing units left; electrical
connectivity and values are unchanged. Logic electrical/workflow checks pass
after this layout-only adjustment.

## Final package and observations

Candidate **0.186.2 /239** is built from clean source
`41936d4a8d904aaf3cd17f973e440c2a6a2d7b57`. Both binaries record that revision
with `vcs.modified=false`; its ad hoc signature verifies strictly. The
[manifest](build-result.json) records the native and agent SHA-256 hashes. The
qualified /205 archive retains its original checksum and is not replaced.

The [final packaged stdio replay](packaged-stdio.json) passes discovery,
manual AND truth, inverter and half-adder transient truth, all four half-adder
input combinations, captures, conservation, kept-record save/reopen and
SVG/PNG/PDF export. Final PNG schematics were visually inspected for complete
wiring, pin symbols, captions and parts lists.

The [native observations](native-result.json) distinguish each tested build.
Running /238 showed the AND LED off with one switch closed and 2.780 mA with
both closed; its inspector showed A/B high and 4.8609 V output. Unsaved-work
protection appeared when changing examples; only those synthetic switch edits
were discarded. The layout-only /239 follow-up separates the ground from the
SB caption. The final half-adder window shows all seven library gates, live
A/B/sum/carry traces, the corrected 2 ms scope interval and gate readings.
The app remains running that clean example in Auto with U1 selected and Apollo
inherited. Earlier owned /237 and /238 test apps were closed normally.

## Broad-check limits

The later full root race run passes **all workbench tests (113.239 s)**,
transport and inventory tests. That run's coil conformance subtest returned an
EOF; its isolated race rerun passes (6.831 s). The first full conformance run
also passed (122.252 s). The full desktop race run reached 545.791 s and failed
one pre-existing flasher workflow on its unchanged 100 ms integration deadline;
that exact workflow passes in isolation under race (13.765 s). The earlier
root timer/coil deadline failure also passes the focused timer race run.
These failures occurred during concurrent race/build load; their precise
scheduling/EOF cause is not established. They are retained in the logs, not
reported as a completely green combined-suite run. Production numerical,
wall-clock and transport limits were not relaxed.

Focused final logic/native/race checks pass after the default sampling change;
logic electrical/workflow tests pass after the final coordinate-only change.
Lint, root/desktop vet and final command-source inventory verification pass.
This is feature validation, not renewed distribution qualification or a
hardware timing claim.
