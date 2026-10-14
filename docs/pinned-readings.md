# Pinned live measurements

Choose **Pin a reading…** in the general right inspector (available for DC and transient circuits) to keep any available voltage or current visible on the sheet. A probe can be pinned even if it is not plotted. Each card shows signed units, explicit voltage reference or current direction, and live/stale/unavailable text. The Inspector exposes the same text and reason through ordinary semantic controls. Drag a card in Select mode, or edit its X / Y fields. Remove it from the Inspector. Eight pins are supported; positions snap to the 10-unit grid inside ±10000 units. Fit and SVG include the complete card bounds and reference labels.

Pins always measure the current **live** experiment, including when the scope is showing a recorded result. They never change to a captured record’s values. Snapshot fields carry the accepted result’s run, electrical revision and physical sample time. DC values come from the accepted steady-state calculation; transient values come from the current accepted sample. Values use SI units in JSON; display text uses V or mA. Positive current follows the descriptor’s named direction, including source delivery and NPN emitter output. Negative values retain their sign.

The numerical kernels supply immutable full-probe frames. Pin reads do not launch a second solve. DC results retain their original compiled graph, so a previous node map cannot be interpreted against new wiring. A paused electrical edit leaves old values visibly stale. Deleting the source removes its value and reports unavailable; undo restores the association. A failed/incomplete run cannot masquerade as a new zero measurement. Recorded-view selection does not affect live pins.

## Shared commands and state

| Command | Parameters | Result |
|---|---|---|
| `measurement.pin.add` | `{"id":"v:C1","position":{"x":350,"y":250}}` | Add an available probe; reject duplicate/unknown IDs and the ninth pin |
| `measurement.pin.move` | `{"id":"v:C1","position":{"x":400,"y":300},"previous_position":{"x":350,"y":250}}` | Move on the grid; optional previous position refuses intervening gestures |
| `measurement.pin.remove` | `{"id":"v:C1"}` | Remove a pin, including an unavailable source |

All use normal revision guards, successful-request retry keys and actor-aware history. A native drag previews geometry locally and commits one shared move on release; one undo reverses the complete gesture. Replacing the project cancels old gestures. An intervening agent move cannot be overwritten by the old pointer release. Numeric fields use the same guarded move command.

`snapshot.pinned_readings[]` includes `definition` (ID/position), `probe` (label, SI unit, positive/reference direction), optional `value`, `time_s`, `run_id`, `electrical_revision`, `state`, optional `reason`, `display`, and `reference_display`. States are `live`, `stale`, or `unavailable`. Missing-source descriptors carry identity/unit without inventing current terminal associations. Responses and cached retry outcomes own their value pointers.

Pin edits preserve time, charge, simulation run/electrical revision, edit selection and graph. They are undoable presentation edits and mark `document.preferences_dirty`; this contributes to the overall unsaved-change guard. `schematic_dirty` continues to mean graph/experiment changes. Undoing to saved preferences becomes clean.

## Persistence and output

Version-4 `.elab` files optionally store `pinned_readings.items`, containing only stable probe IDs and positions. Strict validation rejects null/missing fields, extra fields, duplicate/invalid IDs, non-grid/nonfinite/out-of-range coordinates and excess pins. Missing-source pins remain portable as unavailable preferences. New/load/example replacement installs that document’s pins or clears them. Reopen starts fresh simulation; recorded values and pin values are never trusted from this field. Recovery captures even preference-only dirty changes.

Older builds lacking this optional version-4 extension refuse the field instead of dropping it. Files without pins omit it and remain compatible with previous version-4 readers. This follows the document’s existing optional model-extension policy; it is not a promise that all older binaries read every new version-4 extension.

SVG renders both visible card lines at saved world positions and includes the exact exported measurement/status provenance in metadata. It preserves the source project. Power, arbitrary derived expressions, and pins of captured recorded values remain outside this slice; current/voltage display preferences, full keyboard selection, screen-reader qualification and visible latency remain separate acceptance work.

## Validation

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/pinned-readings-demo.py --output docs/validation/headless-pinned-readings.json \
  --artifacts docs/validation/pinned-reading-artifacts
# Use only an owned development session: replaces its circuit.
python3 scripts/pinned-readings-demo.py --connect build/pinned-readings.endpoint.json \
  --output docs/validation/desktop-pinned-readings.json \
  --artifacts docs/validation/desktop-pinned-reading-artifacts
```

Independent RC equations yield 3.160602794 V at 10 ms, then 2.859831672 V / −0.285983167 mA after 1 ms of discharge. Core tests cover all DC/transient kernels, geometry preservation, original-graph stale frames, source deletion, undo/redo, dirty/save/reopen, preference-only recovery, eight-pin bounds, malformed documents, retry isolation and recorded-view separation. Native tests exercise pin/numeric controls, one-commit drag and conflict/project-generation protection. Built/packaged smoke reads fresh GPU card/background/text regions for both RC pins. This is engineering evidence on the current workstation, not a participant, complete accessibility or distribution qualification.

The inspection update adds direct **Pin** actions beside selected component/terminal/net/probe readings and fixes general DC pin-selector visibility. Shared reading formatting now lives in `workbench/readings.go`; pin JSON fields and persistence remain compatible. See [measurement inspection](measurement-inspection.md).
