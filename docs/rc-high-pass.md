# RC high-pass edge detector

Choose **Examples → RC high-pass filter**. A 0–5 V square source drives a 1 µF capacitor in series with a 10 kΩ resistor to ground. The output is across R1. Input, R1 output and C1 voltage are already plotted; R1 is selected for inspection and editing. Pause and step across 50 ms to see the negative output pulse, or change R1/C1 to compare decay times.

For a constant input between edges, `Vout(t) = Vout(t₀) exp(−(t−t₀)/RC)`. Each ideal source edge adds its voltage change to the output while capacitor voltage stays continuous. Here `RC = 10 ms`: output starts at +5 V, falls to about −4.96631 V at 50 ms, then reaches −1.82700 V at 60 ms. These are ideal-circuit predictions, with no ESR, leakage, component ratings or additional output loading.

The public command is the existing generic loader:

```json
{"command":"example.load","params":{"id":"rc-high-pass"}}
```

Ordinary mutation guards and retry keys apply. `examples.list` supplies the guidance, and filtered `capabilities.list` discovery includes the ID in the loader’s input schema. Native menu aliases invoke the same loader. Example reset restores the circuit, experiment and its three default probes through the existing unsaved-work/document protection. Other examples retain their previous default probes; custom high-pass probes survive Save/Open. Older builds that predate this example ID refuse files retaining its new origin metadata.

The drawing can also be built from blank with the ordinary component/wire/value commands. Live stepping, negative voltage/current inspection, independent Verify, kept captures, saved origin, undo/redo and SVG/PNG/PDF export all use existing public commands. Paused electrical/history edits leave old readings stale until restart/resume. Reset clears session recordings under the existing replacement contract; previously saved captures remain in their files.

Analytic checks found an accuracy issue that was hidden when inspecting the larger capacitor voltage alone. At 17 ms, the previous integration setting put the roughly 0.913 V output about 1.47 mV above its analytic value. Non-waveform networks now check both non-ground terminal voltages of a coupling capacitor as well as its stored voltage. The extra node check uses 5e−8 V absolute plus 1.25e−6 relative tolerance. Grounded capacitors, waveform networks and mixed RC/NPN retain their established error paths. The numerical method, device equations and 512-attempt/100 ms per-output budget are unchanged. The high-pass checks require the PRD-style `max(1 mV, 0.1% of reference magnitude)` bound for each output/capacitor value, plus current and conservation checks and both sides of each pulse edge.

Run the public reference in a fresh owned session after building:

```sh
python3 scripts/rc-high-pass-demo.py \
  --output build/rc-high-pass-report.json --artifacts build/rc-high-pass-assets
```

Use new output paths. Optional `--connect` uses an owned native endpoint; it closes that window unless `--keep-open` succeeds. The reference preserves an edited circuit and its original capture across fresh-process reopen, checks protected reset/history and pure output, and measures twenty transport-inclusive 200 ms Verify completions. Backend timing is separate from visible UI latency, difficult/capacity fixtures and other hardware. The model, physical-device, accessibility, distribution and participant acceptance limits remain.

The 0.98.0 /113 checkpoint measures a maximum analytic voltage error of **0.689 mV** over 125 captured rows; every value passes its pointwise tolerance. On the recorded Apple M4 Pro workstation, twenty 200 ms captures per transport have observed p95 **65.46 ms stdio /63.10 ms native socket**, maximum **66.57 ms** across all forty. These are fresh-kernel backend captures in already running processes, with OS caches unchanged. [Validation receipt](validation/rc-high-pass-validation.json), [captured data](validation/rc-high-pass-capture.json) and [native workflow](validation/rc-high-pass-native.json) preserve the scope and build identities.
