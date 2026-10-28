# Bounded analysis reads

Source development adds `analysis.list` and `analysis.page` through the same
Service.Dispatch used by the UI and agents. ELab 0.200.1, build 279, predates them.
Existing full-result queries, snapshots and exports retain their behavior.

`analysis.list {}` returns current `revision`, `command_revision`,
`document_generation`, and `analyses`. Each entry identifies its `kind`, `id`,
`status`, row count and column descriptors. Kinds are `sweep`, `recording`,
`spectrum` and `tolerance`. Noise sweeps also describe contributor IDs, source
units and integrated RMS budgets in the same order as each point's PSD matrix.
No graph, solver result, source bias or raw rows are included in this listing.

Use those identities and generation to request a page:

```json
{"version":1,"id":"page-1","command":"analysis.page","params":{"kind":"sweep","id":"sweep-1","document_generation":2,"offset":0,"limit":25}}
```

Omitted/null/zero limit uses 25; explicit limits are 1–50. Offset starts at zero
and may equal the row count for an empty terminal page. Follow `next_offset`
until `done` is true. The page's `analysis` repeats the compact column/contributor
header. Exactly one data family is used:

| Kind | Page field | Meaning |
| --- | --- | --- |
| sweep | points | Exact existing SweepPoint values, AC complex responses and noise contribution PSDs |
| recording | samples | Exact time/value rows, preserving probe order and repeated edge timestamps |
| spectrum | bins | Frequency in Hz, peak in the probe's unit and dB relative to one probe unit |
| tolerance | trials | Seeded variation parameters, value or explicit failure for each trial |

These are raw stored units. In an AC sweep, selected voltage outputs produce
paired gain/phase columns; noise density stays in V/√Hz even when UI labels use
nV/√Hz. Full provenance and analysis assumptions remain available through the
existing inspect/export commands.

Queries return detached copies and do not alter live state, history, cursors,
selection or dirty flags. `document_generation` is required in page parameters;
replacement returns `conflict` and requires a fresh listing. Retained completed
records are immutable within that document context. Unknown, evicted or replaced
IDs return `not-found`; known unfinished/failed analyses without a record return
`not-ready`. Invalid kind/range/shape returns `invalid-params`. Offset arithmetic
is bounded before addition.

The listing also supplies compact admission guards for subsequent mutations:
use its command_revision and document_generation as the normal request envelope
expected_command_revision / expected_document_generation, and an idempotency key.
Clock publication can continue; intervening commands still produce ordinary
admission conflicts. Paging does not bypass any guard or add a second command path.
