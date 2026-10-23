# ELab document workspace redesign

Decision and implementation: October 22–23, 2026. Audience: hobbyists and makers, with room to grow. Scope: redesign the existing native app; preserve the electronics engines, file format and shared user/agent command model.

## Design direction

Treat a circuit as a document that the user edits and observes. The schematic is the central work surface. A quiet toolbar identifies the open document and offers familiar file/history actions. A persistent library supplies components. A task-based inspector handles properties, simulation and document settings. A scope dock and compact readings keep live behavior visible.

The primary Apple references are [Keynote's inspectors](https://support.apple.com/en-ca/guide/keynote/tan391376b09/mac), [Logic Pro's main window](https://support.apple.com/guide/logicpro/main-window-interface-lgcpe9cc403a/mac), and [Apple's toolbar guidance](https://developer.apple.com/design/human-interface-guidelines/toolbars). Keynote supplies the document/canvas/inspector relationship; Logic supplies the library/workspace/lower-editor relationship for a technical creative tool. These are workflow references, not a pixel-for-pixel clone or a claim that the controls are all AppKit.

## Baseline audit

An owned copy of /196 was captured before changes, using the RC charge/discharge example. It showed several competing visual layers: large dashboard readings, wide panel gutters, repeated filled buttons, an oversized toolbar, a component library mixed with output/help controls, and a single long inspector containing unrelated tasks. The small window clipped some schematic annotations until Fit was applied. Secondary actions competed with the drawing for attention.

The baseline capture is retained with the validation evidence. Measurements from different window heights are not a controlled quantitative usability comparison.

## Implemented decisions

| Area | Decision | Purpose |
| --- | --- | --- |
| Window | Unified macOS chrome, compact document title, basename and Edited state | Make the circuit feel like the document being edited |
| Toolbar | Quiet file/example menus, icon Save/Undo/Redo/appearance actions with accessible names | Reduce repeated button text and visual weight |
| Library | Dedicated left component search/list, placement and named wiring | Keep building tools in one predictable place |
| Canvas | Narrow gutters, restrained grid, compact tool row | Give drawing and live results more space |
| Inspector | Fixed Inspect / Simulate / Document navigation and one scrolling form | Separate tasks without hiding navigation or nesting scrollbars |
| Inspect | Selected-object editing, measurement and pinned-reading controls | Keep object work together |
| Simulate | Sampling guidance and timing first, then probes, verification, sweeps and circuit information | Put common live-experiment controls within reach |
| Document | Export/preview actions and page settings first; appearance, display, recovery and help below | Make output easy to find without cluttering the library |
| Readings | A compact status strip with clear values and sentence-case labels | Preserve live feedback without a dashboard of large cards |
| Appearance | Neutral light/dark surfaces, restrained blue selection, compact macOS typography | Establish a calmer hierarchy and consistent selection language |
| Plot | Named measurement colors independent of the UI accent | Keep trace legends truthful when interface colors change |

Existing command IDs, value validation, save/recovery behavior and simulation semantics remain in use. The one added presentation command, `workspace.inspector.set`, exposes exactly the same navigation to external and built-in agents. Its selection is session-only and does not enter schematic files.

## Iteration and verification

Early rendered passes exposed title-bar overlap and cramped native tab pages. The final composition uses a fixed application-owned navigation toolbar and a single scroll owner. This avoids wrapped tab headings, low-contrast inherited tab styling and horizontal page scrolling. Ordinary edits retain the current scroll position; document replacement and section selection start a fresh viewport.

Tests were updated to navigate to a real visible inspector section before capturing controls. They continue to exercise retained-action refusal, native field settlement, recovery ownership, output settings, physics and UI/agent parity. Hidden controls are removed from the mounted tree, rather than retained as invisible test targets. A mounted regression scrolls the actual inspector, switches sections through physical control events, and checks that navigation stays visible and the circuit is preserved.

Final package, screenshots, quality-check results and any remaining limitations are recorded in the accompanying validation receipt. Compact scope controls intentionally remain vertically scrollable; the existing divider and scope-collapse action provide more drawing or measurement space.

## Boundaries

The implementation uses Vellum/Cogent widgets, macOS window chrome and the existing AppKit menu bridge. It does not claim a complete AppKit conversion, Liquid Glass materials, full VoiceOver qualification, or new electronics capabilities. The component list still uses Vellum's managed list presentation. Very small workspaces and large circuits still require zoom, Fit or scope resizing. Public signing/notarization remains governed by the existing unsigned-release decision.

## Delivered build and evidence

The final build is **ELab 0.180.0 /198**, from clean source `063ea93567f477aa0ccf91bf22557e057136ec51`. [Open the packaged app](../build/releases/ELab-0.180.0-macos-arm64/ELab.app), or use its [ZIP](../build/releases/ELab-0.180.0-macos-arm64.zip). The earlier /197 candidate is retained as an intermediate pass; /198 corrects the light background discovered behind dark inspector navigation.

The [validation receipt](../docs/validation/native-ui-198/report.json) records the full `scripts/check.sh` pass against all 676 unchanged tracked Go source files: goimports, staticcheck, 67 Python tests, root/desktop race tests, vet and mounted-control race regressions. The ZIP was extracted into a Unicode path containing spaces; all 127 files and modes match, the integrity seal verifies, and app/agent bytes remain unchanged after execution. LED/RC/flasher numerical results match historical references and the paired packaged agent. The native UI also selected R1, accepted 12k, saved through the toolbar, stepped the scope, switched inspector sections and changed appearance through its native menu.

[Before/after visual evidence](ui-audit/native-redesign-198/README.md) includes compact light/dark, document-output and large-window captures. The test app closed with joined workers and its endpoint removed. The previous unsigned release is unchanged. A long filename or agent actor in the footer can still wrap at the compact size; this narrow status-text limitation is recorded, rather than claiming that every possible string/layout has been qualified.
