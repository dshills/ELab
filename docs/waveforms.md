# Per-source sine and finite pulse waveforms

Choose **Sine source** or **Finite pulse** from Components, wire `positive` and `negative`, and select the source to edit its waveform. Negative terminals must be grounded. Native number fields and external agents share `source.sine.set` and `source.pulse.set`.

The existing `pulse-source` appears as **Ideal square** in the native tray. It keeps instantaneous edges and shared experiment timing, including existing RC/NPN examples, old projects and recordings. New waveforms use distinct component/model identities; their parameters belong to each source and survive duplication/save/reopen.

## Parameters and limits

| Kind | Model | Defaults |
|---|---|---|
| `sine-source` | `elab.ideal-sine-source.v1` | Amplitude 2.5 V, offset 0 V, frequency 10 Hz, phase 0° |
| `finite-pulse-source` | `elab.finite-pulse-source.v1` | High 5 V, low 0 V, high/low phases 50 ms each, rise/fall 1 ms each |

Sine voltage is `offset + amplitude × sin(2π × frequency × time + phase)`. Amplitude is nonnegative; the full offset/amplitude envelope must remain within ±12 V. Frequency is 0.1–1000 Hz and phase ±360°. `component.set` changes amplitude; the complete command accepts `{"id":"V1","waveform":{"amplitude_v":2.5,"offset_v":0,"frequency_hz":10,"phase_degrees":0}}`.

Finite pulses begin at the low level at time zero, rise linearly, hold high, fall linearly, then hold low. The period is `high_s + low_s`. Rise time is included in the high phase and fall time in the low phase, so plateau lengths are `high_s − rise_s` and `low_s − fall_s`. These can be zero for a triangular waveform. Reported duty is the high-phase fraction of the period. Levels are ordered `low_v ≤ high_v` within ±12 V; equal levels form a constant source. Each phase is 1 µs–10 s, the period is at least 1 ms, and positive rise/fall times of at least 1 µs must fit their phases. `component.set` changes the high level.

The complete pulse command is `{"id":"V1","waveform":{"high_v":5,"low_v":0,"high_s":0.05,"low_s":0.05,"rise_s":0.001,"fall_s":0.001}}`. Invalid commands refuse before changing any parameter. Required nested fields cannot be omitted or null.

Finite sources also offer direct **Frequency / Period / High phase %** controls through `source.pulse.timing.set`. They retain voltage levels and rise/fall times, and expose derived configured values and exact editable text in `snapshot.pulse_timings`. See [pulse clock controls](source-pulse-clock.md) for quantity syntax, coupled bounds, retry/no-op behavior and the separate ideal-square timing path.

All sources are ideal; there is no output resistance, current limit, battery discharge, noise, thermal model, rating or physical generator identity. They are educational waveform definitions, with complete immutable `waveform_model` metadata (`elab.source-waveforms.v1`) accompanying projects and records.

## Sampling and numerical behavior

The new `elab.network-waveforms-mna-be.v1` engine combines sources with the existing bounded network parts, grounded-emitter NPNs and capacitors. Bounds are 3 sources, 100 electrical nodes including ground and the published [device/object limits](circuit-capacity.md). Ordinary ideal-constraint and direct source/capacitor restrictions apply.

Smooth sources are stamped at integration endpoints. Adaptive backward Euler compares one full step with two half steps, using 5e-7 V absolute plus 2e-5 relative local capacitor tolerance for this engine. Local tolerances are not a global error guarantee. Internal sine steps are at most 1/64 period; ramps are at most 1/8 rise/fall duration. Each source's corners are split and retained exactly. Legacy square discontinuities retain both sides at the same time; finite corners are continuous. Constant configured waveforms avoid artificial corner/substep work.

A single work budget covers the whole requested output sample, including all source corners: at most 512 work units and 100 ms. Coarse/high-frequency requests can refuse with advice to reduce sample step, preserving the previous graph/charge/time/history. This budget also prevents per-corner splits from multiplying allowed work. Existing engines retain their tolerances and legacy phase behavior.

Output sampling can hide waveform detail even when internal integration resolves it. `source_warnings` explain this in snapshots and the selected-source Inspector. **Fit waveform sampling** invokes `simulation.sampling.fit {}`: it reduces the output step toward 32 samples per sine period or 4 per shortest ramp, bounded by the supported 1 µs minimum, and lowers playback speed when needed for 1000 samples per wall second. It follows normal settings-edit restart semantics. It never increases a user's finer step. Global high/low phase/sample coupling applies only when a legacy square source is present.

High/Low controls hold the compiled primary source at its extrema: sine offset ± amplitude, finite pulse high/low, or the existing legacy levels. Other sources remain automatic. Primary selection prefers the first new waveform source, then a legacy square, then the first DC source. The Source ID is exposed in transient state. Auto restores its configured waveform. Model/settings edits restart running experiments; paused edits remain stale until restart/resume.

## Portability and evidence

Live pin voltage/current probes, recorded scope, Verify and exports use the same captured graph and solver. Complete source configuration/model parameters travel in project/recovery/JSON records; canonical hashes identify model dependencies. CSV adds `waveform_model` when present. Strict import rejects missing/modified model data, missing or foreign source parameters and invalid waveform envelopes/timing. Reopening starts fresh at zero with saved pause preferences.

Fixtures compare sine RC response to the exact sinusoidal first-order solution, finite-ramp RC response to its exact linear-ramp solution, and finite pulse/NPN held phases to the independent DC engine. They also cover independent periods, exact corners, endpoint source readings, conservation, negative levels, whole-sample refusal atomicity, sampling guidance, native/agent parity and saved model/record validation.

Run `python3 scripts/waveforms-demo.py --output docs/validation/headless-waveforms.json` after building. `--artifacts PATH` retains a project, SVG and JSON/CSV records. `--connect` is only for an owned development session: this fixture replaces its schematic.

Ideal-square sources expose the same three timing fields through `simulation.pulse.timing.set`. Their entries in `snapshot.pulse_timings` have `source_kind: "pulse-source"` and `shared: true`; changing any one changes the experiment clock for all ideal-square sources. Finite entries have `shared: false`. The shared command retains sample step and playback and rejects phases shorter than the current step. See [shared clock controls](shared-pulse-clock.md).
