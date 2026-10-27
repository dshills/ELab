# Loop return ratio and captured step response — item 5

58 kinds /209 commands /78 examples. Dedicated dynamic-op-amp controlled-current
injection uses the existing cancellable, bounded, source-guarded sweep workflow;
closed-circuit bias and loading remain, with a declared return-ratio method,
complex responses, sampled margin estimates, save/reopen and JSON/CSV export.
It does not label ordinary signal gain as loop gain or certify hardware stability.
Discrete-transistor wire-break/Nyquist/RHP-pole analysis is not claimed.

Step-response set/query commands use explicit initial/final references and saved
linear segments/jump limits. Direction-normalized overshoot and observed settling
require actual timestamp coverage and continuous requested hold through end_s.
No inferred final state, extrapolation, unseen peaks or future guarantee. Session
measurements preserve live state, history and documents; retained recordings
reopen and reproduce the query. Inspection/queries own all result pointers.

Independent loaded two-pole return-ratio and unity/phase reference, separate
ordinary signal gain, analytic three-pole margin estimates, exact crossing
deduplication, malformed/huge-phase refusal, capture/reopen/tamper/cancellation,
analytic RC settling and synthetic overshoot/falling/jump/short/unsettled cases
pass. Unit-bearing set/query times, selection/refusal/clear/purity/ownership pass.
Ordinary/offscreen desktop controls match agent results, reject stale inputs and
validate across themes. Native source inventory passes; native OS queue/hardware
acceptance3/6/9/11 remains open. No numerical/resource/worker limits changed.

Prism: only Anthropic claude-opus-5-5 and same fallback, complete9/9 chunks,
37 files, no skipped/excluded/truncated bytes. Two medium and ten low findings:

- 0b896b8c19aff2e4: openLoopControl already returns an allocated exact-size vector
  or error. Added dimension guard and explicit parameter-based mode dispatch;
  loop mode cannot fall through to sourceIndex=-1.
- 9b9458ecd64b10c9: public record admission already checks canonical bounded
  gain/phase before deriving margins (forged-phase import test passes). Added
  helper-local shape/count/finite/grid/phase-continuity bounds with direct
  malformed/1e300 controls, returning an error; no public exploit is claimed.
- 91b93bb8b8e63f58,7adad0a71fdcef68: clearer test failure formatting/marshal checks.
- d164258b5d5b8b37: inspector explicitly reports unavailable for fewer than two
  saved rows or no positive saved duration, before constructing defaults.
- 146a1b422a6df2eb: module is Go1.25.6, with per-iteration range bindings; no
  pre1.22 compatibility claimed, so no redundant loop-shadow workaround.
- eae1d1ddd01e5dd7: curated OpAmpFollowerExampleGraph creates VIN as a sine source
  via validated MakeComponent. Positive example/loop tests fail if that fixed
  fixture contract changes; no arbitrary older graph is passed to the builder.
- be4c5035e8a393ec: target error lists configured dynamic-op-amp loop analysis.
- 31853f75b280eb5b: discovery explicitly states ascending positive loop bands;
  other target kinds retain descending support. Admission/domain tests agree.
- 0e453c33d33d215b: Service.Dispatch normalizes set inputs before applyLocked,
  while pure query normalizes locally. Both now have explicit unit-text controls;
  no duplicate normalization added to the private setter.
- acfac444e34a1417,1da5977c623ad5f2: helper marshal/type/schema/length diagnostics
  strengthened. Clear settings allow zero tolerance only with the canonical
  disabled request; detailed bounds also describe that exception.

Initial example fixtures used a zero-volt AC source and a 2000-step Verify
request. They were corrected to an explicit5 V step and20 µs/20 ms (1000 steps).
No limit was raised; raw failures remain in build/electronics-next-five.
Review changes have fresh focused race and lint receipts. Full final source and
matching-package qualification follows all five commits. Prior packages/SDK and
the owner's /276 app/documents/settings stay preserved.
