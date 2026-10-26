# Native Save-panel observation

October 26, 2026. **No production repair is justified by this batch. Native
check 3 remains unqualified.** The qualified /267 and all earlier packages are
unchanged; no components, commands, solver limits or release bytes change.
[Structured report](report.json), [app-local trace](native-events.jsonl).

## Bounded diagnostic

An isolated copy of desktop source differs only in absolute module replacements
and app-local observations added to file_focus_darwin.m. Native window/menu
notifications and a local NSEvent monitor record metadata: activation, key/main
window classes/numbers, first-responder/action-target classes, event codes and
modifier masks. No text, window titles or other-app data are logged by that
observer. It returns every event unchanged and never activates a window,
changes a responder or sends an editing action. The private bundle identity
isolates picker preferences; appearance/recovery paths and the LED project are
owned fixtures. This is an unqualified diagnostic, not a replacement /267 build.
[Source identity](source-identity.json), [diagnostic-only patch](diagnostic-only.patch),
[preparation recipe](prepare.py.txt). Build succeeds; its optional Go module-stat
cache write is sandbox-denied, as retained in build.txt. No security settings or
new accessibility/capture grants are used.

The first CUA launch call reports 11,984.9751 seconds before returning its exec
wrapper lookup error. Actual endpoint startup occurs at 07:28:21 UTC; attaching
to that running app takes 0.1349 seconds. It is not launched again. The reason for
the long call is unavailable: no rejection or missing authorization is reported.
Do not infer that the owner must authorize ELab again. Later UI calls use a
15-second execution timeout. [Launch and guard notes](tool-launch-and-guard.txt).

## Observed native behavior

The saved synthetic LED file supplies a private directory and filename. Native
Save As opens NSSavePanel. Replacing its selected name with Alpha Circuit,
collapsing with Right, pressing Command-A and typing Beta produces
**Alpha CircuitBeta**: the shortcut did not select the name. The native Edit →
Select All action selects the complete Alpha CircuitBeta string. A further Right/Command-A
readout reports no selected-text annotation; without a replacement probe it does
not provide an independent selection-success/failure verdict. Escape cancels the
first panel. No Save button or clipboard command is used.

At observed panel/menu transitions NSApp is active and NSSavePanel is key.
The successful Select All menu action has NSRemoteView as first responder and
NSFakeServiceResponder as its resolved target. The menu uses the Command mask
1048576 and a nil explicit target. Both native menu WillSendAction and
DidSendAction notifications are recorded. A simple inactive-app explanation is
not supported by those observations; no activation patch is made.

The parent process records no local key events while the Save panel is active.
A later control with the GLFW window key records both Escape down/up, including
the expected window number and key code 53. This proves the local observer can
see ordinary parent-window keyboard events. It does **not** establish which
process/path received the failing panel chord, or whether the cause lies in
CUA, AppKit, GLFW or ELab. Remote view-service routing is an investigation lead,
not a demonstrated cause or fix.

## Modal-target control and cleanup

A second Save As opens in the same owned process. Reattaching the documented
app binding during that modal control leads to a UI-tool state guard: it says
the app changed and requires a fresh state read. The recorded read shows the
GLFW main window while the picker remains pending; the trace also shows that
key-window transition. No second shortcut result is produced or claimed. This
is not evidence of a direct human return or of a product focus defect. The state
is re-read before another action; a benign Root Escape supplies the ordinary
key-monitor control above.

The first full snapshot baseline preceded a viewport publication, so only
viewport height 927→959 and its revision/command-revision/actor differ after the
first Cancel. Circuit, document, physics, history and measurements stay exact.
The settled snapshot is then entirely unchanged through the inconclusive modal
reattachment control. The saved fixture's recorded bytes and mtime stay exact;
no additional project is saved. Normal public app.close cancels the still-pending
owned picker, and CUA confirms no ELab fixture remains running.

All 1,354 qualified executable/test inputs and eight prior package/SDK identities
are verified unchanged. No unchanged full gates or new package are run. Original
native checks 3/6/9/11 remain open. Owner review is not a prerequisite for further
engineering work.

## Next control

Prepare a minimal owned NSApplication/NSSavePanel reference using the same
production temporary Edit-menu helper. Drive its buttons/fields only through
CUA. Compare its shortcut behavior with this GLFW/Vellum diagnostic before
changing application focus or menu code. A bounded software-test startup runner
and attachment to an already-running fixture should avoid an unbounded fresh
launch call; it must keep existing security and approval settings intact. If
native tooling cannot run that control, continue source/agent/offscreen work and
record the specific technical limit. Do not repeat identical Save trials or
change intentional SDK contracts without a reachable reproduction.
