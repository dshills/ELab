# First-version UI walkthrough and issue list

October 17, 2026. **Package 2 is in progress.** These are direct observations of ELab's actual native window, using pointer, keyboard and accessibility actions. They are not participant results, a screen-reader study or visible-latency measurements.

## Configuration and boundaries

The separate `build/ELab-first-version-ui.app` review bundle used source base `937d4e0` and the pinned Vellum SDK `47b7153`. Its 0.122.0 /139 label identifies a private UI harness, not a promoted release. The cursor repair below was rebuilt into the same review bundle. [Checkpoint metadata](checkpoint.json) records the repaired executable's SHA-256.

Observed app screenshots are **1185×855** and **1044×764** pixels. The smaller size came from dragging the app's resize corner. These are window captures; the planned 1280×800 and 1440×900 **display configurations have not been qualified**. No system display or appearance setting was changed. Theme changes were confined to the test session with appearance persistence disabled.

Both test hosts used new private endpoints/recovery directories and exited normally through `app.close`. No user session, existing project or canonical/preserved app bundle was replaced. The UI-test connection's first attach took about 758 seconds; subsequent observations/interactions were brief. That tooling observation is not evidence of app launch latency or a p95 measurement.

The subsequent readability pass kept the same private review bundle and SDK. [Its checkpoint](readability-checkpoint.json) records the new binary, exact native/headless comparisons and image dimensions. Screenshot bytes returned by computer use are JPEG; their extensions were corrected from `.png` to `.jpg` without changing the pixels. The checkpoint preserves the original-to-current filename/hash mapping. Historical capture sizes above are unchanged; new larger captures are 1184×854 after resizing, and compact captures remain 1044×764.

## Completed observations

| Task | Actual UI path and outcome | Evidence |
|---|---|---|
| LED from blank | Palette and canvas placement of source, resistor, LED and ground; four terminal-to-terminal wires. Closing the loop immediately displayed 2.917 mA. | [Actual initial circuit](led-from-blank-light.jpg) |
| Value edit/history | Typed `2k` in the inspector and applied it: 1.477 mA. Header Undo restored 2.917 mA; Redo restored 1.477 mA. | [Saved edited circuit](led-built-ui.elab); screenshot is the earlier 1 kΩ state |
| Fault recovery | Selected the LED return wire, pressed Delete, saw stale readings and the named disconnected cathode. Reconnected the terminal dots and regained live simulation. | Resulting four-wire project in the LED fixture |
| RC from blank | New schematic; palette/search placement of 1 µF capacitor, ideal square source, resistor and ground; four pointer-drawn wires. Hid scope during assembly, then reopened it. Typed/applied `10k`; live startup reset to zero. | [Saved RC project](rc-built-ui.elab) |
| RC timing | Paused, restarted, pressed Step ten times. At 10 ms the visible capacitor reading was 3.161 V, matching the ideal-step reference. Entered B at `10 ms`, A at `0 ms`; visible Δt was 10 ms. | [Before repair](rc-cursors-before.jpg), [structured observed state](rc-observed.json) |
| Menus/themes/scroll | Opened native File, Examples and theme menus; switched to Dark. Scope scroll exposed its cursor controls. Smaller window preserved scrolling but reduced schematic readability. | [Dark view](rc-dark-before.jpg), [compact view](rc-compact-before.jpg), [compact AX state](rc-compact-before.ax.txt) |
| Native save/open | Saved both circuits through native Save dialogs into this folder. In a fresh repaired app, selected the RC file in the native Open dialog; values/connectivity returned with paused fresh time zero. | Fixtures and [final checkpoint](checkpoint.json) |

The RC IDs are R2/V2/GND2 because the same test session previously created the LED circuit. Session-lifetime ID reservations are expected; no file contents were rewritten to substitute reference IDs. The observed RC result was 3.160602794 V before display rounding (see the captured state); the UI shows 3.161 V.

## One release issue list

All items are owned by ELab implementation/review. P1 means significant workflow/qualification risk; P2 means a bounded usability defect. The alpha decision is stated separately from the full PRD gate.

