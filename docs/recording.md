# Recorded-result import and replay

Open a saved **recorded JSON** from the Verify inspector. ELab validates the complete `elab.verification` version 1 file, retains its captured input and samples, and opens a clearly labeled **Recorded scope**. The live schematic, readings, time, source mode, cursors, pause state, undo history and explicit save point are preserved. The source inspector lists captured components, connections, startup settings and engine identity through the same `operation.inspect` query agents use.

The recorded panel offers Play/Pause, speed, previous/next sample, exact sample index, time seek, A/B cursors and a return to the live scope. The inspector offers captured probe selection, a visible interval, exact interval statistics and cursor voltage/current deltas. Changing the live circuit preserves the recording. Recorded playback never invokes a solver, synthesizes interpolated values or resumes saved physical state.

A time seek/cursor snaps to the nearest saved sample, selecting the latest/right-hand row on equal-time edges or equal-distance ties. Index seek and row stepping expose each actual row, including both sides of ideal pulse edges at the same time. Playback displays the latest saved row at or before its wall-clock target, stops at the final row, and restarts at the beginning when Play is pressed at the end. It can skip display frames at high speed; exact inspection/export remains available. A queued old clock tick cannot rewind a newly positioned playhead.

The plot retains first/last samples and each of 128 ordered buckets' minimum/maximum points (at most 258 points per trace). Reduction is visibly labeled. Cursor lookup, statistics, paged samples and JSON/CSV exports use all original data, including edge rows. Statistics are sample-based, not time-weighted. An interval with no exact samples is labeled unavailable instead of inventing values. A DC record is one point at time zero and has no playback duration.

The source UI now shows a one-row record’s fixed index as readable text; multiple-row captures keep exact numeric seeking. This avoids the zero-span numeric-control failure exposed by native /182 qualification. [RECORD-DC-UI-01 evidence](validation/recorded-dc-control/README.md) retains the failed candidate and source repair; [clean /183](validation/recorded-dc-review-183/README.md) qualifies the copied native/headless sequence. The shared `recording.seek` command still accepts DC index zero.

## Shared commands

All mutations require the ordinary revision and retry key. Inspect `snapshot.recording` for current position, viewing/playback state, readings, scope, reduction policy and complete sample count.

| Command | Parameters / behavior |
| --- | --- |
| `verification.import` | `path`, optional `expected_document_generation`; validate and load JSON, select a new local record ID, show Recorded scope |
| `operation.select` | Existing `{id}`; select a retained operation, pause playback and reset recorded position/probes/window/cursors; empty ID closes the viewer |
| `operation.inspect` | Existing `{id}`; captured metadata and final results; imports include local source path/hash in the operation summary |
| `operation.samples` | Existing `{id,offset?,limit?}`; exact rows in pages, including imported records |
| `recording.view.set` | `mode:"live"|"recorded"`; recorded requires a completed selection; returning live pauses recorded playback |
| `recording.seek` | Exactly one of `index` or `time_s`; pause and seek |
| `recording.step` | `direction:"previous"|"next"`; pause and move one actual row |
| `recording.speed.set` | `seconds_per_wall_second` (.001–10); configure without starting playback |
| `recording.play` | `seconds_per_wall_second` (.001–10); play from current row or restart at end |
| `recording.pause` | Empty object; freeze recorded playback only |
| `recording.window.set` | `start_s`, `end_s` inside the complete recording; positive ordered interval, or 0/0 for DC |
| `recording.probes.set` | `ids` from captured probes only; `[]` hides traces |
| `recording.cursor.set` | `cursor:"a"|"b"`, `time_s` inside the complete recording |
| `recording.cursor.select` / `.clear` | `cursor:"a"|"b"`; choose click target or clear measurement |
| `verification.export` | Existing `{id,path,format:"json"|"csv",overwrite?,expected_document_generation?}`; preserves original captured metadata/rows |

An imported record receives a new local `record-N` ID; its original `metadata.operation_id` and creation time remain intact. Re-export preserves the original portable record and does not add host paths. Imported `source_unchanged` is always false: foreign revision counters cannot establish association with a live session, even when their numbers coincide. The UI states that imported data has no live source association. The file's SHA-256 identifies exactly what was loaded, not who produced it or whether the circuit is physically accurate.

## Validation, bounds and current limits

