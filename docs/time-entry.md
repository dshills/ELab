# Time values with units

Enter a time such as `10 ms`, `250 us`, `250 µs`, `250 μs`, or `0.01 s` in native cursor, interval, sampling, pulse-timing, Verify, recorded-position and autosave-interval controls. These are text fields committed by the native control, so partial keyboard input does not restart physics. Existing numerical voltage, frequency, phase, index and playback-speed controls retain their units.

Bare time text means seconds, as do JSON numbers. Supported case-sensitive units are `s`, `ms`, `us`, `µs`, `μs`, `ns` and `ps`. Decimal and scientific forms are supported, with an optional sign and spaces before the suffix. Commas, compound expressions, hex values, unknown units, nonfinite values and nonzero underflow are refused. Text is bounded to 128 UTF-8 bytes. Command-specific sign/range, retained-history, sample-fit and work limits remain unchanged; parsing a unit never makes an unsupported time admissible.

The same public fields accept either numeric SI seconds or unit text:

| Command | Time fields |
|---|---|
| `simulation.transient.set` | `step_s`, `high_s`, `low_s` |
| `simulation.step`, `simulation.step.start` | `duration_s` |
| `verification.settings.set`, `simulation.verify` | `duration_s`, `sample_step_s` |
| `scope.cursor.set`, `recording.cursor.set` | `time_s` |
| `scope.window.set`, `recording.window.set` | `start_s`, `end_s` |
| `recording.seek` | `time_s` (alternative to sample `index`) |
| `source.pulse.set` | `waveform.high_s`, `low_s`, `rise_s`, `fall_s` |
| `recovery.configure` | `interval_s` |

For example, `scope.cursor.set {"cursor":"a","time_s":"10 ms"}` uses the ordinary retained-sample snapping and timing measurements. `simulation.step {"duration_s":"10 ms"}` uses the same bounded atomic Step worker as `0.01`. Finite pulse timing also accepts text inside supported atomic schematic batches. Batch command scope is unchanged: simulation experiment settings cannot be included in those schematic batches.

Command normalization happens after strict JSON duplicate-key validation and before ordinary typed validation/retry admission. One decimal exponent is composed before floating conversion, avoiding separate scale rounding. Equivalent forms such as `"10 ms"`, `"10000 us"` and `0.01` have the same mutation retry intent. Unknown/duplicate fields, malformed nesting and ordinary revision/key rules remain enforced. Waiting/immediate job commands retain their original actor, cancellation and completion behavior.

`capabilities.list` references local `urn:elab:time:1` for affected command input fields; its registry definition gives accepted types, grammar description, budget and examples. Protocol version 1 and schema registry version 2 remain compatible, and numeric clients keep working. Output snapshots, project files and recorded files continue to store strict numeric SI values; string timing in those files is rejected. No file/model format or numerical bounds changed.

Native controls use the same public string parameters, preserve full float64 values in display round trips and refuse commits from an old rendered command revision. Invalid units/ranges preserve committed settings/physics/history and show the shared error. Entry no longer silently clamps a request to a retained boundary. Cursor requests continue to snap to actual accepted samples, whose floating timestamps can differ slightly from the requested decimal time. Native typing/focus/IME and screen-reader qualification remain separate release checks.

[Core/native/public validation](validation/time-entry-validation.json), [headless workflow](validation/headless-time-entry.json) and [packaged workflow](validation/desktop-time-entry.json) cover live/recorded fields, ideal RC reference samples, complete failure preservation, nested finite pulse timing, Step job behavior, saved numeric data and fresh paused reopening. Unit/cancellation/model/work behavior remains on the same service. Run `python3 scripts/time-entry-demo.py --output REPORT --artifacts DIRECTORY`; `--connect` targets only a designated owned session.
