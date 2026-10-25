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
4. In progress: produce an updated unsigned downloadable release with the current
   electronics, matching agent, examples, use/model limits, checksum and rollback.
   Preserve the qualified /205 download and /256 development candidate. Test
   archive extraction and offline packaged workflows; retain declared deferrals.
5. Pending: searchable categorized examples browser with schematic thumbnails
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
