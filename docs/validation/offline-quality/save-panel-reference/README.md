# Standard AppKit Save-panel comparison

October 26, 2026. The filename Command-A failure also occurs in a minimal
**standard NSApplication/NSSavePanel app without GLFW or Vellum**. This narrows
the integration investigation; it does not identify CUA, AppKit, remote-view
routing or the shared temporary Edit-menu helper as the cause. No product repair,
release or completed native acceptance gate results. [Structured report](report.json).

## Reference and observed behavior

The reference uses NSApplication's ordinary run loop, an NSWindow, a button and
NSSavePanel. Its temporary Edit-menu/focus helper is the exact same instrumented
copy used in the [earlier ELab diagnostic](../save-panel-diagnostic/README.md):
SHA-256 `0d15e35de415953cb77a4fad36702678dda217c5f45cf9dce56f5da712bfdf23`.
That copy adds app-local, nonmutating metadata observations to the production
helper; production helper SHA-256 remains
`f23cddd9c93c5c566c69dd8075e00555d9d4e76aa759974bba17164de31c04c4`.
[Exact reference source](reference.m.txt), [helper](file_focus_darwin.m.txt),
[build identity](source-identity.json), [linked frameworks](linked-frameworks.txt).
It uses a private bundle ID and an empty owned directory, with no owner project,
clipboard command, global observer, extra capture technology or security change.

All interactive actions use documented CUA APIs. Opening the panel, typing Alpha
Circuit, collapsing selection with Right, pressing Command-A and typing Beta
produces **Alpha CircuitBeta**. Native Edit → Select All visibly selects that
entire name; typing Gamma then produces **Gamma**. This replacement probe
confirms the menu control independently of the shortcut failure.
[Failed shortcut](beta-after-command-a-ax.txt),
[menu selection](menu-select-all-ax.txt),
[successful replacement](gamma-after-menu-selection-ax.txt).

At native menu-action observations the app is active, NSSavePanel is key,
NSRemoteView is first responder and NSFakeServiceResponder resolves Select All.
The temporary menu uses the Command mask 1048576 and no explicit action target.
The parent local monitor records no panel key events, as in the ELab run. These
observations are at the recorded transitions, not direct evidence of delivery
or activation at the precise failing chord. [Raw app-local trace](native-events.jsonl).

The helper remains common to both runs. A conventional permanent Edit menu could
separately test menu lifetime, but there is no evidence yet that this would fix
ELab. Do not patch production focus/activation/menu behavior from this result.
Original native checks **3,6,9,11 remain open**. No normal-package, physical
keyboard-rate or precise native queue acceptance is claimed.

## Tool timing and normal cleanup

The ordinary software-test parent starts the fixture and observes readiness in
**0.380 seconds**, within an eight-second startup deadline. The subsequent CUA
getApp attachment to that already-running owned process nevertheless reports
**2,690.6643 seconds**, despite timeout_ms=15000. Its cause is unavailable and no
approval rejection is reported. The process is never relaunched. Later button,
field and menu interactions return in under one second each. This bounds app
startup but demonstrates that the UI attachment timeout does not bound tool
wall time; do not keep repeating fresh native attachments unattended.
[Startup](startup.json), [ready](ready.json), [parent recipe](run.py.txt).

Escape cancels the panel. Its completion reads “Cancelled; no file written” and
the temporary Edit menu is removed. The close button ends the owned app normally,
with parent-observed **exit 0** and CUA inventory showing no reference remaining.
The fixture directory stays empty. No Save button is pressed.
[Cancel state](cancelled-ax.txt), [exit](exit.json), [cleanup](cleanup.json).

All **1,354** frozen compiled/test inputs and nine recorded package/SDK archive
hashes stay exact. No unchanged full gate or package is repeated. The build is
warning-free; [closeout assertions](closeout.py.txt) verify result strings,
menu metadata, cancellation, normal exit, empty directory and preserved inputs.

## Repeatability and next work

The recipes record this workstation's absolute source path. In a disposable
workspace, copy reference.m.txt to the staging reference.m and copy the exact
helper to the prior diagnostic staging path expected by [prepare.py.txt](prepare.py.txt),
then build with that recipe. [run.py.txt](run.py.txt) intentionally refuses an
existing ready file; inspect/close a prior fixture before a fresh run. Drive
actual UI only through CUA, and quit normally. The parent never synthesizes input.
The reference itself never writes a selected project, even if accepted.

With native attachment wall time presently uncontrolled, prioritize useful
source/agent/offscreen reliability work: inspect in-use retained menu actions
across intervening public agent edits and document replacement. First establish
reachability and reproduce through the real driver queue before any repair;
keep intentional programmatic/selector/Switch/ToggleButton contracts unchanged
without contrary evidence. Existing slider origin tests already cover held
agent invalidation and must not be rerun merely to fill this gap. Native work
retains priority when tooling can establish the exact remaining scope safely.
