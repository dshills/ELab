# Remaining web priorities — November 4, 2026

Source baseline: `f1fc39eb`, after items 1/3/5 merged to main. The original list
and current status are [recorded here](../../web-followups.md). Each milestone
receives internal review and a separate commit before subsequent implementation.
The owner has no testers available, is a novice, and requested best judgment
and online knowledge. That context is now in the project instructions.

## Item 2 — beginner activities and engineering rehearsal

Added three guided activities beside the drawing, accessible from Help. Removed
stale native-versus-web comparison instructions. Explanations cover units,
signed readings, changing one variable, pause/restart, captures and reopening.
Replacement uses existing save/recovery confirmation; cancellation preserves
both the circuit and the current activity. No telemetry or learning scores.

Internal review checked lesson values against the actual examples, exact UI
labels, safe text rendering, focus, replacement cancellation, no automatic
completion claims, and a current electronics-novice context in AGENTS.md.
References and human-research limits are in [activity guidance](../../learning-activities.md).
All frontend checks and three real-engine browser journeys pass, including a
resistor edit that reduces solved LED current and a save/reload of that edit.
Logs: `build/web-remaining-learning-frontend.log`, `build/learning-browser.log`.
No human pilot or human acceptance is claimed; the owner explicitly directed
engineering work without depending on recruitment or their circuit expertise.

## Item 4 — group circuit editing

Added box/Shift selection, group drag and keyboard movement, detached copy/paste,
duplicate and delete. Shared guarded commands commit each operation atomically
and create one history action. Copies preserve settings and internal wires,
remap block members/ports/parameters and separate copied visible named nets.
Clipboard IDs prevent silently pasting another actor's replacement clipboard.
Failed destination/capacity/ID/hidden-member checks preserve history and IDs.

Internal review corrected selection normalization for expanded blocks, preserved
SVG node identity and viewport during drags, excluded text/browser shortcuts,
and kept catalog discovery under its existing 64 KiB budget by shortening only
compact summaries. Full descriptions/schemas remain available. Paste searches
free destinations; final engine checks remain authoritative.

Core selection/schematic/discovery race tests pass, including detached clipboard,
capacity refusal, block remapping, copied net labels and one-step undo/redo.
Root/webagent/MCP races, lint and 29 frontend tests pass. All 30 browser journeys
pass across three engines, and ten consecutive Chromium box/drag repetitions
pass. Evidence: `build/selection-reviewed-core.log`, `build/selection-root-race.log`,
`build/selection-frontend.log`, `build/selection-lint.log`,
`build/selection-milestone-browsers.log`, `build/selection-drag-repeat.log`.

## Item 6 — actionable circuit diagnostics

The browser now displays shared structured explanations, corrective advice,
explicit technical details and associated component/wire/terminal markers.
Actions select actual targets, begin wiring at a named terminal, undo a deletion,
fit sampling or review/restart a failed simulation. No automatic circuit repair
or guessed object association is introduced. Request refusals keep scoped local
help and disappear after a correction/new command. The panel no longer overlays
or blocks drawing gestures.

Internal review covered request/source scope, safe text rendering, stale marker
clearing, hidden-block target resolution, disclosure purity and guarded actions.
The repair journey found a fast Delete bug: target actions now await pending
selection and reject document replacement. Drag guards wait for the browser's
own queued selection, check unchanged source positions/generation, then freeze
through pointer-up; concurrent changes remain guarded rather than retried.

Shared diagnostic race tests, lint and frontend checks pass. All twelve targeted
three-engine journeys pass: actual wire deletion/repair, precise resistance
refusal/correction, group edits and fast box/drag. Existing workbench journeys
passed in the preceding combined run; the final whole suite is rerun after the
remaining milestones. Logs: `build/diagnostics-core.log`,
`build/diagnostics-frontend.log`, `build/diagnostics-lint.log`,
`build/diagnostics-final-browsers.log`.

