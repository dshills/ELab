# Live simulation controls for users and agents

Filtered `capabilities.list {"command":"…"}` now describes nine live-control commands with `x-command-contract`: current-state requirements, effects, known errors, completion and observation/cancellation. Native controls use those commands. The ordinary catalog stays compact, and existing output schemas retain numeric SI values.

| Command | Admission and outcome |
| --- | --- |
| `simulation.transient.set` | Requires all five settings. A changed configuration is one undoable source edit; an identical configuration preserves history/physics. Changing settings while paused leaves old readings stale until Resume or Restart. |
| `simulation.input.set` | Requires a complete, fresh transient and an allowed mode. While paused, queue the mode; Step or Resume applies it. A running exact RC/NPN input applies immediately; running network input admits atomic background `input_work`. |
| `simulation.pause` | Pause immediately and request cancellation of clock/initialization/input/Step work. Committed charge/time and queued input remain; workers may still be canceling. |
| `simulation.resume` | Continue fresh transient charge/time and apply queued input. Stale/DC/uninitialized state recalculates. Already running/updating is idempotent. Inspect fresh state and any input job before depending on readings. |
| `simulation.restart` | Begin a new run at configured startup/time zero, retaining pause state; supersede input/Step, clear live cursors/window and preserve graph/history/recordings. Compatible queued mode follows engine startup policy. |
| `simulation.step.start` | Immediate admission of a bounded manual advance. Requires a paused fresh transient and no active input/Step, including canceling. Inspect `step_work` until terminal. |
| `simulation.step` | Waiting compatibility alias for the same atomic job. It can wait for pending initialization before admission, then returns its job's final outcome. After admission, caller disconnect stops waiting without canceling the work. |
| `simulation.work.cancel` | Exact current active `input_work.id` or `step_work.id`. Requests cancellation; observe terminal state. Completed, replaced or unknown IDs refuse with `operation-unavailable`. |
| `simulation.wait` | Observe initialization/status updating and active project/input/Step work for up to five seconds. Ready live transients return immediately. Settled does not imply success; inspect outcomes. |

`simulation.wait` does not launch work or wait for Verify, independent recovery reads, sweeps, printing, appearance or shutdown. Recovery work represented by active `project_work` is included. Use those families' own job state. Timeout leaves admitted work running. All mutations still require the normal observed revision/document guard and retry key.

## Shared input-mode inventory

`snapshot.live_input_modes` is an owned current-analysis list for complete supported graphs, shared with dispatcher admission and native input buttons:

| Analysis | Modes |
| --- | --- |
| RC transient | `auto`, `charge`, `discharge` |
| NPN, mixed, waveform and network transient | `auto`, `high`, `low` |
| DC, incomplete or unsupported graph | empty |

This inventory describes supported choices, not current readiness. A paused electrical edit can retain the transient analysis/modes while old readings are stale; input admission then refuses `unavailable` until a fresh run exists. It differs from `verification_input_modes`, where DC supports `auto` for an offline operating-point record.

The target is `snapshot.transient.source_id`. Network override affects the compiled primary source, leaving other supplies/waveforms alone. Auto follows the configured waveform at current simulated time. For a primary DC source the native buttons say Power on/off; the underlying modes are high/low. Voltage values and polarity still come from the captured source model. Mode actuation is session state and preserves source/history/dirty membership; it is distinct from editing source voltage or waveform settings.

## Work and numeric bounds

Step duration accepts shared unit-time text or numeric seconds. It must be positive, at most **1 s**, and at most the rounded product **1,000 × current sample step**. It must advance the current floating-point timestamp. Network analyses containing capacitors additionally require at least **1e-10 s**. The worker has the existing cooperative **30 s** deadline and unchanged per-sample numerical budgets. No partial physical advance is published on failure, cancellation or supersession. The latest job reports progress, captured revisions/time, actor, final error and outcome; it is not a kept recording or saved file data.

Live sample step is **1 µs–1 s**; playback is **0.001–1 simulated seconds per wall second**, capped at **1,000 output samples per wall second**. The global startup fallback is **0–12 V**; signed per-capacitor startup overrides remain separate. Legacy ideal pulse-source phases are **1 µs–10 s** and must cover one output step whenever that part is present, including an incomplete graph. Finite pulse/sine source parameters have their own rules.

