# Next five electronics capabilities

Authorized October 27, 2026: implement in order, with working examples,
independent electrical tests, shared UI/agent commands, one completed Prism
review/finding disposition per item using Anthropic `claude-opus-5-5`, and local
commits. No OpenAI provider or fallback. No push or public upload. Preserve the
running owner's /276 session and every qualified package/SDK identity.

1. Complete class-AB/loudspeaker transients by reproducing the stiffness/work
   failure and repairing the numerical cause. Keep existing residual, error,
   attempt, wall-time, cancellation and state-publication guarantees. Add a
   positive amplifier/speaker example and refinement/conservation coverage.
2. Optional capacitor ESR/leakage and standalone inductor winding resistance,
   with saved defaults preserving existing ideal models, shared controls,
   losses/energy/probes and a supply/damping comparison example.
3. DC/pulsed current sources and bounded constant-current/constant-power loads,
   shared configuration, load-step/supply examples and independent electrical
   references. Explicit compliance/low-voltage behavior, no singular 1/V law.
4. Generic configurable JFET models, drawing/readings, transient/AC behavior,
   preamplifier/current-source examples, persistence and independent laws.
5. Dedicated feedback loop analysis with explicit injection/return-ratio scope,
   gain/phase margins, and captured overshoot/settling measurements. Never call
   an ordinary closed-loop signal Bode sweep a loop stability measurement.

Every item is integrated with discovery/validation, history where applicable,
model-bearing captures/projects, exported metadata and routed desktop parity.
Derived analyses keep source/result identity and cancellation guards. Named
manufacturer fidelity and hardware validation require separate evidence.

Current starting point: clean main `861e2aa5`, app /276 clean binary source
`323f5e7a`, 54 kinds / 203 commands / 67 examples. Existing native acceptance
3/6/9/11 and participant/separate-account gates remain separate. No new native
OS queue/hardware qualification may be inferred from routed/software evidence.

Item 1 reproduced the 17.7 ns false cross-pair LC ceiling and now uses checked
local storage modes, strict fallback and carried private proposals/endpoint
ceilings. Full 200 ms/two-polarity refinement, independent RLC Jacobian/modes,
original fine-ceiling startup, capture/reopen and routed UI/agent tests pass.
Numerical correctness tests use deterministic test clocks; separate public
service/UI steps retain production wall-clock guards. An unrelated old LED
startup race case hit the work guard in the broad run; three isolated controls
pass. Initial Prism review (Anthropic Opus 5.5) completed all three chunks,
with two medium/four low observations; source changes and dispositions address
them. Follow-up review and final focused checks are running. New Gonum BSD
notices must be included inside the final app notice bundle and outer Licenses.

Item 1 is complete. Three complete Anthropic Opus 5.5 Prism reviews are retained
in `docs/validation/electronics-next-five/transient/`; final report has zero
medium/high and three low observations. Two final test lows are fixed; mandatory
mode-check overhead is a documented bounded tradeoff. Current positive waveform
proof includes full two-polarity refinement and production-clock captured/UI
steps, distinct from deterministic numerical tests. Final focused race controls,
ordinary/offscreen parity and lint pass. All 27 prior package/SDK identities stay
exact. Committed as `361b77ba` before beginning item 2.

Item 2 adds optional versioned capacitor ESR/leakage and standalone inductor
winding resistance without extra storage slots or solver-limit changes. Both
examples, independent RC/RL/RLC and complex-impedance references, energy balance,
Jacobian, capture/reopen/forgery checks and routed ordinary/offscreen UI-agent
parity pass. A numeric minimum endpoint conversion regression was reproduced and
fixed by preserving decimal float32 control input before SI conversion. Command
discovery exceeded its existing 64 KiB bound; sharing existing schema definitions
restored it without deleting fields or raising the limit. The native source
ledger adds only passive.configure and its inspector route; native acceptance
3/6/9/11 remains open. Prism completed 7/7 Anthropic Opus 5.5 chunks; four medium/nine low observations are fixed or dispositioned in the passives evidence README. Fresh focused core/desktop checks cover the fixes.

