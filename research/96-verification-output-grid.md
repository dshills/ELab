# Verification at the output-step boundary

October 23, 2026. Existing shared verification and recording import; no new
feature, model, format or higher simulation limit.

The unchanged /217 packaged agent refuses both numeric `0.001` / `0.000001`
and unit-string `1 ms` / `1 us` requests as exceeding 1,000 output steps.
The immediately smaller duration succeeds, and the immediately larger duration
correctly refuses without changing accepted state. The public reproductions
and the original failing source regression are retained in the validation evidence.

Binary64 division rounds `0.001 / 0.000001` slightly above 1,000, while
`1000 × 0.000001` represents the requested endpoint exactly. Admission used the
division's ceiling, but the worker generates times by integer multiplication.
The repair adjusts the step estimate against that actual output grid. Recording
import uses the same count for bounds and completeness checks. Agent capability
details explain the represented grid and the unchanged 1,000-step limit.

This does not round nearby times together. A distinct final interval still
counts separately, and a duration immediately above the 1,000-step endpoint
still refuses without mutation. Existing tests continue to require final samples
at sub-picosecond and adjacent-representable times, and preserve the documented
mixed-solver minimum-timestep failure rather than publishing partial captures.
Manual stepping and its existing progress reporting are unchanged.

The new regressions cover the exact limit, its lower neighbor, a short grid and
a distinct short final interval; each successful RC capture matches analytic
voltage, imports successfully and validates as a saved embedded recording.
Offline captures preserve live charge/time and the circuit. The focused race
run and existing verification/grid regressions pass in 2.371 s.

The [full quality gate passes](../docs/validation/verification-output-grid-218/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted race and
offscreen vet. Desktop race takes 174.489 s; mounted race 159.428 s. All
888 frozen source/check inputs match. Clean packaging and matching-agent storage
checks follow this source receipt. The qualified /205 download remains preserved;
the accumulated native UI acceptance work remains separate.
