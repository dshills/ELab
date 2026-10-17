# Schematic navigation

The Inspector's Previous, Next and Center selection controls use the same public commands as agents. When the schematic has keyboard focus, Option + Left/Right selects the previous/next object; Option + Home/End selects the first/last. These keys do not replace arrows in text fields. Ordinary canvas arrows retain their existing move/pan behavior.

`selection.navigate` requires `direction` (`next`, `previous`, `first`, `last`). Optional `kind` is `all` (default; an empty string also means all), `components` or `wires`; optional `center` defaults to false. Order is current component order followed by wire order, matching the object selector. Next/previous wrap. With no matching current selection, next chooses the first object and previous chooses the last. Empty inventories refuse with `unavailable`.

Navigation selects a stable object ID, clears explicit measurement focus, returns to the select tool, clears an unfinished wire gesture and normalizes the value draft for the selected object. It does not change components, connectivity, undo history, dirty membership or simulation state. Centering preserves zoom. A component centers at its placement point; a wire centers at half its routed path length. View centers remain within the persisted ±1,000,000-unit bounds.

`view.selection.center` centers the current selection. Optional `id` must match that selection, otherwise `conflict`; no selection returns `unavailable`. The Inspector captures this ID along with the normal command revision and document generation guards. Canvas navigation captures those guards at key-event admission; delayed callbacks cannot navigate a later document or intervening command. It refuses input during file decisions, hidden/replaced schematic views, active pointer dragging or shutdown.

```json
{"command":"selection.navigate","params":{"direction":"next","kind":"components","center":true}}
{"command":"view.selection.center","params":{"id":"R1"}}
```

These are command/parameter fragments; transport requests also need the usual version, request ID, mutation key and revision precondition.

Fit includes component bodies, labels, terminals, source annotations, pinned reading cards and the actual deterministic wire routes. Earlier builds reserved the router's entire search area, leaving excessive whitespace in compact windows. Routes are prepared once per fit, using the existing complete-graph cache; a bounded search then accounts for captions that enlarge at smaller zooms. It preserves the maximum fit zoom of 2 and minimum zoom of 0.01. When a schematic cannot fit at that minimum, `view-limit` refuses without changing the view; navigate to individual objects, move them closer or enlarge the viewport.

The native schematic enlarges part captions toward one bitmap source pixel per screen pixel, capped at twice their original world size. Source annotation spacing grows with its text. Default dark wires are brighter and schematic strokes retain at least 1.25 screen pixels. Large overviews can still require zooming for detail; this does not promise readable labels at every supported zoom. The zoom readout is in the footer, leaving the compact Select/Wire/Pan/Esc/sweep/Fit row fully available.

The full physical keyboard focus path, screen-reader operation and custom mesh accessibility remain unqualified. Vellum's named Canvas2D objects do not apply directly to ELab's MeshCanvas extension. Callback tests and packaged rendering checks do not establish those acceptance gates.
