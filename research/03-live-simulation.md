# Live simulation: preserve the experience, define the meaning

**Research anchor:** you value iCircuit because it shows a live simulation while you draw. The product opportunity is to make electronics feel directly explorable, with enough clarity to support learning and later engineering work.

## “Live” describes several different things

| Meaning | What the user experiences | What this research establishes |
|---|---|---|
| Always-on circuit analysis | Draw or adjust and see behavior without repeatedly configuring a run | Explicit iCircuit positioning. [S01](https://icircuitapp.com/index.html) |
| Interactive simulation | Start once, then change controls while the circuit responds | EveryCircuit and SimulIDE document this. [S07](https://everycircuit.com/help) [S59](https://simulide.com/p/) |
| Interactive virtual instruments | Connect a scope/meter and adjust it during simulation | Multisim and Proteus document this. [S44](https://www.ni.com/en/support/documentation/supplemental/07/explore-circuit-behavior-using-simulation-driven-instruments-in-.html) [S47](https://www.labcenter.com/instruments/) |
| Firmware co-simulation | Buttons, peripherals and code interact | Wokwi and Proteus describe supported MCU workflows. [S50](https://docs.wokwi.com/) [S48](https://labcenter.s3.amazonaws.com/downloads/Tutorials_p9.pdf) |
| Updated analysis after an edit | A new calculation is launched; results replace an earlier run | A possible design choice, not equivalent to continuous state evolution |
| Live electrical/design-rule checking | Connections or layout constraints are checked during editing | Useful verification, but it does not simulate circuit behavior |
| Live 3D view or cloud sync | Rendering or document state updates | Also useful, but unrelated to electrical simulation fidelity |

For every competitor tested later, record exactly which edits remain possible while running: a resistor value, a switch, a new wire, deleting a component, changing a device model and changing firmware. These are separate tests. Marketing language cannot answer all six.

## What to learn from iCircuit

The manual shows a useful interaction chain: vary a component with a slider or exact value, inspect it with a meter, and add a measured quantity to the scope. Connected wires can follow moved components. Examples can be restored after experimentation, and the scope has recording/history controls. These mechanisms reduce the cost of trying an idea. The same manual also documents simulation-bandwidth assumptions, demonstrating that approachable interaction does not remove numerical constraints. [S02](https://icircuitapp.com/usermanual/)

The design lesson is **shorten the experiment loop while preserving the user’s understanding of what happened**. That is more specific than “add animation.”

## Proposed interaction contract

The following requirements are **design hypotheses**, not findings about what any existing tool already guarantees.

1. **The user can see whether a circuit is running, paused, incomplete, stale or unsupported.** Each state has distinct text and visual treatment; do not rely only on color.
2. **Ordinary edits do not freeze the canvas.** If solving is expensive, the editor stays responsive and clearly indicates which results are current.
3. **Incomplete drawings are normal.** Explain the relevant open connection or missing reference on the canvas. Avoid modal error dialogs for every temporary edit.
4. **A failed solve never masquerades as a result.** Keep earlier traces only when labeled as belonging to a prior revision. Never silently substitute zero for failure.
5. **Changing a circuit has defined state semantics.** Identify whether capacitors, inductors, MCU memory and waveform history continue, reset or belong to a prior run. Offer an explicit restart.
6. **Time is honest.** Show simulated time separately from elapsed wall time; slowed simulation should not be labeled real-time without qualification.
7. **Numbers remain available.** Pair animation with units, magnitude, sign and probe direction. Allow conventional/electron-flow choices only if their meaning is explained.
8. **Direct manipulation and exact entry coexist.** A knob supports exploration; exact values, units and reversible changes support reproducibility.
9. **A model’s scope is inspectable.** A logic-level MCU model may answer a code question while omitting supply current or analog behavior. Label these boundaries.
10. **An experiment can be frozen and shared.** Save circuit revision, model versions, stimuli, initial conditions, analysis settings and selected measurements together.

These requirements follow from the gap between immediate interaction and trustworthy interpretation. Wokwi’s explicit analog boundary and TI’s compatibility guidance are particularly useful examples of limits that should remain visible. [S51](https://docs.wokwi.com/chips-api/analog) [S22](https://e2e.ti.com/support/tools/simulation-hardware-system-design-tools-group/sim-hw-system-design/f/simulation-hardware-system-design-tools-forum/692613/faq-tina-spice-note-about-encrypted-models-and-importing-models-from-one-simulator-to-another)

## One project, several levels of investigation

A promising approach to test is a common project with two clearly named activities:

- **Explore:** direct controls, lightweight probes and rapid feedback for a supported circuit.
- **Analyze:** specified conditions, exact analyses, saved results and comparisons.

This does not require two different numerical engines. A shared engine may reduce inconsistency; separate engines may improve responsiveness or model coverage but introduce reconciliation work. Select an architecture only after prototypes establish latency, fidelity, model compatibility and state behavior. Do not silently switch models to preserve animation speed.

A candidate part model should distinguish at least: schematic symbol, behavioral/simulation model, pin mapping, PCB footprint, physical part identity and provenance. These can appear as one usable part in the interface while remaining explicit in project data. EasyEDA’s unified device concept and LibrePCB’s pin/package concepts provide references; extending them with simulation readiness is our proposal. [S29](https://prodocs.easyeda.com/en/introduction/introduction/) [S64](https://librepcb.org/features/library-concept/)

## Experiments that should shape the design

| Experiment | User-visible task | What to observe | Potential failure |
|---|---|---|---|
| LED and resistor | Adjust current and explain brightness | Can the user relate value, current and power? | Animation implies realistic LED behavior without an appropriate model |
| RC charging | Change resistance/capacitance while inspecting voltage | Does the user understand the transient and any state reset? | Edits silently discard stored energy or mix old/new traces |
| Inverting amplifier | Change gain and drive into clipping | Can the user distinguish ideal from supply-limited behavior? | An ideal model looks like a validated real amplifier |
| Oscillator | Start, pause, restart and change a component | Are startup assumptions and history clear? | A solver failure is mistaken for a nonworking circuit |
| Switch and inductive load | Open a switch and inspect the response | Are transients visible and supported? | Slow animation hides a short, large spike |
| MCU plus input/display | Change an input while firmware runs | Can the user identify code versus circuit errors? | Unsupported peripherals appear to work |
| Broken circuit | Remove ground or leave an input floating | Can the user recover without a tutorial? | An error message names an internal node with no visual location |
| Growing circuit | Duplicate blocks and add probes | Does interaction remain usable as cost grows? | Canvas stalls, dropped data or misleading time display |

The later benchmark should use documented component models and independently calculated or measured expected behavior where feasible. Agreement between two simulators alone is not proof of physical validity.

## Provisional targets for testing

These are proposed acceptance targets, not achieved results or industry standards:

- A new maker reaches a correct, changing measurement within five minutes on the LED task.
- A basic edit receives visible UI acknowledgment within about 100 ms; define a separate solve-latency target per circuit class after measurement.
- Users can distinguish a valid current result from a paused or stale one without explanation.
- Users can recover from an intentionally introduced disconnected wire and undo an accidental deletion.
- Users can describe at least one model limitation before deciding whether to build the circuit.

Collect task success, unprompted recovery, time, assistance requests and explanation accuracy. Use satisfaction as supporting evidence; a compelling animation can score well even when it teaches the wrong lesson.
