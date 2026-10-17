# PRD performance fixtures · prepared, not timed

These are the three reference topologies named in PRD section 11. They were constructed in new headless processes through public component/wire/settings commands using the projects in the frozen 0.123.0 /140 four-adapter conformance report. They contain no recordings and are saved paused; RC starts at time zero. [Provenance and numerical checks](provenance.json) identify the source report, agent and resulting files.

| File | Baseline | Representative edits |
|---|---|---|
| [LED](led.elab) | 5 V, 1 kΩ, generic red LED; 2.916796577 mA | Limit 1 kΩ ↔ 2 kΩ; missing-wire repair |
| [RC](rc.elab) | 5 V ideal square, 10 kΩ, 1 µF; 3.160602794 V at 10 ms | Limit 10 kΩ ↔ 20 kΩ while live; pause/restart separately |
| [NPN switch](npn.elab) | 5 V supply/drive, 10 kΩ base, 1 kΩ collector; VCE 70.399219 mV | Drive 0 V ↔ 5 V; base/load value edits |

The NPN fixture has a resistor collector load, matching the PRD reference; the capacitor-timed LED flasher remains an additional stress experiment. Do not replace this NPN case with flasher timing and call the three-reference gate complete.

For a timing run, record hardware/OS/build/model identities, viewport/theme, foreground/visibility state, cold/warm state, sample settings and edit sequence. Retain at least 20 eligible trials per reference. Separate input-event → visible acknowledgment from accepted edit → first valid updated sample. An Updating acknowledgment may precede the valid result; stale readings never count as the new sample. Retain failed/canceled/out-of-scope trials with reasons rather than silently dropping them.

The current computer-use API provides screenshot bytes without the native event, capture-instant or compositor-presentation timestamps needed to resolve the launch targets. Its request-to-return timings include observation overhead. The [calibration receipt](../responsiveness-calibration.json) therefore qualifies **zero** p95 samples. Establish a validated measurement path before collecting a benchmark; do not relabel API, CPU frame-work, Present-call or screenshot-return durations as actual visible latency. Also test cold routing and navigation/cancel/save/undo under a deliberately difficult workload, separately from these baseline edits.
