# Background recorded-result import

`verification.import.start {path, expected_document_generation?}` returns queued background admission. Existing `verification.import` and the native recorded-file picker wait for that same job. Shared `project_work`, `project.work.inspect {id}`, `project.work.cancel {id}` and bounded `simulation.wait` expose reading, validation, cancellation and terminal outcome. Eight retained project jobs/workers remain bounded; concurrent project file work refuses with `busy`.

The worker reads a regular file bounded to 8 MiB and applies the existing strict recorded format/model/graph/settings/probe/sample/statistics validation outside the editor mutex. It trusts no file-supplied live result. Successful attachment adds one complete kept recorded asset with original metadata/rows, a new session ID, admitting actor and input-file SHA-256. `result_operation_id`, `source_sha256` and `viewer_selected` distinguish validated input, imported asset and viewer outcome. Original recorded format and model/work limits are unchanged.

## Live and viewer ownership

Import never changes the live graph, simulation settings, charge/time/run, view navigation or undo history. Editing/stepping can continue during file validation. A successful kept import marks recorded membership dirty, with normal Save/recovery ownership. Reexport retains byte-identical captured data; source path/hash are session provenance, not injected into the portable record. Imported records never claim `source_unchanged` against the current live circuit.

With no intervening viewer action, completion selects the imported record at its first sample. If the user/agent selects a different operation, starts Verify or changes recorded/live viewer controls while import is pending, the new asset is kept without overriding that later choice. Autonomous playback ticks do not count as a new user choice. If an unkept selected asset is evicted at the bounded retention limit, selection clears to live rather than dangling or being forced onto the import.

Recording capacity is checked at admission and again before attachment. If another Verify uses the last available slot during validation, import fails with `record-limit`; existing kept/active records remain intact. A slot is not silently reserved by admission. Excluding an unkept completed asset can make room under the existing retention policy.

## Cancellation and retries

Exact-ID cancellation or Close discards staged data; project replacement supersedes it. An old result/failure cannot attach to a newer or closed document. Within the same document, live electrical/presentation edits are allowed and remain intact. Completion installs the entire validated asset under one mutex acquisition, without a filesystem publication boundary.

An admitted read/validation failure exposes only job bookkeeping and its original error code; it preserves project, physical state, recordings/viewer and history. Waiting request disconnection ends that request while admitted work continues; inspect/cancel or retry the original key. Waiting retries return the original independently copied terminal outcome without duplicate assets. Immediate-start retries retain original admission. Use a new key for a corrected attempt after an admitted failure. Runtime jobs are not saved.

The 30-second context supports cooperative cancellation/deadline checks; a blocked operating-system read or an already executing bounded validation can finish later, but cannot publish canceled data. Named phases avoid fabricated percentages. Native messages identify import and explain when the asset was kept without switching the viewer.

## Evidence

Controlled race tests block reading/validation while editing/stepping, canceling/closing/replacing, changing viewer choice or consuming the last record slot. They prove atomic complete attachment, live-state/history preservation, source/actor attribution, current-source separation, retry ownership and kept Save/reopen portability. Existing malformed-model/row/statistics/version and cross-family replay tests remain active. Native picker tests require shared completed job/provenance; built/packaged GPU smoke imports an owned exported asset and preserves live time/run. Public cross-process workflows check original metadata, hashes, byte-identical reexport, invalid-file preservation and fresh paused startup with kept replay.

Run `python3 scripts/project-import-demo.py --output REPORT --artifacts DIRECTORY`; `--connect` replaces only a designated owned endpoint. Admission timing excludes visible latency. Manual recovery and some shutdown paths remain synchronous; complete events/agent coverage, accessibility, numerical/performance consolidation, distribution and participant gates remain open.
