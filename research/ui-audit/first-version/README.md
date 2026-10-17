# First-version UI walkthrough and issue list

October 17, 2026. **Package 2 is in progress.** These are direct observations of ELab's actual native window, using pointer, keyboard and accessibility actions. They are not participant results, a screen-reader study or visible-latency measurements.

## Configuration and boundaries

The separate `build/ELab-first-version-ui.app` review bundle used source base `937d4e0` and the pinned Vellum SDK `47b7153`. Its 0.122.0 /139 label identifies a private UI harness, not a promoted release. The cursor repair below was rebuilt into the same review bundle. [Checkpoint metadata](checkpoint.json) records the repaired executable's SHA-256.

Observed app screenshots are **1185×855** and **1044×764** pixels. The smaller size came from dragging the app's resize corner. These are window captures; the planned 1280×800 and 1440×900 **display configurations have not been qualified**. No system display or appearance setting was changed. Theme changes were confined to the test session with appearance persistence disabled.

Both test hosts used new private endpoints/recovery directories and exited normally through `app.close`. No user session, existing project or canonical/preserved app bundle was replaced. The UI-test connection's first attach took about 758 seconds; subsequent observations/interactions were brief. That tooling observation is not evidence of app launch latency or a p95 measurement.

## Completed observations

| Task | Actual UI path and outcome | Evidence |
|---|---|---|
| LED from blank | Palette and canvas placement of source, resistor, LED and ground; four terminal-to-terminal wires. Closing the loop immediately displayed 2.917 mA. | [Actual initial circuit](led-from-blank-light.png) |
| Value edit/history | Typed `2k` in the inspector and applied it: 1.477 mA. Header Undo restored 2.917 mA; Redo restored 1.477 mA. | [Saved edited circuit](led-built-ui.elab); screenshot is the earlier 1 kΩ state |
| Fault recovery | Selected the LED return wire, pressed Delete, saw stale readings and the named disconnected cathode. Reconnected the terminal dots and regained live simulation. | Resulting four-wire project in the LED fixture |
| RC from blank | New schematic; palette/search placement of 1 µF capacitor, ideal square source, resistor and ground; four pointer-drawn wires. Hid scope during assembly, then reopened it. Typed/applied `10k`; live startup reset to zero. | [Saved RC project](rc-built-ui.elab) |
| RC timing | Paused, restarted, pressed Step ten times. At 10 ms the visible capacitor reading was 3.161 V, matching the ideal-step reference. Entered B at `10 ms`, A at `0 ms`; visible Δt was 10 ms. | [Before repair](rc-cursors-before.png), [structured observed state](rc-observed.json) |
| Menus/themes/scroll | Opened native File, Examples and theme menus; switched to Dark. Scope scroll exposed its cursor controls. Smaller window preserved scrolling but reduced schematic readability. | [Dark view](rc-dark-before.png), [compact view](rc-compact-before.png), [compact AX state](rc-compact-before.ax.txt) |
| Native save/open | Saved both circuits through native Save dialogs into this folder. In a fresh repaired app, selected the RC file in the native Open dialog; values/connectivity returned with paused fresh time zero. | Fixtures and [final checkpoint](checkpoint.json) |

The RC IDs are R2/V2/GND2 because the same test session previously created the LED circuit. Session-lifetime ID reservations are expected; no file contents were rewritten to substitute reference IDs. The observed RC result was 3.160602794 V before display rounding (see the captured state); the UI shows 3.161 V.

## One release issue list

All items are owned by ELab implementation/review. P1 means significant workflow/qualification risk; P2 means a bounded usability defect. The alpha decision is stated separately from the full PRD gate.

| ID | Finding and reproduction | Next action / closure evidence | State |
|---|---|---|---|
| UI-1 · P1 | Fit reduced the hand-built RC circuit to 44% in the larger split view and 32% in the compact view. Bitmap schematic captions become too small to read; thin dark wires also need review at this scale. The compact registered drawable area was 456×158 px. | Repair excessive fitting margins and/or small-scale presentation while preserving complete framing, source annotations and actual circuit geometry. Repeat these same projects in both themes and compact windows. **Alpha readability blocker.** | Open; next repair |
| UI-2 · P2 | Unset cursor fields displayed `0 s`. Focusing A and pressing Return moved focus but placed no cursor. Changing the text to `0 ms` placed it, proving the same-value/native-commit ambiguity. | Unset fields now render empty, with placement help. The native UI was reopened, `0 s` was typed/committed into A, Clear became enabled, and Clear restored an empty field. | **Repaired and directly retested** |
| UI-3 · P2 | The second Save dialog suggested `Untitled.elab.elab`; the first suggested `Untitled.elab`. Explicit names saved correctly with one suffix. | Repeat default-name Save/Save As across file types. Inspect panel reuse and filename/type ordering; fix the cause before qualifying the ordinary save flow. | Open investigation; not a data-loss claim |
| UI-4 · P1 | Accessibility-based `setValue` on an actually visible scrolled cursor field returned `cannotClickOffscreenElement`. Coordinate focus plus keyboard entry worked. Some freshly selected inspector controls were absent from intermediate AX snapshots despite visible pixels. | Recheck after a stable observation; isolate scrolled geometry versus delayed publication. Do not infer complete keyboard/VoiceOver support from this pass. Full accessibility acceptance remains open; alpha limitation must be disclosed if unresolved. | Open qualification issue |
| UI-5 · P1 | Rapid batched focus/select/type/Return actions sometimes lost text or retained old text: `ground` became `groun`; another rapid replacement inserted into the prior query. Confirming focus/selection and observing text before Return preserved full input. Typed resistor values and cursor units worked with those boundaries. | Isolate native commit/focus ordering from computer-use event timing using a controlled reproduction. Do not assign blame to the app/framework or mark it fixed without evidence. If ordinary fast typing is affected, repair it before alpha. | Open investigation |

## Cursor repair validation

`TestCursorTimeFieldsDistinguishUnsetFromStartup` failed before the repair because A's unset field had text `0 s`. It now checks empty A/B, explicit zero placement, a 10 ms two-cursor interval, and clearing A while retaining B. Existing committed-field stale-context and invalid-unit guards remain in force. No commands, schemas, numerical models or file formats changed.

Validation after the repair:

- Native race tests matching `Test.*(Cursor|Time|Probe|Scope)` passed in 19.723 s.
- `scripts/lint.sh` and native `go vet ./...` passed; native build passed with the existing duplicate `-lobjc` linker warning.
- Direct GUI recheck: reopen RC through native dialog → paused/time zero → scroll to empty cursor fields → type `0 s` → Return → placed cursor → Clear → empty field. [Placed-zero pixels](cursor-zero-after.png), [AX state](cursor-zero-after.ax.txt), [final cleared state](checkpoint.json).
- A full quality gate passed in the preceding [example-entry checkpoint](../../../docs/validation/first-version-examples/README.md). This repair has the focused validation above; those checks do not replace the final release-candidate gate after all repairs.

## Remaining package 2 work

Resolve/retest the listed issues; finish the flasher/NMOS/PMOS walkthrough; test paused edits, complete keyboard construction/navigation, Save/Discard/Cancel and readable exports; repeat at the declared display configurations and choose an evidenced minimum usable window size. Physical print, VoiceOver, visible p95 latency and participant outcomes remain unqualified. A/B construction and partial file work are proven here; the whole package and release checklist are not complete.
