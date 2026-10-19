# Visible responsiveness results

October 19, 2026. Frozen app: **ELab 0.156.0 /173**. Original
[collection declaration](37-visible-latency-cohort-plan.md).

The LED block has all 20 scheduled outcomes: **14 available software-frame bounds
and six unavailable test-harness outcomes**. There is no full-cohort LED p95 or
PRD performance pass. RC collection halts after one uncommitted edit and one
preparation-only abort; NPN collection remains pending.

| Reference | Scheduled outcomes retained | Available bounds | Full-cohort p95 |
|---|---:|---:|---|
| LED | 20/20 | 14 | Unavailable: six harness outcomes |
| RC | 2/20 | 0 | Unavailable: uncommitted edit; preparation abort. Eighteen unattempted. |
| Resistor-loaded NPN | 0/20 | 0 | Pending |

Available LED bounds, in milliseconds; these are **descriptive values for n=14**,
including controller overhead, from a before-input marker to the supplied
composition-frame timestamp:

| Response | Minimum | Median | Maximum |
|---|---:|---:|---:|
| Visible committed parameter | 102.684 | 116.528 | 130.839 |
| Fresh valid result | 102.684 | 132.986 | 147.876 |

The individual fresh bounds are below 500 ms. The incomplete cohort does not
establish its p95 target. Acknowledgment bounds above 100 ms are inconclusive
about product latency because controller overhead is included. These are not
exact OS input/acceptance timestamps or physical display scanout measurements.

The first trial was rejected by an incorrect comparison of generated wire IDs
across sessions. Five more failed preparation because the harness tried to reuse
session-reserved component IDs after Clear. Those six outcomes remain excluded;
none is retried. The revised producer restores the baseline through the public
load command, checks exact native wire identity preservation and compares wire
semantics across independent sessions. Only the remaining declared IDs 07–20 run.
This changes the harness, not the app or its identity guards.

All 14 subsequent edits have matching native/agent results and an independent
LED-law/KVL check. Both native test sessions close normally with settings and
app artifacts unchanged. All 2,462 captured complete frames and 2,317 original
PNGs remain in the [replayable proof](../docs/validation/latency-led-cohort-173/README.md),
including the excluded first recording, initial region/analysis mistakes and
their corrections. No app, solver, model, SDK or command changes occur.

The first RC trial has one completed CUA Apply call after private `2uF` input,
but the accepted capacitor stays 1 µF, with unchanged electrical revision/run
and τ = 10 ms. All 163 complete frames/161 original PNGs remain in the
[halt and diagnostic proof](../docs/validation/latency-rc-halt-173/README.md).
`rc-02` is explicitly aborted before input/capture; IDs 03–20 are untouched.
There is no accepted edit from which to calculate a latency bound.

Separate index-based Apply checks accept 2 µF while live and paused; the paused
result is truthfully stale. A coordinate case has no confirmed input-call
completion and remains inconclusive. The owner confirms manually resizing the
diagnostic window to 1759×1014; that change is not an inferred layout bug or a
controlled timing comparison. The RC headless preflight and retained native
samples agree with an independent whole-period analytic solution. No application
defect is reproduced or source repair made here.

Both RC sessions save synthetic final projects and close normally. The original
collection preserves shared settings. The functional diagnostic detects changed
shared window-settings hashes and its wrapper fails the final equality assertion;
the sandbox denies that process access and the change's source is unproven.
Keep the failure and current settings. App artifacts and all 650 Go sources
remain unchanged. No diagnostic result is promoted into the cohort.

Resume only untouched RC IDs after restoring the declared test geometry in an
isolated session with no concurrent interaction; stop on another uncommitted
edit. Keep these unavailable outcomes and the separate NPN block. If a complete LED p95 is needed after
the harness repair, declare a new cohort separately; do not backfill failed IDs
or calculate a full-cohort percentile from the successful subset. Owner walkthrough,
accessibility/IME, other environments and the hobbyist study remain separate.
