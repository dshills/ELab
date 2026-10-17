# Recovery configuration, work and cached selection

This checkpoint advances the requirement that users and AI agents can discover and run the same operations. Six existing commands now expose `recovery-controls-and-work` details through filtered `capabilities.list`: configuration and its immediate alias, manual work start, waiting scan/checkpoint, and candidate selection. No new application command or runtime/model/file-format change is introduced.

## Behavior made explicit

Automatic scheduling and manual work are separate. A disabled store still accepts manual scan/checkpoint. Creating a different store opens/leases/scans it even when disabled; disabling the current normalized path changes scheduling without accessing that folder. It preserves last-known catalog, error and selection and reports configuration applied/catalog not applied. Physical parent aliases are not normalized into one store identity.

Candidate selection uses the cached valid catalog without reading disk, granting ownership permissions, restoring data or admitting work. Empty selection remains cleared across scans and same-store refreshes; a different store resets that intent. Restore and dismissal retain separate ID/hash and ownership checks.

A completed checkpoint can create a capsule, deduplicate unchanged content, skip an obsolete capture, or retire owned recovery files for clean contents. Only a newly created capsule has a result ID/time. Historical `last_snapshot` and `last_saved_at` can still identify a retired file. Directory limits count ownership leases and unrelated entries too.

The contracts describe captured inputs, retained job IDs, validation and worker bounds, cancellation before publication, fenced publication/cleanup, later editing/replacement, newer catalog ownership and retry outcomes. A failed admitted request consumes its key; repaired storage needs a new key to perform another read.

## Validation

[The qualification receipt](../docs/validation/recovery-controls-validation.json) records the exact source inputs, process outcomes, artifact hashes, separate unpromoted candidate and preserved bundle/SDK identities. Dedicated core tests verify query-owned discovery, unchanged ordinary schemas, no-storage selection and disabled manual checkpoint/dedup/clean retirement. The full gate includes formatting/imports/Staticcheck, 37 Python tests, root/native Go race tests and vet.

Fresh owned headless and native workflows each exercise all six contracts, eight pure admission refusals and ten terminal jobs, including a failed scan, replay after repair and a corrected new-key scan. Both preserve explicit manual file bytes, a complete four-row RC recording and accepted paused physics. Exact capsule bytes and its embedded document checksum are retained and independently reviewed before clean retirement. Independent RC references check voltage/current rows and the retained live sample. Actual public actors remain `agent.stdio` and `agent.socket`.

Initial fixture assertions crossed two intentional component edits: these advance the requested run and mark accepted readings stale. The corrected fixture compares complete source state at each recovery boundary and separately checks unchanged accepted samples. Three initial failures and the correction are retained; no product runtime defect was inferred or repaired.

The private candidate also passes 45 native smoke stages and the existing four-adapter suites with 27,596 numerical checks. A read-only discovery inventory finds **129 scoped contracts among 162 command IDs, leaving 33 without one**; both prior recorded-import schemas remain exactly equal.

## Remaining scope

P0 remains **17 implemented /10 partial**, and AGENT-02 remains partial. Contract annotations are not exhaustive behavior or native-accessibility qualification. Controlled core tests cover blocked OS work, cancellation/Close, newer catalog/selection, cleanup and replacement interleavings; public runs make no nondeterministic race claim. Complete physical input/accessibility/readability, event coverage, general storage/hardware, visible latency, distribution, physical printers and participant acceptance remain open. The canonical and preserved apps are unchanged; this candidate is not promoted.
