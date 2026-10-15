# Compare captured experiments

Use **Verify** to capture an experiment, change a component value and Verify again. Choose **Show recorded**, then use **Compare captures → Reference recording B** in the Inspector. A is the selected recording; B is the chosen reference. Enter **Compare at** as a time such as `10 ms`. Readings show A, B and **B − A** in the probe's units. **Whole-run statistics** expands each reading's range, sample mean/RMS and sample count. The checkbox is a shared session preference, available to agents too.

For example, capture the RC example with R1 at 10 kΩ, change it to 20 kΩ, and capture again in Charge mode. With the second run selected as A and the first as B, capacitor voltage at 10 ms is lower in A. For the default 1 µF capacitor and 5 V source, the analytic values are approximately 1.967 V in A and 3.161 V in B. Comparing two captures does not run or change the live circuit.

## Measurement meaning

The comparison uses each recording's nearest **actual saved row**, without resampling or interpolation. Exact distance ties select the later sample; duplicate timestamps select the latest/right-hand row. Requested time, actual times, sample indices, offsets from the request, sample step and captured durations are returned separately. Differently sampled runs may yield different actual times. B − A is the difference between those explicitly identified readings, not a promise of simultaneous values.

Whole-record statistics use every exact saved row in each complete capture, including duplicate edge rows. They do not follow the visible scope window or cursor interval. Sample mean/RMS depend on sample distribution, rate and duration; they are not time-weighted integrals. Comparing unequal record lengths therefore needs that context.

Only probes with the same ID, SI unit, positive terminal and reference terminal match. Display labels may differ. Unmatched probes are reported on both sides; no common readings yields an unavailable comparison with a reason. Matching terminal identities does not assert matching wiring, device models or initial/input settings. Inspect each capture with `operation.inspect` and the existing Captured source inspector when interpreting a difference. Local IDs and original capture IDs, analysis and engine identify the two data sources. No current live readings are substituted, even after later schematic edits.

DC operating-point captures compare at time zero. Transient times must be within **both** captured durations; requests outside that overlap refuse rather than clamp, extrapolate or change the previous settings. Curves from DC parameter sweeps are a different data type and do not enter this time-record comparison.

## Shared commands

| Command | Inputs and result |
| --- | --- |
| `recording.compare` | Pure query. Required `id` (A), `reference_id` (B), `time_s` (finite numeric seconds or unit text). IDs must name distinct retained completed time/DC recordings. Returns exact sample provenance, matched readings/deltas, separate whole-record statistics and unmatched probes. No key/revision guard needed. |
| `recording.compare.set` | Same required fields and ordinary mutation key/revision guard. `id` must be the selected capture. Configures the shared Inspector comparison; optional `show_statistics` controls the statistics display. Empty `reference_id` clears it. Returns the normal snapshot with `recording.comparison`. |
| `capabilities.list` | Filter either command to discover time bounds, inventory, state/identity rules and known refusals. |

Selection starts a fresh comparison session: selecting any operation clears the comparison. Changing the reference through the UI starts at zero, which exists in every complete capture. A reference that is later evicted remains explicitly unavailable; its old values are never silently reused. Import completion preserves a later comparison configuration through the existing viewer revision guard.

The configuration is session/view state, without undo/history, dirty membership or physics changes. Projects retain the underlying kept captures; reopening resets comparison settings and assigns fresh local IDs. Select those reopened captures to compare their unchanged data again. Exported/imported recordings use the same validated captured data. There is no project-format change.

This addition is a numerical comparison table. Plot overlays, time alignment, threshold/period extraction, relative-error percentages and formal equivalence testing remain separate features. It does not certify a circuit or prove real-device behavior.
