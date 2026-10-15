# Diode rectifier and smoothing

Choose **Examples → Rectifier / smoothing**. C1 is selected for quick editing; the scope starts with diode current, capacitor/output voltage and input voltage. Agents use `example.load {"id":"rectifier"}` or the protected `example.rectifier` shortcut. Discovery includes the circuit's question, expected behavior, experiment and model limitations.

The template uses a 5 V peak, 10 Hz sine source, 100 Ω charging resistor, generic silicon diode, 10 µF smoothing capacitor and 10 kΩ load. During positive peaks the diode charges C1. Between peaks it blocks while the load discharges C1, producing ripple. The capacitor starts at zero. These are educational low-voltage equations, with no transformer, mains circuit or physical power-supply rating.

Run the circuit and compare the input and output traces. Change C1 to **47 µF** and compare the last full period after **two simulated seconds**. With the pinned models and 1 ms output interval, measured ripple decreases from about **2.39 V** to **0.69 V**. This result describes this template, not a universal component sizing rule. More capacitance also changes startup and charging-current behavior.

For a repeatable comparison, Pause, Restart, then advance two simulated seconds through the shared Step command in chunks. A paused electrical edit keeps readings stale until Restart/Resume; it does not silently advance time. Scope cursors, fixed windows, current probes and independent Verify work as in the other examples.

```json
{"command":"example.load","params":{"id":"rectifier","discard_changes":true}}
{"command":"simulation.pause","params":{}}
{"command":"simulation.restart","params":{}}
{"command":"simulation.step","params":{"duration_s":"100 ms"}}
```

Normal transport version/request ID, retry key and revision/document guards still apply. These commands replace only the chosen session's document; omit `discard_changes` to preserve unsaved-work protection. Saving and reopening retain the example origin, component values, exact source waveform and immutable diode model. Reset current example restores the template and uses the existing Save/Discard/Cancel policy.

The model includes finite diode series resistance and reverse leakage, but omits reverse breakdown/recovery, junction storage, capacitor ESR/leakage, temperature adjustment and heating. The 1 ms output interval is adequate for the template's 10 Hz sine; increasing frequency may require a smaller interval. Existing sampling/work/convergence limits still apply.

Electrical tests compare the scalar capacitor ODE against separate RK4 integration with the inverse current-domain diode equation, independent of the nodal Newton/backward-Euler network. The first 200 ms agrees within the declared 3 mV bound; the observed maximum error was about 0.98 mV. Tests also check reverse leakage, KCL/power residuals, live/Verify equality, source preservation, larger-capacitor ripple and fresh-process model/template persistence. This is an independent implementation of pinned educational equations, not measured-device or SPICE qualification.

Native regressions exercise the Examples menu, capacitor field, paused Restart and matching scope/results against an agent. The packaged GPU smoke opens this example, advances it with native Step and renders its schematic/scope meshes. [Validation](validation/rectifier-example-validation.json) records these checks; [headless](validation/headless-rectifier.json) and [native](validation/desktop-rectifier.json) demos also reconstruct the same circuit from a blank sheet using ordinary component/wire commands.

After building, reproduce the public workflow with:

```sh
python3 scripts/rectifier-demo.py --output build/rectifier-check.json --artifacts build/rectifier-check-new
```

The artifact directory must be new. Use `--connect` only for an owned development session because the demo replaces its schematic. The demo saves the modified circuit, captured verification and a PNG; it never submits a printer job. Completed Verify adds a kept recording asset while preserving source history and live physics.

Image review exposed a general export defect: conservative label backgrounds could erase a nearby symbol. SVG/PDF/PNG now draw those backgrounds below symbols and pins. A pixel regression fails with the prior renderer and passes with the repaired layer order. Genuine text/symbol overlap in arbitrary crowded drawings still needs layout attention.

For a captured waveform, **Interval measurements** reports time-weighted average/RMS, AC ripple RMS and peak-to-peak over an explicit interval. Compare the same late-cycle interval and sample step for 10 µF and 47 µF. Linear reconstruction between saved rows does not certify steady state or unseen peaks. See [measurement semantics](interval-measurements.md).
