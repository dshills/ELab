# Vellum widget adoption qualification

October 25, 2026. Five UI migrations described in
[implementation notes](../../../research/114-vellum-widget-migrations.md).
The upstream remains f9dee54; compatibility source 0edd5db extends the existing
decimal step fix to the compact stepper. The new elab3 archive, complete
compatibility patch and all verifier inputs are pinned in sdk.lock.json.
SDK comparison/integrity checks cover all 8,730 members; elab2 is preserved.
The 53-part library, shared commands, project formats, numerical equations and
simulation work/accuracy limits are unchanged. The 30-minute desktop suite
limit is a test-runner deadline, not a solver allowance.

## Source checks

Goimports/Staticcheck lint, Python tooling tests, PRD ledger verification and
complete root, desktop and rendered desktop race/vet suites pass. These are
whole application suites, not just the new-widget selection. Exact commands,
exit status and timing are in quality-complete.json; raw logs are compressed.
Cached Go successes retain normal Go cache semantics. suites.json records skips.

| Suite | Top-level tests passed | Including subtests | Skips |
| --- | ---: | ---: | ---: |
| root-race.jsonl | 861 | 1940 | 0 |
| desktop-race.jsonl | 332 | 765 | 0 |
| mounted-race.jsonl | 453 | 1337 | 0 |

## Compact stepper compatibility

The first complete rendered run caught binary accumulation in the compact
stepper: its addition bypassed the existing managed decimal helper. The corrected
SDK shares decimal arithmetic between modes without changing typed/off-grid
values, boundary policy or service precision. The unchanged ELab playback-step
regression now passes. Full framework portable/ordinary-driver race and vet,
tagged vet and focused rendered number-field checks passed; exact scope and
source are in framework-complete.json. This is not a new full rendered framework
run. No numeric assertion or tolerance was weakened.

## Specific regression coverage

- Search typing/Return preserves commit semantics and the existing field ID;
  Clear routes through the shared command and returns focus to the entry.
  A captured Clear cannot erase a query changed by an agent, even after View
  refreshes the retained SearchBox callback.
- FormField preserves editor/label IDs and field callbacks. Refused component
  value guidance comes from the same response as the agent, appears beside the
  field and in InlineNotice, and leaves accepted values unchanged.
- Compact steppers use existing bounds, steps and ownership. Actual routed
  keyboard stepping and wheel input are checked in the production inspector.
- Synthetic macOS preference observations qualify System dark/accent behavior,
  preservation of explicit Light/Dark/Apollo, contrast/motion policies and no
  service mutation. Motion-only preference changes invalidate static schematic
  geometry; custom scope/sweep paint revisions also track appearance.
- Actual More-menu pointer opening and Fit selection are checked in all four
  themes, with actions restored at wider width. Old menu choices refuse an
  intervening agent edit; an actually open popup retires before it can borrow
  refreshed callbacks. Existing complete button/tab/inspector paint and
  field-correction regressions remain active.

The initial new fixtures omitted the offscreen helper's explicit background;
that test configuration was corrected. A search-only fixture also discarded the
production FocusRequest; it now retains it and still requires Clear entry focus.
The consecutive search/placement focus-order assertion now explicitly includes
the new Clear affordance, retaining the remaining placement checks.
Focus-stroke coverage expects the new scoped accent rather than the old fixed
framework fallback. Geometry and outline-coverage assertions remain active.

The reviewed native inventory is refreshed for the new source/callback shapes.
It retains 182 public commands and the same 150 literal public routes, with no
unknown routes. The complete root test initially caught the stale ledger; it
passes after reviewing the SearchBox/overflow/field bindings and regenerating it.
See inventory-review.json and research/13-native-command-inventory.json.

## Preserved releases

preserved.json rechecks the qualified alpha ZIP and the previous /254 app/agent
hashes. The preceding SDK lock, archive and compatibility patch remain unchanged. The new package is a development
candidate and does not replace or promote the qualified unsigned alpha.


## Completed package check

ELab **0.194.0 /255** is built from clean source
`87b7d47a7b01fe86a419db0b7f24f5e5da83526f`, with its matching agent alongside it. Both build-info
receipts report that revision and `vcs.modified=false`. The unchanged SDK packager
records matching version, build, SDK release and pre-signing source-binary hash.
Strict ad-hoc signature verification passes. The qualified unsigned alpha remains
unchanged.

The packaged replay saves, fresh-reopens and exports all six latest electronics
examples as SVG/PNG/PDF. Both sound examples render and export matching WAV bytes
without autoplay. Artifact hashes and replay receipts are retained here.
The separately profiled native check is described in `native-notes.txt`; its UI
and state evidence are saved. The owned session was closed normally and its
endpoint is absent. Owner projects and existing applications were preserved.

## Remaining native observation

A PWM bounds tooltip remained visible in captures after changing to the LED
inspector and moving focus. The resistor’s accepted domain and validation are
correct; this is an unresolved native hover/tooltip-retirement observation.
See [native notes](native-notes.txt) and saved UI captures. The native check also
records one intermittent custom-mesh capture omission, with the circuit visible
in earlier and later captures. Neither observation is counted as a passing
hover/rendering gate. All automated suites above pass.
