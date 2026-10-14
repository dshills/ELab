# Collapsible scope dock

Use **Hide scope** in the live or recorded scope toolbar to reclaim schematic space. The collapsed strip retains simulated time or recorded position and a **Show scope** action. Collapsing does not pause the live simulation or recorded playback, hide numeric meters, clear measurements, or discard data.

Users and agents share `scope.visibility.set {"visible":false}` and `{"visible":true}`. Snapshot `scope_visible` reports the session preference, initially true. Ordinary command guards/retry rules apply. The preference survives project replacement during the session, and applies to live and recorded scopes; it is not embedded in circuit files or restored across launches. Inspector measurement/probe settings and exports continue to work while collapsed.

Live samples, recorded position/playback, selected probes, cursors/windows/statistics, view center/zoom, history, document dirty state, run identity and electrical revision remain independent of plot visibility. Invalid or missing booleans are refused. A change publishes view/preferences observation hints. Retired native canvases are released, and reopening creates a fresh mounted plot with the same data. Hidden pointer gestures and callbacks from older mount epochs cannot set a cursor on a different mounted plot.

Actor-aware undo/redo now use compact “agent” or “other” captions while retaining the full actor and action in accessible help; their captured history-tip guard and explicit other-actor acknowledgment remain unchanged. Common ID/cursor/probe/playback/input-mode parameter schemas use local registry references, avoiding repeated definitions without increasing catalog or transport limits.

[Validation](validation/scope-visibility-validation.json) includes core/native state invariants, hidden acquisition/playback, native GPU retirement/remount and [compact-window screenshots](../research/ui-audit/scope-collapse/README.md). Scope resizing, persistent workspace layout, automatic Fit, tiny-zoom legibility and full keyboard/screen-reader qualification remain separate follow-ups.
