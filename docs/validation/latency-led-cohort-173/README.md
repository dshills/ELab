# Declared LED responsiveness block

October 19, 2026. **All 20 scheduled LED IDs have retained outcomes: 14 available
software-frame bounds and six unavailable harness outcomes. No full-cohort p95
or latency gate result is reported.** The original declaration is commit
`d5d10d0`, [protocol](../../../research/37-visible-latency-cohort-plan.md).

The available bounds describe a before-input marker → supplied composition-frame
timestamp on frozen signed ELab 0.156.0 /173 and recorder `a8cec4e…923f`.
They include CUA/controller overhead and are not exact OS-input/accepted-edit
latency or physical scanout measurements.

| Available-only descriptive statistic, n=14 | Minimum | Median | Maximum |
|---|---:|---:|---:|
| First visible committed value | 102.684 ms | 116.528 ms | 130.839 ms |
| First fresh valid result | 102.684 ms | 132.986 ms | 147.876 ms |

Each available fresh-result bound is below 500 ms, but the incomplete cohort
cannot establish its proposed p95 target. The acknowledgment bounds exceed
100 ms and are inconclusive about actual product latency because controller
overhead is included. Targets and the nearest-rank/full-cohort rule stay unchanged.
No percentile is calculated from the successful subset.

## Retained failures and producer repair

- `led-01` records one Apply and correct 1 kΩ → 2 kΩ physics, but the producer
  compares generated native wire IDs W5–W8 with fresh-agent W1–W4 and reports
  failure. Independent replay confirms identical endpoints, connected flags and
  numerical results. The original unavailable outcome stays excluded; its raw
  119.412/136.079 ms intervals remain diagnostic data.
- `led-02`–`led-06` fail during preparation because clearing a project does not
  release session-reserved component IDs. No recording or Apply occurs for those
  IDs. The first worker is explicitly halted after six outcomes and exits 1;
  its native session still closes normally. Its traceback and halt decision remain.
- Before `led-07`, the revised producer is recorded with both source hashes.
  A fresh protected native session constructs the same reference graph once and
  restores its saved baseline through public `project.load` for later trials.
  Cross-session wire comparison checks endpoint/connected semantics, while exact
  native wire IDs must remain unchanged by the value edit. Only IDs 07–20 are run.
  No failed ID is retried, and app/agent/recorder/model/duration are unchanged.

All 14 subsequent trials complete, with one CUA Apply/electrical revision/run
increment each, matching packaged-agent physics and an independent bisection of
the declared Shockley LED law plus series resistance/KVL. Baseline saves remain
unchanged; final projects are separate. No application defect or relaxation of
its identity guards is inferred from these harness mistakes.

## Image, clock and cleanup replay

The 15 recordings contain **2,462 complete image frames and 2,317 original PNGs**.
All frames classify against three manually inspected full originals: private draft
with 1 kΩ/2.917 mA, committed 2 kΩ with explicit stale labels, and fresh
2 kΩ/1.477 mA. A fast solve need not expose a captured stale frame. The analysis
finds the first committed value independently of solver completion. All images,
including the excluded first recording, are retained losslessly in per-trial
`frames.tar.xz` archives with member/hash/size/mode manifests.

The first region proposal, inherited from the 65% calibration viewport, clips
the label in the fixture's 57% fit. Inspection corrects it to cover the complete
schematic value label. Both grouping outputs/scripts remain. No recording is
repeated and the replay uses the corrected complete-label region for every trial.

The initial analysis also assumes every supplied display time after the marker
must carry a cached marker interval. `led-08` disproves that: callback 2 is
journaled before acknowledgment but carries a prospective display time after it.
The corrected replay uses journal callback order to determine whether that marker
was already known, without adjusting timestamps. Both analysis versions and the
failed log remain. Eight corruption classes are refused on every recorded dataset;
two additional checks reject attaching a marker retroactively to that prospective
pre-ack frame and removing a required post-ack interval.

Both native sessions report public completed shutdown/exit readiness, `open -W`
exit 0, absent owned PIDs/endpoints and unchanged shared geometry, review, copied
and canonical applications. Native process exit status is unavailable under
LaunchServices. CUA independently reports no running ELab afterward. Host metadata
matches the declaration in both phases; every capture retains exact PID/window,
crop, mode, timestamps, report and controller source/events. Existing access is
used without a privacy grant, audio, upload or other-app input.

Replay is read-only and requires Pillow. It never launches the app or recorder:

```sh
python3 -B docs/validation/latency-led-cohort-173/verify.py
```

All 650 Go hashes remain identical to the passed quality checkpoint. No app,
solver, SDK, command or saved-format changes occur; source-quality and package
evidence retain their recorded scopes. Raw eligibility flags remain false and
gate outcomes remain null. Next: the original RC/NPN blocks. A complete LED p95
would require a separately declared cohort after the harness repair; these failed
IDs must never be backfilled or silently pooled away.
