# Thyristor evidence

Ordered item 5: SCR/TRIAC saved one-variant config, shared configure command,
ordinary wired gate inputs, tentative gate-supported vs actual-current latching,
holding-current and signed commutation guards. Existing hybrid event bracketing,
trial ownership and all solver/resource/work limits remain authoritative. Generic
low-voltage three-terminal law; no physical quadrant/recovery/dv/dt/mains claim.
Native/SVG/browser symbols, terminal/wire/scope currents, source/model provenance,
inspector fields, history, captures and two curated examples are integrated.

Independent controls cover terminal KCL/passivity and full Jacobians, both gate
and main polarities, direct trigger/latching/holding sequences, true load-current
threshold with a gate pulse shorter than the output interval, reverse SCR blocking,
both TRIAC half-cycles, signed commutation and event refinement. Holding cutoff
matches asin((Vdrop+(RL+Ron)*Ih)/Vpeak) against a resistive sine load. Captures test
model/state/current/power forgery; save/reopen/undo/refused-edit and rejected-trial
memory ownership are checked. Ordinary/offscreen user/agent/all-theme controls use
the same shared model command and existing source/button/switch commands.

Initial work retained under build: an edit script selected its prior buffer while
updating DeviceParameters; compile caught duplicate/undefined declarations. That
file was reconstructed from exact committed HEAD and the intended narrow additions
reapplied before electrical tests. First test formatting used `<-` instead of a
spaced negative comparison. First reset test expected a switch change to apply
before an output step; corrected to respect the documented existing deferred
switch-actuation contract. These are not relabeled passing product trials.

Initial broad raced controls exposed two distinct outcomes: the existing relay
capture hit the unchanged 100 ms integration guard under instrumentation/load
(96 attempts; no solver limit changed), and adding shared commands exceeded the
existing 64 KiB ordinary catalog envelope. Catalog teasers now use the existing
word-boundary truncation at 88 instead of 96 runes; full filtered descriptions
and all schemas/contracts remain intact. The bounded transport guard and exact
filtered-description regression are retained. Separate final controls are recorded
without treating the earlier raced timeout as a clean invocation.

Prism initial review complete: 6/6 Anthropic Claude Opus 5.5 chunks, 38 files,
no skips/truncation; one medium/three low observations.
d2504504a0c8b9e5: the premise of an exponential thyristor junction is incorrect.
At frozen memory, both gate and main laws are monotone piecewise linear. Newton
still enforces every residual/correction and state memory is evaluated only after
a converged analog solution; it cannot acquire a latch during an unconverged
trial. A comment makes the intentional L-step exemption explicit. Full Jacobian,
KCL, actual-load and bracket-refinement references verify this path.
f6397cb0900821e5: held field acquired on the app goroutine. 6078619a8c229a8b:
research spacing fixed. 9157face5c0c73a8: probe/startup logic now derives fixed
pin names by kind, avoiding direct indexing of a supplied terminal slice.

The first full root run's only product-suite failure was an old explicit
linearity-expected-kind table, missing all three new semiconductor kinds. The
constructor already classifies them conservatively nonlinear; table corrected.
Other root packages, including local agent conformance/sockets/spooler, passed.
A final source/UI inspection also found noise mode still offering current probes
and plotting tiny flat densities in the old volts-scale padding. Noise now filters
the selector to voltage outputs and plots nV/√Hz while retaining SI capture values.
New optical/temperature x-axis units get readable bitmap-safe unit text. These
are integration corrections to earlier batch items, covered by follow-up review.

Follow-up Prism complete: 2/2 chunks, 9 files, no omissions and zero medium/high
findings. a295a659146b1bb3: bitmap unit replacer is now shared and immutable.
8199fdff792bb5e5: piecewise-linear-law comment moved inside its correct case.
An added noise-selector test initially inspected the default component page;
corrected to follow visible Simulate navigation before checking voltage choices.
Final repair core race passes 1.901 s; ordinary sweep/noise/thyristor UI passes
1.192 s. Full batch tests and exact packaged workflows follow source closeout.
