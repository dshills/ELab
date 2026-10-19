# Visible responsiveness results

October 19, 2026. Frozen app: **ELab 0.156.0 /173**. Original
[collection declaration](37-visible-latency-cohort-plan.md).

The original collection retains **all 60 scheduled outcomes: 46 available
software-composition bounds and 14 unavailable outcomes**. NPN has a complete
20-observation percentile; LED and RC have unavailable outcomes and no full-cohort
p95. There is no overall PRD performance pass.

| Reference | Scheduled outcomes retained | Available bounds | Full-cohort p95 |
|---|---:|---:|---|
| LED | 20/20 | 14 | Unavailable: six harness outcomes |
| RC | 20/20 | 12 | Unavailable: one uncommitted edit, one preparation abort, six covered recordings |
| Resistor-loaded NPN | 20/20 | 20 | 137.294 ms committed value; 169.661 ms fresh result |

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

The original NPN block has **20/20 available bounds**, with one 0 V → 5 V
base-drive edit per trial and no LED in the collector load. All native graphs,
wire identities and results match the packaged agent and existing independent
70-digit Decimal reference, with KCL/KVL/emitter-current/power checks. The on
result is 4.929600780862 mA /70.399219138186 mV VCE. A producer draft's absent
top-level model-field assumption is corrected before any cohort trial; the model
is already checked in the complete nested NPN result. App/solver/SDK remain frozen.

All 3,260 complete image frames and 3,130 original PNG files replay against three
manually inspected states: private draft/original cutoff, committed 5 V with
explicit stale readings, and fresh saturation. Twelve stale frames distinguish
acknowledgment from the new valid result. Nine corruption classes are refused
per recording. The worker and owned session exit normally, with shared settings,
app artifacts and all 650 Go hashes unchanged.
[NPN proof](../docs/validation/latency-npn-cohort-173/README.md).

NPN bounds, in milliseconds; **n=20, nearest-rank p95 is the 19th value**:

| Response | Minimum | Median | p95 | Maximum |
|---|---:|---:|---:|---:|
| Visible committed parameter | 102.820 | 122.881 | 137.294 | 141.870 |
| Fresh valid result | 116.814 | 132.464 | 169.661 | 170.081 |

The fresh-result bound supports the 500 ms target for this NPN fixture within
the software-composition method. The acknowledgment bound is inconclusive against
100 ms because controller overhead is included. Neither is an exact OS-input or
physical display measurement, and neither establishes a general PRD gate pass.

The fixed original collection is now recorded in full. Complete LED/RC
percentiles would require separately declared cohorts with the repaired producer
and exclusive desktop conditions; never backfill failed IDs or calculate their
full-cohort percentile from successful subsets. Owner walkthrough, broader
performance coverage, accessibility/IME, other environments and the actual hobbyist
study remain separate.
