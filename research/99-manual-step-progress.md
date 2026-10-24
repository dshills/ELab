# Manual-step progress matches its actual output grid

October 24, 2026. Existing shared command metadata; no new feature or higher
simulation limit. Source/public reproductions are retained with the repair.

The unchanged /219 agent completes an RC step while reporting `1/2` staged
samples for a duration just above one sample interval, using either numeric or
unit input. A run starting at 5 ms can similarly complete with `3/4`. Its time
and voltage reach the requested endpoint; the total count is misleading. Exact
and ordinary partial-final intervals report consistently.

Admission used a quotient ceiling to estimate the total. The worker instead
uses the captured origin plus an integer sample index and preserves an existing
rule that snaps a final tail below 1 ps to the requested endpoint. The repair
counts that same bounded grid and shares the unchanged target-time calculation
with the worker. Duration/range validation, timing, equations and the 1,000-sample
limit are preserved. Verify keeps its separate strict final-sample policy.
Detailed agent discovery now explains the actual progress-count policy.

Three source regressions fail before the repair. Seven repaired cases cover
exact, adjacent and tiny-tail endpoints, a nonzero origin, a genuine partial
interval, a tail above the snapping threshold and the 1,000-sample boundary.
They verify planned/committed counts, exact final time, analytical RC voltage
and unchanged source/history. Existing cancellation/retry/supersession and
published-bound tests also pass in the focused race run, 1.536 s.

The first full gate hits the existing 100 ms wall limit in the direct flasher
numerical test after 27 attempts. That test does not call manual stepping. The
failure is preserved; no scheduler cause is asserted from this observation.
Its numerical oscillation assertions now use the existing deterministic test
clock, retaining the 512-attempt cap. Production budget code and dedicated
time/attempt-limit enforcement tests are unchanged. The flasher, budget and
progress cases pass together under race instrumentation in 2.398 s. Initial
maximum-case fixture errors used an unknown command and omitted Restart after
changing paused settings; those logs are kept separately.

The [full gate rerun passes](../docs/validation/manual-step-progress-220/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, configured mounted
race and offscreen vet. Desktop race takes 187.764 s, mounted race 168.829 s;
all 890 source/check inputs match. The clean /220 package and matching agent pass eight count cases, including
unit input and the 1,000-sample boundary. Five physical traces/results exactly
match /219; sixteen independent RC voltage/current references pass. The next
larger duration still refuses without mutation, all owned workers join and
eight processes exit 0. The bundle remains intact. Native acceptance of the
earlier UI repairs remains separate. The SDK
and UI implementation are unchanged; /219 and the qualified /205 download remain
preserved. Fresh native acceptance of the earlier UI repairs remains separate.
