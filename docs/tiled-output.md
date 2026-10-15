# Print a larger schematic across pages

In **Output page**, enable **Tile across pages**, choose paper/orientation/margins and an explicit scale. Enabling the checkbox starts at 100% if no explicit scale exists. **Tile overlap** defaults to 5 mm. Open Print preview to navigate the captured pages, save the shown page as PNG, or submit the complete captured document. **Fit to page** restores single-page fit mode.

```json
{"command":"export.page.set","params":{"paper":"letter","orientation":"landscape","margin_mm":10,"scale":1,"tile":true,"overlap_mm":5}}
{"command":"export.schematic.pdf","params":{"path":"/absolute/path/circuit.pdf"}}
```

Normal request IDs, retry keys and revision/document guards apply. Settings are session output choices; they do not change the circuit, dirty state, history or simulation. They also work for incomplete drawings.

Tiles cover the complete generated schematic bounds at the chosen scale, left-to-right then top-to-bottom. Each page reserves an 18-point footer and identifies its number, row/column, scale and overlap. Content is clipped to each printable tile; adjacent pages repeat the chosen overlap for assembly. The drawing is never silently shrunk. A sparse layout can include blank or mostly empty tiles; preview the grid or choose a smaller explicit scale to reduce paper use. Paper is A4/Letter, margins 0–40 mm, scale 0.1–4 points per schematic unit and overlap 0–20 mm. Tiling requires an explicit scale. Layouts over 16 pages fail before publishing output. Existing 8 MiB asset and cooperative operation deadlines remain.

`print.preview.start` captures the current source/settings, starting at zero-based `page_index` (default 0). Inspect `print_preview.status` until completed, failed or canceled. `page_index`, `page_count`, `rows` and `columns` describe the grid. One page PNG is rendered and retained at a time.

`print.preview.page.set {"id":"preview-1","index":1}` renders another page of the same immutable capture under a new preview ID. It preserves captured graph/settings/readings and source identity even if the editor has changed. Same-index re-entry preserves the preview; out-of-range, stale, hidden or unfinished requests refuse. Page navigation does not submit a printer job. The ordinary preview cancellation/close controls still apply. Old image-chunk IDs refuse after replacement, preventing mixed-page reads.

`export.schematic.png` selects one current-source tile through an explicit zero-based `page_index`; omission refuses if there are multiple pages. The native general PNG action explicitly selects page zero. **Save shown PNG** supplies `preview_id` instead: it atomically writes the exact visible completed preview bytes, preserving the original capture. Page/index/DPI overrides and recording IDs are incompatible with `preview_id`. Already admitted captured export can finish after the preview closes; a new export cannot target a closed/replaced preview.

`print.submit` renders and submits **all captured pages** in one PDF job, regardless of the selected preview index. The UI labels this **Print all N pages** and retained jobs expose `page_count`. Source edits never silently change the submitted capture. Existing destination/copy selection, observable work, cancellation, acknowledgment uncertainty and spooler-state rules apply. Local spool options request one-sided output and no scaling; this does not prove a physical printer honors page geometry or produces correct paper output.

Core tests cover independent geometric coverage/overlap, margin clipping, page count limits, immutable navigation, exact captured PNG ownership/lifetime, stale-ID refusal, atomic publication and all-pages print preparation through an isolated backend. Native tests exercise settings, navigation and save actions; packaged smoke loads multiple page images. The public workflow checks Letter and A4 output through headless and native commands; [PDF page/text/render inspection](validation/tiled-output-visual-review.json) has separate evidence, including all 12 Letter/A4 pages. Physical printer, complete keyboard/IME/accessibility and visible-latency qualification remain open. No project format, electronic model or numerical kernel changes.
