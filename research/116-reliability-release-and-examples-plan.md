# Reliability, workflow, release and examples work

Authorized October 25, 2026. Work through these in order without owner-review
blocking progress. No new components or electrical model families.

1. Complete: distinguish intermittent canvas omission from capture artifacts.
   Reproduce on the current app; inspect native composition/retained resources.
   Fix a demonstrated defect and verify scrolling, inspector/theme changes and
   redraw, with direct-source and composed-frame evidence where available.
2. Complete: finish scroll-related tooltip behavior. Retained numeric controls
   leaving the visible clip must not publish displaced help at the window edge.
   Preserve valid help, numeric edits, existing command semantics and menus.
3. Complete: qualify the core drawing/wiring/editing/measurement/save workflows
   through keyboard and public agent commands, including refusal/correction,
   interruption, stale actions, Undo and recovery. Fix demonstrated failures;
   distinguish actual keyboard evidence from descriptor/callback checks.
4. Complete: produce an updated unsigned downloadable release with the current
   electronics, matching agent, examples, use/model limits, checksum and rollback.
   Preserve the qualified /205 download and /256 development candidate. Test
   archive extraction and offline packaged workflows; retain declared deferrals.
5. Complete: searchable categorized examples browser with schematic thumbnails
   and short experiment descriptions, retaining existing protected example-load
   commands and their agent equivalents. Verify keyboard, search, selection,
   unsaved-work protection and theme/size containment. Package a final update
   after its checks, preserving the step-4 release.

Use standing ELab/desktop authorization in AGENTS.md. Preserve owner sessions
and projects; use isolated fixtures. No OS appearance/security changes. Keep
solver capacity, precision and work limits unchanged. New tests must reproduce
real regressions or verify user workflows. Run relevant focused checks during
implementation, then full quality gates for release candidates; do not repeat
unchanged full suites without a new concern.

Initial state: main 29f82f96, clean; no extra ELab branches or worktrees. SDK elab4
pins upstream f9dee54 and compatibility source 2c349fc. The /256 tooltip-retirement
fix and its complete passing suites remain preserved. Scope and progress are
updated here as each ordered item is completed.

## Item 1 result

The full-window preview appeared to omit the circuit, but the stored raw native
frame contains the complete geometry. Exact schematic and scope pixel regions
are identical before/after the purported disappearance; a full-resolution crop
shows the components and labels. Native pan changes the expected pixels, and
focus, scroll, inspector/theme switches and a reversible window resize keep the
circuit available. The original window dimensions were restored. This is an
inspection/preview artifact, not a demonstrated application rendering defect;
no speculative renderer modification is made. The abandoned diagnostic overlay
was built only under ignored build paths and was never run or shipped.

Receipts/captures are staged in build/reliability-five. The new native checking
method compares actual saved pixels and crops instead of relying on the large
preview. This resolves the earlier observation within the tested cases without
claiming exhaustive GPU/platform qualification.

## Item 2 result

The old SDK reproduces visible help retained during scroll and pending help
opening at (0,10) after clipping, with numeric value still 45. SDK elab5 cancels
hover on scroll, rejects stale queued callbacks by origin and refuses delayed
help for invisible widgets. Visible/pending, fresh valid help and unchanged
number tests pass; an explicit old-callback replay cannot steal a new same-widget
hover. Framework portable/ordinary-driver race, scoped rendered lifecycle/menu/
focus/number tests, core queued-hover race and ordinary/tagged vet pass.

Clean /257 native verification scrolls PWM Duty out of view with no displaced
corner help. Accepted components, frequency/duty, electrical revision and run
remain unchanged. The /256 and qualified alpha bytes are preserved. The same
owned /257 fixture is reused for the next keyboard workflow.

## Item 3 result

