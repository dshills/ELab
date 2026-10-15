# Discovering schematic editing behavior

`capabilities.list {"command":"component.move"}` now describes the semantic rules for fourteen schematic operations. The same commands run through user controls, native/headless agent adapters and supported atomic edit batches. These details add no private user-only behavior or numerical model.

| Commands | Key rules |
| --- | --- |
| `parts.search.set`, `editor.value.set` | Up to 128 UTF-8 bytes; raw session text. Search matches every whitespace-separated token across library names/aliases/pins/units/models; an inspector draft does not apply a component value. |
| `component.move`, `component.rotate` | Destination bounds ±10000, nearest 10-unit snapping with ties away from zero; occupied component centers refuse. Rotation adds 90° clockwise. Geometry preserves pin identities, connectivity, accepted results, electrical revision and run. |
| `component.duplicate`, `object.delete` | Clone configured component fields without wires/live state; omitted/null position tries bounded nearby centers. Net labels keep their name and named-net association. Deleting a component atomically removes incident wires. |
| `component.select` | Existing component or wire ID; empty clears. Clears explicit measurement focus and restores the selected committed value draft. |
| `drawing.tool.set`, `drawing.position.set`, `drawing.cancel` | Session planning only. An occupied placement draft is permitted until Place. Tool/cancel clears `wire_start`, while named From/To choices remain. Place + net-label clears the label draft. |
| `wire.begin`, `wire.terminals.set` | Existing exact terminal references; From/To each require a string, with empty clearing a choice. Planning does not connect anything. Equal choices can be planned but cannot be connected. |
| `wire.connect` | Distinct existing endpoints, unused valid ID or automatic W designator, ≤300 wires. Direct duplicate pair refuses in either direction; a different pair on the same net may form an explicit copper cycle. Success clears wire-start/From/To choices. |
| `wire.junction.insert` | 1–8 unique connected routes, capacity for one component and one extra wire per route. Multi-route insertion requires a position; one route can choose a bounded free interior candidate. Only named routes join. |

Detailed inputs include dynamic inventories, current pin-reference bounds, ID lifetime, positions, capacities, state/effects, known errors and completion/observation rules. The normal discovery catalog and its 64 KiB bound remain unchanged. `component.place`, `component.set` and `capacitor.initial.set` retain their existing [value contracts](component-values.md).

## Exact move preimages

`previous_position` is an optional additional gesture guard. Omitted/null disables it. A supplied object requires both numeric `x` and `y`, and must exactly equal the currently stored position before any snapping; a different complete position returns `conflict`. Malformed or missing coordinates return `invalid-params` without moving the component or changing state. This now matches the existing pinned-reading move guard and the already declared point schema.

Before this repair, `{ "previous_position": {"x":0} }` could supply a missing Y as zero and unexpectedly pass for a component at the origin. Four failing-before/passing-after cases cover empty, missing-X, missing-Y and null-coordinate objects. A late malformed move also rolls back an entire atomic batch, including staged ID reservations. Valid native gestures already supply both coordinates.

## State and geometry distinctions

Planning, searching, selection and value drafts preserve graph, physics, undo history and document dirty state. An actual graph change is an ordinary undoable document edit. Electrical changes use running restart / paused-stale behavior; graph normalization may refresh a selected value draft. Invalid input preserves all prior valid public state.

Current terminal references come from `snapshot.components`: concatenate the component ID, a dot, and one of its exact terminal names. Crossings, proximity and overlapping drawing positions never connect nets. Junction insertion additionally requires a free snapped center within half a grid step plus 1e-9 of every named route, at least one grid step from each named endpoint. An explicit copper cycle can make individual wire-current splits indeterminate; it does not invent a unique current split.

Optional empty/null automatic IDs and optional kind inheritance are described explicitly. These are distinct from required endpoint/target fields, whose null values refuse. Detailed schemas describe the current commands; server validation and current inventories remain authoritative.

After building, run `python3 scripts/schematic-contracts-demo.py --output build/schematic-check.json --artifacts build/schematic-check-new`. It discovers all fourteen contracts, constructs an LED from blank, tests preimages/snapping/planning, geometry, clone/delete recovery, explicit junctions and same-net cycles. Reports retain command contracts and registry/full-response digests; one full discovery witness is stored separately. Use `--connect` only for an owned blank session; the demo replaces and closes it.

Coverage remains partial for other command families, complete native bindings/typing/IME/accessibility and release acceptance. These contracts do not close those gates.
