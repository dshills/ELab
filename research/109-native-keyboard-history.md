# Native keyboard and history acceptance

October 24, 2026. [Evidence and checklist](../docs/validation/native-keyboard-223/README.md).

Native items 14 and 15 pass. Fresh and oldest text-history boundaries preserve
initial value and explicit selection, proved by replacement typing. Actual native
text Undo/Redo works with CUA's character-level edits. Search and value deletion,
arrows and Escape stay local. After Tab reaches actual Apply/list controls,
Command-Z and Shift-Command-Z correctly undo/redo a reversible circuit move.

Private public snapshots remain exact; each of four application-history actions
delivers one command, restores the expected history entry/geometry/dirty state,
and preserves electrical results. Two harness assumptions were corrected openly:
typeText is not one coalesced Undo record, and `redo_entry` is optional history
metadata whose exact identity must be asserted after Undo. No product change.

Source, signature, saved fixture and /205 remain exact; the owned app is restored
clean and paused at zero. Fourteen original native checks are complete. Remaining:
3, 6, 9 and 11. AppKit Save shortcuts and exact native queue/rate qualification
remain separate. Continue those routes and current form/parser ownership under
standing authorization without waiting for owner review or another launch prompt.
