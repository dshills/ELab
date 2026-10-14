# Background project open

`project.load.start {path, discard_changes?, expected_document_generation?}` admits background file reading and validation and immediately returns `snapshot.project_work`. The existing `project.load` waits for the same job and returns its terminal outcome. Native Open uses that waiting adapter on a worker; the Inspector exposes the shared phase/status and **Cancel opening project** command. Drawing/navigation and other clients can continue using the dispatcher.

A job has a stable `project-N` ID, actor, original session/command revision, source document generation, path, phase, bytes read and terminal error. Successful publication adds the new document generation and run ID. Phases are `queued`, `reading`, `validating` and `completed`; they are named stages, not invented percentages. Status is `queued`, `running`, `canceling`, `completed`, `failed`, `canceled` or `superseded`. The snapshot shows the latest job; `project.work.inspect {id}` reads up to eight retained jobs, preserving active work.

## Admission and publication

Generation, dirty/discard and path checks happen before admission. File type/size, strict JSON, model dependencies, graph/settings and embedded records are validated outside the editor lock. Existing 8 MiB/legacy 256 KiB limits and link/special-file refusal remain unchanged. No staged document/result is visible before success.

Publication requires the captured document generation, electrical/actuation identities, graph/settings, persisted pin/display/example preferences, recording membership/operation sequence, source path/disk baseline and pending input mode to remain current. An intervening content edit, Save association, replacement or new verification/import supersedes the open, even when the original request allowed discarding earlier unsaved work. That permission does not authorize discarding later edits. Navigation, selection and display disclosure are allowed; successful Open restores the file's saved view and preferences.

A successful current open installs the complete validated document atomically under the mutex, marks its saved baseline, restores pause preference and starts fresh physical state. Calculation may still be `updating` after the file job completes. `simulation.wait` waits for project work and subsequent initialization, bounded to five seconds. Changed graph/experiment/pin/display/example content adds one undo entry for the original admitting actor/revision; reopening identical content adds no redundant history. Undo retains the opened file association and marks restored older content dirty.

## Cancellation, failures and retries

`project.work.cancel {id}` targets the exact active project ID. A second open while that job is active returns `busy`; unknown/old cancellation IDs return `operation-unavailable`. Close cancels work. Canceled or superseded preparation cannot install a document or reset newer physics/history. An admitted Save supersedes a pending Open, with the original Open still inspectable.

The worker uses a 30-second cancellation context. Cancellation prevents publication, but a stalled operating-system filesystem call cannot always be interrupted immediately; status stays `canceling` until it returns. Request disconnection only ends the waiting adapter, leaving the admitted job observable/cancelable. Shutdown waits for workers outside the editor mutex.

An admitted validation/read failure records only operation bookkeeping, admission revision and actor; it preserves drawing, wires, readings, history, save point and file association. It carries the original `file-error`/`invalid-document` code. Legacy `project.load` returns that error plus its terminal job snapshot. Successful admission consumes the retry key even if validation later fails: use a new key for a new attempt after correcting the file. Immediate-start retries return the original queued admission; waiting-command retries return the original terminal outcome. Cache copies are independent and normal 256-key retention limits apply.

## Evidence and remaining work

Controlled blocked-reader race tests cover responsive navigation/save, exact cancellation, failed read purity, content/preference/replacement/recording supersession, original actor/history, disconnected wait and retry ownership. Native file-control tests preserve Save/Discard/Cancel and picker-generation protections while checking shared job status. Actual GPU smoke uses native Save As/Open with an owned temporary picker adapter. Public headless and packaged workflows exercise immediate admission, invalid-file preservation, saved fresh startup and waiting compatibility.

Run `python3 scripts/project-work-demo.py --output REPORT --artifacts DIRECTORY`; `--connect` replaces only a designated owned session. Admission timings exclude visible input-to-pixel latency. Save now uses [background captured work](project-save.md); export/manual recovery still contain synchronous work; full job/event conformance, accessibility, distribution and participant gates remain open.

SVG/live CSV/recorded JSON+CSV export encoding and disk work now run outside the lock via [captured exports](project-export.md). Manual recovery/import/shutdown remain separate paths.

Recorded import now also uses [background project work](project-import.md). Manual recovery and some shutdown calls remain separate paths.
