# Autosave and recovery

The native app enables autosave by default, every two seconds, in a private directory returned by Go's `os.UserConfigDir()/ELab/recovery` (macOS: `~/Library/Application Support/ELab/recovery`). The Inspector shows interval, enable/disable, checkpoint, refresh, candidate selection, restore and dismiss controls. An inactive-session candidate produces a recovery notice without interrupting drawing.

`--recovery-directory /absolute/private/path` chooses another directory. `--no-recovery` skips native startup configuration. Native smoke tests use a temporary directory. Headless sessions remain filesystem-free by default; opt in with `elab-agent --recovery-directory /absolute/private/path` or use the same `recovery.configure` command during a session.

Snapshots contain the complete portable schematic (graph, supported models, simulation settings, view and selected probes), source-path context and integrity metadata. They include kept recorded results and saved pause/viewer preferences (format 4), and exclude live solver state and editor history. A dirty project is captured automatically; an unchanged document is deduplicated. Autosave changes presentation revision when status/list changes, without changing the command revision, live circuit, clock, undo history or explicit save point. View/probe changes are captured while the project is already dirty; by themselves they do not change the existing document dirty policy.

The implementation writes and syncs a temporary file, publishes a new immutable snapshot atomically, then keeps the latest two snapshots it owns for each document. It never overwrites an explicit `.elab` file. Saving successfully retires that session's recovery snapshots for the current document. Cleanup failure is reported separately and does not turn a completed explicit save into failure.

On reopen, scanning validates the whole capsule, embedded document checksum, supported schematic version, values, models and terminals. Only the newest valid snapshot per document is offered; a damaged newer snapshot leaves an older complete copy available. Corrupt entries are reported, never silently restored. A restore requires the listed file SHA-256, protects unsaved changes, supports `expected_document_generation`, and restarts live simulation from the restored startup settings. The restored project is a dirty, unassociated copy with `document.recovered_from` context. Save As is required to choose a destination.

Public commands (also listed with schemas by `capabilities.list`):

| Command | Parameters / behavior |
| --- | --- |
| `recovery.configure` / `recovery.configure.start` | Wait for setup / return immediate admission. `directory`, `enabled`, `interval_s` (1–60 s, unit text accepted); directory must be real/private (0700). Inspect/cancel through shared project-work commands. |
| `recovery.scan` | Empty object; refresh validated candidates and problems |
| `recovery.checkpoint` | Empty object; capture now even if automatic snapshots are disabled; clean projects retire owned snapshots |
| `recovery.select` | `id`; empty string clears selection |
| `recovery.inspect` | `id`, `sha256`; query returns validated candidate, schematic configuration and recording metadata |
| `recovery.restore` | `id`, `sha256`, optional `discard_changes`, `expected_document_generation` |
| `recovery.dismiss` / `recovery.dismiss.start` | Wait for dismissal / return immediate admission. Exact `id`, `sha256`, optional `expected_document_generation`; observe/cancel through project work before removal begins. |

`recovery.samples {id,sha256,recording_index,offset?,limit?}` reads embedded exact rows in bounded pages (default 100, max 200).

All mutations require the normal revision and idempotency key. Failed validation leaves the editor intact. Normal internal shutdown checkpoints dirty work when enabled and releases its lease. Explicit `app.close {"discard_changes":true}` retires the current document's owned snapshots. Store shutdown serializes with pending writes so a late worker cannot recreate discarded data. Other documents and other sessions' snapshots remain intact.

Active-session leases identify ownership. Restoring an active session's snapshot as a copy is allowed; deleting another live session's data is refused. Each session uses a random namespace and only prunes files it wrote. Process liveness is qualified on macOS/Unix; the Windows process-liveness handling conservatively marks another present lease unknown/protected, pending platform qualification. PID reuse can conservatively postpone dismissal; recovery remains available.

Bounds: 128 directory entries, at most two retained snapshots per session/document, at most 8 MiB plus 16 KiB per recovery capsule, bounded diagnostics and unchanged project component/wire limits. A full directory produces an explicit error; ELab does not delete another session's data to free space. Move older snapshots/abandoned leases out of the directory before scanning if this bound is exceeded. Power-loss durability depends on the host filesystem; crash recovery is tested with a killed process, not a power-loss experiment.