Rate/count admission compares against rounded products rather than dividing and rejecting a one-ULP quotient overshoot. Thus a **1 µs step with 0.001 playback**, and a **1 ms manual advance of 1,000 samples**, are accepted. The next larger representable playback or duration above its product bound still refuses. The ceil-derived Step sample estimate is capped at 1,000 at that boundary. Output timestamps are anchored to the captured start plus the output index times the step; repeated addition had allowed a 1,001st final sample at later clock origins. This repairs an unusable exact minimum combination without increasing the work limits. Native number-field range endpoints map back from Vellum float32 events to the declared float64 limits, so the displayed playback minimum sends exactly 0.001.

Network algebraic input work retains the existing cooperative **5 s** deadline and atomic publication. New inputs supersede earlier input/Step work; Pause cancels. Before a dependent Step, require the relevant work's `completed` outcome. See [input jobs](input-work.md) for commit and charge-continuity rules. Numerical failure, cooperative timeout and explicit cancellation remain inspectable outcomes; a successful command admission alone does not prove a solve succeeded.

The [public workflow](../scripts/live-control-contracts-demo.py) discovers all nine contracts, tests inventory/paused queues across five circuit families, exercises stale-state refusals and exact microsecond boundaries, compares 1 ms/2 ms RC charge against `5 × (1 − exp(−t/0.01))`, and preserves source/history/files. Native callbacks and packaged smoke exercise the same minimum rate, queued input and 1,000-sample completion. These tests do not establish complete physical typing/focus/IME/accessibility or visible latency acceptance.

## Waiting for initialization

The waiting `simulation.step` alias normalizes numeric seconds and unit text before deciding whether to observe a pending initializer. For example, `0.001`, `"1 ms"` and `"1000 us"` now follow the same path. Previously, unit text could return `unavailable` immediately while the equivalent numeric request waited. The request still needs a paused, fresh, supported run before a manual step is admitted; observing initialization does not pause a running circuit or guarantee initialization succeeds.

Malformed/duplicate JSON, invalid time text, already-stale revision/document guards and existing retry keys do not wait behind unrelated initialization. Lifecycle and context guards use one shared check, called again by ordinary dispatch after any wait. An intervening command or document replacement can therefore produce `conflict`; the alias never silently refreshes the caller's context. An `expected_revision` can become stale solely because initialization publishes a result. Use `expected_command_revision` when the intended guard should survive solver publication while still rejecting intervening commands.

Before Step admission, waiting reserves no operation or retry key. Canceling that observation leaves the initializer running and permits a later fresh submission. After admission, cancellation of the waiting caller leaves the accepted Step running, with existing exact-ID cancellation/inspection available. These phases remain distinct.

Equivalent-unit retries share the original successful mutation fingerprint. A cached reply retains its original result and precedes newer context guards, matching the existing retry policy; it is not a current snapshot. Refresh `project.inspect` before depending on that result as current state. Reusing the key for a different duration still returns `key-reused`.

Controlled tests hold the real initializer to compare wait/cancel behavior, reject stale contexts promptly, release one successful Step and recheck guards after an intervening command/publication. The [public workflow](../scripts/step-initialization-demo.py) separately verifies unit-equivalent replay, strict duplicate-field refusal and preservation of an unrelated saved document through both transports. No private delay controls are added to the product API.


## Native RC control parity · October 18

A bounded actual native check on the exact **0.125.0 /142** executable now matches the external agent for Charge, Discharge, Step +1 sample and Restart. The private endpoint was observed around every native action. Queuing either mode preserved committed time/voltage; Step applied it atomically. Both one-sample jobs are attributed to `user`. Charge gives 0.47581290982020213 V at 1 ms, then Discharge gives 0.4305333247898886 V at 2 ms, matching the independent 10 ms RC law within 1e-12 V. Restart retains Pause and the discharge mode while resetting time/startup voltage.

Every reported transient field (including retained samples/events, except independent run/electrical IDs) matches the separately executed agent sequence; graph/settings and native source/history are preserved. [Source bindings, raw observations, verifier and limits](../research/ui-audit/first-version/live-control-parity/checkpoint.json). The original private host closed normally after saving its test copy. This qualifies these RC instantiations; it does not establish other input inventories, numeric entry, full mounted binding coverage, VoiceOver/IME or visible latency. No product/SDK source, review package or version changed.
