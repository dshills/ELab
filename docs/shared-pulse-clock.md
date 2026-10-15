# Set an ideal-square clock

Load **Examples → Load blinking NPN LED**, select **VIN**, and edit **Frequency**, **Period** or **High phase · %**. A 20 Hz clock with 25 percent high phase stays high for 12.5 ms and low for approximately 37.5 ms. The existing Pause, Step and scope controls let you inspect the LED's response.

The inspector labels this **Shared ideal-square clock**. Every ideal-square (`pulse-source`) in the document shares the experiment's high/low phase settings. Selecting one source does not make its timing independent. Finite-pulse sources retain their [independent timing and ramps](source-pulse-clock.md).

## User and agent contract

All three fields invoke `simulation.pulse.timing.set` through the same guarded dispatcher agents use:

```json
{"id":"VIN","frequency_hz":"20 Hz","duty_percent":"25 %"}
```

Or supply `"period_s":"50 ms"` instead of frequency. Exactly one timing alternative and percent text are required. Frequency accepts string quantities in Hz/kHz/mHz or bare decimal/scientific Hz; duty accepts percent text with optional `%`. Period accepts finite numeric seconds or the shared time-text grammar. Native text commits preserve decimal precision.

`capabilities.list {"command":"simulation.pulse.timing.set"}` returns typed alternatives, bounds, relationships and errors. `snapshot.pulse_timings` lists each current ideal-square source as `source_kind: "pulse-source", shared: true`, including while drawing an incomplete circuit. Filter those entries to find every affected source. Each entry contains numeric configured frequency/period/duty and exact editable frequency/duty text. These are configuration values, not measured oscillation results.

A change updates only `transient_settings.high_s` and `low_s`. Voltage values, finite/sine definitions, sample step, playback and startup voltage stay fixed. It is one **Change shared pulse timing** undo entry. Running simulation restarts from configured startup; paused simulation remains stale until Resume/Restart. Atomic named `edit.batch` stages and commits the settings with the graph, or preserves both on failure.

Re-entering displayed timing preserves phase bits, physics, dirty state and history when the current phases satisfy sampling limits. Successful command/actor revisions may still advance. An inherited invalid sampling combination is refused even on re-entry. Equivalent unit text normalizes before retry matching; cached replies describe the original admission, so refresh `project.inspect` for current state. Captured native callbacks reject intervening user/agent changes.

## Sampling limits

The broad bounds are 0.05–500000 Hz, period 2 µs–20 s, and duty strictly between 0 and 100 percent. Both phases must be 1 µs–10 s **and at least the current sample step**. For high fraction `d`, the effective frequency limit is `min(d, 1-d) / step_s`; all other experiment settings limits also apply. At the default 1 ms step, 1 kHz/50 percent is refused because both phases would be only 0.5 ms. Changing the sample step or playback requires an explicit separate experiment edit.

Malformed/extra/missing/null fields, both/neither timing alternatives, invalid quantities/types/ranges, wrong component kind or incompatible sampling return `invalid-params` without changing state. A missing source returns `not-found` after parameter validation. No sample step is shortened to make a request fit.

Save/open and recovery use existing experiment phase fields. There is no new saved timing field, file-format version, device model, numerical kernel or sample schedule.

## Evidence and remaining limits

The [public workflow](../scripts/shared-pulse-clock-demo.py) checks both transports: a live blinking NPN LED switches off at 12.5 ms and on at 50 ms, recorded source/LED phases agree, exact corners retain both sides, threshold measurement resolves the 12.5 ms high pulse, and save/open retains the settings and kept recording. A separate incomplete drawing checks the global effect across two ideal-square sources and independence of a finite pulse. Core tests cover unit retries, no-op validation, maximum configured frequency with explicitly fine sampling, atomic rollback and undo. Native tests cover all three fields, the scope notice and an intervening-agent guard; packaged smoke invokes the fields.

These checks establish the listed semantic paths on this workstation. Full physical typing/IME/focus/accessibility, visible latency, physical devices, distribution and participant acceptance remain separate release gates.