Import requires a regular local file at most 8 MiB; links, invalid UTF-8, duplicate/unknown/noncanonical fields, missing required fields and null fields refuse. The entire captured schematic, supported model parameters, analysis/engine identity, source mode and numeric settings are validated. Probe definitions must match the captured graph. Rows must cover every requested output sample and the complete interval, stay ordered, match the channel/sample count, contain finite bounded measurements and preserve at most three rows at one edge time. Reported statistics must match the full rows. Final-result identity and device lists must match the captured supported circuit. No imported data is re-solved or certified.

The source validator also compares selected channels in saved LED/NPN final results against the last captured row, including the right side of a pulse edge, using the existing consistency tolerance. NPN base/load resistor and LED identities must match the captured graph. Standalone imports and embedded project/recovery recordings share this check. [RECORD-FINAL-01 source evidence](validation/recording-final-consistency/README.md) preserves thirteen original /181 acceptances and corrected refusals, intact byte-identical exports and pulse-edge/High/Low controls. [Clean /183](validation/recorded-dc-review-183/README.md) qualifies exact copied native/headless refusals and intact exports; /181 remains affected and preserved. This does not certify unselected scalar fields or arbitrary solver history.

Grid matching tolerates decimal timestamp roundoff up to the smaller of the existing `1e-12 × max(1, abs(target_s))` tolerance and one billionth of the spacing to the nearest distinct planned output time. This includes the final partial step. A startup row cannot substitute for a required tiny final sample, and a preceding full step cannot substitute for a distinct final partial sample. This tightens acceptance of incomplete files previously hidden by the absolute tolerance; complete exports and ordinary rounded timestamps remain supported. Embedded project/recovery recordings use this same validator. File versions and raw saved samples are unchanged.

Existing Verify bounds still apply: 10 seconds, 1000 output steps, 4096 rows and six captured channels. Offline sample overrides do not inherit irrelevant live playback-rate limits; DC step metadata is not a pulse-duration constraint. Explicit empty Verify probe/startup-override arrays are preserved by typed Go requests as well as raw JSON requests; nil/omission inherits defaults.

The project retains at most eight operations/records. Import can coexist with one active verification and evicts an excluded terminal operation, never an active job or a kept recording. Kept capacity requires explicit exclusion before replacement. Imports, selections and replay use immutable records, independent of graph history and live simulation. Session close cancels playback. Failed import/measurement validation leaves the editor/viewer intact.

CSV is an export for external analysis; it lacks the full captured schematic and is not an import format. Arbitrary SPICE/other-app waveform formats are outside this native-record format. Version-4 `.elab` projects and recovery capsules now include kept recordings and saved pause/viewer preferences. Completed/imported recordings mark these project assets unsaved; live physics and schematic history remain independent. See [project format and lifecycle](project-format.md).

Core/native tests cover all five supported analyses, exact metadata/export round-trip, corrupt input refusal, empty captures, DC sampling overrides, pulse edge stepping, timing cursors, plot bounds/extrema and complete statistics, independent playback, snapshot cloning, native picker parameters and user/agent control parity. The native GPU/UI smoke opens the recorded plot and uses its seek/cursor controls. `scripts/recording-demo.py` provides a repeatable headless or owned-desktop integration workflow.

Captured runs can also be compared numerically at a requested time, retaining each actual saved sample time and separate whole-run statistics. See [Compare captured experiments](recording-comparison.md).

For RC delays and flasher edges, [threshold timing](recording-timing.md) reports saved-row brackets and same-direction spacing bounds without interpolating event times.

Recorded cursor time fields and native plot input now pass an optional exact `operation_id` to `recording.cursor.set`. A changed selected capture refuses atomically; clients omitting the field keep current-selection behavior. See [scope input context](scope-input-context.md) for prepared geometry, revision/document guards, drag continuity and remaining OS presentation/input limits.

Full-record plotting exposes the actual saved endpoint as `last_sample_time_s`, independently of rounded capture metadata duration. Empty visible intervals contain no scope statistics or recorded inspector readings; raw sample rows and position remain available. [Narrow-window membership](scope-probes.md#very-narrow-intervals) is shared with reference overlays and live scope/CSV.

## Capacitor engine compatibility

Current general capacitor captures identify SDIRK2. Earlier backward-Euler captures are admitted only within their original analysis family and retain their engine, samples and exact JSON re-export. Captured and final-result engine identities must agree. Current SDIRK2/Radau results require zero `numerical_damping_w`; historical backward-Euler results may retain finite nonnegative damping. These are consistency checks, not authentication of arbitrary solver history. See [integration and compatibility](capacitor-integration.md).
