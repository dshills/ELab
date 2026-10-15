# Background recovery refresh and checkpoint

`recovery.work.start {kind: scan|checkpoint}` returns immediate queued admission through the same semantic service used by the native recovery buttons. Existing `recovery.scan` and `recovery.checkpoint` wait for those workers. Shared `project_work`, stable `project.work.inspect {id}`, exact `project.work.cancel {id}` and bounded `simulation.wait` apply. Project jobs/workers retain their eight-operation bounds; other project file work refuses while a manual recovery operation is active.

## Refresh

Scan reads and strictly validates the bounded recovery directory outside the editor mutex. It preserves the latest selected/explicitly cleared candidate. A catalog revision distinguishes recovery-list publications from selection/navigation: if automatic capture or another recovery publication updates the list while scan is running, the late list is discarded. `recovery_catalog_applied:false` reports that completed work did not replace newer recovery state. Selecting a known candidate remains a normal shared operation.

Scan stays cancelable through its filesystem pass. Cancellation, Close or directory replacement prevents publication. Operating-system reads can return after cancellation/deadline, but cannot overwrite current metadata. Scan is directory-scoped and can refresh the same store after a project replacement without modifying that project.

## Checkpoint

Admission captures the committed document, graph/settings/models/preferences/kept records, viewport, dirty state and command/capture-state ordering. Private serialization runs outside the editor mutex before publication. Native callbacks return admission; assertion helpers wait only to inspect final output. Editing, navigation and stepping remain available, and later edits do not enter that captured capsule or become saved.

A publishing checkpoint writes a complete private recovery capsule, or retires owned snapshots for a captured clean document, outside the editor mutex. `cancelable:false` and `operation-committing` make the boundary explicit before writes/retirement. `result_recovery_id` and `result_recovery_saved_at` identify a newly created capsule; unchanged captures may deduplicate, and clean captures produce no new ID. Existing per-session ownership, two-snapshot retention, checksum/model validation, 128-entry/size bounds and command/capture-state ordering remain intact. Explicit saved `.elab` files are never overwritten.

Replacement before publication discards work. A publishing older capture can complete after replacement, but cannot relabel the new project's metadata; association is false and original source generation remains reported. Late metadata cannot overwrite a newer recovery catalog or selection. Failed follow-up scanning does not undo a successful checkpoint; its recovery error is separate. Captured document memory is released after completion. Capsule/project formats are unchanged.

## Failures, retries and limits

Admitted read/write failures retain job bookkeeping while preserving project/physical/history/save state. Waiting request disconnection ends the request, leaving admitted work observable. Waiting retries return the original terminal result without duplicate writes; immediate-start retries retain admission. Use a new key after correcting an admitted failure. The 30-second context supports cooperative pre-publication cancellation, while filesystem calls or publishing work may finish later. Phases are named stages, not invented percentages.

Recovery document serialization was separated from store publication so manual, automatic and Save-cleanup callers can encode outside their locks. Store ordering still rejects older dirty/clean captures before publication/retirement, including background commits sharing one command revision. Shutdown retains its existing serialized final-capture behavior and remains a separate responsiveness task.

## Evidence

Blocked-worker race tests prove original capture versus later editing/stepping, preparation cancellation/replacement, scan selection preservation, checkpoint publication fencing and preservation of a newer automatic catalog. Existing store crash/ownership/retention/order tests remain active. Native recovery callbacks use immediate admission; native tests/GPU smoke exercise manual checkpoint/restore with assertion-only waits. The public workflow checks captured/later values, capsule identity, original saved-file preservation, explicit cleared selection, waiting compatibility and a final clean paused project.

Run `python3 scripts/project-recovery-work-demo.py --output REPORT --artifacts DIRECTORY`; `--connect` replaces only a designated owned endpoint. Backend timing excludes visible latency. Recovery configuration now uses [background setup](recovery.md#background-recovery-setup). Dismiss, inspection/sample queries and some shutdown paths still include synchronous work. Complete events/agent coverage, accessibility, numerical/performance consolidation, distribution and participant gates remain open.

Recovery Restore now uses its own shared background worker; see [Restore contract and evidence](project-restore.md).
