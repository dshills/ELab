# Vellum pushed-source qualification

ELab now pins **v0.1.0-personal.1**, upstream
`f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40` on the owner's pushed
`codex/mac-personal-app-stabilization` branch. The private compatibility source is
`218c0dec8cfb52de4d63b91e0701f340396b7d84`. This source is newer than the published alpha.3 assets.
The owner Vellum checkout was not modified. See the
[active dependency and full patch](../../../third_party/vellum/README.md).

## Integrity and preserved artifacts

All 8,730 SDK members match the manifest. Archive, manifest, base/derived
source, compatibility patch and six verifier imports are pinned before execution.
All four replacements agree with the lock. Native preparation verifies all 110
ordered dependency stages, including upstream's new group-source stage and the
four retained ELab stages. The prior SDK archive/cache were fully reverified;
/205 alpha and /253 app/agent digests are unchanged (`before.json`, `integrity.json`).

## Resolved compatibility issues

The compatibility overlay retains ELab's existing control ownership, queued field
and held-gesture callbacks, numeric drafts/decimal steps, system cursors, current
event coordinates, wheel routing, partial-scroll accessibility, divider gestures,
themed disclosure/window colors and text Undo boundaries.

Explicit focus intent and the retained layout-reveal fallback now share lifetime
rules. A cancelled request cannot revive during paint. Native Return/Tab can move
focus before reconciliation; its new target receives one post-layout reveal.
Focus-only cancellation schedules a bounded host paint to retire pending state.
Clearing explicit focus still permits later pointer focus, while a completed
programmatic reveal cannot override a later manual wheel. Existing and new routed
focus/wheel tests cover these cases, and ELab's four interval-validation cases
pass without changing their assertions.

Toolbar customizer frame lookup and parent identity use `core.AsFrame` for the
accessibility scroll-host subtype. The upstream number-stepper restoration check
now requires its retained widget, icons and working base step function because
ELab supplies decimal stepping rather than the upstream nil/default step. No
regression is deleted or skipped as a repair. ELab adjusts explicit action padding
for the new two-point managed focus border so constrained toolbars and tab
underlines retain their original outer geometry. The painted focus-color check
now expects the documented managed macOS accent fallback rather than the old
Cogent global color. Existing geometry, outline-coverage and shared-command
assertions remain active across all four themes. The agent-selected stroke is
checked within the same three-pixel border region, accommodating its one-pixel
inset inside the reserved focus outline instead of assuming the outermost pixel. No electronics model, work limit,
application command or project format changes.

## Check scope

The complete portable and ordinary native-driver race/vet suites pass on the final
source. Final focused rendered tests cover focus, cancellation, routed pointer/
wheel input, toolbar dragging, numeric fields/drafts, queued callbacks and partial
scroll geometry; tagged vet passes. `framework-final-complete.json` keeps exact
commands and the final source identity. An earlier complete rendered framework
run also passed at intermediate source `fe1965c`; `framework-complete.json` and its
raw log retain that scope. It is not presented as a complete rendered run of the
final focus adjustment. Initial and intermediate failure logs are labeled separately.

ELab's native build, goimports/Staticcheck lint and 67 Python tooling tests pass.
The final ordinary desktop selection covers UI/command contracts; the rendered
selection exercises all `TestMounted` controls plus symbols, workspace and theme
cases. Exact commands are in `application-complete.json`; these are compatibility
checks, not a rerun of every electronics test already recorded for /253.

| Final ELab suite | Top-level passed | Including subtests | Skips |
| --- | ---: | ---: | ---: |
| desktop-compatibility-race.jsonl | 10 | 16 | 0 |
| mounted-compatibility.jsonl | 120 | 557 | 0 |

Raw logs are compressed alongside this file. The initial consumer runs found interval-field visibility failures and toolbar/
tab painting failures. Their labeled logs are preserved. The final runs include
those exact production cases with corrected focus coordination and layout.


## Completed package check

ELab **0.193.0 /254** is built from clean source
`ce3fb5af717a0a796aa3f79dba28a57bd316ea23`, with its matching agent alongside it. Both build-info
receipts report that revision and `vcs.modified=false`. The current SDK packager
records matching version, build, SDK release and pre-signing source-binary hash.
Strict ad-hoc signature verification passes. The qualified unsigned alpha remains
unchanged.

The packaged replay saves, fresh-reopens and exports all six latest electronics
examples as SVG/PNG/PDF. Both sound examples render and export matching WAV bytes
without autoplay. Artifact hashes and replay receipts are retained here.
The separately profiled native check is described in `native-notes.txt`; its UI
and state evidence are saved. The owned session was closed normally and its
endpoint is absent. Owner projects and existing applications were preserved.
