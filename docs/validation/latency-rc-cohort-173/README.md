# Declared RC responsiveness block

October 19, 2026. Frozen signed **ELab 0.156.0 /173**. All 20 original RC
IDs have retained outcomes: **12 available software-composition bounds and eight
unavailable outcomes**. No full-cohort percentile or PRD performance pass.

| Available-only descriptive statistic, n=12 | Minimum | Median | Maximum |
|---|---:|---:|---:|
| Marker → visible committed value | 110.283 ms | 122.461 ms | 144.572 ms |
| Marker → fresh valid result | 110.283 ms | 122.461 ms | 144.572 ms |

These bounds include controller overhead. They are not exact OS input/acceptance
times or physical display scanout measurements. Each available fresh bound is
below 500 ms; the incomplete cohort does not establish its p95 target. Bounds
above the 100 ms acknowledgment target are inconclusive about product latency.
The [original declaration](../../../research/37-visible-latency-cohort-plan.md),
three-second duration, thresholds and original failed IDs remain unchanged.

## Outcomes and interaction isolation

- `rc-01`: one completed CUA Apply does not commit private `2uF`; accepted 1 µF,
  electrical revision, run and τ = 10 ms persist. Its visible original recording
  and separate functional checks remain in the immutable
  [halt proof](../latency-rc-halt-173/README.md). Cause unresolved.
- `rc-02`: explicitly aborted after preparation, before recording or UI input.
- `rc-03`–`rc-08`: a separate protected session preserves the fixed geometry and
  refreshes the Apply reference after the public before-state query. Five edits
  commit and match agent/analytic physics. The sixth does not commit, and the
  worker stops immediately. **All six recordings show another app covering the
  target**, so none provides a visible latency bound. The uncommitted edit is
  confounded by foreground/interaction conditions and does not establish an
  application defect. Original producer statuses are preserved separately from
  this image-based unavailable classification.
- `rc-09`–`rc-20`: the owner offers five minutes without desktop interaction,
  then explicitly leaves the machine available. A fresh protected session raises
  only its own window through CUA. The first complete recording is manually
  inspected before continuing. All twelve edits commit once and have visible
  first-parameter/fresh-result bounds. No failed or covered ID is repeated.

The initial attempt to load the CUA controller with dynamic code evaluation is
refused before any input. The controller is then declared directly through CUA;
both intended and actual sources are retained. The app, agent, recorder, solver,
models, SDK and saved format do not change. There is no speculative input repair.

## Frame and numerical replay

The twelve available recordings contain **1,991 complete image frames and 1,968
original PNG files**, retained losslessly in per-trial archives with hashes,
sizes and modes. Every frame's relevant regions classify as accepted 1 µF with
the original result, or accepted 2 µF with fresh τ = 20 ms. The first committed
parameter and first fresh result share a frame in these recordings; no captured
stale frame is required for a fast solve.

Four full original images are manually inspected. Exact schematic-label hashes
identify accepted values. Freshness requires exact RGB `τ = 20 ms` text within
the scope caption plus unchanged live/header regions; it never relies solely on
a moving current/voltage template. Three exact text raster variants accommodate
the initial zero, millisecond and centisecond caption layouts. Only translation
is allowed, with no fuzzy threshold or OCR. The initial one-template grouping
and the complete three-template check are retained. All complete frames match;
unknown/occluded states refuse. Eight corrupted-data cases are rejected on every
available recording, including old tau/old parameter labeled fresh, unfinished
input, wrong run, missing images, crop, clock and eligibility changes.

Mach/display/presentation conversion, callback/marker journal order, prospective
timestamps, bounds, PNG bytes and exact native identity are replayed without
changing raw eligibility flags. Every new native electrical revision/run advances
once. Exact native wire IDs survive; cross-session wire semantics, graph/model,
source settings and samples agree with the packaged agent at the native sample's
actual time. All retained native/agent samples satisfy the independent
whole-period RC closed form. The first fresh zero-time image also visibly shows
the valid startup sample: 0 V capacitor voltage and 0.500 mA current.

The six covered recordings have 16 complete frames and six original PNG files
with one identical full-image hash. Those other-app images remain only in their
private local capture directories; they are **excluded from Git**. Their paths,
hashes, sizes, journals and unavailable decisions are preserved here. Replay of
those original bytes therefore needs the local paths in
`remaining/trials/*/capture/private-image-storage.json`. Available RC image replay
is self-contained apart from the explicitly checked frozen runtime/source files.

Both new sessions save separate synthetic finals and close normally: public
completed shutdown/exit readiness, `open -W` exit 0, absent owned processes and
endpoints, no forced cleanup, unchanged shared settings and app artifacts. The
first worker exits 1 on its declared stop condition; the tail worker exits 0.
The earlier functional diagnostic's settings-equality failure remains separate
and unchanged. No settings are restored or overwritten. All 650 Go sources still
match the passed quality checkpoint.

Read-only replay requires Pillow; it launches no app, recorder or UI input:

```sh
python3 -B docs/validation/latency-rc-cohort-173/verify.py
```

Next: original resistor-loaded NPN block. Any new complete LED/RC cohort needs a
separate declaration; do not backfill these unavailable IDs or pool successful
subsets into a full-cohort percentile. Owner walkthrough and wider PRD gates
remain pending. This checkpoint does not rebuild or promote the app.
