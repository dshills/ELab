# Browser responsiveness at capacity

The November 4 engineering workload uses disposable native-format fixtures padded
with grounded junctions to 100 drawing objects. It separately exercises the
100-node boundary (51 simulated parts) and 64-part boundary (60 nodes), each with
six scope traces. It does not claim all three maxima are reached simultaneously.

Local headless Chromium runs with a 4× main-thread CPU slowdown and a 50%-duty
background worker. Each case measures eight accepted keyboard moves of all 100
objects, twenty pointer movements and two seconds of live scope publication.
Accepted-edit latency ends at the acknowledged SVG update plus two animation
frames; drag latency ends at the next frame after the event handler. These are
synthetic laboratory proxies, not physical event-to-pixel or field INP results.

| Case (100 drawing objects each) | Edit median ms, before → after | Edit p95 ms | Drag frame p95 ms | Same-revision full → conditional snapshot bytes |
| --- | ---: | ---: | ---: | ---: |
| 51 parts / 100 nodes | 109.3 → 74.2 | 365.5 → 340.3 | 11.2 → 1.5 | 274,835 → 209,681 |
| 64 parts / 60 nodes | 96.4 → 68.3 | 121.7 → 88.0 | 10.4 → 1.1 | 417,938 → 357,299 |

The first 100-node edit remains the slow tail in this small sample; p95 with eight
samples is effectively the largest observed value. The results do not establish
performance on another machine or calibrated mobile hardware. Both cases stayed
running with six traces, ten observed scope updates in two seconds and zero
component-root replacements during the final live interval. Snapshot-transfer
comparison uses the same paused revision, avoiding different trace-history sizes.
The before/after live median payloads are also retained, but their trace counts
can differ and should not be treated as a fixed-size bandwidth comparison.

## Changes and correctness

Opaque scene tokens scope geometry to the host, workspace, document and command
admission without revealing the HttpOnly cookie. Matching polls deliver live
markers/control paths/display segments instead of retransmitting static geometry.
Edits, workspace switches and host restarts require a full scene. Clients without
the token keep the existing full response.

An optional browser command projection includes the acknowledged scene and
workspace metadata with the unchanged native result. This avoids immediately
fetching a second full snapshot. Retries and superseded snapshots use the normal
fresh-read path; cached native results and guards are unchanged. Recovered dirty
status is overlaid for the browser without relabeling the native result.

Geometry-only edits update existing component, terminal and wire nodes; structural,
value or selection changes rebuild them. Live segment/control updates preserve
focused node identity. Cached drawing/selection lookups remove repeated scans.
SVG-native coordinate conversion handles aspect/layout transforms; drawing
gestures suppress native text selection and keep focus. Toolbar layout is
synchronized before starting the next interaction. No solver model, bound or
result sample is changed.

## Reproduce

The later multi-sheet browser uses only the active sheet for drawing geometry. It reuses routed geometry during live polling and keeps unchanged SVG elements when a neighboring component is edited. The expanded Chromium check adds a 200-object drawing with 64 simulated parts, 60 nodes and six scope traces. Under the same synthetic CPU load, eight group edits had a 113 ms median and a 215 ms maximum, and the scope updated 11 times in two seconds.

Build `build/elab-web-browser`, install the pinned browser dependencies/runtimes
from [browser qualification](../browser/README.md), then run:

```sh
cd browser
ELAB_BROWSER_TESTS=1 npm test -- responsiveness.spec.mjs --project=chromium
```

The test attaches `responsiveness.json`. Set `ELAB_PERF_REPORT` to a new local
report filename if needed. Owned hosts, temporary libraries, worker and CPU
throttling are cleaned up. The suite's readiness probe uses the static homepage
so it does not consume a workspace. Timing is reported; functional capacity,
progress, same-revision geometry omission and correct edits are asserted.

[Before](validation/web-remaining-20261104/responsiveness-before.json) and
[after](validation/web-remaining-20261104/responsiveness-after.json) are development
measurements tied to baseline `47a7b010` and this milestone's working tree. They
are not shipping/release receipts. The test approach follows Chrome's
[CPU throttling guidance](https://developer.chrome.com/docs/devtools/settings/throttling).
[Field INP](https://web.dev/articles/optimize-inp) uses a different population and
percentile; these laboratory accepted-edit timings are not labeled as INP.
