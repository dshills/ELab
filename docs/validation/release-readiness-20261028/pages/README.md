# Item 3: bounded analysis queries

analysis.list and analysis.page are two new pure shared-service queries: 222
commands, 61 physical kinds. Legacy queries/snapshots/exports remain unchanged.
Pages support sweeps, recordings, spectra and tolerance trials with explicit
kind/id/document-generation identity, 1–50 rows and detached output arrays.
Listing contains compact admission guards without constructing a full snapshot.
The agent transport and headless lifecycle loop use a small ReadyToExit getter,
so pure page requests do not trigger a hidden full-snapshot shutdown check.

Core tests reconstruct every data family, own nested arrays/scalars, preserve
all 201 capacity noise points, enforce bounded responses, zero/negative/over-range
and terminal offsets, stale document generation, missing/not-ready IDs and real
retention eviction (exclude a kept sweep first; eight kept captures are protected).
Transport tests compare exact stdio/socket replies. Matching native, socket and
stdio software fixtures reconstruct 201 rows with maximum 50-row response 274687
bytes under the controller's JSON encoding; all agents and owned hosts exit 0.
The native fixture is a copied/rebuilt test bundle, not a replacement of /279.

Final complete portable suite passes 104.636 s, affected rendered race UI 17.8 s,
focused race suites, both vets and lint pass. Initial compile used an unknown
ClearParams fixture type; initial page clone normalized nil AC metadata wrongly;
initial eviction test tried to evict a kept record. These are retained and fixed;
no production retention protection or existing metadata semantics were weakened.
The first complete portable run failed only that initial eviction test; the
corrected complete invocation passes all packages.

Complete initial/follow-up Prism reviews use Anthropic Claude Opus 5.5. Initial
findings resolved: assert the transport fixture is paused; guard empty tolerance
IDs; clarify 0.200.1 /279 reference; handle marshal errors/use test contexts;
reject unknown internal record types; preserve slices with slices.Clone.
Follow-up has no medium/high findings. Slice normalization is a new unreleased
query, with exact source preservation intentionally tested; legacy commands were
not changed. Close waits for shutdown done and is idempotent, so the readiness
assertion is synchronous by contract. Its misplaced doc comment is corrected.

This is shared-command/runtime evidence, not native OS queue/hardware acceptance.
No components, electrical limits or solver/work guards changed; no push/upload.
