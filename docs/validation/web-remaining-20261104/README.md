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
