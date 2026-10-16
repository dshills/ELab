# Fitting the schematic workspace

Examples open with **Fit** enabled. The native app measures the drawable schematic area after layout and fits component bodies, terminals, labels, routed wires, pinned readings and source annotations using the existing bounds calculation. Hiding/showing the scope or resizing the window refits the circuit. The highlighted Fit button indicates this mode.

Pan, zoom, Center selection and navigation that centers an object return to manual positioning. Later workspace changes preserve that view. Click Fit to follow resizing again. Selecting an object without centering it does not leave follow mode. Editing the graph does not continuously reframe it; click Fit when new geometry needs reframing.

New schematics use manual mode. Opening a saved schematic or recovery document restores its saved center/zoom and disables following. Measured canvas dimensions remain session state. No project format changes, dirty-state changes, history entries, solver restart, transient time advance or capacitor-charge changes are caused by fitting.

## Shared commands

```json
{"command":"view.viewport.set","params":{"width_px":594,"height_px":137}}
{"command":"view.fit","params":{"follow_viewport":true}}
{"command":"view.pan","params":{"dx":20,"dy":0}}
```

These fragments omit the ordinary request envelope, mutation guards and retry key. `view.viewport.set` reports a client's drawable area; it does not resize an OS window. Both dimensions are required finite numbers from 64 to 100000 pixels. An external/headless agent can report its own virtual canvas. In the native app the mounted canvas reports its actual size, so fabricated agent sizes may be replaced by the next layout observation.

`view.fit` retains its existing one-shot behavior when `follow_viewport` is omitted or false, with omitted dimensions defaulting to 650 × 400. With `follow_viewport:true`, omitted dimensions inherit the registered canvas size, or use those defaults before a first measurement. Supplied dimensions register the size. `project.inspect` and event snapshots expose `viewport` with `width_px`, `height_px`, `registered`, `following` and optional `fit_failure`.

If a valid size would require zoom below the existing 1% minimum, the size report succeeds, preserves the previous view and exposes `fit_failure.code:"view-limit"`. Following remains enabled so a larger reported area can recover. An explicit Fit that cannot fit still refuses without changing mode/view. Invalid reports refuse before mutation.

Native reports are deferred past the current paint callback, coalesced to the latest pending size and guarded by command revision, document generation and widget identity. Stale reports retry from a later view/frame. A bounded pending delivery handles a full owner mailbox; shutdown cancels admission waiting. Reporting does not replace existing error/file messages or file-job completion identity.

## Evidence and remaining work

Core tests project actual terminals, wire routes and annotation corners into small and large canvases for all ten examples. Native tests cover deferred/coalesced publication, mailbox backpressure, stale navigation/document/widget rejection and Fit/agent parity. Packaged smoke waits for the current command/generation to reach actual mesh preparation before checking mounted scope resize and manual positioning. Animation checks also wait for initial layout publication before measuring their unchanged revision invariant.

The public reference is `scripts/viewport-fit-demo.py`. Run it against a fresh owned native endpoint, then run headlessly with `--sizes-from` pointing to that native report. It retains discovery evidence and checks fitting, preserved physics/document/history, manual navigation, refusal, one-shot compatibility and saved views.

Direct native inspection at the default window measured **594 × 137** with scope expanded, fitting the NMOS example at **18.27%**. Hiding scope measured **594 × 379** and enlarged it automatically to **50.53%**. Manual pan survived expansion, and native Fit restored 18.27%. The whole schematic fits, but labels at 18% remain too small to read comfortably. Minimum useful canvas space, responsive scope/inspector layout, general label/crossing readability and complete physical input/accessibility qualification remain open.
