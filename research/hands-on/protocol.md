# Circuit comparison protocol and reproduction

## Scope and environment

Test date: October 12, 2026. Host: macOS 26.6.2, build 25G83. Audience: hobbyists and makers who may later need deeper analysis and a path to hardware.

| Product | Tested environment | Version evidence |
|---|---|---|
| iCircuit | Existing native Mac application | Bundle version 1.15.1, build 11510277 |
| Falstad | Hosted CircuitJS in the in-app browser | File → About displayed 4.1.5js |
| LTspice | Official Mac distribution installed with user approval | Simulator logs report 26.0.2 for MacOS; bundle 26.0.2.1 |
| KiCad | Existing Mac schematic editor and integrated simulator | 10.0.3; visible schematic title block and installed application |

The circuits were generated as local fixtures, then opened and exercised through each application's UI. Native symbol libraries were used for the KiCad and LTspice schematics. Simulation results were observed in the applications; analytical expectations were calculated separately. KiCad's CLI was used to inspect exported connectivity, not as a substitute for its UI simulation tests.

The interaction sample covers opening/importing, running, component inspection, LED resistor editing in three apps, RC scope configuration, numerical reading, and NPN switching. Manual drawing from blank, PCB layout, physical construction, vendor model import, large circuits, AC analysis, digital logic, and comprehensive fault recovery are outside this pass. KiCad resistor editing and tuning were attempted but not established through the automation interface; this is an evidence gap, not a finding that KiCad lacks them.

No task-duration or click-count ranking is reported. Window sizes differed, some apps reused existing preferences, and automation focus/launch problems would confound a human usability measurement.

## Shared circuits

### 1 LED and current limiting resistor

Connect 5 V DC → 1 kΩ resistor → LED anode; connect cathode to ground. Inspect current and forward voltage. Change the resistor to 2 kΩ and observe the response.

- iCircuit: generic LED, forward-voltage-at-1-A parameter 2.102426 V, red channel 100%.
- Falstad: imported legacy LED identified by the UI as `old-default-led`.
- LTspice and KiCad: educational model `D(Is=1e-20 N=2 Rs=1)`, at the engines' reported 27 °C.

These are **not one identical LED model across four tools**. Compare how users discover and change the assumptions; do not rank accuracy from different currents. The SPICE model is not a validated model of a purchasable LED.

### 2 RC charging and discharging

Drive a 10 kΩ resistor and 1 µF capacitor with a 0–5 V square wave, 10 Hz, 50% duty. Measure capacitor voltage relative to ground.

- iCircuit/Falstad source: 2.5 V amplitude plus 2.5 V offset.
- SPICE source: `PULSE(0 5 0 1u 1u 50m 100m)`.
- LTspice: `.tran 0 200m 0 10u`; measurement at 10 ms.
- KiCad: `.tran 10u 200m`; cursor at 10 ms.
- iCircuit: scope duration 300 ms; compare observed automatic bandwidth of 256 Hz with manual 100 kHz.
- Falstad: 10 µs timestep in exported test file; scope displayed 20 ms/div.

Time constant is `RC = 10 ms`. For an ideal step starting at zero, `V(10 ms) = 5(1 − exp(−1)) = 3.160602794 V`. A 1 µs rise gives a slightly different value, so this ideal expression is a close check rather than an exact SPICE target.

For periodic steady state with ideal 50 ms high/low phases, let `a = exp(−5)`. Then `Vmin = 5a/(1+a) = 0.033464255 V` and `Vmax = 5/(1+a) = 4.966535745 V`. Use these extrema when interpreting the continuously running tools. The iCircuit legacy capacitor file retains an initial 1 mV value, so the tools' startup conditions are not identical.

### 3 NPN low side switch

Connect a 1 kΩ collector load to 5 V, emitter to ground, and base through 10 kΩ to the same 0–5 V, 10 Hz source. The collector should go low with a high base drive and return toward 5 V with a low drive.

