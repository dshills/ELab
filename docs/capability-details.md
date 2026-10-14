# Detailed command discovery

Use `capabilities.list {}` for the normal 112-command catalog. To inspect one command's input without inflating that catalog, use:

```json
{"version":1,"id":"pulse-contract","command":"capabilities.list","params":{"command":"source.pulse.set"}}
```

The response keeps the ordinary discovery envelope, schema registry, protocol/schema versions, output schema and mutation/retry descriptions. Its `capabilities` array contains exactly the requested command, with a fully expanded root input schema. An omitted or empty filter lists every command. Unknown IDs return `not-found`; malformed, null, duplicate, extra or oversized filter fields return `invalid-params`. Filters are limited to 128 UTF-8 bytes. Queries do not change schematic/history, physics, document state or command revision and require no mutation key.

Detailed bounds/examples currently cover `source.sine.set`, `source.pulse.set`, `simulation.transient.set`, `potentiometer.wiper.set` and `example.load`. Other commands return their existing type schemas with the root local reference expanded. Outputs and saved files remain numeric SI; command-only time text is not added to output schemas. The ordinary catalog remains within its original 65536-byte limit.

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

`x-details-scope` explicitly limits these annotations to parameter bounds and examples. Stateful admission preconditions, all failure codes, every other command's semantic limits and exhaustive native capability coverage remain unfinished. This addition improves agent discovery; it does not close the full agent-parity or accessibility release gates.

The [headless workflow](validation/headless-capability-details.json), [packaged workflow](validation/desktop-capability-details.json) and [release receipt](validation/capability-details-validation.json) record compatibility, paused query purity, examples, unit text and refusal preservation. Unit/race checks exercise all command filters and caller ownership, every sine/pulse numeric boundary, coupled pulse restrictions and the original catalog budget. Run `python3 scripts/capability-details-demo.py --output REPORT`; `--connect ENDPOINT` replaces only a designated owned validation session.
