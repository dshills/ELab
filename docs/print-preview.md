# Captured schematic print preview

Use **Print preview** in the component rail. It replaces the drawing pane with a captured white-paper page. Preview puts page settings in the left rail and temporarily hides the live scope and summary cards to leave room for the page; returning restores those panels without changing acquisition or visibility preferences. The Output page controls supply A4/Letter, orientation, margin, grayscale/color, fit or explicit scale, and optional live overlays. **Refresh preview** captures current settings and source; **Back to circuit** releases the image. Editing remains available in the Inspector and through agent commands. A source/settings change shows “source changed; refresh to update” while preserving captured pixels.

The preview and PNG/PDF file output use the same captured schematic renderer. Preview pixels are a 150 DPI PNG; the native image uses Vellum's immutable tiled image and contain fit. PDF remains vector output. Existing **Export PDF** exports the current editor source; it does not export an older preview. This release does not submit a printer job.

## Shared commands

| Command | Parameters/result |
| --- | --- |
| `print.preview.start` | Optional `page` override; omission captures Output page defaults. Returns admission snapshot. |
| `project.inspect` / `events.poll` | Inspect `print_preview`: stable ID, queued/running/canceling/completed/canceled/failed status, visibility, source freshness, page, captured run/electrical/document identity, actor, dimensions, size, SHA-256 and failure. Completion publishes `preview`. |
| `print.preview.image` | Exact preview `id`, optional byte `offset` and `limit` (default/max 65536). Returns base64 PNG chunk, offsets, total and digest. Query is read-only and returns owned bytes. |
| `print.preview.cancel` | Exact active preview `id`; requests cancellation. |
| `print.preview.close` | Exact current preview `id`; cancels active rendering, hides the pane and releases retained bytes. An old ID cannot close a newer preview. |

Mutation revision/document guards and retry semantics are unchanged. Preview completion does not advance the command revision. `simulation.wait` retains its existing simulation/project-work semantics; use inspection/events to await preview completion. One render worker is admitted at a time, with an 8 MiB encoded-image limit and a cooperative 30-second context deadline. Closing/canceling does not admit another worker until the old worker finishes. Application close cancels and joins rendering. CoreGraphics calls can only observe cancellation at their existing checkpoints; the deadline is not preemptive.

`source_current` compares document generation, graph/experiment settings and source annotation content; default-page captures also compare current page settings. Explicit page overrides are independent of later default-page edits. Opt-in live overlays additionally compare captured display/readings, so live numeric changes can stale that preview. View pan/zoom and theme changes do not alter a captured page. Snapshot/event payloads contain metadata only, never the image. Native decoding verifies chunk identity, size, digest and dimensions off the UI owner thread, then publishes only if the same preview is still open.

The macOS cgo page renderer is required; unsupported builds return `unsupported`. Incomplete drawings are allowed. Existing explicit-scale fit errors remain explicit. No printer discovery/submission, printer selection, tiling, physical output or universal crowded-page readability claim is made.

Validation includes captured PNG/export byte parity, edits during blocked rendering, owned chunks/pages, stale IDs, cancellation and close joining, timeout classification, bounded discovery, shared native controls, hidden-canvas gesture refusal and an actual packaged native smoke. The public [demo](../scripts/print-preview-demo.py) exercises both stdio and native socket transports. Visual evidence and exact release checks are recorded separately.
