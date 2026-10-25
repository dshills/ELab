# Building a circuit with the keyboard

A fresh window focuses Search once. Search by component name or a hobbyist alias such as `battery`. Commit with Return to focus the matching list. Up/Down, Home/End, page keys and initial letters navigate the Vellum ListBox; Return or Space chooses its focused row. Tab reaches the stacked X and Y fields, then **Place at X / Y**. Commit each coordinate before continuing. A successful placement returns to search, so the next part starts nearby. No-match search keeps search focus; invalid/occupied placement preserves the drawing and does not request a success focus change.

**Connect by name** opens the From/To terminal selectors together in the left rail. Space opens each native menu, arrows choose and Return commits. Tab reaches **Connect wire** once both ends are selected. Success resets both ends and returns to From. Escape cancels the drawing tool and requests search focus. Canvas wiring remains available.

| UI operation | Shared command |
| --- | --- |
| Component search | `parts.search.set` |
| Choose part / Connect by name | `drawing.tool.set` |
| Commit X / Y | `drawing.position.set` |
| Place | `component.place` |
| Choose wire ends / Connect | `wire.terminals.set` / `wire.connect` |
| Escape | `drawing.cancel` |
| Select / draft / apply value | `component.select` / `editor.value.set` / `component.set` |
| Save | `project.save.start` through the existing file worker |

Every domain edit retains the ordinary command revision and document guards. Focus continuation is native presentation following success; repeated redraws reuse the request sequence and do not continually redirect focus. A stale callback refuses after a newer agent command. All agent adapters use the same operations directly without simulating input.

## Mounted evidence and remaining work

A fresh owned macOS app was operated through CUA keyboard input: search/select/place DC source at (-200,0), resistor at (0,-110), LED at (200,0), ground at (0,110); connect V1.positive → R1.1, R1.2 → D1.anode, D1.cathode → GND1.reference, and GND1.reference → V1.negative. The completed 1 kΩ circuit showed 2.917 mA. The Selected object menu and value field changed R1 to `2.2k`; the updated display showed 1.345 mA. Cmd+S and the native directory chooser saved the four-part/four-wire graph with exactly 2200 Ω. Read-only inspection verified the saved graph and user attribution. The left rail and live result were visually inspected.

This establishes one LED workflow. Automation required observations between dependent focus, text-selection and menu events; fast batched input was inconclusive. The native dialog produced `Untitled.elab.elab` from the suggested filename, and after it closed subsequent keys did not yield an exposed focused element. Filename handling and native-dialog focus restoration remain follow-ups. RC plotting, recovery, IME, screen-reader use, visible input latency and participant testing remain open. The broader automated smoke is callback/semantic evidence, not proof of those interactions.

## Inspector and clock focus

Selected component controls now put values and switch/wiper actuation ahead of general network summaries and longer model explanations. Back, Next and Center share a row; their existing commands and accessible names remain. All pin/model assumptions remain in the inspector. This improves the default-window first edit without claiming every long form fits above the fold.

A successful native Pause/Resume action explicitly continues focus to the replacement Resume/Pause button once. Tab can then return to Search. Repeated view refreshes do not issue new focus requests. Unavailable commands do not change focus or the clock. Direct agent commands retain their ordinary domain behavior and do not request presentation focus.

The [current-run review](../research/ui-audit/keyboard-inspector/README.md) and [checkpoint receipt](validation/keyboard-inspector-validation.json) record actual search/placement, unit-bearing value apply/undo, and clock-focus checks. CUA omitted several visible selected-component controls from its accessibility tree even while Tab reached them; this unresolved discrepancy prevents full assistive-technology acceptance. Physical pointer/wheel, IME, dialog restoration and complete RC/recovery scenarios remain unqualified.

The 0.101.0 pass gives rail labels bounded containers, repairing the observed long-line clipping without changing label IDs, text, focus order or commands. Native shaped-text bounds and default-window light/dark captures qualify the bounded fix. See [rail text review](../research/ui-audit/rail-labels/README.md). The CUA accessibility omission remains open; a separate native Cocoa probe sees the selected controls, which does not establish reader acceptance.

## Current /257 keyboard qualification

The [October 25 workflow receipt](validation/reliability-five/workflow/report.json)
repeats construction and four named wires using physical keyboard input. Invalid
0 Ω is refused; 2 kΩ applies, Undo/Redo restores the right values, and Resume
shows 1.477 mA. Cmd+S saves `Keyboard LED.elab` with one suffix. The dialog
returns focus to Pause, and Tab reaches the component category. The matching
packaged agent freshly reopens the exact graph cleanly. These cases supersede
the older doubled-filename and missing post-dialog-focus observations above.

In the RC example, keyboard sample steps reach 10 ms and 3.1606 V. Keyboard
cursor entries of 0 ms and 10 ms show Δt = 10 ms. New replaces the untouched
example without a save prompt; an edited restoration shows the protected
Save/Discard/Cancel decision, and Escape preserves the graph. Dependent native
menu choices were observed before continuing; fast input batches are not a
latency or keystroke-throughput qualification. Accessibility participant/IME
and separate-account checks remain outside this engineering evidence.