Validation: core race tests cover automatic capture, retention, copy restoration, unchanged explicit files, checksum damage/fallback, stale hashes, generation and dirty guards, active ownership, failed configuration and late-worker rejection after save/discard/shutdown. Native control tests compare recovered graph/document state with external-agent commands. `scripts/recovery-demo.py` tests actual process-crash recovery through the public JSON interface. Native GPU/UI smoke includes checkpoint and restore controls.

Pinned-reading presentation edits participate in document dirty state and recovery captures, including a document whose graph and recordings are otherwise clean. Capsules retain IDs/positions, never trusted measurement values; recovery reconstructs fresh physics and remains an unsaved copy.

Save cleanup now runs outside the editor lock. Captures carry a private state-order stamp in addition to admitted command revision, preventing late clean retirement from erasing newer dirty background commits at the same command revision. On-disk recovery format is unchanged. [Captured Save behavior](project-save.md).

Manual Refresh/Checkpoint now use [background recovery work](project-recovery-work.md). Native callbacks return admission; existing agent commands retain waiting behavior. Captured source/capsule identity, cancellation and newer catalog/selection protections are observable through the shared project-job interface. Configuration now also uses the shared background worker; Restore has its own background path. Inspection queries now execute outside the editor mutex; explicit read-job aliases and serialized shutdown remain separate work.


## Background recovery setup

Native enable/disable and interval controls submit `recovery.configure.start`. Directory creation, private lease creation, scanning and old-lease cleanup run outside the editor mutex and native owner callback. `recovery.configure` remains the waiting compatibility command; startup helpers may use it before opening the window. Directory text is nonblank/no NUL and at most 4096 UTF-8 bytes. Both command inputs share a local schema with the time-text grammar; recovery state and saved output remain numeric seconds. Filter `capabilities.list` by either command for bounds/examples.

The shared `project_work` reports queued admission, phases, actor/input identity, cancellation and terminal errors. `recovery_configuration_applied` is initially false and becomes true only after atomic installation. Existing recovery remains active during preparation. Failure, cancellation or Close before installation preserves the previous configuration and cleans only a newly staged lease; a newly created empty directory may remain. A new setup does not delete older recovery capsules or explicit saved files.

On success, a same-directory setup reuses its lease and preserves newer catalog/selection state. A different directory replaces the active recovery store only when validated; the old loop is canceled and its lease is closed outside the editor lock. The `closing-previous-recovery` phase is non-cancelable (`operation-committing`): installation has already happened. A lease-cleanup failure is a `cleanup_warning`, not an undo of installed settings. `recovery_catalog_applied` reports list publication separately from configuration installation. The result reports the document generation present at installation; recovery is session state, so editing or New does not invalidate preparation for the same previous store.

[Disabling the already configured canonical path](recovery-disable.md) is a settings-only exception: it does not open/scan that folder or wait for its writer lock. It preserves last-known error/catalog/selection, reports configuration applied with `recovery_catalog_applied:false`, and fences late automatic status. An already captured checkpoint may finish under normal retention. The UI identifies the retained failure as a last error while autosave is off. New/different-store setup and enable still perform ordinary preparation; restoring storage while disabled does not restart autosave until Enable is requested.

Setup shares the eight file-worker/retained-job bounds and excludes other file operations until completion/cancellation. A waiting request may disconnect while accepted work continues; retrying its original key returns that operation's outcome. Immediate-start retries retain original admission, so inspect the operation for current status. The cooperative 30-second budget applies before installation; OS calls and committed lease cleanup can finish later. Service Close joins setup workers and cleanup. Inspection queries run outside the editor mutex; observable read jobs and [background shutdown](shutdown.md) provide separate operation lifecycles.

The [public setup workflow](../scripts/recovery-configuration-demo.py) runs through stdio or an owned native endpoint. Controlled tests hold setup and old-lease cleanup while edits/steps proceed, and cover staged-lease cancellation/Close, cleanup warnings, retry/disconnect, document replacement and newer catalog preservation. Directory privacy/entry limits, snapshot formats, retention and model validation are unchanged.


## Background recovery dismissal

