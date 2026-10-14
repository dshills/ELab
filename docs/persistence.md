# Portable schematic persistence

The current writer now uses version 2 and saves transient settings. Version 1 remains readable and migrates to default settings in memory. See the [RC update](rc-transient.md). The record below describes the original save/load slice.

Implemented October 13, 2026. The desktop and portable application service now save and reopen circuits through `project.save` and `project.load`. The shared catalog has 31 commands.

## File format and behavior

`.elab` files are readable JSON with `format: "elab.schematic"`, `version: 1`, the component/wire `graph`, saved `view`, and embedded `led_model` parameters. Built-in ideal source/resistor/ground versions and generic LED metadata are validated against the supported library. Future/unknown versions are rejected explicitly. The file is self-contained for this supported circuit slice; it does not store OS paths, simulation results, session state, history, or agent credentials.

Saving writes a private temporary file beside the destination, flushes it, and atomically publishes the complete file. New paths are published without clobbering a concurrently created destination. Existing files need explicit overwrite unless they are the currently tracked, unchanged file. A SHA-256 baseline detects outside edits before an ordinary Save. Directory metadata is flushed where supported. File size is bounded to 256 KiB; paths must name regular files, and parent directories must exist. New files use owner-only permissions. No automatic backup, autosave, or crash-recovery journal has shipped yet.

Opening reads and validates before changing the session. It restores stable IDs, positions, rotations, units/values, wiring, and view, clears tool/selection drafts, marks the restored graph clean, and resumes a fresh live calculation. Incomplete or unsupported but structurally valid drawings are worth saving and reopen with diagnostics. Invalid files leave the existing document intact. Opening another document invalidates pointer gestures captured against the previous document generation.

Dirty state compares the complete graph to its saved baseline, so undoing to that baseline becomes clean. Navigation/selection/pause are session state and do not make the circuit dirty; an explicit Save also captures the current view. Loading a changed graph adds one undo edit. Undoing that load restores the graph while retaining the newly opened file association; a subsequent Save writes that current path. New detaches the current path. Save itself does not add a history entry.

New, Open, example loading, and close protect unsaved changes. The desktop offers Save and continue, Discard changes, or Cancel; agents get `unsaved-changes` and can save first or supply `discard_changes:true`. File dialogs run on workers through Vellum's native services; all actual reads/writes call the window-independent shared dispatcher. Dialog/prompt state blocks background desktop drawing commands while external agents retain the same explicit semantic API.

## Validation

- Root and desktop tests with race detection and `go vet` pass.
- Model/graph/view round trips preserve IDs, exact 2.2 kΩ values, moved and rotated components, wires, and numerical live results in a new process.
- Tests verify save-point/undo/redo dirty state, protected New/Open/close, Save and continue, canceled native-picker callbacks, incomplete drawing loading, and old pointer gesture invalidation.
- Invalid/truncated/future-version files, bad terminal references/model metadata/canonical fields, oversized data, nonexistent directories, existing destinations, external edits, and link destinations are refused without corrupting the current drawing. Successful retry keys do not rewrite the file.
- Actual macOS Save completed into `test.elab`; the file contained all four parts and four wires. Reopening it through the native Open picker restored the live 2.916796577 mA operating point. The native unsaved-change prompt blocked background canvas input; Cancel preserved the drawing, and Discard during Open restored the saved file. An external agent then loaded that native-saved file into the final desktop build with clean state and matching results.
- The native drawing/value smoke check passes after integration. Packaging remains a private unsigned development build.

Evidence: [headless/new-process workflow](validation/headless-persistence.json), [desktop agent workflow](validation/desktop-persistence.json), [native save observation](validation/native-persistence-observation.json), [native reopened schematic](validation/schematic-persistence-ui.jpg), [unsaved-change prompt](validation/unsaved-schematic-prompt.jpg), and a [portable moved/rotated fixture](validation/persistence-led.elab). Run `scripts/persistence-demo.py` for repeatable checks. See [agent protocol](agent-api.md) for exact command parameters and refusal codes.

Remaining work includes autosave/recovery, migrations as new models ship, recent-file/document management, collaborative file conflict merging, general solver/transient capabilities, exports, and qualification on other operating systems and filesystems.

## Version 3 probe persistence

The next workflow update writes version 3 documents with `scope_probes: {ids:[...]}`. Save captures the valid selected channels; opening validates all IDs against the graph before restoring them. Files from versions 1/2 still load. New versions fail clearly in older apps. Cursors, scope time windows, sample history, and artifacts remain session state; load starts a new live run. Probe selection alone does not mark the graph dirty, like viewport navigation, but an explicit Save captures it. See the [next-ten record](../research/09-next-ten-implementation.md).

Version-4 documents may include `pinned_readings: {"items":[{"id":"v:C1","position":{"x":350,"y":250}}]}`. The optional extension stores only up to eight stable probe IDs and snapped card positions; measurements/status/solver state are never accepted from disk. Current and missing-source IDs are validated, so deleting a source can preserve its unavailable pin across save/reopen. Newer readers continue to load versions 1–4; older builds that do not recognize this optional field reject it instead of silently dropping the preference. Files without pins omit it. Pin add/move/remove affects `preferences_dirty` and the document dirty guard, independently of `schematic_dirty` and `recordings_dirty`. Undoing to saved pin preferences becomes clean. Save/load/recovery capture and restore preferences; a reopened valid circuit starts fresh and updates values from the solver. See [pinned readings](pinned-readings.md).
