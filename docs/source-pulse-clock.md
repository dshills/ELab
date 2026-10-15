# Set a finite pulse clock

Select a **Finite pulse** source to edit **Frequency**, **Period**, or **High phase · %** directly. Frequency and period are two views of the same setting. Each committed field uses `source.pulse.timing.set`; agents can supply frequency/period and duty together in one command. The existing detailed high/low phase, rise/fall and voltage controls remain available.

Enter `20 Hz`, `0.02 kHz`, or `50 mHz` for frequency, `50 ms` for period, and `25 %` for the high-phase percentage. Frequency and duty use quantity text, including bare decimal/scientific text; bare duty `25` means 25 percent. Period uses the existing numeric-seconds or time-text grammar. Text commits preserve decimal precision without a native float32 conversion.

Rise belongs to the high phase; fall belongs to the low phase. The percentage describes the configured phase lengths, rather than measured time above a voltage threshold. For 20 Hz and 25 percent, the period is 50 ms, high phase is 12.5 ms, and low phase is approximately 37.5 ms. The existing 1 ms rise/fall stay unchanged. Voltage levels, other sources and project sampling/playback settings stay unchanged too.

## Shared command and configured state

```json
{"id":"Clock","frequency_hz":"20 Hz","duty_percent":"25 %"}
```

Or use a period:

```json
{"id":"Clock","period_s":"50 ms","duty_percent":"25 %"}
```

Supply exactly one of `frequency_hz` or `period_s`, with required `id` and `duty_percent`. Frequency/duty fields are strings; period accepts a finite number in seconds or time text. Query `capabilities.list {"command":"source.pulse.timing.set"}` for typed alternatives, units, bounds, model relationships, errors and examples. Current eligible IDs come from `snapshot.pulse_timings`.

Each configured timing entry contains `id`, numeric `frequency_hz`, `period_s` and `duty_percent`, plus editable `frequency_text` and `duty_text`. These are derived configuration values, available even while drawing an incomplete circuit or holding a source high/low. They are not measured oscillation results. `components[].pulse` remains the authoritative saved phase/ramp definition; floating-point phase arithmetic can differ by a last bit from decimal requests.

Re-entering the displayed timing preserves configured phase bits, physics, dirty state and undo/redo. Ordinary successful command/actor revisions can advance. A changed timing is one named **Change pulse timing** electrical edit: running simulation restarts from configured startup; paused simulation stays stale until Resume/Restart. The command also works inside an atomic named `edit.batch`.

Equivalent frequency/percentage unit text normalizes before retry fingerprinting. Frequency and period alternatives are distinct request forms. Retries return the original admission snapshot; refresh `project.inspect` for current state. Common revision/document guards protect intervening edits, including old native field callbacks.

## Bounds and refusals

Frequency is 0.05–1000 Hz; period is 1 ms–20 s; duty is strictly between 0 and 100 percent. These broad ranges are further constrained by the existing pulse model: each phase is 1 µs–10 s, and retained rise/fall times must fit the derived high/low phases. No transition is shortened or rescaled to make a request fit. For example, 1 kHz/50 percent refuses while rise and fall remain 1 ms; shorten those transitions explicitly first if that is the desired experiment.

Invalid types/units, missing/both alternatives, nonfinite/underflow values, out-of-range or incompatible derived phases return `invalid-params` without mutation. An absent source returns `not-found` after parameter validation. A different component kind requires a finite pulse source. The older **Ideal square** (`pulse-source`) continues to use shared experiment high/low timing and is not redirected by this command.

Changing the configured waveform does not automatically change output sampling. Existing source-sampling warnings and **Fit waveform sampling** remain available; solver/model/work bounds still apply. Source parameters save/open/duplicate/recover through their existing phase fields, without a file-format or model-version change. No live measured value is trusted from configured timing.

## Evidence

The [public workflow](../scripts/source-pulse-clock-demo.py) builds a finite-pulse RC circuit from blank, configures an incomplete source, checks units/retry/no-op/refusal behavior, steps the live circuit against an independent finite-ramp RC reference, verifies source sample values and high-width bounds, and saves/reopens the configured phases and kept recording. Core tests cover strict bounds, half-cycle relationships, retained ramps, arbitrary displayed-value round trips, atomic batches and snapshot/retry ownership. Native callbacks cover exact text, invalid preservation and each field's intervening-agent guard. Complete physical typing/IME/focus/accessibility, visible latency and physical-device qualification remain separate.