Native valid/damaged snapshot buttons use `recovery.dismiss.start`; the existing `recovery.dismiss` command waits for the same worker. Reading, hash/ownership validation, commit checks, removal and follow-up scanning run outside the editor mutex. Preparation remains cancellable through `project.work.cancel`. The `dismissing-recovery` phase fences removal with `cancelable:false`; a later cancel returns `operation-committing`. The shared 30-second context is cooperative before that boundary; OS calls and committed work can complete later.

`project_work.result_recovery_id` identifies the requested snapshot. `recovery_snapshot_removed` starts false and becomes true only when removal actually succeeds; `source_sha256` reports its confirmed digest on success. Waiting disconnection/retry and immediate-start admission retention follow existing file-job rules. A successful removal followed by a scan error stays completed with a warning/recovery error, so it cannot be mistaken for a preserved file. An error before removal leaves the removal flag false.

The worker validates the requested hash and active owner using the same captured bytes, then rechecks content, regular-file identity and active-session ownership before unlinking. This store's checkpoint/pruning paths are serialized with commit; the editor lock remains free. Symlinks, replaced/changed files and newly active other-session ownership refuse. Damaged capsules may still be explicitly dismissed by their listed hash. These checks protect ordinary cooperating application/filesystem use; they do not claim an atomic conditional unlink against a hostile process changing paths during the final OS call.

A later recovery publication or explicit cleared selection is preserved. A removed ID is filtered out of a newer list without replacing its other entries. Dismissal is directory-scoped: editing/New does not invalidate accepted work for the same store. An optional input document-generation guard applies at admission; project state, live physics/history, other snapshots and explicit saved schematics are preserved. Close before commit cancels the dismissal; shutdown finalization now runs outside the editor mutex with an observable lifecycle; complete platform shutdown remains a separate qualification task.

The [public dismissal client](../scripts/recovery-dismiss-demo.py) uses isolated fixtures and a real separate-process lease owner. It proves active-owner refusal, inactive/damaged removal, exact result ID/digest and unchanged saved/live state through headless or native endpoints. Controlled tests cover preparation/Close cancellation, changed bytes/inode/symlink and late owner guards, blocked commit responsiveness, late catalog/scan failure, missing worker results and retry/New interleavings. Native callback tests and packaged GPU smoke activate the same Dismiss control.


## Pure recovery reads outside the editor lock

`recovery.inspect` and `recovery.samples` retain their query-only contracts. They capture the configured store and requested ID/hash under the editor mutex, then read, validate models/checksums, inspect the lease and build the result outside it. No public state, command revision, history, physics, catalog or file bytes are changed by a read. Inspection now includes `directory`, identifying the captured recovery directory even if another command reconfigures recovery while the read is in progress. A completed read is an immutable capture by exact hash, not a claim that the source file still exists afterward.

At most four recovery reads run concurrently per service, shared between these compatibility queries and observable jobs. Additional reads return `busy`; slots release after success, error, timeout or cancellation. A caller's context and a cooperative 30-second bound govern the read. OS reads are not forcibly interrupted; cancellation is checked before/after validation and result construction. Close cancels and joins admitted readers. New filesystem reads on a closed service return `unavailable`; open a new session to inspect retained data. Final recovery I/O is serialized outside the editor mutex; see [shutdown](shutdown.md).

These compatibility queries remain pure and do not create operation jobs or progress publications. The observable aliases below provide job IDs without changing these compatibility contracts. Serialized shutdown and broader job coverage remain open under AGENT-04. Existing sample pages remain exact, default 100/max 200, with copied probes/rows. Results retain the original embedded recording even after live values/time change.

The [public query workflow](../scripts/recovery-query-demo.py) compares inspection metadata and sample rows against the original Verify record, verifies later live edits are not substituted, and checks both successful/refused query purity plus saved/capsule file hashes. Core blocked-reader tests cover editor/step availability, capacity, caller cancellation, Close joining, timeout cleanup, reconfiguration identity and owned responses.


## Observable recovery inspection and recording pages

Choose a recovery snapshot and **Inspect saved copy** to read its captured component values and embedded recording list. **Read a recording…**, **Previous rows** and **Next rows** read exact stored rows in pages of 20. Reads show progress and **Cancel read** independently of Save/Open work. Inspecting preserves the open circuit, simulation time, history and recovery selection; it does not restore or select a recording. The native summary displays at most 40 components, while the public result contains the full validated graph.