| ID | Finding and reproduction | Next action / closure evidence | State |
|---|---|---|---|
| UI-1 · P1 | Fit reduced the hand-built RC circuit to 44% in the larger split view and 32% in the compact view. Bitmap schematic captions become too small to read; thin dark wires also need review at this scale. The compact registered drawable area was 456×158 px. | Fit now uses actual routes and shared caption bounds; captions grow up to 2× world size, source lines retain spacing, strokes retain 1.25 screen pixels, and dark wires are brighter. Native checks in both themes show readable RC captions and full framing. | **RC cause repaired and retested**; broader flasher/device walkthrough remains open |
| UI-2 · P2 | Unset cursor fields displayed `0 s`. Focusing A and pressing Return moved focus but placed no cursor. Changing the text to `0 ms` placed it, proving the same-value/native-commit ambiguity. | Unset fields now render empty, with placement help. The native UI was reopened, `0 s` was typed/committed into A, Clear became enabled, and Clear restored an empty field. | **Repaired and directly retested** |
| UI-3 · P2 | The second Save dialog suggested `Untitled.elab.elab`; the first suggested `Untitled.elab`. Explicit names saved correctly with one suffix. | Repeat default-name Save/Save As across file types. Inspect panel reuse and filename/type ordering; fix the cause before qualifying the ordinary save flow. | Open investigation; not a data-loss claim |
| UI-4 · P1 | Accessibility-based `setValue` on an actually visible scrolled cursor field returned `cannotClickOffscreenElement`. Coordinate focus plus keyboard entry worked. Some freshly selected inspector controls were absent from intermediate AX snapshots despite visible pixels. | Recheck after a stable observation; isolate scrolled geometry versus delayed publication. Do not infer complete keyboard/VoiceOver support from this pass. Full accessibility acceptance remains open; alpha limitation must be disclosed if unresolved. | Open qualification issue |
| UI-5 · P1 | Rapid batched focus/select/type/Return actions sometimes lost text or retained old text: `ground` became `groun`; another rapid replacement inserted into the prior query. Confirming focus/selection and observing text before Return preserved full input. Typed resistor values and cursor units worked with those boundaries. | Isolate native commit/focus ordering from computer-use event timing using a controlled reproduction. Do not assign blame to the app/framework or mark it fixed without evidence. If ordinary fast typing is affected, repair it before alpha. | Open investigation |
| UI-6 · P1 | Loading the fixture through an agent widened Undo to “Undo agent” and clipped the theme menu while header space remained unused. The compact schematic tool row also scrolled horizontally. | Reserved the complete header action width and reduced its gaps; tested **both** Undo agent/Redo agent. Moved zoom to the footer and narrowed horizontal tool-button padding while keeping their height and Toolbar semantics. | **Repaired and directly retested** at both window sizes/themes |

## Readability and compact-control repair

The original Fit used `WireRoutingBounds`, which includes the path finder's search margins, plus a large generic part rectangle. The repair uses actual cached deterministic paths, body/terminal geometry and caption bounds. Native and agent Fit use the same core calculation. A 40-iteration bounded search checks each accepted view against its own enlarged caption geometry; route preparation occurs once. Exported schematic data, saved models, electrical state, dirty membership and history semantics are unchanged.

The native schematic grows primary part captions at smaller zooms, with a 2× world-size cap for overviews. Source blocks remain right aligned with adequate line spacing; horizontal source blocks keep their last baseline above their leads. Stroke widths have a screen-space floor. The normal/dark wire color uses the existing trace contrast adjustment. These changes do not claim that arbitrarily large drawings have readable labels at minimum zoom.

Observed results:

