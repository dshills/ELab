# Background captured exports

`export.start {kind, path, overwrite?, id?, expected_document_generation?}` returns queued admission for `schematic-svg`, `scope-csv`, `verification-json` or `verification-csv`. Recorded kinds require the retained completed record's `id`; live kinds refuse a recording ID. Existing `export.schematic.svg`, `export.scope.csv` and `verification.export` wait for the same workers and return `last_export`, preserving native picker behavior and file formats.

The existing shared `project_work`, `project.work.inspect {id}`, `project.work.cancel {id}` and `simulation.wait` cover exports. Eight project jobs and filesystem workers remain bounded. A second project file operation while an export is active refuses with `busy`. Native status identifies Export and disables cancellation when publication begins.

## Capture and preservation

Admission validates generation/path and refuses the current schematic path. Live scope CSV requires fresh selected traces in a valid retained interval. Its window, probes and retained samples are captured together. Later stepping or changing probes/windows does not alter the file. Existing row labels, SI units, duplicate edge timestamps and source run/electrical identity remain intact.

SVG captures graph positions/values, voltage/current display and pinned readings under the shared mutex, then renders outside it with motion frozen and existing static direction cues. Later geometry/value changes stay in the editor and cannot enter that captured drawing. Incomplete drawings remain exportable; stale/unavailable readings retain their labels.

Recorded JSON/CSV captures immutable record rows and independent metadata. Excluding/evicting that record, altering the live circuit or changing recorded viewer controls cannot substitute another dataset. JSON retains the complete captured graph/models/settings; CSV retains original model/operation/units attribution. Existing formats and 8 MiB bounds are unchanged.

Encoding, destination checks, temporary write/sync, atomic publication and cleanup run outside the editor mutex. Exports do not change project association, saved baseline, physics, scope, selection or undo history. Later user edits remain intact. `last_export` adds exact `sha256` and capture `document_generation` beside its existing path/format/run/electrical/record ID and byte count. Project-job completion carries the same hash and source generation/run. These identify a capture, not a claim that the current circuit still matches it.

## Cancellation and publication

Preparation is cancelable; replacement/Close before publication discards work without replacing the destination. Phases include `encoding`, `checking-destination`, `writing`, `publishing`, `cleanup` and `completed`. They are named stages without fabricated percentages. Once atomic publication starts, `cancelable:false` and `operation-committing` refusal make that irreversible boundary explicit. A publishing artifact can finish after replacement, with false current-document association and original provenance; the replacement project is preserved.

Save and Export now share one file-publication module. Regular bounded destinations are rechecked for identity/content just before replacement; new targets use exclusive link publication. A destination changed during preparation is refused, even after explicit initial overwrite. Temporary files are removed before terminal completion, with separate cleanup warnings. The current schematic is protected by path/generation and serialized project-file operations. Portable final-check/rename cannot guarantee compare-and-swap against every unrelated writer, and operating-system calls may outlast cancellation/deadline.

Waiting command retries return their original terminal outcome without another write; immediate-start retries return original admission. Admitted file errors retain only operation bookkeeping and preserve prior artifact/project/history/readings. Use a new key for a corrected attempt. Jobs are runtime state and are absent from saved/recovery documents.

## Evidence

Controlled race tests block encoding and durable preparation/publication, then navigate/edit/step/change scope, evict recorded assets, cancel or replace documents. They prove captured source ownership, existing-destination preservation, external-change refusal, temporary cleanup and publication fencing. Native tests compare user/agent bytes and require shared job/artifact identity. Built/packaged GPU smoke invokes all four native export controls with temporary picker adapters. The public workflow verifies all four immediate-start kinds, captured intervals/metadata, hashes/byte counts/generation, protected current state and byte-identical legacy output.

Run `python3 scripts/project-export-demo.py --output REPORT --artifacts DIRECTORY`; `--connect` replaces only a designated owned session. Backend timings exclude visible latency. Manual recovery, recorded import and some shutdown calls still include synchronous work; full events/agent conformance, accessibility, numerical/performance consolidation, distribution and participant gates remain open.

Recorded import now also uses [background project work](project-import.md). Manual recovery and some shutdown calls remain separate paths.