Users and agents share five commands:

| Command | Behavior |
| --- | --- |
| `recovery.read.start` | Required `kind` (`inspection` or `samples`), candidate `id` and exact `sha256`; captures the configured directory. Returns admission with a stable job in `snapshot.recovery.reads`. |
| `recovery.read.inspect {id}` | Pure in-memory metadata: status/phase, actor, input revisions/generation, captured input/directory, document bytes, result size/digest and error. |
| `recovery.read.cancel {id}` | Requests cooperative cancellation of that retained active job; inspect until terminal. |
| `recovery.read.result {id}` | Owned completed typed result (`inspection` or `samples`); no disk read. Decode runs outside the editor lock. |
| `recovery.read.chunk {id,offset?,limit?}` | Owned base64 JSON bytes for the same complete result, with offset/next/total and whole-result SHA-256. Default/max chunk 65,536 bytes; offset may equal total. |

Sample starts require `recording_index >= 0`; `offset >= 0`, default/zero `limit=100`, maximum 200. Inspection forbids those three keys, including explicit zero/null. The exact candidate hash is checked against the read bytes. A changed hash fails with `file-conflict`; invalid/missing files fail with `invalid-recovery`. Recording indices and page bounds are validated by the worker. Detailed discovery publishes bounds, conditions, errors and completion behavior for all five commands.

Four active readers are shared with legacy pure queries. A fifth returns `busy`; a canceled worker keeps its slot until it returns. Eight jobs are retained, evicting the oldest terminal job when necessary. Edits, New and directory reconfiguration do not supersede admitted reads: results retain their original directory/ID/hash and never overwrite newer catalog/selection state. Caller disconnect does not cancel accepted work. Explicit cancellation, cooperative 30-second deadline and Close are observed before publication; no partial result is exposed. Close joins workers. OS I/O is not forcibly interrupted. [Background shutdown](shutdown.md) joins readers and finalizes recovery outside the editor mutex.

Snapshots contain metadata only. Immutable result JSON is capped at 16 MiB; larger results fail with `result-limit`. Structured delivery is capped at 1 MiB minus 1,024 envelope bytes; `result-too-large` directs clients to chunks. Consumers should finish/reassemble retained chunks before admitting enough other jobs to evict their source. Queries return `not-found` after eviction and `not-ready` without a complete successful result. Chunked and structured responses own their data and change no revisions. Native result decoding runs off the UI owner; a cache is displayed only for the latest matching candidate hash/directory/job.

The [public read-job workflow](../scripts/recovery-read-demo.py) checks captured RC values and exact original Verify rows through both public transports, compares legacy queries, reassembles digest-checked chunks, preserves saved/capsule bytes and tests result retention after reconfiguration. Controlled core tests additionally cover blocked-read editing/Save, shared capacity, cancellation/deadline/Close, in-flight reconfiguration, eviction and ownership. Native tests and packaged smoke use the same Inspect and recording controls. These checks do not establish complete physical typing/accessibility or general latency acceptance.

Recovery candidates and inspection/read results expose [ownership status](recovery-ownership.md) as `active`, `inactive` or `unknown`. A present but unverifiable foreign lease keeps the capsule protected; `active_session:true` includes this conservative unknown case. Native controls distinguish it from confirmed activity, allow restoring a copy and disable dismissal. Preparation and commit recheck the same guard; repairing/closing the owner can restore a verifiable state. Source/capsule formats and explicit damaged-file handling remain unchanged.

The [repeated automatic recovery protocol](recovery-timing.md) passes 240 paused/advancing native/headless samples and 20 exact process-crash restores on the recorded pilot workstation. Maximum dispatch-to-validated-capsule completion is 2.544445 s with the ordinary two-second schedule. This named-fixture timing result leaves general storage/hardware, maximum recording storage, automatic failure-display and broader release qualification separate.

The subsequent [automatic failure/retry repair](recovery-failure.md) keeps storage errors and completed retry metadata observable across drawing edits. Newer recovery work and manual-save cleanup still fence stale observations, so a retired capsule is not re-listed after an intervening edit. The public owned-folder interruption workflow checks complete recorded RC retry and unchanged manual/source recovery bytes through both interfaces.
