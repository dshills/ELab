# Printable schematic PDF

Choose **Tools → Export → Download pdf** in the browser. The shared output-page settings cover paper, orientation, margin, fit/explicit scale, color and live-overlay settings. Defaults are Letter landscape, 10 mm margins, fit the complete schematic, grayscale on white paper, and no measurement overlays. The source circuit need not be complete or solved.

PDF export captures the same complete symbol, terminal, wire and label layout
as SVG. The renderer produces vector PDF with an embedded font and selectable
text. It runs on Linux, macOS and Windows without system fonts or a printer.
Characters outside the embedded font appear as question marks; SVG remains
available when you need the full original label text.

| Shared command | Behavior |
|---|---|
| `export.page.set` | Set session defaults; omitted fields reset to defaults. No circuit/history/dirty/physics change. |
| `export.schematic.pdf` | Capture and atomically write the PDF, waiting for the project operation's terminal outcome. |
| `export.start` with `kind:"schematic-pdf"` | Admit the same capture asynchronously; inspect/cancel via the existing project-work commands. |

```json
{"command":"export.page.set","params":{"paper":"a4","orientation":"portrait","margin_mm":12}}
{"command":"export.schematic.pdf","params":{"path":"/absolute/path/circuit.pdf"}}
```

These are command fragments; normal transport version/request ID, revision guard and retry key still apply. An export can supply its own `page` object to override session defaults. Paper is `letter` or `a4`; orientation is `portrait` or `landscape`; margins are 0–40 mm. Omit `scale` for fit, or use 0.1–4 PDF points per schematic world unit. If an explicit scale would clip the complete schematic in single-page mode, the operation fails without publishing a partial/cropped file. Enable `tile:true` with an explicit scale to cover the complete drawing across up to 16 pages; `overlap_mm` defaults to 5 and accepts 0–20 mm. Tiles preserve scale and follow row-major order, reserving an 18-point footer with page/grid/scale/overlap labels. Oversized grids fail before publication; no implicit shrinking occurs. See [tiled output](tiled-output.md).

Color output and live measurement overlays are opt-in. Overlays are a captured live snapshot with original stale/unavailable labels, never relabeled recorded data. Artifact page settings and PDF Subject metadata record source run/electrical/document identity, page configuration and overlay provenance. Compare page, text and captured source identity when checking output from different ELab versions.

Page settings are session output choices, not saved circuit preferences. Output runs in the existing bounded background export worker with immutable captured graph/settings, guarded file publication, cancellation and preserved original project/physics. It refuses exporting over the project path and uses the existing 8 MiB asset limit. Later edits can continue while preparation is in progress; the artifact retains captured source identity.

The historical [validation receipt](validation/pdf-export-validation.json), [headless exports](validation/headless-pdf-export-final.json) and [native exports](validation/desktop-pdf-export.json) record six reference circuits and shared settings. Rendered checks cover page dimensions, all text within bounds, Greek units, grayscale and representative legibility. The smallest reference text was about 7.5 pt in the flasher. This is not a universal readability guarantee for arbitrary capacity/layout.

This implements PDF output and shared page preparation. [Print preview](print-preview.md) and [local printer discovery/submission](printing.md) are implemented separately. Tiled PDF/preview/PNG output is now implemented; physical printer qualification remains pending. Saving a PDF does not prove printing. No printer job is submitted by these commands.

[PNG image export](schematic-image.md) now shares this page preparation and drawing path.
