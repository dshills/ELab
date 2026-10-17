# Background captured saves

Native Save/Save As and `project.save` wait on the same worker that `project.save.start {path?, overwrite?, expected_document_generation?}` admits immediately. `project_work` exposes queued status, actor/source revisions, path, named phase, bytes and exact artifact SHA-256. `project.work.inspect {id}` reads one of eight retained project jobs, preserving active jobs. Pending filesystem workers are also capped at eight. Queries are read-only and return the job's original command revision, rather than a current session mutation precondition.

## Captured data and later edits

Admission captures the current committed graph, models, experiment, selected probes/view, pin/display/example preferences, pause preference and kept recordings. Serialization, destination checks, temporary-file write/sync, publication, temporary cleanup and recovery cleanup occur outside the editor mutex. Existing document/model formats and size/device/work bounds are unchanged.

The file contains the admitted snapshot. Changes made while saving stay in the editor. Success sets the saved graph/preferences/recording baseline to the captured content, so later content remains dirty. View changes alone retain the existing dirty policy. Save adds no undo entry and does not restart or advance physics. Saving during pending input work captures its committed graph; await that input if the requested wiper should be included.

An admitted Save supersedes a pending Open, even if Save subsequently fails. A second Save or Open while Save is active refuses with `busy`. Superseded Open remains inspectable by its ID and its waiting caller receives its own terminal result. Successful waiting retries do not write again; immediate-start retries retain their original queued admission. An admitted failure consumes its retry key. Retention is bounded and runtime jobs are never stored in a project.

## Atomic publication and cancellation

Phases include `serializing`, `checking-destination`, `writing`, `publishing`, `cleanup`, `recovery-cleanup` and `completed`; these are named stages without invented percentages. `cancelable` is true during preparation. `project.work.cancel` targets the exact current job. Cancellation, Close or project replacement before publication discards the prepared file and preserves the existing destination.

Once `publishing` begins, cancellation refuses with `operation-committing`, and the native cancel control is disabled. A filesystem replacement may already have happened, so the app cannot honestly claim that cancellation prevented it. A 30-second context bounds preparation/cooperative cancellation; operating-system calls may return later. A publishing save may finish its captured artifact after Close or replacement, but it cannot attach that artifact to the replacement/closed document. `associated_with_current_document` describes the association at completion; `result_document_generation` identifies the captured document and `artifact_sha256` identifies the bytes written. Terminal outcomes are captured history, rather than live association claims after subsequent replacements.

Existing destinations require normal explicit overwrite/current-file hash rules and regular-file checks. Just before rename, ELab rechecks destination identity and content; a change during preparation returns `file-conflict`, including after an initial explicit overwrite. New files use exclusive link publication, refusing a destination created meanwhile. Temporary files are removed before terminal completion; a cleanup failure is a separate warning. Portable filesystem APIs do not provide a universal compare-and-swap across unrelated writers between the last check and rename. Directory sync remains best effort, with power-loss/platform qualification still open.

## Recovery and continuation protections

After a successful associated publication, recovery captures current state: a clean project retires its snapshots; later dirty edits retain their recoverable copy. Cleanup does not block the editor and failures do not turn a published file into a failed save. Recovery ordering compares both command admission and capture-state revisions before clean retirement or dirty writes. A late clean capture cannot erase a newer wiper snapshot committed at the same command revision.

Native messages distinguish a saved capture with later unsaved edits from a prior-document artifact. Save-and-continue does not act on a replacement project or discard later unsaved content. New/example/Close requests accept the same optional document-generation guard, and native pending choices retain their original generation. A stale discard/continuation refuses instead of replacing a newer project.

## Validation

Controlled race tests block durable preparation and publication to prove navigation/editing/cancellation, unchanged prior destinations, temporary cleanup, later-edit dirtiness, external-change rechecks, replacement before publication, replacement during publication/recovery cleanup, disconnected waits/retries, bounded inspection and recovery capture ordering. Native tests protect pending choices and shared file controls; built/packaged GPU smoke exercises Save As/Open. The public workflow validates captured bytes/hashes, inspection/precondition semantics, explicit conflict/overwrite and fresh paused offline reopening.

Run `python3 scripts/project-save-demo.py --output REPORT --artifacts DIRECTORY`. `--connect` requires a fresh owned blank session. Backend admission timing excludes visible latency. [Export](project-export.md), [import](project-import.md), [manual recovery scan/checkpoint](project-recovery-work.md) and [shutdown](shutdown.md) also expose shared background jobs. Complete event/agent conformance, accessibility, distribution and participant gates remain open.

The waiting/start Save commands now expose [on-demand semantic contracts](project-command-contracts.md), including literal/null paths, captured data, overwrite guards, deadlines, retry consumption, publication boundaries and historical association. The ordinary catalog/file formats are unchanged.
