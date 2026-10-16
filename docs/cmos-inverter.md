# Complementary MOSFET inverter

Choose **Examples → CMOS inverter**. The shared input drives a generic PMOS above a generic NMOS, with both drains joined as the output. The 5 V supply and explicit 10 kΩ load make the signed branch readings easy to inspect. MN is selected; the scope starts with input voltage, output voltage, PMOS drain current and NMOS drain current.

The input starts high for 200 ms, then low for 300 ms, repeating every 500 ms. At high input, MN conducts and output is near ground. At low input, MP conducts and output is near 5 V. Pause and Step across 200 ms and 500 ms to inspect each change. Captures retain both sides of the ideal edges. The time scale is deliberately slow for exploration; it is not a model of a physical gate's speed.

Select VIN, set its high value to **2.5 V**, and Restart. Both generic channels now conduct during the high phase. Their drain currents are roughly 50 mA in opposite signed conventions, even though the output's 10 kΩ load draws much less. This is a useful experiment in intermediate gate drive and supply current, not a safe operating rating. Reset current example restores the 5 V pulse, geometry, selection and default plots through the normal unsaved-work guard.

The native menus and external agents use the existing commands:

```json
{"command":"example.load","params":{"id":"cmos"}}
{"command":"scope.probes.set","params":{"ids":["v:VIN.positive","v:MN","i:MP.drain","i:MN.drain"]}}
{"command":"component.set","params":{"id":"VIN","value":"2.5 V"}}
{"command":"simulation.restart","params":{}}
```

The bottom region card and scope caption identify the first MOSFET, MP; selecting MN does not change that whole-network summary. Use the selected inspector or probes for MN. Supply current is available through `i:VCC`; PMOS drain current is negative when it supplies the output, while NMOS drain current is positive into its drain. `v:MN` is drain minus source, equal to output relative to ground in this circuit. `v:VIN.positive` is input relative to ground. The load resistor prevents a floating output when neither channel conducts. It also causes a small high-level voltage drop; it is not a hidden logic-input model.

These are the unchanged [generic NMOS](nmos.md) and [generic PMOS](pmos.md), with ±1.5 V thresholds, β = 0.1 A/V², λ = 0.02 V⁻¹ and internally tied body diodes. They are not a physical matched pair or a named CMOS IC. There is no intrinsic gate/junction capacitance, stored charge, propagation delay, breakdown, heating or rating model. Ideal edges have no modeled transition loss. Explicit external capacitors can add circuit timing through the existing transient solver; they do not introduce a real device's intrinsic charge behavior. The existing two-total-MOSFET and other network/work limits remain.

Drawing, wiring, rotating, editing, saving, reopening, Verify, probes, scope, output and reset all retain their shared user/agent commands. Verify captures both versioned model dependencies without changing the accepted live state. Edited projects retain their example origin and recordings; reopening starts fresh rather than trusting prior live readings.

Validation includes independent scalar load-line checks at 0, 1.5, 2.5, 3.5 and 5 V; live and captured pulse inversion; signed currents and Kirchhoff/power residuals; both ideal-edge sides; native blank construction/menu parity; protected reset/history; and public save/fresh-process reopen/output. The reproducible public workflow requires a fresh owned blank session:

```sh
python3 scripts/cmos-demo.py --output build/cmos-report.json --artifacts build/cmos-assets
```

Add `--connect /absolute/path/to/owned-endpoint.json` for native testing. That mode closes the owned app unless `--keep-open` succeeds. Build evidence and remaining qualification limits are recorded in [the validation receipt](validation/cmos-validation.json).
