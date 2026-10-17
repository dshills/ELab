# Source output-sampling limits

**Fit waveform sampling** keeps the shortest-ramp target visible when the scope cannot reach it. A 1 µs finite ramp has a 0.25 µs target under ELab's four-samples-per-ramp policy, while the supported output interval stops at 1 µs. Fit reaches that floor; it does not satisfy the target. The warning remains, an ineffective Fit button disappears, and guidance suggests longer rise/fall times.

This repairs the former behavior where clipping the target to the supported minimum also cleared the warning. It changes guidance and observability; waveform definitions, internal integration, numerical tolerances/work budgets and output-interval limits stay unchanged. The target is an engineering display policy, not a guarantee of solver accuracy or sufficient sampling for every response.

`project.inspect.source_sampling` exposes owned, derived entries for configured nonconstant sine/finite-pulse sources:

| Field | Meaning |
| --- | --- |
| `id`, `source_kind` | Configured source identity/kind |
| `target_step_s` | Raw period/32 for sine, or shortest rise/fall divided by four |
| `recommended_step_s` | This source's supported contribution, bounded by the current interval and 1 µs floor |
| `minimum_step_s` | Supported output interval floor |
| `unresolved` | Current interval exceeds the raw target |
| `minimum_limited` | Raw target is below the floor |
| `can_fit` | This source can improve the current interval |

Constant waveforms are omitted. Advice uses configured live-graph values, including while paused/stale or incomplete; it does not describe a recorded schematic or held source's measured waveform. Other sources and LC advice may require an even finer supported interval. The existing shared command remains:

```json
{"command":"simulation.sampling.fit","params":{}}
```

Its guards, retries, undo, staged batch and restart/stale behavior remain. At the floor, repeating an identical fit preserves physics/history/dirty state. Native actions use the same command and guard captured context against intervening agent edits. Focus continues at the output-interval field when available. Source warnings, diagnostic guidance and structured advice all preserve the unresolved minimum. With a recorded viewer open, the Inspector explicitly says this changes live sampling; existing captures keep their original samples.

Advice is not authoritative saved project data. Saved settings/source parameters recompute it on fresh reopen, preserving the kept recording and starting new live physics. The [public workflow](../scripts/source-sampling-demo.py) checks a 1 kΩ /1 nF RC circuit driven by a 1 µs ramp against the exact first-order response, retains both stored sides of the continuous ramp corner, checks minimum/no-op behavior, freshly reopens the project, and verifies a 4 µs ramp meets the policy at the floor without changing earlier recorded bytes.

[Validation evidence](validation/source-sampling-validation.json) includes the failing pre-repair test, controlled unit/native checks, complete public reports and source/binary identities. This is bounded source-sampling guidance; complete physical input/accessibility, visible latency, arbitrary circuit/hardware and release gates remain open.
