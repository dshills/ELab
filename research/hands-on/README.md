# Hands on comparison of four circuit simulators

Tested October 12, 2026, for a product aimed at hobbyists and makers with room to grow.

**The most promising direction is iCircuit-style immediate feedback with the measurement and reproducibility tools found in LTspice and KiCad.** Falstad adds a particularly accessible way to explore a circuit and expose its internal behavior. This is a design hypothesis supported by the workflows below, not a user preference ranking.

Three prepared circuits were opened and simulated in **all four applications**: an LED with a current-limiting resistor, an RC charging circuit, and an NPN transistor switch. The work included real UI interaction, probing, scope configuration, and screenshots. It did **not** benchmark drawing every circuit from a blank sheet or measure human task completion time.

## Results at a glance

| Application tested | LED operating point | Resistor edit from 1k to 2k | RC waveform | NPN switching | Most useful observed pattern |
|---|---|---|---|---|---|
| iCircuit 1.15.1 | Completed | Immediate readout after editing | Completed; bandwidth affected displayed results | Completed | Select a part to see live measurements; add a scope from its meter |
| Falstad CircuitJS 4.1.5js | Completed | Immediate readout after accepting edit | Completed | Completed | Animated circuit with a contextual scope and explicit transistor quantities |
| LTspice 26.0.2 for MacOS | Completed | Completed with explicit rerun | Completed; 3.16051 V at 10 ms | Completed; 70.3993 mV on state | Probe the schematic and save repeatable numerical measurements |
| KiCad 10.0.3 | Completed | Not established in this pass | Completed; cursor reads 3.16 V at 10 ms | Completed; cursor reads 70.4 mV on state | Select signals, type a cursor time, and show operating-point values on the schematic |

## Read the evidence

- [Workflow findings and screenshots](findings.md) — numbered steps, strengths, friction, accessibility risks, and product implications.
- [Measurements and interpretation](measurements.md) — actual readouts, analytical checks, and model differences.
- [Protocol and reproduction instructions](protocol.md) — exact circuits, versions, test boundaries, and fixture corrections.
- [Design opportunities](design-opportunities.md) — proposed features and concrete acceptance criteria.
- [Screenshot register](screenshots.md) — accepted evidence and captures excluded from the main findings.
- [Structured measurements](measurements.json) — values with units and evidence paths.
- [Circuit files](circuits/) and [observation logs](logs/) — native fixtures, exported netlists, simulation workbooks, and LTspice output.

## What changed our understanding

1. **Live feedback makes the edit and its consequence visible together.** iCircuit and Falstad updated the LED current without a separate Run action after the resistor edit. Preserve this interaction in the new product.
2. **Defaults and model identity belong in the experience.** The three LED models produced different currents. In iCircuit, changing the RC bandwidth setting changed the scope extrema and frequency estimate. A convincing animation is not enough to communicate simulation assumptions.
3. **Precision can be approachable.** KiCad lets the user enter a cursor time directly; LTspice saves numerical measurements in a log. Both patterns could sit behind a simple “Measure at…” action.
4. **Schematic and simulation should remain connected.** KiCad's operating-point annotations and LTspice's node probing both reduce the need to translate between the circuit and its results.

The evidence supports an interaction direction, not a conclusion that any simulator is universally more accurate or easier to learn. These were small generic circuits, on one Mac, operated by an agent rather than a sample of hobbyists.
