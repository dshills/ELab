# All-discrete class-AB amplifier

October 27, 2026. Owner authorized expanding transistor capacity, a working
all-discrete example, electrical tests and identical user/AI commands.

## Scope and implementation

Combined NPN/PNP capacity rises from 2 to 16, including embedded optocoupler
phototransistors. All other resource bounds, models, file format and numerical
accuracy/work guards remain unchanged. The example uses eight BJTs: differential
input, PNP mirror load, diode-referenced tail sink, PNP voltage-gain stage and
complementary biased output with global feedback. No op-amp is present.

The initial valid cold DC negative-input trial failed Newton convergence.
Adding emitter degeneration alone and an experimental Newton backtracking trial
did not repair it. Backtracking was removed. A private eight-stage source ramp
now serves only as a cold-start fallback after ordinary attempts fail, for more
than two BJTs in purely algebraic discrete networks. The final full authored
sources must pass ordinary strict residual/uniqueness guards. Reactive initial
conditions and existing one/two-BJT successful paths remain unchanged.

The first negative-input helper incorrectly tried to author a negative-valued
DC source and ignored its validation error; those preliminary negative numbers
are excluded. Subsequent trials use the existing sine offset command/field.
A first desktop test used the legacy quick-menu ID, which is not an item in
the current browser; the final test uses actual search/collection/Open controls.
The public reconstruction initially exceeded the 100-edit batch bound, then
tried reusing retired component IDs after Clear. Final tests preserve both
contracts: batches have at most 80 edits and reconstruction uses new BUILT_ IDs.
These controller setup failures do not justify API-limit changes.

The first recording test requested a page above the public 200-row maximum;
correct pagination fixes the test without changing the API limit.

## Verification plan

- Independent scalar tail-current and 16-device load-line references.
- Both polarity limits, combined refusal at 17 and editable/savable excess.
- DC gain, offset, mirror loading, idle overlap and both output half-cycles.
- Two transient periods, step refinement, captured model provenance, persistence.
- Desktop handlers versus public service commands, fresh package runtime and
  extracted example reopening. Full application race/lint/vet/build gates.
- Preserve all prior packages, owner sessions, SDK archives and caches.

The 1 nF compensation capacitor starts at an explicit biased 7.55 V condition.
This is an educational low-frequency amplifier, not a physical power-on,
thermal, bandwidth or stability qualification. Original native input gates
3/6/9/11 remain open; software/native-service tests do not close them.

The first complete portable race run detected a real compatibility regression:
`CompareCapacity` derived historical envelopes from the new 16-device value,
rejecting the frozen genuine pre-expansion report. The validator now explicitly
retains the exact historical two-device envelopes plus the current envelope.
Tests retain every corruption refusal and add six unpublished hybrid-envelope
refusals. The frozen historical report and all its numerical checks are unchanged.
Only this package and its command consumer require a new portable race run;
unchanged packages retain exact-input coverage from the initial complete run.

## Complete: runnable candidate /275

[ELab 0.197.0 /275](../build/ELab-0.197.0-275-Discrete/ELab.app) is qualified
from clean source `4e79a78c057ba76b6cc7bc74e5bb7e804334d428`. It includes the
matching agent, 63 examples and the unchanged Vellum v1 runtime/icon. Full
coverage passes 885 portable /348 desktop /485 rendered top-level race tests
(1981 /849 /1520 including subtests), with no skipped tests, plus lint/imports,
77 Python tests, PRD audit, all vets and build. The repaired conformance package
and its only command consumer were freshly rerun; unchanged portable packages
retain exact-input passing coverage from the first complete run.

Exact copied native smoke and four public adapters pass under development/IP
and shared-geometry write denial. All 102 external conformance processes exit
normally. Exact matching headless and copied native amplifier workflows pass
401/801-row captures, half-step refinement, both output currents, reconstruction
through existing editing commands, exports and recorded reopen. The owned
native fixture closes normally with endpoint removal; owner sessions are untouched.

All 63 projects reopen paused/clean with exact graphs. All 124 original project
and SVG bytes plus all 62 original catalog entries remain exact; copied README
and catalog metadata add the new entry. All 23 protected prior archive/executable/
SDK/lock identities remain exact. No new component kind, public command, model,
file version or other resource/accuracy/work limit is introduced.

The final exported schematic was visually checked after correcting crowded
feedback labels. A reused native-controller scope label was corrected from
“New public commands” to “Existing shared commands”; its original receipt is
retained and no electrical/test outcome was changed. Original native acceptance
3/6/9/11, participant and separate-account gates stay open. Nothing is pushed,
uploaded or promoted over the existing qualified download.

[Complete evidence](../docs/validation/discrete-class-ab/README.md),
[summary](../docs/validation/discrete-class-ab/summary.json),
[user/agent guide](../docs/discrete-class-ab.md).
