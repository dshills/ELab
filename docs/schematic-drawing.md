# Schematic drawing and clean Vellum UI

Persistence has since shipped; see the [save/load update](persistence.md). The scope below records the drawing slice before that update.

Implemented October 13, 2026. This extends the [first LED slice](implementation.md) with blank schematic construction and a cleaner native editor. The portable graph and command service remain independent of Vellum. The current catalog contains 29 commands shared by users, headless clients, and external agents.

## Working behavior

The editor has a neutral white/slate layout with teal for active tools and live status. The left rail holds four component tools and a short first-circuit guide. The center contains a spacious dot-grid sheet, compact Select/Wire/Pan controls, zoom/Fit, context hints, and diagnostic feedback. The right inspector exposes selection, exact values, presets, position/rotation, and deletion. Exact placement appears while placing; named terminal connection controls appear while wiring. Separate bottom cards show LED current, voltage, and power, with paused/stale status identified separately.

Users can place a source, resistor, LED, and ground by clicking the sheet, wire terminal dots, select and drag components with attached wires, rotate, delete, enter values such as `2.2k ohm`, and undo/redo. Keyboard/semantic placement and named terminal connection paths are available alongside pointer interaction. All committed edits call the same `Service.Dispatch` used by agents. One drag produces one undoable move; a position precondition prevents that drag from overwriting a concurrent move.

The graph stores stable components, named pins, explicit wires, and derived nets. Connectivity does not depend on geometry. Moving/rotating preserves the electrical run; deleting a part and its wires is one recoverable edit. Missing connections, reversed LEDs, extra unsupported parts, and shorts produce diagnostics. A prior successful result remains explicitly stale after invalid/incomplete electrical changes.

On macOS, a small app-local Cocoa pointer adapter uses the event's own location, correcting a discrepancy observed in the pinned native toolkit's cursor-location lookup during targeted mouse input. It monitors only this app's sheet and feeds the same canvas gesture handlers. Vellum sources were not changed. Other platforms retain the toolkit input path and are not qualified by this validation.

## Validation and evidence

- Root and desktop race tests and `go vet` pass. Tests compare complete construction/value editing through actual UI callbacks versus direct agent commands, and check graph connectivity, deletion/history, validation, alternate supported topology order, and stale-result handling.
- Native smoke passes: construct from blank through UI controls, verify rendered circuit pixels, edit resistance, and verify the new live result.
- Manual macOS input passes in the redesigned window: place all four parts, connect all four wires by terminal clicks, inspect the resistor, apply `2k ohm`, and drag it from (0, −110) to (40, −130) with its wires retained. Current changes from 2.916796577 mA to 1.476723549 mA.
- The same construction/edit/retry/delete/undo/pause/resume/selection/view workflow passes through headless stdio and an external agent connected to the visible redesigned app.

Saved reports: [headless drawing](validation/headless-drawing-led.json), [desktop agent drawing](validation/desktop-drawing-led.json). Saved final UI: [live schematic screenshot](validation/clean-schematic-ui.jpg).

## Remaining scope

This is a drawing editor with a deliberately bounded series LED DC solver, not the full MVP solver. General junction/wire waypoint editing, more component models, transient simulation/scopes, persistence/recovery, export, atomic batches/events, screen-reader qualification, performance percentiles, and other desktop platforms remain to implement. The current app is a private local development bundle. See the [PRD](../research/07-product-requirements.md) for the full acceptance contract.
