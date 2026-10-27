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

Qualification and package results will be linked here at completion.
