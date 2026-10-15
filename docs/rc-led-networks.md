# Capacitor-buffered LED and RC high-pass circuits

**Examples → LED fade** opens a 5 V ideal pulse source, 1 kΩ current limiter, generic red LED, 100 µF capacitor across the LED, and ground. C1 is selected. The input is high for 200 ms and low for 300 ms. The scope compares source voltage, capacitor voltage and LED current. The capacitor charges and LED current rises; when the source falls, capacitor voltage stays continuous and stored charge keeps LED current flowing briefly. The low source also drains the capacitor through R1.

Change C1 to **220 µF**, then Restart and compare the same time after the falling input edge. A larger capacitor changes startup as well as the tail. Brightness is illustrative; ideal capacitance and the generic LED omit ESR, leakage, temperature changes, device ratings and calibrated light output. This is an educational transient experiment, not a physical component sizing specification.

Agents use `examples.list` and **`example.load {"id":"led-fade"}`**. The native menu ID `example.led-fade` is a presentation alias for that existing shared command, not a new public API. Save/Discard/Cancel, generation/revision guards, undo, origin/reset guidance and saved projects follow the existing example contract. Ordinary component placement, named-terminal wiring, value, pulse-clock, probes, Verify, interval measurements and exports work on the same circuit.

## Routing repair

Previously, one resistor + one capacitor + one LED entered the legacy RC-only compiler and was refused before reaching the capable network solver. It now enters the bounded general network compiler. The exact one-pole RC low-pass path remains unchanged. If the specialized RC compiler declines another topology, its general network compiler now validates it; it may still refuse unsupported sources, ideal constraint loops or other existing limits. Incomplete circuits remain incomplete, with stale previous results and explicit diagnostics.

This also enables a series-capacitor **RC high-pass**: source positive → C1.1, C1.2 → R1.1, R1.2 → ground, source negative → ground. With 5 V DC, R=10 kΩ, C=1 µF and zero capacitor startup voltage, the resistor starts at 5 V and decays as **5 exp(-t/10 ms)**. Plot `v:R1` for output, and `v:C1` for stored voltage. At 10 ms the analytic resistor voltage is about 1.8394 V and capacitor voltage about 3.1606 V. Terminals 1→2 define signed capacitor voltage/current.

No numerical kernel, model, integration settings or file version changes were required. Existing network/source-grounding, capacitor/source impulse and independent startup-constraint rules remain. The routing regression was demonstrated failing before the repair for both connected circuit families.

Independent tests compare capacitor-buffered LED charging against a scalar RK4 integration using an inverse current-domain Shockley equation, with a 3 mV capacitor-voltage envelope and KCL/power checks. High-pass output is compared with the independent exponential. Pulse tests verify duplicate-time left/right rows preserve capacitor voltage across the input edge, then LED current decays; save/reopen restores configured parts and fresh charge/time.

After building, run `python3 scripts/rc-network-demo.py --output build/rc-network-check.json --artifacts build/rc-network-check-new`. It reconstructs the fade circuit from blank using ordinary commands, compares it with the guided template, tests two capacitances, captured measurements, save/reopen, high-pass response and schematic SVG/PNG output. Use `--connect` only for an owned blank session; the demo replaces and closes it.

The public reference workflow finds **29.1904 µA** LED current at 210 ms with 100 µF, versus **236.799 µA** with 220 µF, ten milliseconds after the source falls. The high-pass output at 10 ms is **1.84049 V**, versus the analytic **1.83940 V** (about **1.09 mV** difference). Complete native/headless discovery and observations match except transport. These values describe the pinned template and sampling settings; see [public evidence](validation/rc-network-public-parity.json).
