# Recorded controls retain private state only for their capture

October 18, 2026. A wrong-target measurement edit reproduced in the /155 source and repaired without changing commands, models, project format or the frozen SDK.

## Failure

Create two 20 ms RC captures. On capture `verify-1`, type a private `8 ms` edit without committing. An external public `operation.select` switches to `verify-2` before one redraw. Both captures have the same displayed initial time, and the native renderer retains the same widget/private text. Focus that new capture’s visible field and press Enter: the old draft is accepted through the freshly rebound callback for `verify-2`.

All three reproduced cases advance revision 8→9 with no error: cursor A is placed at 8 ms, recorded seek moves to sample 8, and plot-window start becomes 8 ms. The native incarnation stays 2→2 across captures. Same-capture repaint correctly preserves the draft and is a passing control. [Before run](before.txt).

## Repair

Recorded inspector, panel and cursor factories scope private native control state to `recording:<capture ID>`. The existing document-generation binder still composes the document identity afterward. A different capture therefore creates a fresh native editing/pressed state even if its public control ID and controlled text match the old capture. Same-capture redraw retains its private input.

The existing component scoping function now delegates to the shared ownership helper with exactly its previous component ID. Its keys and semantics remain unchanged. Scoping occurs in each recorded factory, so isolated controls also carry ownership; reapplying the same capture key in the assembled panel is idempotent. All domain edits still use the existing guarded public commands.

## Evidence

Mounted routing types `8 ms`, verifies no public mutation before commit, switches captures through an agent command and redraws. The repaired widget changes incarnation 2→3 and shows canonical `0 s`. The test explicitly focuses it before Enter; the complete public snapshot stays unchanged. It then types a fresh `6 ms` edit, which changes only the selected new capture’s intended cursor/seek/window through one command. This rules out apparent protection caused merely by absent focus. All three same-capture control cases preserve their native incarnation/draft and commit normally. [After run](after.txt).

The tests check unchanged live components/wires, run/electrical identities, transient/mixed physics, document and circuit Undo/Redo. Broader focused tests also cover recorded one-click/deferred cursor handling and the existing retained component text, numeric and button cases. [Focused race log](focused.txt).

Only `desktop/field_context.go` and `desktop/recording.go` change production native hashes. All non-line source-site records and command/alias/coverage sets match the prior ledger: 68 files /393 sites /160 binding-review sites /207 literal routes /162 public IDs /132 referenced IDs. The SDK remains `677548d`.

This proves the three mounted private-text target-switch cases and source ownership propagation. It does not qualify all recorded controls, physical OS interleaving, IME, external reader or visible p95. Source/native checks and a clean replacement package are separate steps. /155 and canonical /120 are preserved. Owner walkthrough is still “Not tried yet”; P0 remains 17 implemented /10 partial.

The final complete `scripts/check.sh` passes: audit freshness, import formatting/Staticcheck, 51 Python tests, root/native race tests and vet, required mounted regressions and tagged vet. Native race: 153.344 s; mounted race: 17.538 s; broader focused race: 11.431 s. The development executable also passes full native smoke. [Quality log](full-check.txt), [native smoke](native-smoke.txt), [build metadata](development-build-info.txt), [checkpoint](checkpoint.json). These are source/development-binary checks, not a clean replacement release qualification.
