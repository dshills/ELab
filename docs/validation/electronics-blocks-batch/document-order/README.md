# Document inspector order repair

The complete final rendered race invocation reproduced clipped document output
controls in light, dark, system and Apollo themes. PDF margin/label/following
controls had zero visible bounds in the existing 320 by 1100 rendered viewport.
Other tests in that invocation passed; the failed invocation is retained exactly.

The new Circuit blocks group had no explicit sort priority, so the map default
placed its seven-field form before Schematic output. Assigning blocks priority 6
and Help priority 7 preserves the existing Share/print and output priorities.
There is no command, component, solver, SDK or test-assertion change.

Existing form-spacing/paint checks across all themes, inspector navigation,
block commands and user/agent parity pass together under the rendered race driver
in 9.296 seconds. Complete Anthropic Claude Opus 5.5 Prism review covers the
single production file (one chunk, no skipped/truncated input, zero findings).
The affected complete rendered suite is rerunning before packaging; final
receipts belong in the batch qualification directory. Root race, ordinary desktop
race, all vets, lint, 77 Python tests, PRD and inventory had otherwise passed.

These tests use the rendered driver and do not establish native OS queue or
hardware-rate acceptance. Native checks 3/6/9/11 remain open.
