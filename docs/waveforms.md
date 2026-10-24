# Per-source sine and finite pulse waveforms

Choose **Sine source** or **Finite pulse** from Components, wire `positive` and `negative`, and select the source to edit its waveform. Negative terminals must be grounded. Native exact text fields and external agents share `source.sine.set` and `source.pulse.set`.

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

The unreleased `elab.network-waveforms-mna-sdirk2.v1` engine combines sources with the existing bounded network parts, NPN/PNP transistors and capacitors. Bounds are 3 sources, 100 electrical nodes including ground and the published [device/object limits](circuit-capacity.md). Ordinary ideal-constraint and direct source/capacitor restrictions apply.

Smooth sources are stamped at both implicit stage times. [Adaptive SDIRK2](capacitor-integration.md) compares one full step with two half steps, using 5e-7 V absolute plus 5e-6 relative local capacitor tolerance for this engine. Local tolerances are not a global error guarantee. Internal sine steps are at most 1/64 period; ramps are at most 1/8 rise/fall duration. Each source's corners are split and retained exactly. Legacy square discontinuities retain both sides at the same time; finite corners are continuous. Constant configured waveforms avoid artificial corner/substep work.

A single work budget covers the whole requested output sample, including all source corners: at most 512 work units and 100 ms. Coarse/high-frequency requests can refuse with advice to reduce sample step, preserving the previous graph/charge/time/history. This budget also prevents per-corner splits from multiplying allowed work. Existing engines retain their tolerances and legacy phase behavior.

Output sampling can hide waveform detail even when internal integration resolves it. `source_warnings` explain this in snapshots and the selected-source Inspector. **Fit waveform sampling** invokes `simulation.sampling.fit {}`: it reduces the output step toward 32 samples per sine period or 4 per shortest ramp, bounded by the supported 1 µs minimum, and lowers playback speed when needed for 1000 samples per wall second. It follows normal settings-edit restart semantics. Raw targets below the 1 µs output floor remain explicitly unresolved after fitting; the Inspector hides an ineffective Fit action and suggests longer ramps. Shared `source_sampling` snapshot entries expose target/recommendation/minimum flags. See [sampling-floor behavior](source-sampling.md). It never increases a user's finer step. Global high/low phase/sample coupling applies only when a legacy square source is present.

High/Low controls hold the compiled primary source at its extrema: sine offset ± amplitude, finite pulse high/low, or the existing legacy levels. Other sources remain automatic. Primary selection prefers the first new waveform source, then a legacy square, then the first DC source. The Source ID is exposed in transient state. Auto restores its configured waveform. Model/settings edits restart running experiments; paused edits remain stale until restart/resume.

## Portability and evidence

Live pin voltage/current probes, recorded scope, Verify and exports use the same captured graph and solver. Complete source configuration/model parameters travel in project/recovery/JSON records; canonical hashes identify model dependencies. CSV adds `waveform_model` when present. Strict import rejects missing/modified model data, missing or foreign source parameters and invalid waveform envelopes/timing. Reopening starts fresh at zero with saved pause preferences.

Fixtures compare sine RC response to the exact sinusoidal first-order solution, finite-ramp RC response to its exact linear-ramp solution, and finite pulse/NPN held phases to the independent DC engine. They also cover independent periods, exact corners, endpoint source readings, conservation, negative levels, whole-sample refusal atomicity, sampling guidance, native/agent parity and saved model/record validation.

Run `python3 scripts/waveforms-demo.py --output docs/validation/headless-waveforms.json` after building. `--artifacts PATH` retains a project, SVG and JSON/CSV records. `--connect` is only for an owned development session: this fixture replaces its schematic.

Ideal-square sources expose the same three timing fields through `simulation.pulse.timing.set`. Their entries in `snapshot.pulse_timings` have `source_kind: "pulse-source"` and `shared: true`; changing any one changes the experiment clock for all ideal-square sources. Finite entries have `shared: false`. The shared command retains sample step and playback and rejects phases shorter than the current step. See [shared clock controls](shared-pulse-clock.md).

