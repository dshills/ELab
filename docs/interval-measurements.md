# Time-weighted captured waveform measurements

Run **Verify**, select a completed capture and **Show recording**, then choose a probe under **Interval measurements**. Enter measurement start/end with time units. The readout shows average, RMS, AC ripple RMS, peak-to-peak and extrema in that captured probe's volts or amperes. This interval is independent of the plot window. **Measurements off** clears it. A DC operating point has no duration to average.

These measurements integrate a piecewise-linear reconstruction of the retained adjacent samples. Clip segments at the requested boundaries by linear interpolation; do not extrapolate. Duplicate-time edge rows retain their left/right limits, while their zero-duration vertical segments receive no weight. Extrema include the limits of positive-duration segments; an isolated zero-duration excursion contributes neither area nor a peak. At a starting edge the following segment supplies the right-hand limit; at an ending edge the preceding segment supplies the left-hand limit.

For duration T, average is integral(v)/T, RMS is sqrt(integral(v²)/T), and AC ripple RMS is sqrt(integral((v-average)²)/T). Each linear segment from a to b contributes dt*(a+b)/2 and dt*(a²+a*b+b²)/3 to the first two integrals. AC RMS centers and scales the small ripple separately before integrating, avoiding subtraction of nearly equal RMS² and average². Voltage and current signs/references follow the captured probe. RMS and AC RMS are nonnegative; peak-to-peak is maximum minus minimum. These are waveform statistics, not power in an arbitrary load.

The existing **sample mean/RMS** readouts remain distinct: they give every stored row equal weight. Time-weighted values account for partial final steps, irregular sample spacing and duplicate pulse edges. A linear reconstruction cannot establish unseen peaks or transitions, a true analog RMS value, periodicity or steady state. Reduce the sampling step and compare results for a demanding circuit.

## Shared commands

| Command | Contract |
| --- | --- |
| `recording.statistics` | Required `{id,probe_id,start_s,end_s}`; pure query against a retained completed capture. |
| `recording.statistics.set` | Same input, selected capture only; cache the same result in `snapshot.recording.interval_statistics`. Empty `probe_id` with zero start/end clears. |

Input times accept numeric SI or unit text such as `1.5 ms`. Require 0 ≤ start < end ≤ captured duration; no clamping. Uncaptured probes refuse. Output `settings` has numeric SI times; `probe`, `origin_id`, `engine`, `method:piecewise-linear.v1`, sample/segment counts and duration identify the input and method. Output fields are `minimum`, `maximum`, `peak_to_peak`, `time_mean`, `time_rms` and `ac_rms`. Query results and snapshots own their data. Full on-demand capability contracts describe bounds, errors and effects.

Set uses ordinary revision/document guards and retry retention. Selection/reopen clears session measurements; save retains the original recording for exact recomputation. Later live edits, plot decimation, playback, cursors and plot windows do not substitute or alter captured inputs. Measuring preserves live time/charge, electrical/run identity, document/dirty state and undo history. At most 4096 retained rows are scanned; this is bounded synchronous derived work with no solver or file worker.

After building, `python3 scripts/interval-statistics-demo.py --output build/interval-check.json --artifacts build/interval-check-new` checks analytic RC and ideal-pulse values, saved replay and rectifier ripple. Use `--connect` only for an owned development endpoint: the demo replaces that session's circuit and closes it afterward.

The public reference run uses the rectifier template, 2 ms saved output intervals and a 1.9–2.0 s measurement window. At 10 µF it reports average **3.09954 V**, RMS **3.18643 V**, AC ripple RMS **0.739084 V** and peak-to-peak **2.38801 V**. At 47 µF it reports **3.82757 V**, **3.83315 V**, **0.206793 V** and **0.679913 V**, respectively. Separate Python Simpson integration of every saved linear segment agrees within 1e-12 V. This is a measured late-cycle interval of this captured model, not a steady-state detector or hardware specification. See [headless/native parity](validation/interval-statistics-public-parity.json).
