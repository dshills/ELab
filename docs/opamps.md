# Powered op-amps

**Later addition:** [optional bandwidth and slew dynamics](hobbyist-devices.md#optional-op-amp-dynamics) are now available through the inspector and `opamp.dynamics.set`. The original model and old saved circuits remain the default; the historical qualification below describes that base model.

Source implementation in progress after 0.105.0. The canonical `build/ELab.app` remains the previously qualified 0.105 app. A separate pinned-SDK candidate now passes packaged smoke, public electronics suites and amplifier output checks. [Current native/output review](../research/ui-audit/opamp-output/README.md) records small fitted labels, a persistent tooltip, incomplete external accessibility and remaining release gates. The later [native mouse review](../research/ui-audit/native-mouse/README.md) advances the earlier value-editor findings; it does not establish complete native input acceptance.

The shared part library now includes `opamp`, searchable as **op-amp**, **operational amplifier**, **buffer**, or **voltage follower**. Its automatic IDs start at **U1**. Native palette placement, terminal wiring, source/resistor value edits, rotation and movement use the existing public commands. The gain and internal parameters are fixed generic model assumptions, rather than editable component values.

## Build a follower

Start on a blank sheet. Place a powered op-amp, three voltage sources, a 1 kΩ load resistor and ground. Set the signal source to 1 V and the two supply sources to 5 V. Connect:

| Connection | Purpose |
| --- | --- |
| Signal positive → `U1.noninverting`; signal negative → ground | Input relative to ground |
| Positive-rail source positive → `U1.vplus`; negative → ground | +5 V rail |
| Negative-rail source positive → ground; negative → `U1.vminus` | −5 V rail, using an ordinary nonnegative source magnitude |
| `U1.out` → `U1.inverting` | Negative feedback |
| `U1.out` → load pin 1; load pin 2 → ground | Resistive load |

The output should be close to 1 V. A single 5 V/ground supply also works for inputs within its modeled output range. For an inverting amplifier, ground `noninverting`, feed the signal through an input resistor into `inverting`, and connect a feedback resistor from `out` to `inverting`. The frozen 10 kΩ input /20 kΩ feedback reference gives approximately −0.999969336 V for a 0.5 V input with ±5 V rails and a 10 kΩ load.

Agents use the same `component.place`, `wire.connect`, `component.set`, `component.rotate`, `component.move`, `scope.probes.set`, `simulation.restart` and `simulation.step` commands. For example, on a fresh sheet:

```json
{"command":"component.place","params":{"id":"U1","kind":"opamp","position":{"x":0,"y":0}}}
{"command":"wire.connect","params":{"from":"U1.out","to":"U1.inverting"}}
{"command":"scope.probes.set","params":{"ids":["v:U1","v:U1.input-difference","i:U1","i:U1.vplus","i:U1.vminus"]}}
```

These are intent fragments. Real requests need the normal protocol envelope, revision/document guards and retry keys. Inspect `parts.list`, `capabilities.list` and the current snapshot before constructing or editing an existing document. [Connected-demo admission rules](demo-sessions.md) apply to standalone demo programs; general agents retain normal guarded editing access.

## Measurements and signs

All five pins have individual voltage-to-ground and current probes. The selected native inspector reports output voltage/current, input difference, supply span, signed supply currents and device power. Its plot shortcuts use `scope.probes.set`, including the same six-trace limit as agents.

| Probe | Meaning |
| --- | --- |
| `v:U1` or `v:U1.out` | Output relative to ground |
| `v:U1.input-difference` | Noninverting minus inverting input |
| `v:U1.supply` | Positive minus negative supply |
| `i:U1` or `i:U1.out` | Current delivered out of the output pin |
| `i:U1.noninverting`, `i:U1.inverting` | Current into that input pin |
| `i:U1.vplus`, `i:U1.vminus` | Signed current into that supply pin |

The output-current convention is explicitly opposite to internal current into the output. Negative supply current can be negative during ordinary operation. Terminal currents, unique wire currents and device power use these declared signs consistently. A positive supply span reports `powered`; zero reports `unpowered`; negative reports `reversed-supply`. Reversed supply disables the active stage and is an explanatory model state, not a simulation of protection or damage.

Input common mode needs an external reference. The compiler connects the input pair through its declared 10 MΩ resistor, and output to V− through its declared 1 GΩ bleed. Supply/control coupling supplies no reference to the input pair, and V+ requires an external reference. In op-amp networks an ideal MOS gate also supplies no reference path. Floating groups refuse. Follower feedback, tied inputs and equal power rails are intentional joins; an unloaded output remains measurable through the declared bleed.

## Fixed model and storage

`elab.generic-opamp.v1` is an ELab educational macromodel. It has 100,000 V/V center-region gain, 10 MΩ differential resistance, a 50 Ω output-resistance scale, a 20 mA active-current scale, 1 mA quiescent-current scale, 0.1 V supply soft-enable scale and an always-present 1 GΩ output-to-V− bleed. The voltage target is smoothly limited around the supply midpoint; output current approaches its active bound smoothly. Total output current also includes the bleed.

There is no intrinsic bandwidth, pole, slew, input offset/bias mismatch, common-mode limit, protection diode, noise, thermal/rating or named-device behavior. External capacitors and inductors supply circuit timing. The generic model cannot establish a physical amplifier's speed, stability or capacitive-load suitability. Equations and independently generated references are in the [integration plan](../research/16-opamp-integration-plan.md).

Projects, recovery, Verify records and DC sweep sources carry the complete `opamp_model`. Missing or altered parameters refuse. Captured dependencies hash those parameters; imported final results require matching device IDs/models, a zero ground reference and readings consistent with the recorded terminal voltages. CSV exports include `opamp_model`. Unrelated circuits omit the new optional model fields, preserving existing model hashes. Older readers reject the unknown new model/part rather than silently dropping it.

Native and exported symbols share the same triangle/sign/power-lead paths and five named terminals. SVG metadata includes the exact model. Existing PDF/PNG/print paths consume schematic output; mounted visual, print and broad device-envelope acceptance remain separate qualification work.

There are at most two op-amps per network, within the existing total part/node/source/storage budgets. The numerical foundation retains the original equations and work limits, with a differential-input Newton proposal limit and a voltage-correction acceptance check for op-amp networks. This does not guarantee convergence for every possible feedback topology.


[Source validation record](validation/opamp-source-contracts-validation.json) retains the focused/core/native/stdio scope, full quality gate, schema-budget repair, corrected fixture attempts and remaining release requirements.

## Guided experiments

Choose **Examples → Op-amp follower** or **Inverting amplifier**, or discover `opamp-follower` / `opamp-inverting` through `examples.list` and load with `example.load`. Both use a 10 Hz bipolar sine, explicit ±5 V supplies and a 10 kΩ output load. They select U1 and plot input voltage, output voltage, delivered output current and differential input voltage. Unsaved replacement and reset use the ordinary protected commands.

Pause, Restart and step 25 ms for the positive input peak. The follower produces approximately +1 V. The inverter uses RI 10 kΩ / RF 20 kΩ: +0.5 V input gives approximately −1 V output. Changing RF to 10 kΩ halves that amplitude. Reducing the follower load to 10 Ω demonstrates the generic active current scale, producing approximately ±0.2 V rather than following ±1 V. The live and Verify paths use identical model equations; no intrinsic frequency response is modeled.

Warm overload testing exposed excessive differential-input damping. The proposal limiter now permits geometric outward growth in saturation while bounding zero-crossing overshoot; equations, work/accuracy limits and iteration bounds remain unchanged. Tests cover both current signs, zero recovery, a full sine capture, conservation, gain edits, source identity, save/reopen and reset. Actual native socket and headless stdio captures match all 101 samples in each lesson.

The inspector places live readings and two rows of plot shortcuts before assumptions. Full accessible action names and public commands are retained. [Native screenshots and findings](../research/ui-audit/opamp-guides/README.md) show the changes and unresolved mounted text-field/focus, accessibility, tooltip and small-label issues. This is an unreleased source checkpoint, not native-input or release acceptance.

## Mixed driver qualification

The optional conformance `--opamp-suite` also builds 10 kΩ /1 µF input timing with a buffered NPN LED driver and a two-buffer NMOS LED driver. Both use 0–5 V pulses and explicit ±5 V amplifier rails. The LED turns on by 50 ms and off by 100 ms, with signed current/power conservation and a retained project recording. Finite output resistance visibly matters: the loaded NPN buffer reaches approximately **4.95385 V** at 50 ms while the capacitor is **4.96602 V**. Treating it as an ideal follower would incorrectly label this expected headroom loss as an error.

These public tests found two actionable numerical issues. Powered circuits now use the existing junction-voltage LED evaluation to resolve the off NPN's approximately 20 fA leakage without roundoff-driven Newton correction oscillation. Op-amp capacitor networks also use the existing coupling-node error scales (5e−8 V absolute plus 1.25e−6 relative), across their analysis families. This tighter local control meets the unchanged analytic RC acceptance bound. Constitutive laws, iteration/work/voltage limits and all no-op-amp arithmetic remain. Local error control is not a universal global accuracy guarantee.

All four public transports pass the four-case suite and **931 checks** each. [Evidence and encoding](validation/opamp-adapter-conformance.json), [suite contract](agent-conformance.md#powered-op-amp-suite). The earlier [mounted value-editor recheck](../research/ui-audit/opamp-mixed/README.md) failed; subsequent [settlement and pinned native-mouse work](../research/ui-audit/native-mouse/README.md) provides scoped physical-edit evidence. Repeated backend performance and buffered RLC/PMOS storage checks are now separately qualified in the integration plan. Full device/capacity, visible-latency, native-input/accessibility and release acceptance remain open.

## Schematic output qualification

The guided layouts now space their supplies, input resistor and ground reference to keep captions and feedback wires visible. Horizontal waveform captions sit above their terminal leads; ground captions share rotation-aware baselines on the canvas and in SVG/PDF/PNG. These are presentation changes without changed electrical inputs. Existing `example.load`, movement/rotation, export and preview commands remain shared by users and agents.

Run `python3 scripts/opamp-output-demo.py --artifacts build/new-opamp-output --output build/new-opamp-output.json` against a fresh owned session. `--connect` accepts an explicitly owned blank native endpoint and closes that host afterward; [demo admission](demo-sessions.md) protects other work. Both examples, all source/ground rotations, grayscale and tiled pages are covered. [The retained receipt](validation/opamp-output-validation.json) records ten native/headless captures, thirteen pages, independent PDF rendering, exact PNG preview parity and unchanged numerical evidence. Arbitrary neighbor placement and physical printer output remain separate work.
