# N-channel MOSFET circuits

In a small window, **Hide scope → Fit** gives the circuit more room; **Show scope** returns to the measurements. Automatic opening fit remains a layout follow-up.

Choose **Examples → MOSFET LED switch** to see a 5 V pulse drive an LED through M1. The high phase lasts 200 ms and the low phase 300 ms. Pause, then Step to compare gate-source voltage, drain-source voltage and drain current. Select M1 for operating-region and signed-current readings or its three plot shortcuts. Search **MOSFET**, **NMOS** or **FET** to place one in a blank schematic.

M1 has named **gate**, **drain** and **source** terminals. Current enters drain and leaves source; the source-current reading uses that outgoing convention. Gate current is zero in this model. Cutoff removes channel conduction; the body diode still contributes reverse leakage. The example's 100 Ω gate resistor and 100 kΩ pull-down form a small voltage divider, so high VGS is about 4.995 V. The 1 kΩ series load resistor limits the LED current. A fully conducting MOSFET normally operates in **triode** (also called the linear/ohmic region), unlike the saturated NPN switch. MOSFET **saturation** means its current-source region, not its lowest-resistance state.

Build from blank using the same parts palette, named-wire controls and value inspector as other parts. Native buttons, menus and agents all reach the same public commands:

```json
{"command":"example.load","params":{"id":"nmos"}}
{"command":"component.place","params":{"kind":"nmos","id":"M1","position":{"x":100,"y":60}}}
{"command":"wire.connect","params":{"from":"M1.source","to":"GND1.reference"}}
{"command":"scope.probes.set","params":{"ids":["v:M1","v:M1.gate-source","i:M1.drain","i:M1.gate","i:M1.source"]}}
```

These fragments illustrate separate actions; placement follows a blank sheet rather than a loaded template that already contains M1. Mutations use the existing revision/document guards and retry keys. `parts.list` exposes pins, model identity, aliases and limitations. There is no editable numeric MOSFET value; this version uses fixed generic parameters. The native `example.nmos` alias invokes generic `example.load`, including Save/Discard/Cancel replacement protection.

`project.inspect` includes `nmos_model` when the graph contains an NMOS. Fresh `mixed.mosfets` entries report `vgs_v`, `vds_v`, `gate_current_a`, `drain_current_a`, `source_current_a`, `power_w`, the model and the region (`cutoff`, `triode`, `saturation`, or `reverse`). A reverse state means VDS is negative; channel and body-diode contributions are combined in drain current. Probe `v:M1` measures drain minus source; `v:M1.gate-source` measures gate minus source. Individual terminal-to-ground voltage probes remain available. Move/rotate preserve connectivity and accepted physics. Electrical edits retain stale readings while paused until restart/resume.

## Model and bounds

`elab.generic-nmos.v1` is an educational enhancement-mode three-terminal model. Body is tied internally to the physical source; beta already includes geometry. Its fixed parameters are **VT = 1.5 V**, **β = 0.1 A/V²**, and **λ = 0.02 V⁻¹**. With overdrive `u = VGS − VT` and `VDS ≥ 0`, channel current is:

- `0` for `u ≤ 0`.
- `β (u·VDS − VDS²/2) (1 + λ·VDS)` for `0 ≤ VDS < u`.
- `β u² (1 + λ·VDS)/2` for `VDS ≥ u`.

This uses the conventional long-channel square-law regions described in [MathWorks' Level 1 NMOS equations](https://www.mathworks.com/help/simscape-electrical/ref/spicenmos.html) and the parameter meanings documented in the [ngspice manual](https://ngspice.sourceforge.io/docs/ngspice-manual.pdf). It is a chosen ELab model, not a claim of full SPICE MOS1 or named-device equivalence. For negative VDS the intrinsic channel ends exchange roles. A separately oriented source-to-drain body diode uses the existing generic silicon junction (Is 1 nA, n 1.7, Rs 1 Ω, 27 °C); its complete parameters are embedded in the NMOS model.

The implementation stamps analytic derivatives into the existing bounded network Newton solve. Gate and source need external reference paths; floating ideal gates fail rather than acquire a fabricated leakage path. Gate may explicitly join source or drain. A direct drain/source short retains the existing shorted-component diagnostic. A source above ground is supported; voltage sources retain the network's grounded-negative restriction. Limits retain **two NMOS** as a per-kind ceiling and now expose **two MOSFETs total across N/P polarities**, in addition to the existing NPN and global network limits. No numerical work limits are raised.

There is no subthreshold channel, body effect, intrinsic gate/junction capacitance, charge storage, switching loss, breakdown, heating, rating or physical device identity. Instantaneous pulse switching is therefore quasistatic device response. Add an explicit external gate-source capacitor to study circuit timing: changing RG to 10 kΩ and adding 1 µF across gate/source produces a 9.0909 ms time constant with the example's 100 kΩ pull-down. The capacitor starts at the configured initial voltage and retains charge across pulse edges. This does not model a real device's gate charge or Miller plateau. [PMOS high-side switching](pmos.md) is also available; configurable device models remain future work.

## Persistence, captures and validation

Schematics, document/recovery inspection, captured verification and DC sweeps retain the fixed NMOS parameters. Stored model changes or missing required NMOS metadata refuse import. Saved recordings hash the entire model, including its body diode; verification CSV adds `nmos_model`. Existing file versions remain; older builds without this component will refuse it. SVG/PDF/PNG and preview/printing use the new symbol through the existing schematic output path.

`workbench/nmos_test.go` checks independent load lines, exact channel reference points, analytic Jacobian differences, source offsets, body-diode orientation, pulse edges, capacitor startup/analytic RC curves, conservation, snapshots, model guards, saved recordings and gate-voltage sweeps. Native tests construct from blank through palette/wire/value controls and compare agent results, exercise protected template loading and paused staleness. Packaged smoke exercises both pulse phases, inspector/plot controls, circuit and scope meshes.

Run the public reference against a fresh owned blank session:

```sh
python3 scripts/nmos-demo.py --output build/nmos-report.json --artifacts build/nmos-assets
# Add --connect /absolute/path/to/owned-endpoint.json for a native session.
```

The reference checks blank/template equality, gate/drain/source probes, pulse corners, geometry preservation, captured model CSV/JSON, save/reopen, tampered-model refusal and analytic external gate RC charge/discharge. Native runs close the owned app afterward. Outputs and asset directories must be new. Full generic-device hardware qualification and complete mounted input/accessibility testing remain separate gates.
