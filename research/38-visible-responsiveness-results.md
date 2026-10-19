# Visible responsiveness results

October 19, 2026. Frozen app: **ELab 0.156.0 /173**. Original
[collection declaration](37-visible-latency-cohort-plan.md).

The LED block has all 20 scheduled outcomes: **14 available software-frame bounds
and six unavailable test-harness outcomes**. There is no full-cohort LED p95 or
PRD performance pass. The RC block also retains all 20 outcomes: **12 available
bounds and eight unavailable outcomes**. NPN collection remains pending.

| Reference | Scheduled outcomes retained | Available bounds | Full-cohort p95 |
|---|---:|---:|---|
| LED | 20/20 | 14 | Unavailable: six harness outcomes |
| RC | 20/20 | 12 | Unavailable: one uncommitted edit, one preparation abort, six covered recordings |
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
`rc-02` is explicitly aborted before input/capture. Neither has a timing bound.

Separate index-based Apply checks accept 2 µF while live and paused; the paused
result is truthfully stale. A coordinate case has no confirmed input-call
completion and remains inconclusive. The owner confirms manually resizing the
diagnostic window to 1759×1014; that change is not an inferred layout bug or a
controlled timing comparison. The RC headless preflight and retained native
samples agree with an independent whole-period analytic solution. No application
defect is reproduced or source repair made here.

Those two RC sessions save synthetic final projects and close normally. The original
collection preserves shared settings. The functional diagnostic detects changed
shared window-settings hashes and its wrapper fails the final equality assertion;
the sandbox denies that process access and the change's source is unproven.
Keep the failure and current settings. App artifacts and all 650 Go sources
remain unchanged. No diagnostic result is promoted into the cohort.

The next protected session retains fixed geometry and refreshes Apply after the
public before-state query. Five edits commit and match analytic/agent results;
the sixth does not commit and immediately stops the worker. **All six actual
recordings show another app covering ELab.** IDs 03–08 are therefore unavailable
for visible timing. The covered uncommitted edit has confounded foreground/input
conditions; it does not prove an application defect.

The owner then offers an exclusive desktop interval and subsequently leaves the
machine available. A separate protected session raises its own window through
CUA, checks the first original recording, and collects only untouched IDs 09–20.
Every edit commits once with matching native/agent physics at the same actual
sample time and independent RC laws. The 1,991 complete frames and 1,968 original
PNGs replay against accepted schematic values and exact 20 ms scope-text
templates, accommodating moving transient captions without OCR or static
voltage/current assumptions. Eight corruption classes are rejected per recording.
[RC cohort proof](../docs/validation/latency-rc-cohort-173/README.md).

Available RC bounds, **descriptive values for n=12**, in milliseconds:

| Response | Minimum | Median | Maximum |
|---|---:|---:|---:|
| Visible committed parameter | 110.283 | 122.461 | 144.572 |
| Fresh valid result | 110.283 | 122.461 | 144.572 |

The two responses share a captured frame in these twelve trials. No full-cohort
p95 or performance pass follows from the available subset. Both new sessions
close normally with shared settings and app artifacts unchanged. Covered images
remain in private local capture directories, with hashes and unavailable records
in the repo; those other-app images are excluded from Git. No app/SDK/source
repair or replacement trial occurs.

Continue with the separate original NPN block. If a complete LED or RC p95 is
needed, declare a new cohort separately; do not backfill failed IDs
or calculate a full-cohort percentile from the successful subset. Owner walkthrough,
accessibility/IME, other environments and the hobbyist study remain separate.
