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
| `recovery.dismiss` | `id`, `sha256`; removes exactly that confirmed snapshot |

`recovery.samples {id,sha256,recording_index,offset?,limit?}` reads embedded exact rows in bounded pages (default 100, max 200).

All mutations require the normal revision and idempotency key. Failed validation leaves the editor intact. Normal internal shutdown checkpoints dirty work when enabled and releases its lease. Explicit `app.close {"discard_changes":true}` retires the current document's owned snapshots. Store shutdown serializes with pending writes so a late worker cannot recreate discarded data. Other documents and other sessions' snapshots remain intact.

Active-session leases identify ownership. Restoring an active session's snapshot as a copy is allowed; deleting another live session's data is refused. Each session uses a random namespace and only prunes files it wrote. Process liveness is qualified on macOS/Unix; the Windows placeholder conservatively treats a present positive-PID lease as active, pending platform qualification. PID reuse can conservatively postpone dismissal; recovery remains available.

Bounds: 128 directory entries, at most two retained snapshots per session/document, at most 8 MiB plus 16 KiB per recovery capsule, bounded diagnostics and unchanged project component/wire limits. A full directory produces an explicit error; ELab does not delete another session's data to free space. Move older snapshots/abandoned leases out of the directory before scanning if this bound is exceeded. Power-loss durability depends on the host filesystem; crash recovery is tested with a killed process, not a power-loss experiment.

Validation: core race tests cover automatic capture, retention, copy restoration, unchanged explicit files, checksum damage/fallback, stale hashes, generation and dirty guards, active ownership, failed configuration and late-worker rejection after save/discard/shutdown. Native control tests compare recovered graph/document state with external-agent commands. `scripts/recovery-demo.py` tests actual process-crash recovery through the public JSON interface. Native GPU/UI smoke includes checkpoint and restore controls.

Pinned-reading presentation edits participate in document dirty state and recovery captures, including a document whose graph and recordings are otherwise clean. Capsules retain IDs/positions, never trusted measurement values; recovery reconstructs fresh physics and remains an unsaved copy.

Save cleanup now runs outside the editor lock. Captures carry a private state-order stamp in addition to admitted command revision, preventing late clean retirement from erasing newer dirty background commits at the same command revision. On-disk recovery format is unchanged. [Captured Save behavior](project-save.md).

Manual Refresh/Checkpoint now use [background recovery work](project-recovery-work.md). Native callbacks return admission; existing agent commands retain waiting behavior. Captured source/capsule identity, cancellation and newer catalog/selection protections are observable through the shared project-job interface. Configuration now also uses the shared background worker; Restore has its own background path. Dismiss/inspection and serialized shutdown remain separate work.


## Background recovery setup

Native enable/disable and interval controls submit `recovery.configure.start`. Directory creation, private lease creation, scanning and old-lease cleanup run outside the editor mutex and native owner callback. `recovery.configure` remains the waiting compatibility command; startup helpers may use it before opening the window. Directory text is nonblank/no NUL and at most 4096 UTF-8 bytes. Both command inputs share a local schema with the time-text grammar; recovery state and saved output remain numeric seconds. Filter `capabilities.list` by either command for bounds/examples.

The shared `project_work` reports queued admission, phases, actor/input identity, cancellation and terminal errors. `recovery_configuration_applied` is initially false and becomes true only after atomic installation. Existing recovery remains active during preparation. Failure, cancellation or Close before installation preserves the previous configuration and cleans only a newly staged lease; a newly created empty directory may remain. A new setup does not delete older recovery capsules or explicit saved files.

On success, a same-directory setup reuses its lease and preserves newer catalog/selection state. A different directory replaces the active recovery store only when validated; the old loop is canceled and its lease is closed outside the editor lock. The `closing-previous-recovery` phase is non-cancelable (`operation-committing`): installation has already happened. A lease-cleanup failure is a `cleanup_warning`, not an undo of installed settings. `recovery_catalog_applied` reports list publication separately from configuration installation. The result reports the document generation present at installation; recovery is session state, so editing or New does not invalidate preparation for the same previous store.

Setup shares the eight file-worker/retained-job bounds and excludes other file operations until completion/cancellation. A waiting request may disconnect while accepted work continues; retrying its original key returns that operation's outcome. Immediate-start retries retain original admission, so inspect the operation for current status. The cooperative 30-second budget applies before installation; OS calls and committed lease cleanup can finish later. Service Close joins setup workers and cleanup. Existing dismiss/inspection and final shutdown filesystem paths remain separate responsiveness work.

The [public setup workflow](../scripts/recovery-configuration-demo.py) runs through stdio or an owned native endpoint. Controlled tests hold setup and old-lease cleanup while edits/steps proceed, and cover staged-lease cancellation/Close, cleanup warnings, retry/disconnect, document replacement and newer catalog preservation. Directory privacy/entry limits, snapshot formats, retention and model validation are unchanged.