Item 2 committed as `92372827`. Item 3 adds versioned Norton DC/pulsed current
sources and one-quadrant CC/CP electronic loads. Source compliance, explicit
bleed, bounded low-voltage continuation and finite pulse ramps are documented.
Independent resistor/compliance roots, CC/CP supply roots, RC ODE, derivatives,
passivity, current-driven capacitor, storage-free pulse classification, combined
source budget, capture/reopen/forgery, initial voltage trace and ordinary/offscreen
shared inspector/agent checks pass. Source count remains three; no solver-limit
changes. Catalog is 56 kinds /206 commands /73 examples. Model/print/export
metadata and sampling advice retain full settings. Prism completed all eight Anthropic Opus 5.5 chunks; findings are fixed or explicitly dispositioned against tests and declared model scope in the current-devices evidence README. Fresh controls cover the review changes.

Item 3 committed as `fa12ddfa`. A documentation/test closeout corrects the source
budget description to the actual unchanged three-source limit and tests exactly
MaxNetworkSources+1. Item 4 adds configurable generic N/P-channel JFETs within the
existing combined two-FET capacity, explicit gate PN junctions/bleeds, analytic
Jacobian, DC/transient/AC, signed terminal readings and model-bearing captures.
Three examples teach a preamp and complementary self-biased current sink/source.
Independent square law, complete Jacobian/polarity/reciprocity, self-bias quadratic,
transconductance/pole AC and nonlinear transient ODE checks pass, as do
capture/reopen/forgery, history, capacity and ordinary/offscreen shared controls.
Discovery keeps its 64 KiB bound by sharing existing save/load/step schemas.
Catalog: 58 kinds /207 commands /76 examples. Prism completed all seven Anthropic Opus 5.5 chunks; one medium and eight low findings are fixed or dispositioned in the JFET evidence README. Fresh race controls cover review changes.

Item 4 committed as `89a367f7`. Item 5 adds a dedicated dynamic-op-amp controlled-
source return-ratio target through existing sweep commands, closed-bias injection,
all sampled gain/phase crossings, explicit unavailable margins, captured method,
source identity, keep/export/reopen and existing cancellation guards. It is not
ordinary signal gain or a stability certificate. Discrete-transistor physical-wire
loop injection is not silently claimed. Captured step-response set/query commands
require explicit initial/final references, band and hold; saved linear segments
and jumps give direction-normalized overshoot and observed settling only.
Loaded two-pole electrical return ratio/unity reference, independent three-pole
margin estimates, RC settling, synthetic overshoot/jump/falling/short/unsettled
controls, purity/ownership/reopen, cancellation and both desktop drivers pass.
Two teaching examples added. The initial RC fixture had zero source and an
excess 2000-step capture; it now explicitly uses 5 V and the existing 1000-step
bound. No simulation limit changed. Catalog:58 kinds /209 commands /78 examples.
Discovery, source inventory and lint pass. Prism completed all nine Anthropic Opus 5.5 chunks; two medium/ten low observations are fixed or dispositioned in analysis/README.md. Fresh focused controls cover the changes before final combined qualification.

Item 5 committed as `9984b36c`. Combined ordinary desktop and offscreen race suites
pass completely; all vets, lint, 77 Python tests, PRD coverage and native inventory
pass. The initial portable invocation was blocked by sandbox Unix socket binds,
three outdated metadata/test expectations, and three historical 100 ms race work
refusals. Raw output and the frozen 1117-input manifest remain under build.
Corrections preserve exact historical capacity envelopes including paired absence
of new FET discovery fields, resolve the pooled preview-byte schema in its test,
and classify all four new nonlinear kinds explicitly. A new 24 V current-source
boundary test also reproduced ordinary cold/pulse initialization failure. A narrow
single-current-source/pure-resistor monotone fallback repairs it only after
ordinary Newton refusal; accepted caller guesses remain unchanged; the unchanged Newton,
residual, range and work checks accept/refuse the guess. Floating source, zero
drive, excluded-device scope and both desktop drivers pass. No capacity/accuracy/
work limit changes. Final correction review and portable qualification follow.

All five feature items are committed, followed by reviewed boundary/compatibility
repairs in 8976ae93. Corrected portable coverage is complete in aggregate: three
relay/sensor race time-guard failures in the full invocation all pass together
in the isolated exact-source control. No test/solver time guard was increased.
Complete desktop suites plus final focused fallback controls, vets, lint, Python,
PRD and inventory checks are retained separately. A fresh 0.199.0 /277 candidate
and matching agent are being built from clean committed source; older packages
and all 27 protected package/SDK identities remain exact.
