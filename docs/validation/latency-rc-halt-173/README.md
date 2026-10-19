# RC collection halt and functional checks

October 19, 2026. Frozen signed **ELab 0.156.0 /173**. The original RC
block has two retained unavailable outcomes and **zero available latency bounds**.
IDs rc-03–rc-20 are unattempted. No failed ID is replaced, no full-cohort p95 is
calculated, and no performance gate passes.

`rc-01` records one completed CUA Apply call after typing private `2uF`, but the
public capacitor remains 1 µF. Electrical revision and run remain unchanged,
and the scope remains τ = 10 ms / run 12. The producer's post-capture Pause is
the only subsequent command revision. The recording retains 166 callbacks,
163 complete image frames and 161 original PNGs, losslessly archived with
member hashes and sizes. Every image preserves the accepted schematic label,
time constant and run in the manually inspected baseline regions. The captured
waveform continues animating. There is no committed edit or fresh edited result
from which to measure responsiveness.

The next ID, `rc-02`, has prepared baseline/reference files but is explicitly
aborted before UI input or capture. The block halts for diagnosis. The controller,
worker, session sources, original reports, assertion and abort are retained.
The original native session closes normally with shared settings, copied/review
and canonical applications unchanged.

Three separate functional cases are declared before their inputs, excluded from
the cohort and performed without a native recorder:

| Case | Observation |
|---|---|
| Live, index-based Apply | Accepts 2 µF with one new run and fresh τ = 20 ms. |
| Live, coordinate-based Apply | Invocation-intent file exists; no completion or immediate after-observation is returned. Later public state remains 1 µF. **Inconclusive**, not a confirmed completed no-op. |
| Paused, index-based Apply | Accepts 2 µF with one new run; explicitly paused/stale until simulation resumes. |

The owner confirms manually resizing the diagnostic window from 1053×802 to
1759×1014. These cases do not constitute a controlled layout or performance
comparison. The earlier `live-index-inspect` file is a pre-click observation;
`live-index-after` records the actual accepted result. Do not pool functional
checks with timing trials or infer a reproducible application defect from them.
No application or SDK repair is made on this evidence.

The diagnostic saves its synthetic final project and closes through public
`app.close`, reports completed exit readiness, `open -W` exit 0, absent owned
process/endpoint and no forced cleanup. Its wrapper exits 1 because the final
shared-window-settings equality assertion fails. The test process's sandbox
denies read/write access to that directory; the source of the concurrent change
is unproven. Both before/after hashes and the assertion are preserved. **No
settings are restored or overwritten.** Review, copied and canonical application
hashes still match. Normal shutdown and settings equality are separate outcomes.

The RC producer uses an independent whole-period geometric-series solution,
then the exact partial high/low phase, with no ELab timestep iteration. Two
packaged headless sessions agree at 10, 50, 51, 100 and 133.7 ms for 2 µF.
The retained 1 µF native before/after samples also satisfy that reference. These
numerical checks verify backend physics, not delivery of a UI click. The initial
restricted preflight failure and successful escalated run are both retained.

Read-only replay requires Pillow and never launches an app or recorder:

```sh
python3 -B docs/validation/latency-rc-halt-173/verify.py
```

All 650 Go sources still match the passed quality checkpoint. Keep the original
unavailable outcomes. Resume only untouched RC IDs under the declared method
after re-establishing its fixed geometry and exclusive test interaction; stop on
another uncommitted edit. NPN collection, owner walkthrough and other release
gates remain pending. No new build or promotion occurs here.