- iCircuit/Falstad: generic NPN, beta 100, native models.
- LTspice/KiCad: `NPN(Is=1e-14 Bf=100)`.
- SPICE runs: 200 ms, 10 µs nominal/max step as configured in the native fixtures.
- LTspice measurements: collector voltage at 25 ms and 75 ms.
- KiCad cursor: collector at 25 ms; off state also observed in the waveform and an earlier cursor position.

This resistive-load circuit tests basic switching. It does not test a motor, relay, inductive kick, protection diode, transistor heating, or suitability of an actual transistor.

## Reproduce the runs

1. **iCircuit:** open each `.cir` in [circuits/icircuit](circuits/icircuit/). Select the LED or transistor for live values. For RC, select the capacitor and add Vd to the scope using its meter control. Set the time window to 300 ms. To reproduce the finer RC result, turn off Automatic Bandwidth and enter `100k` in Circuit settings. The saved RC file has a 10 µs step; do not treat cached scope extrema in the text file as a fresh measurement. Restore `1k` after the LED edit if retaining the baseline.
2. **Falstad:** visit the [CircuitJS application](https://www.falstad.com/circuit/circuitjs.html), choose File → Import From Text, and paste a fixture from [circuits/falstad](circuits/falstad/). The `-tested.txt` RC and NPN exports include scope settings. Right-click a component for View in New Scope. The resistor edit uses its value dialog; accept the edit to observe the new current.
3. **LTspice:** open `.asc` files from [circuits/ltspice](circuits/ltspice/) and press Run. The LED uses `.op`; transient circuits use `.tran`. Click the output wire to plot it, then the input wire for a comparison trace. Read `.meas` results in the generated `.log`. `01-led.asc` is the 1 kΩ baseline; `01-led-2k.asc` reproduces the edited condition. The latter fixture was assembled from the baseline after the UI test; its saved output files are copied from the actual edited run and retain the original internal circuit path.
4. **KiCad:** open each `.kicad_sch` from [circuits/kicad](circuits/kicad/), choose Inspect → Simulator, and Run. The schematic's analysis directive supplies the initial analysis. Select signal plot checkboxes and enable an output cursor; double-click its time cell to enter `10m` for RC or `25m` for NPN. Saved `.wbk` files retain trace and cursor configuration, not simulation data; load them through the simulator's workbook controls and rerun as needed. The schematics are for simulation, with no PCB footprint assignment.

Reference [plain SPICE decks](circuits/spice/) describe the intended topologies. They were not separately executed in a fifth simulator.

## Fixture and environment corrections

- The Falstad transistor's collector/emitter endpoints differ from the legacy iCircuit geometry. The generated import initially left a gap. This was corrected before accepting switching results. It is not scored as a product defect.
- The generated KiCad LED had an invalid `Sim.Type` value. Removing that value selected its built-in diode model; the final exported netlist verifies `D1 /OUT 0 __D1`. A temporary rotation experiment was reverted. Only the corrected forward-biased result is included.
- KiCad required numeric reference annotations for the NPN fixture's `RB` and `RC`. Its guided annotation dialog produced `RB1` and `RC1`; the saved schematic and final netlist contain those references.
- Older-format notices came from the generated KiCad file version. The tested files were saved by KiCad. These notices are not evidence of a normal new-project problem.
- KiCad CLI cache initialization failed under the initial restricted execution environment; a successful permitted export and final re-exports verified all three netlists. Fontconfig warnings were unrelated to circuit results.
- LTspice's official package was downloaded, its Apple signature/notarization checked, and installation approved by the user. The temporary unpacked launch was not usable through automation. The installed application completed the tests. Installation/automation delays are excluded from usability comparisons.
- Native focus and coordinate targeting were intermittently unreliable. An unexpected empty KiCad window was recovered by reopening the named fixture. We did not diagnose an application crash and do not count it as one.

## Supporting documentation

The observations above come from this run. Documentation was consulted to locate features and resolve preparation issues: [iCircuit manual](https://icircuitapp.com/usermanual/), [official LTspice download page](https://www.analog.com/en/resources/design-tools-and-calculators/ltspice-simulator.html), and [KiCad simulation model documentation](https://docs.kicad.org/8.0/en/eeschema/eeschema.html). The older KiCad documentation was used as background only; the tested behavior is from 10.0.3.