- The fixed 456×158 RC regression initially failed at zoom 0.319192. The complete enlarged-caption fit is 0.421365, with graph, accepted sample and project state preserved. Tighter framing without caption enlargement alone reached 0.447592; the final calculation deliberately retains space for the larger text.
- Direct native windows displayed approximately 42% and 60% after repair. [Compact light](rc-compact-light-after.jpg), [compact dark](rc-compact-dark-after.jpg), [larger light](rc-large-light-after.jpg), [larger dark](rc-large-dark-after.jpg). Compare [the original compact view](rc-compact-before.jpg).
- Both “Undo agent” and “Redo agent,” Theme and Pause/Resume remain fully visible. The tool row has no horizontal scrollbar; zoom is readable in the footer. [Observed clipped header before repair](rc-header-before.jpg), [final compact accessibility state](rc-compact-after.ax.txt).
- Public native and headless Fit outputs match exactly at measured drawable sizes **456×159** and **594×217**. Both paths preserve identical components, wires, settings and accepted time-zero RC samples; the source project SHA-256 is unchanged. The one-pixel height difference from the fixed 456×158 test is a measured resize/layout result, not a substituted test size.
- Core viewport tests now contain both ordinary and enlarged native caption bounds for all 18 templates. Native mesh tests require at least eight ink pixels in the tested compact resistor caption, a 1.25-pixel wire, and source line spacing of at least the scaled 13-pixel font height. These geometry checks complement the actual screenshots.
- One fresh 52-object/53-wire reference Fit call took 9.7 ms after the text change; an earlier tighter-bounds-only call took 13.4 ms. These are individual workstation observations, not p95 timing or general worst-case responsiveness evidence.

Focused root viewport/label/routing race checks and native canvas/label/workspace race checks passed before the final control-padding adjustments. The complete final quality gate also passed, as recorded below. Actual multi-display input, declared display configurations, full accessibility and the remainder of package 2 remain open. The first test and subsequent review hosts exited normally; only the disposable review bundle/development executables were rebuilt.

The first full gate passed numerical conformance and workbench race tests but correctly rejected a stale native source ledger. Review confirmed that only three source hashes/site line numbers changed: all 388 site expressions, commands, functions and categories are identical after excluding line numbers. The ledger was refreshed and the focused inventory race check passed; the complete gate was then rerun. This is a source-freshness repair, not an increase in claimed runtime coverage.

**Final gate passed:** `scripts/check.sh` completed lint/import formatting/Staticcheck, 38 Python tests, all root/native race tests and both vet passes. Root numerical conformance and workbench race tests passed in the first run (119.213 s /76.283 s) and were cached on the successful rerun; the refreshed inventory test passed in 1.375 s and the native suite in 142.992 s. These are suite durations, not UI latency. The [checkpoint](readability-checkpoint.json) identifies source/binary hashes and preserved images/projects. Cold-route worst-case cost, visible p95 timing and broader family readability remain release qualification work.

## Cursor repair validation

`TestCursorTimeFieldsDistinguishUnsetFromStartup` failed before the repair because A's unset field had text `0 s`. It now checks empty A/B, explicit zero placement, a 10 ms two-cursor interval, and clearing A while retaining B. Existing committed-field stale-context and invalid-unit guards remain in force. No commands, schemas, numerical models or file formats changed.

Validation after the repair:

- Native race tests matching `Test.*(Cursor|Time|Probe|Scope)` passed in 19.723 s.
- `scripts/lint.sh` and native `go vet ./...` passed; native build passed with the existing duplicate `-lobjc` linker warning.
- Direct GUI recheck: reopen RC through native dialog → paused/time zero → scroll to empty cursor fields → type `0 s` → Return → placed cursor → Clear → empty field. [Placed-zero pixels](cursor-zero-after.jpg), [AX state](cursor-zero-after.ax.txt), [final cleared state](checkpoint.json).
- A full quality gate passed in the preceding [example-entry checkpoint](../../../docs/validation/first-version-examples/README.md). This repair has the focused validation above; those checks do not replace the final release-candidate gate after all repairs.

## Remaining package 2 work

Resolve/retest the listed issues; finish the flasher/NMOS/PMOS walkthrough; test paused edits, complete keyboard construction/navigation, Save/Discard/Cancel and readable exports; repeat at the declared display configurations and choose an evidenced minimum usable window size. Physical print, VoiceOver, visible p95 latency and participant outcomes remain unqualified. A/B construction and partial file work are proven here; the whole package and release checklist are not complete.
