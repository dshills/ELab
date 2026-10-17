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

Core tests project actual terminals, wire routes, enlarged native captions and source annotation corners into small and large canvases for all 18 examples. Native tests cover deferred/coalesced publication, mailbox backpressure, stale navigation/document/widget rejection and Fit/agent parity. Packaged smoke waits for the current command/generation to reach actual mesh preparation before checking mounted scope resize and manual positioning. Animation checks also wait for initial layout publication before measuring their unchanged revision invariant.

The public reference is `scripts/viewport-fit-demo.py`. Run it against a fresh owned native endpoint, then run headlessly with `--sizes-from` pointing to that native report. It retains discovery evidence and checks fitting, preserved physics/document/history, manual navigation, refusal, one-shot compatibility and saved views.

At the 0.90.0 checkpoint, direct native inspection at the default window measured **594 × 137** with scope expanded, fitting the NMOS example at **18.27%**. Hiding scope measured **594 × 379** and enlarged it automatically to **50.53%**. Manual pan survived expansion, and native Fit restored 18.27%. The whole schematic fits, but labels at 18% remain too small to read comfortably. Minimum useful canvas space, responsive scope/inspector layout, general label/crossing readability and complete physical input/accessibility qualification remain open.

The October 17 first-version repair replaces routing search envelopes with actual paths and shares canvas caption geometry between drawing and Fit. The hand-built RC fixture now fits at 42.1% in the fixed 456×158 test, versus 31.9% before repair, while retaining larger captions. Direct native checks in compact/larger windows cover both themes, brighter dark wires, agent history labels and the complete schematic tool row. The native/external clients return exactly matching views at both observed viewport sizes. [Evidence and remaining UI work](../research/ui-audit/first-version/README.md). This closes the demonstrated RC issue; complex examples, full input/accessibility and visible latency remain separate checks.

The 0.91.0 [adjustable workspace](workspace-layout.md) replaces the fixed expanded-scope allocation with a session-controlled divider and scrollable scope controls. Its default canvas is larger; the measurements above retain the original viewport-fit checkpoint evidence.

## Route cache publication repair

Concurrent misses previously appended the same route key more than once to the 512-entry FIFO. A deterministic reproduction occupied 64 slots with one route; a late duplicate at capacity evicted an unrelated route. Publication now checks membership again under the cache mutex. Numerical routing still runs outside that mutex, and callers still receive independent point slices. This repairs cache bookkeeping; it does not deduplicate simultaneous route computations or change connectivity, routing geometry or public commands.

Regression tests cover concurrent publication, duplicate publication at capacity, normal FIFO eviction and caller ownership. The full workbench race suite, focused native routing/viewport race tests, lint/Staticcheck and root/native vet pass. Controlled blocked-solver checks also pass navigation, cancel, save, undo, pause and obsolete-result rejection; they establish backend availability rather than visible latency under a real difficult solve.

On this Apple M4 Pro, `BenchmarkCapacityWireRoutes` measured the existing 70-wire capacity fixture over 20 iterations per case: **7.083 ms cold** (5,846,705 B /35,703 allocations) and **2.969 ms warm** (1,171,719 B /281 allocations) per whole drawing. Cold iterations clear only the benchmark process's cache outside the timed section. These are isolated routing elapsed means, excluding solver/render/presentation work; they do not close the visible p95 gate. The private review and canonical bundles were not rebuilt or promoted for this repair. Run the final release gate when incorporating it into the next candidate.

## Preserve value errors through layout

A value-refusal message itself reduces the canvas height. Although the native viewport reporter retained the message fields, its accepted command advanced the revision; the next view then discarded the structured refusal as stale. The reporter now continues that refusal's context through only its own accepted response. It never adopts an unrelated newer snapshot. A subsequent user/agent command still invalidates the old refusal normally, and stale/document/widget guards remain intact.

The regression reproduces the disappearing error before the repair and verifies retention plus later-agent invalidation afterward. Actual native entry of `0` keeps the valid 1 kΩ circuit and its fresh reading, with the error visible after layout settles. Correcting to `2k` clears the error and yields 1.477 mA. Focused native race tests, lint/Staticcheck and vet pass; public command mappings and the SDK are unchanged. [Checkpoint and input limits](../research/ui-audit/first-version/README.md#value-refusal-and-settled-input-checkpoint). This development repair is not yet in the handed-off review bundle.
