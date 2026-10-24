# Circuit capacity

The tested envelope admits **64 simulated parts and 100 electrical nodes including ground**, with up to six scope traces. Drawing capacity is separately **100 objects and 300 wires**. Ground, junctions and net labels are helper objects; they do not consume the simulated-part budget. Ideal-source branch-current unknowns are not counted as electrical nodes.

`Limits()` is the shared core definition. Every snapshot exposes `limits` and `simulated_component_count`; the native Inspector shows physical/drawing/node counts against those same values. Placement and persistence use the drawing bounds. General simulation uses the physical/node/device bounds, with explicit `mixed-limit` diagnostics. A drawing beyond simulation capacity remains movable, editable, undoable and savable within the drawing bounds. No model substitution or hidden connection is introduced to fit the limits.

## Published bounds

| Resource | Bound |
|---|---:|
| Simulated parts, combined | 64 |
| Electrical nodes, including ground | 100 |
| Drawing objects, including helpers | 100 |
| Wires | 300 |
| Sources | 3 |
| Capacitors | 4 |
| Inductors | 4 |
| MOSFETs, NMOS/PMOS combined | 2 |
| Powered op-amps | 2 |
| 555 timers | 2 |
| Bipolar transistors, NPN/PNP combined | 2 |
| Resistors / potentiometers | 64 each, subject to combined part/node bounds |
| SPST switches / silicon, Schottky and Zener diodes combined | 8 switches; 8 total diodes |
| Generic red LEDs | 4 |
| Selected scope traces | 6 |

Existing device ranges, grounded waveform-source negative terminals, floating/open-pin checks, ideal-constraint restrictions, numerical tolerances, output work guards and operation/document size bounds remain in effect. DC sources may float within a ground-referenced network. The original capacity expansion retained its numerical kernel. Current unreleased general capacitor networks use [SDIRK2 with new engine identities](capacitor-integration.md); device models and the published work/accuracy bounds remain. The envelope does not imply convergence for every arrangement of parts.

## Fixtures and evidence

A source plus 50 series 10 kΩ potentiometers has 101 nodes when every wiper is unloaded. Tying the final wiper to pin 1 creates a valid rheostat and reduces the graph to **exactly 100 nodes**, using **51 simulated parts / 52 drawing objects**. Its effective series resistance is 49 × 10 kΩ + 5 kΩ = 495 kΩ; the 5 V current is 10.1010101 µA. Independent resistor equations check each unloaded wiper voltage/current, the tied-wiper branch and KCL/power conservation. The initial 49-track candidate had 99 nodes, so it was corrected rather than counted as a 100-node proof.

DC and ideal-square versions run at the node boundary. Four live channels capture the source, two wipers and a branch current through a pulse edge. Verify, embedded recording validation and fresh paused project reopen retain all 100 nodes. A separate source plus 63 series 1 kΩ resistors tests the **64-part boundary** against 5/63,000 A. A 101-node drawing and a 65-part drawing are diagnosed and remain savable. These meet the PRD's original minimum-size target in representative supported fixtures; they do not establish universal topology, hardware, visible-latency or usability acceptance.

Core/native race tests, lint and vet cover the changes. Native construction uses the same placement/connection batches and bounds; the GPU smoke additionally checks actual schematic and four-trace scope pixels at 100 nodes. Public `scripts/capacity-demo.py` builds the fixtures, compares independent values, verifies/embeds a record, saves/reopens, exports SVG and saves excess complexity. Use `--connect` only for a fresh owned blank endpoint; the admission guard refuses existing work. Use `--output REPORT --artifacts DIRECTORY` to retain results.

The existing `.elab` format remains version 4; model parameters and dependency hashes are unchanged. Current readers use the expanded drawing bounds, while legacy file byte limits remain unchanged. Older application versions with smaller numerical/drawing limits may refuse the newly larger projects.

The optional `elab-conformance --capacity-suite` adds all four adapters, complete 200 ms captures for these two pulse fixtures and a 64-part four-stage RC/NMOS LED circuit, half-step refinement, exact recording reopen and editable/savable boundary refusals. It recomputes 4,900 numerical references per adapter. See [capacitor integration and public capacity workflows](capacitor-integration.md). The initial checkpoint left repeated timing open; the [capacity timing protocol](capacity-performance.md) measures the three named boundary fixtures separately.

The [555 extension](timer555.md) admits two stateful timers without expanding existing node/storage/work limits. Historical capacity fixtures do not qualify timer capacity; exact pre-timer envelopes remain accepted for those passive/RC/MOSFET fixtures.

The [bipolar extension](bipolar-transistors.md) admits PNPs and ungrounded emitters within the combined two-transistor bound. The original capacity fixtures below remain passive/RC/MOSFET evidence; their exact historical limit envelope does not claim PNP capacity qualification.

Basic logic gates count toward the existing 64 simulated-part and 100-node
envelope. No separate gate-count limit is introduced. Each single gate has
four (NOT) or five named terminals; supply, input and output nets count normally.