## Item 7 — measured browser responsiveness

Measured the 100-object/100-node and 100-object/64-part envelopes with six traces
under Chromium 4× CPU slowdown plus a 50%-duty worker. [Measurements and scope](../../browser-responsiveness.md)
retain both baseline and final results, sample sizes, cold-tail limitations and
same-revision transfer sizes. Scope stayed live and no component roots were
replaced during the live interval.

Implemented opaque conditional scene frames, optional command presentation
without duplicate snapshot transfer, stable geometry nodes, cached lookups and
SVG coordinate/focus handling. Internal review covered host/workspace token
isolation, no cookie exposure, restart/edit invalidation, dynamic controls and
segments, idempotent replies, recovered dirty status, stale acknowledgements,
transformed geometry/pins/wires and cancellation. Ten repeated marquee/drag
trials pass after coordinate conversion and text-selection suppression.

Root/webagent/MCP race suites, lint and 31 frontend tests pass. Forty browser
journeys pass; two Chromium-only load cases are deliberately skipped in the other
engines. Live counter tests verify changing segments, stable focused node identity
and unchanged command revision. Recovery tests verify selection acknowledgements
retain unsaved recovery status. Logs: `build/responsiveness-reviewed-race.log`,
`build/responsiveness-reviewed-lint.log`, `build/responsiveness-reviewed-frontend.log`,
`build/responsiveness-all-browser.log`, `build/responsiveness-coordinate-drag.log`.

## Item 8 — keyboard and accessibility qualification

Added keyboard placement preview movement, skip links and instructions, specific
diagnostic descriptions, stable live status updates and valid captured-analysis
definition lists. Internal review checked that placement arrows cannot move the
old selection, pointer movement cannot flood spoken status, text editing stays
native, and analysis terms/descriptions retain their association.

The keyboard-only construction/save/capture/timing journey traverses actual Tab
order and validates accepted solver state without API mutations. Expanded axe
coverage found and fixed a serious nested definition-list defect. Desktop AX
verified named controls and skip-to-schematic focus. Native VoiceOver output is
explicitly unqualified because desktop isolation blocked the native Chrome
surface; see [scope and limitation](../../web-accessibility.md). No owner or
participant acceptance is claimed.

Twenty-four targeted journeys pass across all three engines; lint and 31 frontend
tests pass. Logs: `build/accessibility-reviewed-browser-r2.log`,
`build/accessibility-frontend.log`, `build/accessibility-lint.log`.

## Item 9 — repeatable release automation

Added `scripts/release_web.py` and manual main-only release workflow. Staging
requires clean exact source, complete matching macOS gates, three actual browser
engines and extracted shipping-binary qualification. Candidate branch rehearsals
cannot publish. GitHub publication creates a draft, downloads and hashes each
public asset, qualifies the downloaded binary, rechecks main and only then
optionally publishes. Existing tags/releases are refused. Artifacts use an
explicit upload allowlist; private runtime credentials are excluded.

Internal review covered wrong commit/hash/partial quality receipts, missing or
flaky engines, candidate refusal, existing tag preservation, changed downloads,
main movement and failed-runtime behavior. Four release-focused Python tests
exercise those refusals without external publication. The whole Python suite
also found a stale agent catalog count after the five new group commands; its
expected inventory is now 234. Lint and all Python tests pass. Full clean-source
candidate rehearsal follows the final feature milestone. No GitHub release or
hosted CI result is claimed. Logs: `build/release-automation-python-final.log`,
`build/release-automation-lint-reviewed.log`.

## Item 10 — running build identity and user-controlled reports

Added visible version/source information and an explicit report preview with
copy/download. The backend projects known scalar fields and codes; it excludes
project contents, references, raw errors, paths, identifiers and credentials.
Browser identity is parsed into bounded family/version/platform fields without
exporting the raw user-agent string. Closing clears previews and invalidates late
responses. Packaged executables embed their actual version/build; shipping
qualification compares their live VCS identity against the sealed package.

