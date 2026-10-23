# Inspector guidance and native file editing

October 23, 2026. Follow-up to the held /206 cursor candidate and native Save
baseline. /205 remains the downloadable maintenance build; /204 is preserved.
These source repairs introduce no components, simulation modes or public commands.

## Guidance in the active task

Native /206 retains and corrects cursor drafts, but Simulate filters out the
group containing the shared bounds and unit instructions. Switching inspector
pages clears that request's feedback, so sending the user to Inspect cannot
recover it. The earlier source regression fails on Simulate and Document.

The shared guidance group now belongs to all three inspector pages. Only the
selected page is mounted, preserving unique IDs and existing command guards.
On Simulate and Document it precedes task controls; Inspect keeps its prior
ordering. The current error summary also exposes its matching action as
accessibility/hover help. Unit checks cover all pages, unchanged accepted state
and identical shared instructions. Mounted Vellum tests at 320×360 in light and
dark confirm readable wrapped action text inside the scroll viewport. These
renders use the managed test font and do not replace a fresh macOS app check.

The command ledger was compared before refreshing source hashes: 166 public
commands, 134 source routes, 404 binding sites and 76 production Go sources.
Every non-line site field, alias and coverage set is unchanged. Only `view.go`
and `document_workspace.go` hashes change. The first audit attempt used an old
compiled auditor and correctly failed on newer command IDs; rerunning the
current auditor resolves that tool mismatch. Objective-C menu delivery remains
outside this Go source inventory.

## Native filename editing

The [Save baseline](83-native-save-keyboard.md) establishes exact path behavior,
including intentional repeated suffixes. No path normalization is added.
Command-A fails in the native filename field; a missing standard Edit menu is
a plausible cause, with actual key acceptance still to verify.

The native focus bridge now supplies a temporary AppKit editing group during
the file-dialog interaction. Undo/Redo, Cut/Copy/Paste and Select All use nil
targets and standard key equivalents, allowing Cocoa's first responder to
handle text. A token prevents an old completion from removing a newer group.
Only the group owned by this helper is removed; an existing native Select All
menu is preserved. The original focus continuation retains its window checks.

A native object fixture includes the actual production Objective-C helpers and
compiles with warnings treated as errors. It verifies shortcuts/modifiers,
responder targets, group placement, replacement and stale/repeated completion,
preservation of original and pre-existing editing items, and nil/zero inputs.
It opens no window, changes no global menu and leaves the clipboard untouched.
Its focused test passes. This establishes the helper contract, not actual
Command-A, panel Cancel/Save, or application shutdown behavior in a rebuilt app.

## Authorization and remaining verification

The owner repeatedly authorized all ELab launches and desktop testing. That
standing instruction is saved in `AGENTS.md`. The launch tool nevertheless
presented repeated prompts, so the owned paused /206 session is kept available.
Subsequent scope/inspector wheel and scrollbar checks reuse it successfully.
Document also scrolls to its final help rows and back to the export buttons.
Command-Shift-S opens the native Save panel in this same /206 session; Cancel
returns to the drawing. The shortcut is registered and works here, so an
earlier unsuccessful attempt is not grounds to add or change that binding.
Continue agent/source/offscreen work without leaving unattended work waiting
for another launch prompt. No approval or security setting is weakened.

The [full source gate passes](../docs/validation/inspector-guidance-207/README.md):
lint/staticcheck, 67 Python tests, complete root/desktop race and vet, required
mounted-control race and offscreen vet. Desktop race takes 176.694 s and mounted
race 150.333 s. The first run stopped
because non-graphical ordinary agents could not bind their Unix sockets; an
independent socket probe reproduces `EPERM`. The passing rerun uses automatically
reviewed local socket access. Both outcomes are preserved. Fresh native checks of
the changed guidance layout and editing keys remain required before promoting
the next maintenance candidate; existing /206 images cannot establish them.

The clean [0.181.6 /207 candidate](../docs/validation/inspector-guidance-207/candidate-report.json)
builds from `99564f53`, verifies its strict ad-hoc signature and passes the
matching headless cursor protocol. Both live/recorded modes preserve state on
refusal, accept correction and reject stale ownership; shutdown completes with
workers joined. All 865 captured source/check input hashes still match. The
candidate remains held for those native checks; /205 remains the download.
