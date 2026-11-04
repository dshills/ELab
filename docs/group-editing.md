# Select and edit several components

In Select mode, drag on empty schematic space to draw a selection box. Components
whose bodies fit inside the box are selected. **Shift-click** adds or removes a
component. Drag one selected component to move the group; connecting wires remain
attached. Choose Pan (H) or use the middle mouse button to pan instead. Escape
cancels an unfinished gesture; Clear selection exits the group.

The selection toolbar offers Copy, Paste, Duplicate and Delete. With the schematic
focused, Command/Control-A selects all, C copies, V pastes and D duplicates. These
shortcuts do not intercept text-field/browser copying. Arrow keys move a selected
group one grid step; Shift-arrow moves ten steps. The toolbar is an alternative
to every copy/paste/delete shortcut.

Each move, duplicate, paste or delete is one undo action. Group moves validate
final positions together, so a member can move into another selected member's
former position. A collision outside the group or an out-of-bounds position
rejects the whole edit. Concurrent edits during a drag produce a conflict rather
than moving a newer circuit silently. Delete includes wires touching deleted
components; Undo restores the whole operation.

Copy captures an independent session clipboard, including internal wires and all
component settings. Later edits to the originals do not change the clipboard.
Paste/duplicate allocate fresh IDs and copy wires whose endpoints are both inside
the selection. External wires are not copied. Visible named net labels receive
new shared names within the copied group, preserving its own connections without
joining the originals. Block copies remap members, ports and parameter bindings;
selecting an expanded member selects its owning block for group operations.

Clipboard content is session-local, absent from projects/backups and cleared when
the host ends. A clipboard replaced by another actor is explicitly refused using
its observed ID. Paste and Duplicate search for free grid positions around the captured group.
The engine rechecks the destination before committing. Collision errors preserve
existing work. Exact offsets are also available through the command API.

The engine commands are `selection.copy`, `.paste`, `.duplicate`, `.move` and
`.delete`, with ordinary retry/document/revision guards and full detailed
discovery. Compact catalog descriptions are shortened to keep the existing
64 KiB discovery budget; detailed command descriptions and schemas remain intact.