## Exact parameter entry

Sine **Amplitude / Offset / Frequency / Phase** and finite-pulse **Low / High level** now use exact committed text fields. Enter `1234.56789012345 mV`, `.02 kHz` or `30 °` without a float32 conversion. Existing `source.sine.set` and `source.pulse.set` accept numeric values or decimal/scientific quantity text in those waveform fields. Voltage units are V/mV/uV/µV/μV; frequency units Hz/kHz/mHz; phase units deg/°. Bare text uses V/Hz/degrees respectively. Pulse phase/ramp fields retain their existing time-text units.

Unit suffixes are case-sensitive and dimension-specific. Invalid units, nonfinite/hexadecimal text and nonzero underflow refuse without changing the source. Each text quantity is at most 128 UTF-8 bytes; parsed values still obey all existing coupled model bounds. These new waveform quantity strings use one decimal conversion with the unit exponent included. Numeric SI inputs remain compatible. Equivalent quantity/numeric intent normalizes before retry matching, including in atomic batches; existing displayed values round-trip without creating an electrical/history edit. Native callbacks retain command/document guards against intervening edits.

This is command-input convenience: saved waveforms, embedded models, numerical results and output schemas remain numeric SI with the existing file/model versions. Output sampling/playback do not change automatically. `capabilities.list` returns three local waveform-quantity schema definitions; detailed command queries attach normalized SI bounds and relationships. [Public validation workflow](../scripts/waveform-quantities-demo.py) checks precision, refusal, persistence, sine and signed finite-ramp RC references, and captured source samples through both transports. Physical typing/IME/focus/accessibility and visible latency remain separate qualification gates.

## Complete waveform editing contracts

Filtered discovery for `source.sine.set` and `source.pulse.set` now includes `x-command-contract` with scope `per-source-waveform`, alongside the existing bounds, quantity definitions and coupled relationships. Find targets in `snapshot.components` by kind; the target need not be selected or connected into a complete circuit. Both missing IDs and wrong kinds return `invalid-params` for these two setters. Their frequency/period convenience command has its own error contract.

Each call replaces a complete definition. Every nested field is required and non-null, including a zero offset, phase or low level; an omitted field never retains its previous value. Quantity parsing and coupled model validation precede target lookup, subject to the common protocol/revision/document guards. Constant sine amplitude or equal pulse levels still obey all frequency/phase/ramp bounds. A triangular pulse can have rise equal to its high phase and fall equal to its low phase.

An actual change is one undoable electrical edit. Running simulation restarts from configured startup; paused accepted readings, capacitor charge and time stay visible as stale until Restart or Resume. Electrical edits clear live scope cursors/window, preserve independent captured rows and leave experiment sampling/playback unchanged. Identical normalized fields preserve physics/history/dirty state, although successful command revisions can still advance. Atomic batches validate against staged components and either commit together or preserve the entire prior state. Ordinary recording `source_unchanged` follows electrical/document/actuation revisions rather than a later numerical resemblance to an old graph.

These contracts add discovery information without changing the commands, device models, solver, work limits, file formats, ordinary catalog or output schemas. [Qualification evidence](validation/waveform-contracts-validation.json) distinguishes native callback coverage and owned native-socket workflows from full physical input/accessibility or hardware validation.

## Full-phase ramp validation

The [exact /179 boundary review](validation/finite-phase-review-179/README.md)
checks eight RC configurations with rise/fall equal to their phases, negative
and constant levels, minimum-length phases and decimal cycle boundaries. All
1,685 captured rows match an independent piecewise-linear/exponential reference;
retained live/Verify channels and complete final mixed results agree. All 49
unique corners survive and eight kept recordings reopen byte-identically.
Maximum capacitor error is 60.687 µV within the existing 3 mV bound. The minimum
cases retain unresolved sampling warnings; corner samples and internal accuracy
do not establish that the output grid meets the four-samples-per-ramp target.
No model/solver/format/budget change or additional defect results from this
bounded review; broader topology, native input and hardware gates remain separate.
