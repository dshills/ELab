# Preserved /271 native smoke refusal

The first clean 0.196.0 /271 candidate is preserved. Its exact copied native
smoke reached the interval workflow with six occupied trace slots. The old
self-test then requested `v:C2`, expecting silent eviction. The new explicit
slot-preservation contract correctly refused; the smoke exits 1 normally.
This is a self-test sequence failure, not a crash or a native queue verdict.
The four-adapter suite was not started after the smoke refusal.

The corrected sequence explicitly clears the sixth visible selector through
`smokeUI` before plotting, just as the already-passing ordinary workflow does.
Only `desktop/smoke.go` changes after the complete source qualification; all
1,370 other frozen executable/test/dependency inputs remain exact. A separate
/272 candidate will test that adjusted sequence. Existing full-slot atomic
refusal/preservation tests remain unchanged.

The /271 matching headless feature demonstration and all 62 packaged example
opens passed. Those receipts belong to /271; final /272 results are separate.
