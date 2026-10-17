# Recovery reads, restore and dismissal contracts

Filtered `capabilities.list` exposes `recovery-choice-and-query` details for `recovery.inspect`, `recovery.samples`, both Restore commands and both Dismiss commands. Users and agents invoke the same service. The annotations describe existing behavior; ordinary catalog/output definitions, command IDs, device models, work limits and file formats are unchanged.

## Exact choices and pure reads

Use an observed `id` and exact case-sensitive `sha256`, independently of current selection. A cached list is an observation, not proof of current bytes or deletion permission. These legacy commands accept hash strings without the stricter observable-read job's shape check. Restore/read validate the capsule before comparing its digest; invalid/missing files can therefore report `invalid-recovery` before `file-conflict`.

`recovery.inspect` and `recovery.samples` capture the store and read/validate/project outside the editor lock. They publish no jobs, revisions, events, history, catalog changes or live results. Later editing, New or store changes do not supersede an already captured query. Inspection reports its captured `directory`. Caller cancellation, a cooperative 30-second bound and Close govern delivery; OS calls can outlast cancellation. Four reader slots are shared with observable recovery-read jobs until their workers actually return.

Inspection returns the full validated graph/settings/model and embedded Verify metadata, not recorded rows. Its shared input type accepts `discard_changes` and `expected_document_generation` as ignored compatibility fields. They do not authorize or guard a replacement because inspection performs none.

Sample requests require a zero-based embedded Verify `recording_index`. Saved DC sweeps are not this list. Omitted/null/zero `offset` starts at zero; omitted/null/zero `limit` uses 100, otherwise 1–200. Offset equal to total returns empty EOF. Pages retain exact probes/rows, including duplicate pulse-edge times, and use the capsule ID as page ID. File/hash errors precede recording/page errors. Returned data is independently owned and does not become live or selected waveform data.

## Restore a copy

`recovery.restore.start` returns queued work; `recovery.restore` waits for the same job. After structural/common checks, admission checks shared work capacity, optional document generation, unsaved-change permission, then configured storage. File/ID/hash validation is worker work. `discard_changes:true` authorizes replacement of the document captured at admission, not later edits.

Active or unknown foreign ownership permits copy restoration. A successful restore installs complete validated contents as a dirty document with no saved path and `recovered_from` provenance. It advances generation/run, uses the saved pause state and restarts physical state/time. Kept recordings retain exact captured data. Await `simulation.wait` for fresh startup after the project job completes. Explicit original files and the recovery capsule remain untouched.

Graph/transient settings, pinned/display/example preferences, kept membership, new Verify/sweep admission, input actuation, document/file association and store changes can supersede pending installation. Session navigation/probe/cursor changes need not do so; restored captured view/probes replace them. Content changes add at most one undo entry attributed to the admitting actor/revision. Restoring identical graph/preferences can still create a new generation without another undo entry.

## Dismiss an exact file

`recovery.dismiss.start` returns queued work; `recovery.dismiss` waits. A configured store and syntactically valid `rec-` ID are required. Catalog membership and a clean current document are not required. The compatibility `discard_changes` field is ignored; dismissal preserves current work. An optional generation guard applies only at admission, and later editing/New need not invalidate directory-scoped removal.

Preparation and commit check exact raw bytes/hash, regular-file identity and parseable foreign ownership. Active/unverifiable foreign ownership refuses removal. Inactive/current ownership permits it. A damaged file may be explicitly dismissed by exact hash; payloads without a parseable owner are not treated as valid owned capsules. The checks serialize cooperating store writers and recheck before unlink, without claiming a hostile-process atomic conditional unlink.

`dismissing-recovery` sets `cancelable:false`; cancellation then returns `operation-committing`. `result_recovery_id` is the requested ID even at admission. `recovery_snapshot_removed` is true only after confirmed removal. Follow-up scan failure leaves removal completed with `cleanup_warning`; `recovery_catalog_applied:false` does not mean the file was preserved. Newer catalog entries and later cleared selection survive; only the removed ID is filtered when a full refresh cannot apply.

Both mutation families share eight retained project jobs/eight actual workers, stable inspection/cancellation and retry ownership. A failed admitted request consumes its key; repair requires a new key. Immediate retries retain admission, waiting retries retain terminal outcome, and waiting caller disconnection does not cancel admitted work.

## Evidence and limits

[Source-frozen qualification](validation/recovery-choice-validation.json) includes controlled race tests and fresh native/headless public workflows. Each workflow exercises six contracts, eleven pure-query refusals, five admission refusals, five failed jobs and four successful Restore/Dismiss jobs. A real foreign process supplies active ownership; an owned damaged lease supplies unknown ownership; graceful owner close supplies inactive ownership. Exact record export and fresh saved-copy reopen preserve the captured data and both original schematics.

Run `python3 scripts/recovery-choice-contracts-demo.py --output REPORT --artifacts DIRECTORY`; add `--connect ENDPOINT` only for a fresh owned native session. Full physical input/accessibility, arbitrary storage/hardware, visible latency and release acceptance remain separate gates.
