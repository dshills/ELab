# Circuit measurements and interpretation

Values below are **displayed readings or saved simulation output**, not bench measurements. Precision differs between instruments. Extra digits in a log are not evidence of greater real-world accuracy.

## LED current

| Application | Current at 1 kΩ | LED voltage at 1 kΩ | Current at 2 kΩ | Evidence |
|---|---:|---:|---:|---|
| iCircuit | 3.271 mA | 1.729 V | 1.658 mA | [Baseline](screenshots/icircuit-02-led-running.jpg), [edit](screenshots/icircuit-03-led-2k.jpg) |
| Falstad | 3.195 mA | 1.805 V | 1.615 mA | [Baseline](screenshots/falstad-02-led-running.jpg), [edit](screenshots/falstad-03-led-2k.jpg) |
| LTspice | 2.91679 mA | 2.08321 V | 1.47672 mA | [Baseline](screenshots/ltspice-07-led-1k-readable.jpg), [edit](screenshots/ltspice-06-led-2k-readable.jpg) |
| KiCad | 2.91678 mA | 2.08322 V | Not tested successfully through UI | [Console](screenshots/kicad-05-led-op.jpg) |

At 1 kΩ, each result satisfies `I ≈ (5 − VLED)/1000` within displayed rounding. LTspice and KiCad use the same specified diode parameters and agree to approximately 0.00001 mA at the displayed precision. iCircuit and Falstad use different native models, so their different values do not demonstrate an accuracy defect. The expected qualitative result—less current after increasing resistance—was observed in all three completed edit workflows.

## RC response

| Reading | Expected reference | Observed | Interpretation |
|---|---:|---:|---|
| LTspice Vout at 10 ms | 3.160603 V for ideal instantaneous step | 3.16051094075 V | Close agreement; actual source has a 1 µs rise |
| KiCad Vout at 10 ms | Same ideal reference | 3.16 V | Agreement to cursor display precision |
| iCircuit minimum, initial automatic 256 Hz bandwidth | 33.4643 mV periodic steady state | 67.4482 mV | Noticeable difference; investigate resolution before trusting extrema |
| iCircuit maximum, initial bandwidth | 4.96654 V | 4.93255 V | Same qualification |
| iCircuit minimum, manual 100 kHz bandwidth | 33.4643 mV | 33.55 mV | Much closer to analytical reference |
| iCircuit maximum, manual 100 kHz bandwidth | 4.96654 V | 4.97 V | Consistent with coarse displayed rounding |
| Falstad minimum | 33.4643 mV | 33.481 mV | Close agreement in this fixture |
| Falstad maximum | 4.96654 V | 4.967 V | Agreement to displayed precision |

Evidence: [LTspice log](circuits/ltspice/02-rc.log), [KiCad cursor](screenshots/kicad-04-rc-cursor.jpg), [iCircuit initial](screenshots/icircuit-04-rc-scope.jpg), [iCircuit finer setting](screenshots/icircuit-06-rc-fine-result.jpg), [Falstad scope](screenshots/falstad-04-rc-scope.jpg).

The displayed scope frequency estimates were 10.24 Hz for the initial iCircuit setting, 10.01 Hz at the finer setting, and 9.952 Hz in Falstad. All sources were configured to 10 Hz. These estimates depend on the scope's sampling/windowing behavior and are not accepted as independent measurements of oscillator error. No engine implementation cause was established.

The first-step measurement and periodic extrema test different conditions. They must not be combined into one simulator error score. Startup voltage, timestep policy, numerical method, and display precision also differ.

## NPN switch

| Application | On-state collector/emitter voltage | Off state | Evidence and limit |
|---|---:|---:|---|
| iCircuit | Scope minimum 50.60886 mV | Scope maximum 5 V | [Scope](screenshots/icircuit-08-npn-scope.jpg); native model, initial automatic bandwidth |
| Falstad | Not accepted as a numerical measurement | 5 V in cutoff readout | [Scope](screenshots/falstad-05-npn-scope.jpg); captured 0 V minimum includes startup and is not a saturation measurement |
| LTspice | 70.399299264 mV at 25 ms | 5 V at 75 ms | [Saved measurements](circuits/ltspice/03-npn-switch.log) |
| KiCad | 70.4 mV at 25 ms | Approximately 5 V on waveform; cursor at 79.9 ms displayed 5.00 V before final capture | [Final cursor](screenshots/kicad-07-npn-cursor.jpg) |

All four showed the expected switching behavior. Only the SPICE pair used the same specified NPN parameters. No named physical transistor was characterized, and no physical measurement was made.
