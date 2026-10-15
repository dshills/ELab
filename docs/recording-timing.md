# Threshold timing from captured samples

Capture a run with **Verify**, choose **Show recorded**, then select a captured probe under **Threshold timing**. Enter a threshold such as `2.5 V` or `1 mA`, choose rising/falling/both arrivals, and set a timing interval. The Inspector lists the exact saved rows before and after each observed arrival. Pages show eight arrivals by default. Choose **Timing off** to clear the measurement.

For the default RC charging example, choose capacitor voltage at 2.5 V. With 10 kΩ and 1 µF, the analytic half-charge time is about 6.931 ms. A capture sampled every 1 ms brackets it between the 6 ms and 7 ms rows. With R1 changed to 20 kΩ, the same threshold occurs around 13.863 ms. Discharge from 5 V uses the falling direction and has the corresponding half-decay time. These measurements use the captured resistor, capacitor and input settings, even after later live edits.

## What is measured

A rising arrival requires a previous saved value strictly below the level and the next value at or above it. Falling reverses those inequalities. Entering a plateau exactly at the level counts once; remaining on the level or leaving it does not repeatedly trigger. Starting at the threshold does not invent a preceding arrival. There is no hysteresis or debounce: noisy crossings remain separate observations.

Each result includes direction, both actual row indices, timestamps and values, plus the captured probe's signed SI units. Duplicate timestamps retain their stored order, so an ideal pulse can have a zero-width before/after bracket. No point time is interpolated, and no new samples are solved. The timing interval includes only adjacent saved rows whose exact timestamps are both inside it. It is independent of the plot window and cursors. An interval boundary never fabricates a sample; use the reported stored times when choosing very tight boundaries.

For each direction, the first two observed arrivals also yield a spacing interval. Its minimum is `second.before.time − first.after.time`; its maximum is `second.after.time − first.before.time`. These are bounds derived from the sampled brackets. They do not certify that a waveform is periodic, that a signal stayed monotonic between rows, or that no short transitions were missed. For a flasher, choose an interval after startup before interpreting a repeated edge spacing as a period. Startup spacing can differ substantially. The UI deliberately calls this **spacing**, not an exact period or frequency.

The full-window crossing count and first spacing summaries do not change when browsing pages. Fewer than two saved rows or no arrivals is a valid empty result. DC operating-point recordings therefore have no temporal arrivals. An incomplete/failed job has no complete samples to measure. DC parameter-sweep curves use a parameter axis and are not time recordings.

## Shared commands

| Command | Contract |
| --- | --- |
| `recording.timing` | Pure query. Required `id`, `probe_id`, `level`, `direction`, `start_s`, `end_s`; optional `offset`, `limit`. Reads a retained complete capture without changing selection or public state. |
| `recording.timing.set` | Same measurement parameters with normal mutation guards/key. `id` must match the selected capture. Publishes a cached derived page at `snapshot.recording.timing`, leaving live physics, history and dirty membership unchanged. |
| `capabilities.list` | Filter either command for domains, interval/page relations, required captured inventory and error contracts. |

`level` is finite text in the captured probe's units. Bare numbers mean SI; explicit unit suffixes may use `m`, `u`, `µ` or `μ` prefixes. Examples: `2.5 V`, `2500 mV`, `1 mA`, `20 µA`. Wrong units, nonfinite values, nonzero values that underflow to zero and magnitudes over 1e12 SI refuse. Text is limited to 128 UTF-8 bytes. Returned `level_si` is numeric.

Direction is `rising`, `falling` or `both`. The interval requires `0 ≤ start_s ≤ end_s ≤ captured duration`; command times accept finite seconds or unit text such as `300 ms`. Output timestamps stay numeric SI. Offset is nonnegative and may equal the total for an empty end page. Limits are 1–100; omitted or zero defaults to eight. Counts/summaries describe the whole requested interval, while `crossings` and `next_offset` describe the page.

To clear using `recording.timing.set`, supply the selected ID, empty `probe_id`, `level:"0"`, `direction:"both"`, zero start/end, and omitted/zero offset/limit. The UI's Timing off selector sends exactly that command. Selecting any operation, starting a new capture or reopening a project clears the session measurement. Projects/record exports keep underlying captured data, which can be measured again after import/reopen. Later timing choices survive an import finishing in the background. No project-format change is needed.

The computation scans at most 4096 retained rows, has no filesystem/solver work and returns a bounded page. Configured results are cached as owned derived data, so playback and ordinary snapshots do not repeat the scan. There is no long-running timing job, automatic edge-time estimation, time alignment, hidden-transition recovery or device certification.


## Pulse widths and duty cycle

The same timing query and configured UI now include first observed high/low pulse widths and the first complete rising cycle. **First high time** spans consecutive rising → falling arrivals; **First low time** spans falling → rising. Each contains both original crossing brackets and minimum/maximum duration. A cycle needs three consecutive observed rising → falling → rising arrivals. Repeated same-direction threshold touches break that sequence; the code does not skip them to invent a cycle. Starting high does not invent its initial rising edge.

These summaries use both arrival directions over the complete timing interval, independently of arrival direction filtering and paging. The existing filtered crossing counts, first same-direction spacings and page behavior remain unchanged. A simple one-edge RC charge or a DC record produces no pulse summary. Where an observed pulse has no complete rising cycle, the UI explains the missing sequence. Imported/reopened raw recordings yield the same derived measurements; the session timing choice still clears on selection/reopen.

