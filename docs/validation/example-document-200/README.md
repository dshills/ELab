# Example clean baseline and title repair /200

October 23, 2026. ELab 0.180.2. [Receipt](report.json),
[public protocol checks](protocol-report.json), [package](package-report.json).

The owner reported a Save prompt when switching untouched examples, and an
Untitled circuit heading for a loaded example. The shared loader now establishes
the graph/settings and presentation baseline from the loaded template. Its file
association remains empty; explicit Save can still keep a copy. Live physics,
pause/step and navigation do not create an edit. Real graph/settings, display/pin
and kept-result changes remain protected. Reset returns to a clean template;
undo/redo compares against the same baseline. No dirty-guard bypass is added.

The title area and footer use the example catalog title. A saved filename takes
precedence; genuinely unnamed blank schematics retain Untitled circuit. The
existing example metadata already supplies the same name to agents. No file
format, model or public command is added.

The initial regression reproduced schematic and preference dirty flags on an
untouched LED template. Old protection/recovery fixtures that depended on that
bug were updated to make real edits. The initial desktop suite stalled waiting
for a picker its obsolete fixture no longer opened; that owned process was
stopped and the corrected fixtures passed. The first full check could not bind
private Unix sockets in the sandbox; the complete check passed with the local
socket access those tests need. No production guard or solver limits were relaxed.

The packaged agent checked all 18 shipped templates without discard consent,
ran/stepped RC without setting dirty, confirmed real-edit refusal and undo-to-clean,
saved a clean example, switched again, and closed with workers joined. UI tests
cover legacy/generic menu routes against the agent, all example names, saved-name
precedence, edited indicators, and real-edit dialog/recovery protections. The
full source check and all 67 Python tests passed.

ZIP extraction into a Unicode/spaces path preserved all members and modes and
passed strict ad-hoc signature verification. No desktop control was used for
this patch. Open owner apps and previous packages were preserved. Physical UI
acceptance, earlier pointer/flasher limits, clean-account installation and full
PRD/VoiceOver/IME acceptance remain outside this receipt.
