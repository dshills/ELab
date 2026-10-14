# Schematic image output

**Export PNG** in the component rail writes the complete schematic on white paper. The Inspector's **Output page** settings apply to both PNG and PDF: A4/Letter, orientation, margins, fit/explicit scale, color and optional captured live measurements. PNG defaults to 150 pixels/inch. The default Letter landscape page is 1650 × 1275 pixels; dimensions round upward to preserve the complete page.

Users and agents use the same captured background export and guarded file publication. `export.schematic.png` waits for completion; `export.start` with `kind:"schematic-png"` returns admission for inspection/cancellation through `project.work` commands. Both accept `path`, `overwrite`, optional document generation, `page` and `dpi`. Resolution is an integer from 72–300; omitted/zero means 150. `capabilities.list` with this command returns a detailed resolution schema.

```json
{"command":"export.page.set","params":{"paper":"a4","orientation":"portrait","margin_mm":12}}
{"command":"export.schematic.png","params":{"path":"/absolute/path/circuit.png","dpi":150}}
```

These fragments still need normal transport version/request ID, retry key and revision guard. Per-export page options override session defaults without changing them. Page choices are output configuration, not saved circuit preferences. Explicit scale fails rather than cropping when the complete drawing cannot fit.

PNG shares the PDF geometry interpreter and CoreGraphics/CoreText primitives. The macOS bitmap backend encodes sRGB PNG through [Apple ImageIO](https://developer.apple.com/documentation/imageio/cgimagedestinationcreatewithdata%28_%3A_%3A_%3A_%3A%29?language=objc); no window or external conversion program is needed. Other/non-cgo builds report unsupported and retain SVG/CSV output. Allocation is bounded to 12 million pixels before context creation; encoded output retains the existing 8 MiB limit. Cancellation is checked during drawing and before publication; native encoding itself is a bounded synchronous call.

PNG Description metadata (encoded by ImageIO as XMP dc:description) records source run, electrical revision, document generation, normalized page, resolution and live-overlay provenance. The PNG resolution chunk rounds to the nearest whole pixel/metre; 150 dpi reads back as about 150.0124 dpi. Artifact metadata carries the same source/page identity and file hash. Default grayscale output omits live overlays regardless of app theme. Opt-in overlays retain their captured stale/unavailable labels; they do not become recorded verification data.

Tests cover asymmetric orientation/color, opaque page corners, allocation/resolution limits, all six reference circuits, original-capture isolation after later page/component edits, cancellation/destination preservation and the native PNG picker. Export preserves project/history/paused physics. The [validation receipt](validation/png-export-validation.json) records public native/headless image comparison and visual inspection.

PNG output is a file export. In-app print preview, printer discovery/submission, tiled pages and physical-device printing remain pending. Large fitted diagrams can have small labels; PNG does not add a universal readability guarantee.

The rectifier checkpoint repairs label-background layering: conservative masks now sit below component symbols/pins, so a nearby long label no longer erases a resistor. This repair also applies to SVG and PDF through their shared layout.

Waveform excitation now appears in [shared source annotations](source-annotations.md), with exact captured component/settings metadata and explicit applied/pending/stale state.
