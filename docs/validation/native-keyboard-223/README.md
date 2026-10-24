# Native keyboard and text history /223

[Receipt](report.json), [captures and checks](raw.tar.gz), [hashes](raw-manifest.json),
[remaining checklist](native-acceptance-pending.json).

Original native items 14 and 15 pass on the unchanged code-equivalent /223 copy.
The same owned session is reused. No product change, new feature or app launch.

## Text history and selection

A fresh `1 kΩ` component-value field has available circuit history from one agent
move. Command-Z and Shift-Command-Z preserve its initial text. An explicit Select
All remains visible across those empty-history shortcuts; typing `47k` without
reselecting replaces the whole value exactly, proving selection preservation.

Native text Undo changes `47k` to `47`; Redo restores `47k`. Walking the actual
character-level history gives `47`, `4`, then `1 kΩ`. Three further Undos preserve
that boundary. Select All is explicitly established at the oldest entry, then
three additional Undos preserve it. Typing `3.3k` without reselecting again replaces
the whole initial value exactly. A new edit invalidates the previous Redo branch.
This does not claim that Undo itself restores a pre-edit full selection: the
oldest-entry selection is deliberately established before testing its preservation.

The first harness assertion incorrectly expected one Undo to reverse a complete
CUA typeText call. Actual input creates individual character edits; that retained
capture shows a successful local character Undo. No product repair is inferred.

## Editor ownership and positive application controls

Both value and search editors handle Left, Delete, Left, Right, Backspace and
Escape locally. Real deletions produce the expected strings, and Escape restores
the accepted field text without changing focus or circuit state. Search text
Undo/Redo consumes its own character edit while circuit Undo remains available.
Seven value-phase and two search-phase public snapshots equal their respective
accepted baselines completely, including command revisions. Native captures also
retain component position and editor focus after every individual editing key.

After Tab leaves the value editor for Apply, Command-Z restores the resistor's
original position and Shift-Command-Z restores the move. Tab out of search focuses
the component list; the same two shortcuts again undo and redo the circuit move.
All four positive controls deliver exactly one shared command each. Exact history
entry identity, component geometry and dirty-state restoration are checked. After
each Redo, the entire accepted state equals baseline except revisions. Electrical
results, net connectivity, values and unrelated state are exact throughout.

An initial verifier omitted optional `redo_entry` from its allowed history fields.
It stopped before socket access; the corrected verifier asserts that entry's exact
identity rather than weakening electrical/state checks. Both harness corrections
are recorded in the receipt; no observed application defect is hidden.

## Preservation and remaining scope

All 897 frozen source/check inputs, saved fixture, strict installed signature,
original candidate and /205 download remain unchanged. Existing full source
gates still apply; no redundant full suite. The owned disposable fixture is
restored clean, paused at zero, Light, with playback speed 0.1. Raw JPEG bytes
are retained with matching extensions.

Fourteen of eighteen original native checks now have scoped evidence; four remain:
3, 6, 9 and 11. These custom-editor/native-scene checks do not qualify the separate
AppKit Save-panel Command-A issue, controlled slow/fast divider drags, or exact
pre-agent callback queue origins. /223 stays held and /205 remains the preserved
download. Continue remaining form/parser ownership and investigate those distinct
native routes under the saved authorization, without owner-review or launch waits.
