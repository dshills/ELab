# P-channel MOSFETs and high-side switching

Choose **Examples → High-side MOSFET LED**. The PMOS source connects to the 5 V supply, drain feeds the current-limited LED, and RPU pulls gate toward source. A high 5 V gate turns the LED off; a low gate turns it on. This reverses the control sense of the NMOS example. Examples now [fit the actual workspace and follow resizing](viewport-fitting.md); **Hide scope** gives the schematic more room. Small-canvas text readability remains a layout follow-up.

The component has named **gate**, **drain**, **source** terminals and an insulated-gate symbol with the opposite body arrow to NMOS. The example rotates it 180° to put source above drain. J1 is an explicit two-wire routing junction for the input; geometric wire crossings alone remain unconnected. The rotated transistor export label now moves to the side away from the gate/base terminal.

Use the existing palette, named wiring and value inspector. Search **PMOS**, **P-channel**, **high-side switch** or **FET**. A generic MOSFET search returns both polarities. Both allocate unused lifetime `M` IDs. The fixed generic device has no editable numeric value; resistor/source values and explicit capacitors control the experiment.

The native menu alias uses the same public generic load command:

```json
{"command":"example.load","params":{"id":"pmos"}}
{"command":"scope.probes.set","params":{"ids":["v:M1","v:M1.gate-source","i:M1.drain","i:M1.gate","i:M1.source","i:D1"]}}
```

As with all mutations, use current revision/document guards and retry keys. Loading protects unsaved work. `parts.list` supplies pins, default model identity and assumptions; filtered `capabilities.list` supplies command details and the complete local schema registry. Native construction/menu/value/plot actions and external clients use the same dispatcher. There is no hidden simulation-only UI path.

## Signed readings and model

Voltage conventions stay **VGS = gate minus source** and **VDS = drain minus source**. Current is positive **into drain** and **out of source**, for both polarities. Normal high-side PMOS conduction therefore has negative VGS, VDS, drain current and source current. The LED current remains positive anode-to-cathode. Power `VDS × ID` is positive dissipation. Gate current is zero; channel cutoff still permits the body diode's reverse leakage.

`project.inspect` exposes `pmos_model` and entries in `mixed.mosfets`. The inspector and existing probes show voltage, current, power and `cutoff`, `triode`, `saturation` or `reverse` region. A normal fully conducting switch uses triode; MOSFET saturation is its current-source region. For PMOS, reverse means VDS is positive. The source-to-drain current convention and normalized VSG/VSD model equations are described in [MathWorks' PMOS reference](https://www.mathworks.com/help/simscape-electrical/ref/spicepmos.html). ELab exports drain-to-source signed values consistently with its NMOS probes.

`elab.generic-pmos.v1` uses **threshold −1.5 V**, **β 0.1 A/V²**, **λ 0.02 V⁻¹**, with body internally tied to source. The channel uses the same square-law implementation as NMOS after reversing voltage/current polarity; analytic gate/drain derivatives retain the appropriate signs through this transformation. Reverse channel flow exchanges the intrinsic ends. Its body diode flows from physical **drain to source**, opposite to NMOS; complete generic silicon-junction parameters are captured with the model.

The two generic devices deliberately share illustrative beta magnitude. They are not a physical matched pair. There is no intrinsic gate/junction capacitance, subthreshold channel, body effect, stored switching charge, breakdown, heating, rating or physical device identity. Use explicit external capacitors for circuit timing. A pull-up to the supply defines the off state in the example; changing gate drive relative to the source changes behavior, rather than treating any nonzero gate as a logic high.

## Gate RC and complementary circuits

Change RG to **10 kΩ** and add **1 µF from gate to source**. With the existing 100 kΩ pull-up, the time constant is `(10 kΩ || 100 kΩ) × 1 µF = 9.090909 ms`. Zero initial capacitor voltage means gate initially equals source. After the input falls at 200 ms, VGS approaches −4.545455 V; after input rises at 500 ms, it returns toward zero. Charge stays continuous at those source edges. This is external RC timing, not a real device's gate charge or Miller behavior.

NMOS and PMOS can share a network. The public reference builds a resistor-loaded complementary inverter with both drains joined and both gates driven together. A low input produces nearly 5 V output, high input nearly zero; an intermediate input conducts through both devices. Results and captures retain both model dependencies and signed branch currents.

`limits.nmos_transistors` remains for existing clients. New `pmos_transistors` and `mosfets_total` expose the envelope: **two MOSFETs total across both polarities**, not two of each. Other drawing/global/device/source/solver budgets remain. DC source terminals may float in a referenced network; waveform-source negatives remain grounded. MOSFET sources themselves may sit above ground. Floating gates require an external reference path. Gate may join source/drain explicitly; a direct drain/source short retains the existing diagnostic.

## Files and validation

Saved projects, recovery inspection, verification and DC sweeps retain optional `pmos_model` parameters. Captured JSON/hash dependencies include the model and its body diode; CSV adds `pmos_model`. Mixed-polarity captures require the model matching each graph part. Missing/tampered parameters, wrong-polarity final models, duplicate result IDs and foreign IDs refuse import. Existing file versions remain; older builds without PMOS will refuse the new kind.

Existing NMOS parameters and serialized field order are unchanged. A frozen model-dependency hash from the 0.88 recording guards that promise; public NMOS reconstruction and actual 0.88 file reopen provide compatibility evidence. The shared Go model/result types retain the original exported NMOS names as aliases. Discovery now references one local `urn:elab:mosfet-model:1` definition for either polarity, avoiding repeated model structures while preserving complete input/output information.

Core tests cover independent channel/load-line references, finite-difference derivatives, body-diode direction, source offset and gate-current conventions, pulse edges, analytic RC timing, complementary conservation, captures/model corruption, snapshot ownership, capacity, gate sweeps and rotated export pin visibility. Shared native construction/example tests and packaged smoke cover both polarities. The public reference requires a **fresh owned blank session** because clearing a document does not release its lifetime IDs:

```sh
python3 scripts/pmos-demo.py --output build/pmos-report.json --artifacts build/pmos-assets
# Add --connect /absolute/path/to/fresh-owned-endpoint.json for native testing.
```

It reconstructs the example, verifies signed pulse/RC behavior, exports and reopens captured models, refuses wrong polarity, then builds/records a complementary inverter. Native mode closes that owned app afterward. No hardware, physical matched-pair, complete mounted input/accessibility or release qualification is inferred from these checks.

The unreleased [recording-consistency repair](recording-consistency.md) validates unprobed signed PMOS readings and stored nodes on import. Its public RC-buffered high-side workflow adds independent reference and failed-file preservation checks alongside the earlier PMOS cases.
