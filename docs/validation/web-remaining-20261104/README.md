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
