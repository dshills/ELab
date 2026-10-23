# UI mouse and scrolling maintenance checks

October 23, 2026. Scope: existing ELab controls, mouse gestures and scrolling.
No electronics features, device models, public commands or document formats are
added. The starting /202 document was clean and empty. Built-in examples and
owned temporary edits/recordings provided the test content. Synthetic circuit
edits were undone or discarded. Autosave was temporarily disabled for the drawing/recording tests and restored
after cleanup. A preliminary sweep was also run with the normal autosave setting;
its owned unsaved result was discarded. The existing 20-component recovery copy
remained listed with its original timestamp and was never opened or dismissed.

## Findings and fixes

1. **Short live/recorded scope drags lost their movement.** A native queue can
   coalesce a quick drag to MouseDown/MouseUp before any slide classification,
   or to SlideStart/SlideStop without SlideMove. The scope now applies both raw
   and classified final release positions using its existing original-document/run/recording/
   cursor/revision guard. Repainting its own successful cursor commands is
   allowed; an intervening actor or selection change still retires the gesture.
2. **Short divider drags could do nothing.** Vellum's controlled split handle
   also listened only to SlideMove. It now accepts raw MouseUp as well as
   SlideStart/SlideStop positions, retaining the original primary press.
   Regression tests send actual manual-window input with a single movement and
   a different final release, in both orientations, retaining the existing
   bounds and controlled owner callbacks.
3. **Secondary clicks edited measurements.** A right-click on the scope moved
   cursor A. Plot/canvas editing now requires the primary button. The macOS
   canvas bridge passes Control-click to the ordinary secondary-button path.
   This does not change native text-field context menus or other macOS controls.
4. **A completed placement could show a false stale-input warning.** Placement
   completes on down; auto-fit may advance the revision before mouse-up. A
   non-dragging release now clears that finished gesture without dispatching a
   command or reporting a stale edit. Actual drag commits still use their guard.
5. **Sweep drags lost their final point.** The discrete sweep cursor now handles
   start/movement/release through a retained gesture context, including the
   native bridge's release packet across its own cursor repaint. Another actor, document, sweep or trace
   change cannot be adopted by the release. Point selection remains discrete
   and uses the same sweep.cursor.set command available to external agents.

## First pass through /202

| Surface | Observed result |
| --- | --- |
| Inspect rail | Wheel and scrollbar thumb reached lower guidance and returned to the top. |
| Simulate rail | Wheel and thumb traversed probes, statistics, verification, sweep and circuit information. |
| Document rail | Wheel and thumb reached recovery/help and returned to output controls; recovery copies were not opened or dismissed. |
| Scope dock | Wheel over the plot exposed cursor fields; thumb returned to the toolbar. |
| Scope cursor | Click placed A; a short drag stayed at its original time; right-click incorrectly moved it. Fixed as above. |
| Divider | A short drag kept the accepted 0.65 fraction. Fixed as above. |
| Preview | Settings wheel/thumb, margin increment/decrement, captured-page layout and Back to circuit worked. No print job was submitted. |
| Native Save panel | Toolbar opened it; Cancel returned to the same circuit without writing a file. |
| Potentiometer slider | A drag changed the wiper from 50% to about 75.1%; Undo restored it, then removed the synthetic component. |
| Recorded viewer | Next and sample-index plus moved from sample 0 to 1 to 2; plot scrolling revealed cursor controls. |
| Sweep | Mouse-focused point entry accepted 7 points; Run completed; click selected point 3. Short drag stayed there. Fixed as above. |
| Schematic | Pan moved the drawing; wheel changed zoom from 43% to 172%; Fit returned to 43%. |
| Menus and decisions | Example/File menus, Save/Cancel and the owned-data discard decision worked. |
| Window geometry | Tests used a 2214×1264 starting window, a compact 1098×804 window and a later 1221×969 window. Automated title-bar/size restoration did not prove an exact original geometry restoration. |

Saved captures distinguish baseline failures from final packaged-app checks in
[ui-input-203](ui-audit/ui-input-203/README.md). Unit/manual-window tests cover
short drags, final release positions, secondary clicks and stale agent admission.
The final validation receipt will identify the exact source, SDK and app tested.
No claim of complete OS input, accessibility, IME or every possible device is made.

## Native verification and rejected preliminary candidate

The first /203 candidate passed source, race and mounted checks but failed the
actual fast scope drag: A remained at 4.460052986512524 s. It is preserved as a
preliminary candidate, not the current qualified release. This exposed the raw
MouseUp case absent from the first classified-drag regression.

The /204 preview moved scope A from 0.5523362235067438 s to
894.340077071291 ms with a short drag. Right-click and Control-click left it
unchanged. A divider drag changed 0.65 to 0.47680414. A separate sweep check
exposed the native frame-token rejection across cursor repaint; after repair,
a short drag moved point 2 (0.833333333 V) to point 6 (4.16666667 V).
Secondary clicks left point 6 selected. On the LED template, right/Control-click
in placement mode kept four components; a primary click placed exactly one fifth
component without a false stale-input warning. Undo restored the clean template.
The native bridge regression now exercises its actual owner delivery after
repainting, both with its own cursor write and an intervening agent trace change.

Final packaged /204 checks and exact artifact hashes are recorded separately in
`docs/validation/ui-input-204`. Preview evidence does not qualify a different binary.
