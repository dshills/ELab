# Continued MVP implementation and code quality

**Started October 13, 2026.** The product owner asked for continued backlog work until they return and explicitly stop it, with attention to reviews, lint, structure and Go practices. This record tracks completed and active slices; it does not declare the entire MVP complete.

## Quality checkpoint and review

The previously uncommitted implementation/research was checkpointed locally on `codex/mvp-continuation` (`8ef5cf4`). No changes were pushed. Subsequent green slices receive separate local commits so their scope is reviewable and reversible.

The initial source review found and repaired:

- **File picker/project replacement race.** Save/Open/export choices could apply to a project an agent loaded while the picker was open. Shared file commands now accept `expected_document_generation`; native pickers capture/preserve it, including pending unsaved-change decisions. A mismatch refuses before reading, writing or publishing. Tests replace a project while the native picker adapter waits.
- **Cursor endpoint precision.** Fixed-window cursors could be called unavailable when decimal endpoints differed only by floating-point timestamp roundoff. Cursor validity now uses the same inclusive time predicate as plots, statistics and CSV.
- **Duplication at bounds.** The default copy position could fail at the positive corner despite nearby free space. Bounded search now tries eight directions while preserving the usual first placement and all existing validation.
- **Lint findings.** Removed an unused hash helper and dead axis initialization; replaced deprecated title-casing of known transistor states with explicit labels; normalized imports and checked previously ignored simulation/CSV return values.

## Repeatable quality gate

`scripts/check.sh` runs project-local Staticcheck and import formatting, then root/desktop race tests and Go vet. `scripts/lint.sh` is separately callable. The nested `tools` module pins Staticcheck 0.8.1, x/tools 0.50.0 and checksums; runtime core dependencies remain standard-library-only. Tools build into ignored `build/tools`, leaving global installations unchanged.

The older installed analyzers could not read the Go 1.27.2 export-data format. Compatible tools were built from existing cached modules and validated offline. No lint checks were suppressed to hide that incompatibility. A numerical race test hit the mixed engine's existing 100 ms guard while a heavy analyzer build ran concurrently; the serial quality gate passes with the guard unchanged. Heavy analysis/build tasks are kept separate from numerical acceptance tests.

## Verify, operation lifecycle and recorded exports

The next slice implements [bounded Verify](../docs/verification.md) with captured graph/startup/settings, independent kernels, operation IDs/progress/cancellation, retained immutable results and paged SI samples. Native settings/readouts and external agents use the same commands. JSON/CSV exports retain the captured source after editor changes. Live physical state, graph, undo and dirty state are preserved. Limits and failure semantics are explicit; partial failed/canceled computations are never published as recorded results.

Discovery uses a complete local schema registry (`schema_version:2`) rather than repeating the full Snapshot definition per command. This preserves the existing 1 MiB response bound and keeps operation discovery scalable.

Completed queue items: bounded Verify, operation lifecycle, and portable recorded exports. Dedicated record-file import/replay remains a follow-on item. Validation: [headless report](../docs/validation/headless-verification.json), [desktop report](../docs/validation/desktop-verification.json), core/native parity regressions and native GPU/UI smoke.

Packaged slice: **0.10.0 / 24**, with [final validation](../docs/validation/verification-validation.json). The prior application bundle was preserved.

## Autosave and crash recovery

Implemented [autosave/recovery](../docs/recovery.md) through seven shared commands and native Inspector controls. Native startup defaults to two-second private snapshots; headless sessions opt in explicitly. Snapshots contain complete validated portable schematics, with immutable publication, checksums, per-session ownership and two-copy retention per document. Saved projects are preserved. Restoring opens a dirty unassociated copy for Save As; dirty/generation/hash guards and the existing native unsaved-change prompt apply.

Review fixes include serializing final shutdown with pending writes so explicit discard cannot be undone by a late autosave, rejecting old captures after an explicit save, retaining a valid older snapshot if the newest is damaged, preserving other live sessions' snapshots (including damaged bodies with identifiable ownership), bounding scans/diagnostics, preserving an explicitly cleared candidate selection, and returning through CLI deferred cleanup before choosing a nonzero process exit. Filesystem work for automatic capture occurs outside the editor mutex. Shared document serialization/installation was extracted from persistence rather than duplicated.

Validation includes core/desktop race tests, Staticcheck/import formatting, Go vet, native/agent control parity and actual GPU/UI smoke. An owned headless process was killed with SIGKILL and reopened: [crash recovery report](../docs/validation/crash-recovery.json) shows successful automatic capture, validated copy restoration, Save As and byte-for-byte preservation of the original `.elab` file. This is process-crash validation, not power-loss qualification.

Packaged slice: **0.11.0 / 25**, with [final validation](../docs/validation/recovery-validation.json) and [desktop endpoint validation](../docs/validation/desktop-recovery.json). Recovery copies, leases, build artifacts and running older application sessions remain local; no user files or earlier bundles were removed. Recorded results/history are not recovery payloads; Windows liveness and public distribution remain unqualified.

## Next work queue

1. Recorded result-file import and replay as explicitly recorded data.
2. Explicit schematic junction/net-label primitives and supported topology expansion.
3. Generic diode, switch and potentiometer models with electrical fixtures.
4. Capacity/latency measurements, keyboard/screen-reader checks and distribution qualification.

Every shipped user operation continues to require the same public agent command/state path. Each slice adds meaningful regression checks and records its actual limits. Participant usability and physical hardware validation remain future gates.