`first_high_pulse`, `first_low_pulse` and `first_cycle` are optional additions to `recording.timing` and `snapshot.recording.timing`. Cycle fields `rising`, `falling` and `next_rising` retain the exact saved rows. Its `high`, `low` and `period` contain numeric `minimum_s`/`maximum_s`. `high_fraction` contains numeric `minimum`/`maximum` fractions (0–1); the UI displays percentages. An unresolved positive period omits that fraction and supplies `reason`. Absence is distinct from a valid zero-width sampled high pulse with a zero fraction.

For rising bracket `[r0,r1]`, falling `[f0,f1]` and next rising `[n0,n1]`, the sampled high duration is bounded by `max(0,f0−r1)` through `f1−r0`; low uses `max(0,n0−f1)` through `n1−f0`. The cycle duration is bounded by `max(0,n0−r1)` through `n1−r0`. If its minimum is positive, high-fraction bounds are:

```
minimum = (f0 − r1) / (n1 − r1)
maximum = (f1 − r0) / (n0 − r0)
```

Those expressions retain the shared rising time in numerator/denominator; separately dividing unrelated high/period extremes loses that dependence. If the cycle's minimum duration is zero, saved brackets cannot resolve a strictly positive denominator and no fraction is reported. Durations and fractions use ordinary floating-point arithmetic from captured SI timestamps. They are bounds based on observed sample brackets, not certified real-device bounds.

The default ideal NPN source yields 50 ms high, 50 ms low, 100 ms cycle and 50% high fraction. For a finite pulse rising from 0 to 5 V in 2 ms, with programmed high duration 7 ms, 3 ms fall and 20 ms period, the 2.5 V arrivals are analytically 1 ms, 8.5 ms and 21 ms. Threshold high time is therefore 7.5 ms and duty 37.5%, rather than programmed `7/20 = 35%`. The captured sample brackets contain those values. Both flasher LED currents can be measured at 1 mA in a post-startup interval with the same commands.

Pulse widths and the first complete cycle do not establish steady state, periodicity, continuous above/below-threshold behavior or absence of hidden transitions between saved rows. Plateaus use the original arrival rules, without interpolation, hysteresis or debounce. A partial interval never synthesizes a starting/ending edge. The [public pulse workflow](../scripts/pulse-timing-demo.py) compares exact user/agent results, analytic ideal/finite-pulse references and both flasher sides, verifies fraction extrema by exact rational corner enumeration, and preserves raw records across import/save/reopen. Controlled core/native regressions cover broad brackets, repeated touches, zero/unresolved cycle durations, filters/pages, ownership and explanatory UI states.

## Two-threshold rise and fall timing

Set the optional **Other threshold** in the recording Inspector to measure travel between two levels of the same captured voltage or current probe. For a 0–5 V RC charge, set Threshold to `500 mV` and Other threshold to `4.5 V` for 10–90% timing. Leave Other threshold empty to retain ordinary single-threshold behavior. The thresholds are explicit values; the app does not infer a waveform's amplitude or percentages.

The existing `recording.timing` and `recording.timing.set` accept optional `second_level`, using the same unit grammar and finite ±1e12 SI limits as `level`. Omitted or empty disables this addition. Nonempty text must normalize to a distinct level; `2.5 V` and `2500 mV` refuse as an equal pair. Either entry order is accepted and the output orders the levels numerically. Changing interval, filter or page retains this setting. Timing off requires empty/omitted `second_level` as part of its canonical clear.

When enabled, optional output `transitions` contains numeric `lower_level_si`, `upper_level_si`, and optional `first_rise` / `first_fall`. A rise begins at an observed rising arrival at the lower level and ends at a subsequent observed rising arrival at the upper level. A fall begins at the upper falling arrival and ends at the lower falling arrival. An opposite arrival back at the starting threshold aborts an unfinished traversal; a later start replaces it. The first complete traversal in each direction is retained. Plateaus use the same arrival rules described above. Intermediate fluctuations within the threshold band are allowed: this is not a monotonicity test.

Each traversal uses the existing spacing structure: `first` and `second` are the exact original crossing brackets, with saved row indices/times/values; `minimum_s = max(0, second.before.time_s − first.after.time_s)` and `maximum_s = second.after.time_s − first.before.time_s`. One saved row interval can span both levels, yielding a zero minimum and an interval-wide maximum. An ideal duplicate-time jump can produce exactly zero for both bounds. No hidden intermediate row or interpolated crossing time is created.

Transition summaries use both directions over the entire timing interval, regardless of the primary arrival filter, offset or page size. Counts, crossings, spacings and pulse/duty summaries still describe the original `level` and are unchanged by enabling `second_level`. Missing traversal means no complete observed pair; the native readout explains that. Starting inside the threshold band or clipping away the starting bracket does not invent an arrival.

For an ideal RC charge from zero to a constant voltage, the 10–90% duration is `R × C × ln(9)`. With 10 kΩ and 1 µF, that is approximately **21.972 ms**; 1 ms captured samples bound it by **21–23 ms**. Decay from 5 V has the same 90–10% duration. A linear 0–5 V source rising in 2 ms and falling in 3 ms has 10–90% durations of 1.6 ms and 2.4 ms. These are educational-model checks, not device qualification.

```json
{"command":"recording.timing","params":{"id":"verify-1","probe_id":"v:C1","level":"500 mV","second_level":"4.5 V","direction":"both","start_s":0,"end_s":"40 ms"}}
```

This is a command/parameter fragment; configured mutations need the usual request envelope, key and revision precondition. The query remains pure. Configured results remain cached, cloned session data, and selection/reopen clears them. Original exported/imported/project recording rows reproduce the same measurements without a saved-format change. The [public transition workflow](../scripts/transition-timing-demo.py) checks RC, finite ramps, ideal NPN source edges and both flasher currents, with user/agent parity and raw-record persistence.
