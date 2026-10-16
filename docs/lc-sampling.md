# LC output sampling

A circuit can be integrated accurately while widely spaced scope samples miss its ringing peak. When a drawing contains both an inductor and a capacitor, **Live LC sampling** in the Inspector shows a value-based reference and offers **Fit sampling** if the current interval can be improved. The notice is available with no component selected. Hide scope to enlarge a compact workspace; Tab can reach Fit sampling and Return activates it. After activation, focus moves to the live sample-interval field.

The shared user/agent command is unchanged:

```json
{"command":"simulation.sampling.fit","params":{}}
{"command":"capabilities.list","params":{"command":"simulation.sampling.fit"}}
```

Mutations use the usual revision/document guards and retry keys. Fit takes the smallest of the current output interval, existing source recommendations and `2π√(Lmin Cmin)/32`, then applies the existing 1 µs minimum. It never coarsens a finer interval. Playback decreases only when required by the existing 1000-samples/wall-second limit. Part values, startup conditions, model equations and numerical work limits stay unchanged. The command is allowed in `edit.batch`: preceding staged values determine its recommendation, and a later invalid edit rolls back the entire batch.

Here `Lmin` and `Cmin` are the smallest positive finite configured values. The elementary LC natural-period relationship motivates the reference; **32 samples per reference period is ELab's engineering policy**, consistent with sine guidance. This is not a measured period, circuit-pole calculation or resolution guarantee. Connections, resistance, damping, active-device timing and multiple storage modes can change actual behavior. Compare finer sample intervals for demanding experiments. The [implementation plan](../research/15-lc-sampling-guidance.md) records the relationship's source and this policy's distinction from it.

`project.inspect` exposes optional `lc_sampling` data:

| Field | Meaning |
| --- | --- |
| `inductor_id`, `capacitor_id` | Parts contributing the smallest values |
| `reference_period_s` | Value-based LC period in seconds |
| `target_step_s` | Raw LC-only period/32 target |
| `recommended_step_s` | Actual supported fit, including current/source constraints |
| `minimum_step_s` | Supported 1 µs floor |
| `unresolved` | Current interval is above the raw LC target |
| `minimum_limited` | Raw LC target is below the supported floor |
| `can_fit` | Fitting can decrease the current interval |
| `note` | Interpretation and limitation text |

Advice is derived from the live graph, omitted without both valid storage types, and never saved as authoritative project data. A read preserves live state/history. At the supported floor, a faster LC scale remains explicitly unresolved and the ineffective Fit button disappears. Larger L/C values may provide a slower experiment. The warning does not promise that every circuit is solvable at the minimum step.

Fitting changed settings is one undoable electrical edit. Running experiments restart; paused experiments retain their accepted time/current/charge as stale until Restart or Resume. An identical fit preserves physics and history. Projects save the actual interval/playback and recompute advice on reopen, starting a fresh paused experiment. Existing recorded samples remain unchanged. Verify uses its configured interval; an explicit Verify override still takes precedence over the live settings.

The independent 5 V /50 Ω /10 mH /1 µF step check illustrates the issue: 1 ms samples report a 5.424 V peak, whereas the fitted 19.635 µs interval reports 7.212 V against a 7.222 V analytic peak. These are sampled maxima over a 2 ms experiment, not a general accuracy claim. Reproduce the public workflow with newly named output paths:

```sh
python3 scripts/lc-sampling-demo.py --output build/lc-report.json --artifacts build/lc-assets
# Add --connect /absolute/path/to/fresh-owned-endpoint.json for native execution.
```

The workflow checks exact retries, paused state, immutable captures, explicit Verify overrides, save/fresh-process reopen, undo/redo, staged batch rollback and the minimum-step warning. Native callback tests and packaged smoke use the same command. [Rendered evidence](../research/ui-audit/lc-sampling/README.md) separately records actual keyboard input and its pointer/accessibility limitations. Complete physical input, screen-reader, visible latency, device and participant qualification remain open.

The native implementation also uses [original-opening window references](window-lifetimes.md) for delayed work after Vellum's window API transition. This does not change the shared sampling command or saved schematic format.

Qualified local checkpoint: **0.105.0 /120** at `build/ELab.app`. The entire 0.104.0 /119 bundle is preserved at `build/ELab-0.104.0-119-preserved.app`. [Exact release, SDK and validation evidence](validation/lc-sampling-validation.json).
