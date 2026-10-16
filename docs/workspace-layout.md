# Adjustable schematic and scope workspace

When a live or recorded scope is expanded, drag the divider between the schematic and scope to choose how much space each gets. The default assigns 65% of the central workspace to the schematic. The divider accepts 35–85%, leaving both panes available. With divider focus, Up/Down changes the share by two percentage points and Home/End selects the limits. The accessible divider exposes the same accepted value.

Scope contents fill the available width and scroll vertically when their controls do not fit. Wheel input over the plot scrolls its containing scope pane; it never zooms the trace. Collapsing the scope removes the divider and keeps the chosen fraction for the next expansion. Print preview and DC sweep hide the source/scope split while retaining its preference.

## Shared command

```json
{"command":"workspace.layout.set","params":{"schematic_fraction":0.8}}
```

This fragment omits the ordinary request envelope, mutation guards and retry key. `schematic_fraction` is a required finite number from 0.35 to 0.85. Missing/null/text/out-of-range fractions and unknown fields refuse before mutation. `project.inspect.workspace_layout` exposes the accepted fraction; detailed command discovery explains effects, guards, errors and completion.

The preference is session-only. Example load/reset, New, file open and recovery retain it; schematic files do not import another session's layout. It changes no graph, simulation/run/time/charge, history, dirty flag, scope visibility, probes, cursors or recorded playback. The native divider uses this command with captured command revision and document generation, so a stale callback cannot overwrite a newer agent choice. Widget float32 range endpoints map to the exact declared bounds.

The command updates its preference immediately; native layout follows on a later frame. The existing `view.viewport.set` reports the resulting drawable area. [Fit mode](viewport-fitting.md) then reframes the schematic if following is enabled. Manual views remain unchanged. A headless client can report virtual canvas dimensions explicitly; the layout fraction alone does not resize an OS window or fabricate measurements.

## Evidence and limits

Tests cover layout bounds and refusal atomicity, owned/pure discovery, physical and measurement invariants, recorded cursors/playback, session/file separation, native callbacks and stale refusal. Mounted smoke checks actual canvas resizing, full-width scope allocation and scrollbar movement from the plot without changing its zoom. It preserves manual navigation and restores its own split fixture before later pixel tests. Recorded GPU assertions scroll their plot into view before demanding a fresh frame.

The public reference `scripts/workspace-layout-demo.py` runs a fresh native session first, then a headless replay using `--sizes-from`. Both inspect the same commands, measured-view transitions, recorded cursor preservation and file/session boundary. Direct native accessibility inspection sets the divider from 65% to 80% and shows the fitted NMOS example grow from about 29% to 41% at the default window. This is an improvement over the previous fixed scope allocation, not a claim that every label is readable at every size.

The fraction is a preference rather than a guaranteed minimum pixel size. Large schematics still need zooming or scope collapse for readable labels. Full responsive rail/tool layouts, complete keyboard/IME/accessibility acceptance and visible latency remain open. Callback and mounted event tests do not establish every physical pointer/scrollbar gesture.
