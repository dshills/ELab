# Divider callback origin

October 25, 2026. A rendered ELab regression reproduces a queued divider
request overwriting a later agent layout after the retained descriptor rebinds:
0.75 becomes 0.6699999570846558 and command revision 6 becomes 7, actor user.
The originating KeyChord is dispatched with a one-event budget; its owner
callback remains queued. A public service agent command then changes the layout,
and a benign routed event refreshes the descriptor before owner delivery.
This uses real driver queues and ELab's workspace callback/context binding, with
minimal pane contents. It is not a replay of a captured Change callback, nor
proof of a hardware/native OS queue interleaving.

A held off-center pointer gesture also borrows a refreshed callback after a
layout change, RC input change or document replacement. The corrected baseline
has four failures: one queued-key case and three held-agent cases. The initial
trial used invalid RC mode `high`; the fixture was corrected to `charge`, and
both raw logs are preserved. Unchanged repaints and fresh gestures are positive
controls, alongside forward/inverse quick and classified-slide geometry tests.

SDK elab6 captures the callback when a queued event is admitted and when a
pointer gesture begins. Latest bounds still clamp and disabled/retired or
reordered/reoriented splits still refuse. ELab's existing field context advances
only after its own accepted dispatch, so an uninterrupted held drag continues.
The driver's former latest-callback test is intentionally updated: the concrete
ELab/agent reproduction demonstrates why callback refresh is unsafe. A fresh
independent request still uses the rebound callback.

No public commands, component kinds, models, solver limits or project formats
change. The SDK's full archive member comparison changes only split.go,
split_test.go and generated source documentation; no dependency contents change.
All qualified packages and earlier SDK archives remain preserved.

Status: focused rendered ELab regressions pass on the isolated repaired SDK;
full dependency and application qualification is in progress. Original native
checks 3, 6, 9 and 11 remain open. This repair does not explain or qualify the
previous CUA inverse-drag behavior or native Save-panel shortcuts.
