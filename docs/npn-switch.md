# NPN transistor LED switch

Implemented October 13, 2026. The subsequent [NPN transient slice](npn-transient.md) adds pulse base drive, a live clock, and a voltage/current scope; the DC record below describes this earlier slice. This slice adds a drawable generic NPN with named `base`, `collector`, and `emitter` terminals, live DC switching, and shared user/agent operation. It extends the LED and RC work without treating transistor timing as a transient simulation.

## Try it

Open ELab and choose **NPN**. The editable example contains a 5 V supply, a separate 5 V base-drive source, a 10 kΩ base resistor, a 1 kΩ LED load resistor, red LED, NPN, and ground. The load resistor is vertical for readable wiring. **0 V**, **0.8 V**, and **5 V** buttons edit the base-drive source through `component.set`; these are ordinary undoable, saved circuit edits. They are available without selecting the source. Pause an experiment to keep its reading; an electrical edit marks that reading stale until Resume.

To build from blank, place these parts and connect:

- Supply positive → load resistor → LED anode; LED cathode → NPN collector.
- Drive positive → base resistor → NPN base.
- NPN emitter and both source negatives → ground.

Either resistor pin order is accepted. LED-before-load-resistor order is also supported. A single shared supply/base source or a resistor-only collector load is supported. Separate sources make base drive convenient to control independently. Extra connections, shorts, open pins, reversed LEDs, pulse sources, and mixed capacitor/transistor circuits produce explicit diagnostics and stop valid-result publication. Coordinates, rotation, IDs, and line crossings do not define electrical connections.

The schematic labels B/C/E and illustrates LED brightness. The readout shows collector/LED current, collector voltage VCE, and cutoff/active/saturation state. Base current, VBE, emitter current, transistor power, and an explanation appear alongside the drive controls. Selecting the NPN reveals its model assumptions. The complete structured result also contains net voltages, individual powers, KCL/KVL residuals, device IDs, engine/model identity, run/revision identity, and stale status.

## Model and numerical method

Model `elab.generic-npn.v1` uses reciprocal Ebers–Moll transport equations at 27°C with IS = 1e−14 A, forward beta = 100, and reverse beta = 1. The [official ngspice manual](https://ngspice.sourceforge.io/docs/ngspice-manual.pdf) documents the simplified Ebers–Moll case and the corresponding IS/BF/BR parameters. This is an educational model, not a characterized physical transistor.

With `F = IS expm1(VBE/VT)` and `R = IS expm1(VBC/VT)`, the engine uses `IB = F/BF + R/BR` and `IC = F − (1 + 1/BR)R`. It solves base KCL by bisection for each collector voltage, then bisects collector KCL against the resistive/LED load. LED forward voltage uses the existing generic red LED model; reverse LED current is neglected. Saturation emerges from the junction equations rather than a fixed VCE clamp. The solver checks convergence, supports cancellation, and publishes only for the current run/electrical revision.

State explanations use both junction biases; cutoff requires |IB| and |IC| below 10 nA. The generic model excludes Early effect, breakdown, heating, junction capacitance, and switching/storage delay. Source range remains 0–12 V and resistors 10–100000 Ω. This engine handles a single grounded-emitter switch, not arbitrary transistor networks or pulse timing. Existing RC transient behavior is unchanged.

## Agent and persistence contract

`parts.list` discovers kind `npn`, model ID, and the three named pins. `component.place`, `wire.connect`, `component.set`, move/rotate/delete, undo/redo, pause/resume, save/load, and inspection are the same service commands used by the UI. `example.npn` loads the editable example with the existing unsaved-change protection. `simulation.wait` waits for the DC solve; it does not advance time.

Snapshot `npn` is a separately typed result; LED-only `result` and RC `transient` remain their own contracts. `npn.engine` is `elab.npn-switch-dc.v1`, with embedded `npn.model`. Legacy top-level `model` remains LED library metadata. Clients should inspect analysis, status, run/revision, and stale state before consuming readings. Socket response allowance is now 512 KiB so the complete 36-command schema catalog remains discoverable; request and schematic-file limits are unchanged.

Version 2 schematic files containing an NPN embed `npn_model` in addition to each component's pinned model ID. Loading requires the supported exact model metadata. Missing or changed parameters are rejected. Existing version 1 LED and version 2 LED/RC documents continue to load. Earlier app builds reject the new field/kind explicitly. Source drive is saved as its component value; solver results and device state are recomputed on opening.

## Validation and evidence

| Example drive | State | Collector / LED current | VCE |
|---|---|---:|---:|
| 0 V | Cutoff | 2 × 10−14 A leakage | 4.249469 V |
| 0.8 V | Active | 1.367808 mA | 1.589712 V |
| 5 V | Saturation | 2.860330 mA | 57.533808 mV |

The cutoff collector voltage with an LED is affected by the generic LED's forward drop at tiny leakage current; it is not the resistor-only 5 V off-state benchmark and does not represent visible light.

The resistor-only research fixture produces **70.399219138 mV** VCE, versus **70.399299264 mV** in the saved LTspice measurement (difference about 80 nV). This passes the PRD's 1 mV tolerance. The comparison is a DC on-state check against the existing [research measurements](../research/hands-on/measurements.md), not a new transistor transient or hardware validation. The new LED topology has model/conservation checks; it has not received a separate external SPICE run.

Validation includes root/desktop race tests and vet, three transistor states, current/voltage/power balance, source/resistor bounds, cancellation, geometry/pin-order independence, LED-first annotation, broken/shorted/unsupported graphs, paused stale readings, undo, embedded-model rejection, and fresh-session reopening. Actual UI callbacks draw from blank and agree with direct agent results. Native smoke exercises LED, RC, and NPN controls and GPU pixels. Headless stdio and the real desktop socket both run the repeatable construction/save/reopen workflow.

Evidence: [final validation summary](validation/npn-validation.json), [headless data](validation/headless-npn.json), [desktop data](validation/desktop-npn.json), [portable schematic](validation/npn-led.elab), and [native UI](validation/npn-switch-ui.jpg).

Run `python3 scripts/npn-demo.py --output docs/validation/headless-npn.json`; add `--connect /absolute/path/to/endpoint.json` for a desktop session. The script uses generated IDs and can be repeated in a persistent session. `./scripts/check.sh`, `./scripts/build.sh`, and `./build/elab --smoke` check the implementation.

Remaining MVP work includes general DC/transient networks, transistor switching timing, selectable probes/cursors, waveform captures/export, recovery, command batches, performance/accessibility qualification, and broader platform acceptance.

For emitter followers, emitter resistance, complementary pairs and PNPs, use the [broader transistor network support](bipolar-transistors.md). This article describes the retained specialized grounded-emitter engine.
