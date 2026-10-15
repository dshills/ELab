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
