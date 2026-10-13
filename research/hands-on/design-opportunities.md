# Product opportunities from the circuit comparison

These proposals translate observed interactions into features to test with hobbyists. They are not a claim that combining every feature would make the easiest product. Keep the starting surface small and reveal detail when a question requires it.

| Priority | Proposed behavior | Evidence | Acceptance criterion for a prototype |
|---|---|---|---|
| P0 | Simulate while editing a valid circuit; keep numerical meters beside the selected part | iCircuit steps 1–2; Falstad steps 5–6 | Changing the LED resistor updates current visibly after value acceptance, without opening another workspace |
| P0 | Offer explicit Plot voltage and Plot current actions | iCircuit step 3; Falstad step 7 | A first-time user can find the capacitor voltage plot without knowing a hidden icon or context-menu convention |
| P0 | Show model identity and assumptions in plain language | Different LED results; NPN native-versus-SPICE models | A user can tell whether a part is an educational approximation or a named device model, and inspect the source and parameters |
| P0 | Expose simulation health and stale results | iCircuit bandwidth experiment; explicit rerun workflows | A plot says when it is stale, paused, unconverged, or potentially under-resolved; automatic resolution choices remain inspectable |
| P0 | Keep results attached to the circuit | KiCad step 13; LTspice step 10 | Selecting a plotted trace highlights its node/component; selecting the circuit exposes the corresponding measurements |
| P1 | Provide editable measurement cursors and named measurements | KiCad steps 12,14; LTspice steps 10–11 | “Voltage at 10 ms” can be entered directly, saved, and reproduced after reopening |
| P1 | Explain device state using the measured quantities | iCircuit step 4; Falstad step 8 | A transistor's cutoff/saturation explanation identifies the relevant base and collector conditions |
| P1 | Save an experiment with its model, analysis, traces, and settings | Falstad text export; KiCad workbooks; LTspice logs/decks | Reopening preserves the question being asked; a result records the circuit revision and settings that produced it |
| P1 | Separate startup, current window, and steady-state measurements | Falstad NPN minimum; RC reference conditions | An extrema readout states its interval and lets the user exclude startup explicitly |
| P1 | Repair missing prerequisites in context | KiCad annotation prompt | A missing model, reference, ground, or connection links to the relevant object and offers an applicable correction |
| P2 | Preserve previous results while exploring a change | Immediate edit loops and explicit result snapshots | A user can compare 1 kΩ and 2 kΩ currents and revert the change without reconstructing the experiment |

## Suggested initial experience

Start with a schematic canvas, a compact component inspector, and an optional scope. The inspector shows value, current, voltage, power, and model identity. A clearly labeled Plot action adds a trace. A compact simulation status explains whether the display is live and what approximation it is using.

“Measure” expands to cursor times, extrema intervals, and repeatable named checks. “Model details” expands to parameters and model provenance. “Verify” saves a bounded analysis and a result snapshot. These are proposed layers of the same workflow; the tests do not establish that two simulation engines are necessary.

## User tests needed before committing the design

1. **Draw the LED circuit from blank.** Observe part search, rotation, snapping, junctions, ground, undo, and value entry. This is the largest missing part of the current hands-on pass.
2. **Ask what changes when resistance doubles.** Compare prediction, discovery of the edit control, and ability to explain the result. Test novices and experienced makers separately.
3. **Find the RC time constant.** Compare a live meter, a scope cursor, and a named measurement. Check whether users understand the difference between initial charging and periodic operation.
4. **Diagnose a deliberately disconnected transistor pin.** Evaluate connection feedback and whether the repair explanation teaches the user what happened.
5. **Replace a generic part with a real device model.** Check whether users notice the change in assumptions and can carry the circuit toward a physical build.

Measure successful completion, help requests, recoverable errors, interpretation accuracy, and confidence versus correctness. Human task time becomes meaningful only with controlled versions, equivalent starting states, counterbalanced order, and a consistent viewport. Do not substitute agent automation time for that study.