Internal review covered source fallback, report field/size bounds, unknown-code
redaction, origin/method policy, no session allocation for build information,
recovered dirty state, request/clipboard failure, stale modal responses and no
automatic transmission. Three root race tests, 32 frontend tests, all 75 Python
tests and lint pass. Three report journeys pass with light/dark axe checks, exact
download bytes, explicit request counting, keyboard dismissal/focus return and
error/retry/late-response checks. Logs: `build/support-reviewed-go.log`,
`build/support-reviewed-browser.log`, `build/support-frontend.log`,
`build/support-python.log`, `build/support-lint.log`.

A frozen-source full macOS gate and clean-branch release-candidate rehearsal are
run after this milestone to qualify the combined implementation.

## Combined rehearsal corrections

The first full macOS plan passed on `4bad6822`, with unchanged source inputs.
Its following release rehearsal correctly refused to stage a package because
browser qualification failed. It found two integration defects: the capacity
fixture loader assumed a `browser/` working directory, and the new visible
version text was absent from the version button's accessible name. Fixture paths
now resolve from the test module location; visible version/failure text is always
included in the accessible name. No gate or accessibility assertion was relaxed.

Internal review and all 32 frontend tests pass. The entire three-engine suite,
launched from repository root, passes 46 journeys with two deliberate non-Chromium
load skips. Evidence: `build/web-remaining-browser-corrected.log` and
`build/web-remaining-corrected-frontend.log`. A new frozen-source macOS plan and
release rehearsal follow these committed corrections; the earlier receipt is
not reused for changed inputs.

## Final combined qualification

Qualified clean source: `9e660ea39034b801f343f5093080af290a6238c3`.
The subsequent evidence commit changes only documentation/receipts.

- [Full macOS plan](final-macos-quality.json): lint, 75 Python tests, 32 frontend
  tests, complete Go race suite, vet and package gate pass; source inputs stayed
  unchanged. Race execution took 521.8 seconds.
- [Browser summary](final-browser-summary.json): 46 passed, two planned
  Chromium-specific load skips in Firefox/WebKit, zero unexpected or flaky
  results. Includes keyboard construction/wiring/save/capture/analysis,
  activities, connected group edits, diagnostics, agent revocation, backup,
  stable live scenes, load measurement, support-report privacy and light/dark
  accessibility. Full local report: `build/web-remaining-qualified-rehearsal/browsers.json`.
- [Package gate](final-package-check.json) and [archived shipping runtime](final-shipping-runtime.json)
  pass exact file/mode/signature checks, version/build/VCS identity, embedded
  assets outside the checkout without Go/Node in PATH, normal Quit, restart,
  unsaved recovery and whole-library backup/restore.
- [Release rehearsal](final-release-rehearsal.json) exercised the actual staging
  controller with the matching full macOS receipt. It uses test version
  `0.0.0-alpha.1`, build `0`, and `publication_ready: false`. The ZIP checksum is
  recorded. `verify_stage` rechecked actual staged bytes, identities, all browser
  results and current source hashes successfully. No GitHub release/publication
  or hosted CI run was performed; the published alpha remains preserved.

Local raw receipts: `build/web-remaining-qualified-quality/` and
`build/web-remaining-qualified-rehearsal/`. Private startup/control replies are
excluded from committed evidence. All owned desktop tabs, fixture hosts and
browser hosts closed normally. No listener remains on the browser-test ports.

Developer ID/notarization/clean-account acceptance remain deferred to the owner.
Direct native VoiceOver output remains unqualified due to the documented desktop
isolation block. Automated/engineering evidence is not represented as participant
usability acceptance. The owner's novice/no-testers context is persisted in
`AGENTS.md`; future engineering must continue without making the owner supply
circuit expertise or recruit participants.
