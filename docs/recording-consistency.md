# Recorded device and storage consistency

Unreleased source change. Imports now reject mixed-network recordings whose final device state contradicts their captured circuit, nodes or final selected probe values. The same decoder handles standalone recorded JSON and recordings embedded in project files, through the ordinary user/agent import and project-load commands.

Before this repair, a recording could change MOSFET gate/drain/source current, gate-source/drain-source voltage, power or operating region while leaving valid-looking selected probes unchanged. A missing ground or terminal node, or an extra foreign node, could also pass. Separate RLC tests found that final capacitor voltage/energy or an inductor current with correspondingly altered energy could contradict the saved nodes or last plotted sample. The retained failing-before tests exercise actual public imports.

The decoder now requires the complete bounded graph-node inventory and zero ground reference. It recomputes MOSFET model, region, signed currents, voltage and power from those captured nodes, sharing the unchanged device-result arithmetic with live simulation. Gate current remains exactly zero and source/drain current conventions agree. Op-amp validation uses the same node-inventory check. Capacitor voltage/energy must match its nodes/value; existing inductor voltage/energy checks remain. Every selected mixed-network probe must match the last saved row, including the right side of a pulse edge.

These checks validate internal consistency without rerunning the transient simulation. They are not a replay, digital signature or proof that arbitrary sample history matches hardware. Existing model parameters, solver accuracy/work limits, capacities and file versions remain unchanged. Failed jobs retain observable errors while preserving the existing project, live physics, history and recordings.

Regression coverage includes both MOSFET polarities without MOSFET probes, standalone and embedded imports, asynchronous failed-import publication, legitimate cutoff/triode/saturation/reverse-body-diode records, and a complementary pair. Valid recorded JSON re-exports byte for byte. The op-amp storage suite adds buffered RLC startup/refinement and an RC-driven high-side PMOS load through the public adapters.

```sh
GOPROXY=off ./scripts/build.sh
./build/elab-conformance --opamp-storage-suite --opamp-suite \
  --inductor-suite --electrical-suite --artifacts build/storage-conformance-new \
  --native-binary build/elab
```

Use the development binary: the qualified 0.105 application predates op-amps and this repair. Full physical input/accessibility, maximum capacity, distribution and application promotion remain separate qualification work.

The [qualification receipt](validation/opamp-storage-validation.json) retains failing-before regressions, four-adapter results, source/binary identities and reproducible input files.
