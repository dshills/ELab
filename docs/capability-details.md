# Detailed command discovery

Use `capabilities.list {}` for the normal command catalog. To inspect one command's input without inflating that catalog, use:

```json
{"version":1,"id":"pulse-contract","command":"capabilities.list","params":{"command":"source.pulse.set"}}
```

The response keeps the ordinary discovery envelope, schema registry, protocol/schema versions, output schema and mutation/retry descriptions. Its `capabilities` array contains exactly the requested command, with a fully expanded root input schema. An omitted or empty filter lists every command. Unknown IDs return `not-found`; malformed, null, duplicate, extra or oversized filter fields return `invalid-params`. Filters are limited to 128 UTF-8 bytes. Queries do not change schematic/history, physics, document state or command revision and require no mutation key.

Detailed bounds/examples currently cover `source.sine.set`, `source.pulse.set`, `simulation.transient.set`, `potentiometer.wiper.set` and `example.load`, plus PNG resolution, recovery setup aliases and the sweep command family described below. Other commands return their existing type schemas with the root local reference expanded. Outputs and saved files remain numeric SI; command-only time text is not added to output schemas. The ordinary catalog remains within its original 65536-byte limit.

Numeric bounds use JSON Schema `minimum` / `maximum` with `x-unit`. For time fields, the existing local time-grammar reference still accepts numbers or unit text. `x-normalized-si-range` applies the same inclusive limits **after** the text is converted to seconds. Standard JSON Schema numeric keywords do not validate the magnitude of a string such as `"60 ms"`; clients must normalize it before checking this extension. Schema-only numeric checks on time references require a dialect supporting `$ref` siblings (2019-09 or later), or explicit evaluation of the normalized-SI range. Server validation remains authoritative.

`x-normalized-si-relations` is an array of structured relationships, evaluated after normalization. Fields are dot paths into the input:

| Operator | Meaning |
|---|---|
| `less_or_equal` | First field ≤ second field |
| `sum_minimum` | Sum of fields ≥ `limit` |
| `sum_absolute_maximum` | Sum of absolute field values ≤ `limit` |
| `ratio_maximum` | First field divided by second field ≤ `limit` |

An optional `when_component_kind` applies a relationship only if the current graph contains that kind. The legacy pulse high/low settings must each cover a sample interval when `pulse-source` is present. Independent field bounds still apply when the relationship is conditional. All denominators in published ratio rules have strictly positive lower bounds.

Sine amplitude/offset must fit ±12 V; frequency is 0.1–1000 Hz and phase ±360 degrees. Finite pulses require ordered levels within ±12 V, 1 µs–10 s phases/transitions, transitions inside their phases and a total period of at least 1 ms. Transient playback/sampling also obeys the 1000 samples per wall-second ratio. Example loading advertises IDs directly from the immutable template catalog. Parameter examples use illustrative IDs such as V1 or P1; place the appropriate component in a fresh session or replace that ID with an existing one. Cleared/deleted session IDs are not reusable.

The original waveform/transient/wiper/example `x-details-scope` limits those annotations to parameter bounds and examples. Their stateful admission/error coverage, other command families' semantic limits and exhaustive native capability coverage remain unfinished. Sweep-specific state contracts are described below. This addition improves agent discovery; it does not close the full agent-parity or accessibility release gates.

The [headless workflow](validation/headless-capability-details.json), [packaged workflow](validation/desktop-capability-details.json) and [release receipt](validation/capability-details-validation.json) record compatibility, paused query purity, examples, unit text and refusal preservation. Unit/race checks exercise all command filters and caller ownership, every sine/pulse numeric boundary, coupled pulse restrictions and the original catalog budget. Run `python3 scripts/capability-details-demo.py --output REPORT`; `--connect ENDPOINT` replaces only a designated owned validation session.

## Sweep command contracts

Filtered discovery for every `sweep.*` command includes `x-command-contract` version 1: preconditions, effects, and known command-specific errors with a `phase` (`admission`, `operation`, `work`, or `admission-or-publication`) and `when` condition. Computation failures are inspected on the sweep operation; file-job errors are inspected on `project_work`. `completion` distinguishes immediate admission from waiting file work, gives observation/cancellation commands, and publishes the cooperative 30-second budget. Successful admission does not promise successful computation or file publication. `simulation.wait` does not wait for sweeps.

Common request/revision/retry rules remain in the discovery envelope. Contract scope explicitly excludes exhaustive OS/filesystem errors and other command families. These are descriptions of validation and behavior, not a separate permission system or a promise that a request is currently admissible. Snapshot state and runtime validation remain authoritative.

`sweep.settings.set` and the non-null `sweep.start.settings` branch expose the same settings schema. The latter may be omitted or null to use the current draft. `x-parameter-domains` gives the parameter name, SI unit and inclusive normalized bounds: voltage −12–12 V, resistance 10–100000 Ω, wiper 0–1. These domains also drive the target inventory. `x-inventory` identifies the current target/probe lists. Endpoints are floating-point text, optionally suffixed with V or Ω/ohm; resistance accepts k/K/M. Wiper is unitless. Endpoint byte bounds use `x-max-utf8-bytes`; `maxLength` alone counts Unicode characters, not bytes.

`x-relations` adds required semantics after parameter-unit normalization: `different` rejects equal endpoints; `positive-if-logarithmic` requires positive endpoints for logarithmic spacing; `distinct-representable-axis` requires every inclusive point to be distinct and monotonic in the selected direction. Optional `spacing` chooses linear (default) or logarithmic equal-ratio points. Descending ranges are supported. Point count is 2–201. Omitted/null probe IDs select automatic target readings; explicit arrays require 1–6 distinct current probe IDs, and empty arrays are refused. Standard JSON Schema cannot validate normalized magnitudes or floating-point representability of text, so clients must apply these annotations or defer to runtime validation.

Cursor `index` is required and is either a zero-based integer below the selected curve's point count or explicit null to clear. Its integer schema publishes the dynamic upper bound in `x-maximum-exclusive`; it is not a JSON Schema numeric keyword. An omitted index remains invalid, and an optional mismatched operation ID refuses even a clear. Native **Clear point** uses this same command.

Import details publish format/engine/version and the 8 MiB source-file limit; import/export paths are nonblank local text without NUL, up to 4096 UTF-8 bytes. Relative paths resolve against the process working directory. Imports and runs share eight retained curve slots, checked at their respective admission/publication points. Imported data does not claim fresh live-source association. Exports require complete captured data and explicit overwrite; only JSON/CSV are accepted.

The [public contract workflow](../scripts/sweep-contracts-demo.py) fetches the sweep family, executes discovered LED examples, exercises limit refusals, clears the point with explicit null, and verifies live/captured state preservation through headless or native sockets. Core regressions additionally check every published parameter endpoint against real admission on LED and loaded potentiometer graphs, descending ranges, count/probe restrictions and command error outcomes. Native regressions/GPU smoke exercise the repaired Clear point control. This expands AGENT-02 evidence without declaring all command discovery, mounted input or accessibility gates complete.

Recovery setup details expand both `recovery.configure` and `recovery.configure.start` from the same input-only local schema, preserving `interval_s` unit text, 1–60 s normalized limits and directory byte bounds. They do not widen numeric recovery-state outputs. See [background setup](recovery.md#background-recovery-setup) for job admission, installation, cleanup and retry semantics.
