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

## Next work queue

1. Autosave/recovery that preserves explicitly saved files and validates recovery before replacement.
2. Recorded result-file import and replay as explicitly recorded data.
3. Explicit schematic junction/net-label primitives and supported topology expansion.
4. Generic diode, switch and potentiometer models with electrical fixtures.
5. Capacity/latency measurements, keyboard/screen-reader checks and distribution qualification.

Every shipped user operation continues to require the same public agent command/state path. Each slice adds meaningful regression checks and records its actual limits. Participant usability and physical hardware validation remain future gates.
