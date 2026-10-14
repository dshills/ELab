# Printable schematic PDF

Choose **Export PDF** in the component rail. The Inspector's **Output page** section shares paper, orientation, margin, fit/explicit scale, color and live-overlay settings with agents. Defaults are Letter landscape, 10 mm margins, fit the complete schematic, grayscale on white paper, and no measurement overlays. The source circuit need not be complete or solved.

PDF export captures the same complete symbol/terminal/wire/label layout as SVG. It uses the macOS CoreGraphics/CoreText backend in both native and headless builds, without opening a window or contacting a printer. CoreGraphics supplies a PDF drawing context rather than a screenshot ([Apple context documentation](https://developer.apple.com/documentation/coregraphics/cgcontext/init%28consumer%3Amediabox%3A_%3A%29?language=objc)). Other platform/non-cgo builds return an explicit unsupported result; their existing SVG/CSV workflows remain available.

| Shared command | Behavior |
|---|---|
| `export.page.set` | Set session defaults; omitted fields reset to defaults. No circuit/history/dirty/physics change. |
| `export.schematic.pdf` | Capture and atomically write the PDF, waiting for the project operation's terminal outcome. |
| `export.start` with `kind:"schematic-pdf"` | Admit the same capture asynchronously; inspect/cancel via the existing project-work commands. |

```json
{"command":"export.page.set","params":{"paper":"a4","orientation":"portrait","margin_mm":12}}
{"command":"export.schematic.pdf","params":{"path":"/absolute/path/circuit.pdf"}}
```

These are command fragments; normal transport version/request ID, revision guard and retry key still apply. An export can supply its own `page` object to override session defaults. Paper is `letter` or `a4`; orientation is `portrait` or `landscape`; margins are 0–40 mm. Omit `scale` for fit, or use 0.1–4 PDF points per schematic world unit. If an explicit scale would clip the complete schematic, the operation fails without publishing a partial/cropped file. Extremely large drawings may need a smaller scale; automatic tiling and minimum-font guarantees are not implemented.

Color output and live measurement overlays are opt-in. Overlays are a captured live snapshot with original stale/unavailable labels, never relabeled recorded data. Artifact page settings and PDF Subject metadata record source run/electrical/document identity, page configuration and overlay provenance. CoreGraphics creation timestamps mean separate exports need not have identical bytes; compare page/text/rendered content and captured identity rather than file hashes alone.

Page settings are session output choices, not saved circuit preferences. Output runs in the existing bounded background export worker with immutable captured graph/settings, guarded file publication, cancellation and preserved original project/physics. It refuses exporting over the project path and uses the existing 8 MiB asset limit. Later edits can continue while preparation is in progress; the artifact retains captured source identity.

The [validation receipt](validation/pdf-export-validation.json), [headless exports](validation/headless-pdf-export-final.json) and [native exports](validation/desktop-pdf-export.json) record six reference circuits and shared settings. Rendered checks cover page dimensions, all text within bounds, Greek units, grayscale and representative legibility. The smallest reference text was about 7.5 pt in the flasher. This is not a universal readability guarantee for arbitrary capacity/layout.

This implements PDF output and shared page preparation. **Native print preview, printer discovery/submission, tiled pages and physical printer qualification remain pending.** Saving a PDF does not prove printing. No printer job is submitted by these commands.

[PNG image export](schematic-image.md) now shares this page preparation and drawing path.
