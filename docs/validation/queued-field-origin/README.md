# Queued field callback ownership

[Receipt](report.json), [raw checks and source](raw.tar.gz),
[raw member hashes](raw-manifest.json), [frozen application inputs](source-inputs.json).

Queued native text/number Change events previously replaced their originating
callback with the latest descriptor's callback at owner delivery. A repaint could
therefore lend an earlier edit a newer form's request context, bypassing ELab's
stale-callback refusal guard. Both retained-widget regressions fail before the
repair: the original callback receives zero calls and the refreshed one receives
one. Text and number events now retain their admitted callback, matching existing
slider behavior. Fresh independent events use the new callback with exact payloads.
Settlement epochs, lifetime guards and other discrete-control delivery are unchanged.

Focused race regressions and related settlement/slider checks pass. Full framework
preparation, normal/offscreen race and vet checks pass. The first sandboxed broad
run failed temporary Unix socket binding; its log is retained separately from the
successful authorized rerun. Full ELab `scripts/check.sh` passes, including lint,
67 Python checks, root/desktop race tests and vet, mounted-control race checks and
offscreen vet. Existing AppKit deprecation/linker warnings remain in the raw log.

Private SDK elab14 is exported from clean source
`23063545f871d8de0724077744d53f8f90d32c69`. All 8,440 members match the verified
manifest. Compared with elab13, only `drivers/cogent/controls.go`, its new regression
and generated `SDK_DOCUMENTATION.md` provenance change. Dependency, notice,
release-metadata and bootstrap bytes are unchanged. Both SDK caches and all frozen
application/check inputs are reverified after the gate. No application/backend Go
production source, public commands, components, models or formats change.

The installed /223 native executable, matching agent, original candidate native
executable, saved RC fixture and /205 ZIP retain their recorded hashes; the installed
app passes strict signature verification. No app is launched, updated or closed
for this checkpoint. No new candidate is built or promoted.

The owner returned and asked to finish the current work and stop. The source repair
is saved; fresh candidate/native qualification is deferred. The four original
native checks (3, 6, 9 and 11) remain open. These retained native-widget queue tests
do not themselves establish exact AppKit event admission ordering or close those
checks. Earlier /223 evidence retains its original scope; /205 stays the preserved
download. Work stops at this checkpoint.
