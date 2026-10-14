# Source settings on the schematic

Sine, finite-pulse and ideal-pulse sources now show their configured excitation beside their symbol, using the same owned annotation data in native drawing and SVG/PDF/PNG output. Agents read `snapshot.source_annotations` through `project.inspect` or ordinary mutation responses; source edits still use the existing shared waveform and transient-setting commands.

| Source | Visible configuration |
|---|---|
| Sine | Peak amplitude (`Vpk`), frequency, offset and phase in degrees |
| Finite pulse | Low/high voltages, high/low phase lengths and rise/fall durations |
| Ideal pulse | Low/high voltages, ideal-edge identity and global high/low durations |

Finite-pulse phase lengths include rise/fall respectively, as in the existing source model. Labels use six significant display digits and `s`, `ms` or ASCII `us`. Exact source parameters remain in component/settings snapshots and saved projects. SVG metadata, PDF Subject and PNG Description/XMP also retain exact captured source components/settings alongside their displayed annotations.

Each annotation contains `id`, `position`, owned `lines`, and `state` (`configured`, `current` or `stale`). A fresh primary source includes its applied `input_mode` and any `pending_input_mode`. A paused Hold request stays pending until the next accepted sample; the caption does not claim it is already applied. Applied holds and pending changes appear on separate lines. After a paused electrical edit, the current configuration is shown with **configured | live stale**, without relabeling the old held reading as current. An incomplete/not-yet-simulated drawing shows configured data without claiming a current source state.

Annotation lines are horizontal and right-aligned beside the source, regardless of symbol rotation. Native bitmap glyph metrics and CoreText glyph advance determine their actual text placement; shared conservative bounds protect exports and Fit. Source masks sit below symbols/pins, preserving the prior nearby-symbol repair. **Fit** includes the complete annotation block and pinned readings while preserving physics, history and dirty flags. RC and rectifier preset views were adjusted to keep their initial source captions in view.

Exports capture owned annotation lines, global timing settings, graph and applied/pending/stale identity at admission. Later source edits can continue without changing that captured output. Default white/grayscale output still omits measurement overlays; source configuration and its state label are part of the schematic itself. This is a live/configured schematic, not a recorded-scene export or printer submission.

The [validation receipt](validation/source-annotations-validation.json), [headless](validation/headless-source-annotations.json), [native](validation/desktop-source-annotations.json), [render inspection](validation/source-annotations-render-inspection.json) and [contact sheet](validation/source-annotations-contact.png) cover sine, finite pulse, global ideal pulse, applied hold plus queued change, and stale configuration. All annotation lines appear in extracted PDF text and within page bounds; native/headless SVG/PNG bytes and PDF pixels match. Visual inspection covers all five pages and a full-resolution sine page. The smallest reference text was about 7.04 pt, not a minimum-font guarantee for arbitrary layouts.

Core tests verify units/wording, snapshot ownership, queued/applied/stale distinctions, unchanged physics during Fit and original timing/hold capture while later edits occur. Native field/mesh/Fit tests consume the same lines. A raster test checks actual end-anchored glyph placement. Existing symbol-layer, capture, source/history and numerical regressions remain active. No source model, solver tolerance, numerical quota or simulation command changed.

Reproduce the output workflow after building:

```sh
python3 scripts/source-annotations-demo.py \
  --output build/source-annotations-check.json \
  --artifacts build/source-annotations-check-new
```

The artifact directory must be new; `--connect` is only for an owned development session because the demo replaces its schematic. Crowded text/symbol overlaps, very long IDs and large fitted drawings still need layout judgment. Native print preview/submission, recorded-scene schematic interaction, complete physical-input/accessibility and distribution/participant gates remain separate.

The expanded Fit frame exposed a smoke-test assumption: partially covered glyphs were counted only when their decoded color matched solid text. An owned GPU snapshot confirmed readable cards with quarter/half/three-quarter multisample coverage. The smoke now measures linear-light equivalent ink inside the card, excluding borders, with the original threshold; actual pixel tests reject background and unrelated colors. This changes validation, not drawing or simulation.