The owned /257 app passes actual keyboard LED placement/wiring, rejected zero
resistance and correction, Undo/Redo, live DC calculation, native save and
post-dialog focus. The matching packaged agent freshly reopens the exact four-
part/four-wire file with 2000 Ω and 1.476723549 mA, clean. The earlier doubled
filename and dialog-focus observations do not reproduce in this candidate.
Native RC stepping reaches 3.1606 V at 10 ms; keyboard cursors measure 10 ms.
An untouched example replaces without a spurious prompt, while an edited
restoration shows Save/Discard/Cancel and Escape keeps its graph.

The existing LED/RC/NPN, project-contract and independent electrical suites
pass across in-process, headless stdio/socket and native socket adapters. All
45 owned external processes exit normally with code zero. Detailed receipts
are in docs/validation/reliability-five/workflow. This does not claim participant,
screen-reader/IME or clean-account acceptance. Full release gates run next.

## Item 4 result

ELab 0.194.5 /260 is an updated unsigned macOS arm64 download with the matching
agent, all 62 existing paused example projects and SVGs, model notes, Vellum
notices, usage and rollback instructions. ZIP SHA-256:
4132046779dd14442187e37baa426de55ccff4c9f5f7b587d691bc31c5ae857a.
All 374 archived files match a macOS extraction's bytes and modes, including the
ad-hoc integrity seal. The copied runtime passes complete native smoke and
extended public suites with source/development paths and outbound IP denied.
The extracted agent reopens all 62 examples with exact graphs, paused clean
state and accepted model startup. The /205 ZIP is unchanged. Owner walkthrough
and separate-account/Gatekeeper testing remain explicitly deferred. No upload
is performed. Evidence: docs/validation/reliability-five/release-260.

The source/full-suite and later smoke-only correction scopes are kept separate
in the receipts. The native harness now navigates the appropriate inspector and
observes stable native publication; semantic bulk fixtures do not flush editors
that have not mounted. Mounted and actual keyboard settlement remain covered.
The final ordered item begins from this preserved release.

## Item 5 result

ELab 0.195.3 /264 completes the examples browser using the existing 62 templates.
Start here, 11 topic/navigation choices, global word search, actual schematic
thumbnails and larger previews include short experiments and model limits.
The five added public browser/query commands bring discovery to 187 IDs;
component kinds remain 53. Protected `example.load`, file/model versions,
ordinary export bytes and simulation limits are retained.

Native checks cover all four themes, the compact 1053×802 window and larger
window, category popup, Return search/open, Clear/no results, Escape/focus,
mouse wheel and scrollbar dragging. Exact public snapshots are unchanged by
wheel/drag. Native Cancel keeps the edited circuit and chooser; Save and continue
writes R1=2000 Ω before replacement, and final Save/Discard restore component
Search focus. Held callbacks refuse later agent context. The same matching
packaged headless/native agents return identical 62 SVG/source hash pairs.

The complete quality matrix passes: 866 portable, 336 ordinary-desktop and
459 rendered-desktop top-level tests (1,945 / 836 / 1,414 including subtests),
with race checks, lint, imports, both desktop vet configurations, Python, PRD
and build. Scoped source receipts distinguish the unchanged portable source
from final desktop compilation. The copied exact package passes native smoke
and extended four-adapter public electrical suites with development-file and
outbound-IP denial; all 102 external processes exit normally.

The unsigned final ZIP is `build/releases/ELab-0.195.3-macos-arm64.zip`,
38,746,820 bytes /375 files, SHA-256:
`31fa3386d57850e5d68ead285d65f2c8a7319ef1e4866ba3e6f070ea76c85fe1`.
A macOS extraction matches every file's bytes/mode and integrity seal; its
matching agent reopens all62 included paused projects cleanly with exact graphs
and accepted startup. The /260 and /205 ZIP hashes remain unchanged. No upload
or push is performed. Owned test apps close normally; owner projects/sessions
are preserved. Recorded owner/clean-account deferrals remain in force.

All five ordered items are complete. No further component or optional feature
work is started. Evidence and original failures are indexed in
[the qualification record](../docs/validation/reliability-five/README.md).
