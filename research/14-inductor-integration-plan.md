# Ideal inductors: integration plan

October 16, 2026. Status: design; implementation pending. This advances the expanded-parts P1 item in [the PRD](07-product-requirements.md), with the same live, editable and agent-operable workflow. The inspected application currently has capacitors and generic NPN/N/P-channel MOSFETs; inductors are absent from the library, compiler, kernel and file validators. This plan is not evidence that inductor support has shipped and does not replace the outstanding P0 acceptance gates.

The intended result is a two-terminal ideal inductor that a hobbyist can place, wire, rotate, edit, give a signed initial current, probe, run live, Verify, save/reopen and export. Three experiments should establish its usefulness: RL current buildup/decay, RLC ringing, and inductive switching with a flyback diode. Native controls and external agents must use the same validated commands and produce the same accepted state.

## Model and workflow contract

- Use a versioned ideal model, tentatively `ideal-inductor.v1`, with inductance in H, voltage pin 1 minus pin 2, current positive into pin 1/out of pin 2, and stored energy `L I² / 2`. Explicitly describe omitted winding resistance, saturation, mutual coupling, parasitic capacitance, heating, ratings and physical part identity. An external resistor supplies winding/load resistance in the experiments.
- Start with zero current by default. A saved, undoable, per-component signed initial-current command should accept unit-bearing text and reset-to-default. Verify needs an explicit scenario override with documented precedence. Existing capacitor startup and global initial-voltage semantics must retain their meanings.
- Proposed defaults are 10 mH and four inductors per network. Numeric inductance/current domains must be qualified and published before palette exposure; they are engineering limits, not physical ratings. Retain current global part/node/source limits, ±30 V node envelope and cancellation/integration work budgets.
- Topology/value/initial-current edits follow existing restart/stale rules. Move/rotate/view operations preserve accepted current and charge. Source and switch actuation at the same accepted time preserve both capacitor voltage and inductor current; a failed, canceled or superseded solve publishes neither partial memory nor a new time.
- Show signed voltage/current, energy and provenance through ordinary selection, terminal/net readings, pins and scope probes. Current cues must include the new branch in the existing unique-wire-current calculation. Timing measurements and captured comparisons continue to operate on stored samples with their existing resolution limits.

## Kernel design to validate

The current [network kernel](../workbench/mixed.go) stores node voltages and source branch currents in `mixedState.x`, with separate capacitor voltage memory in `capV`. Its zero-time solve gives capacitors explicit voltage constraints; its time step uses backward-Euler conductances. [Input and edge processing](../workbench/mixed_transient.go) already recomputes algebraic states at the same time while preserving capacitor memory.

An inductor needs a persistent branch-current unknown. A candidate ordering is **nodes, voltage-source currents, inductor currents, zero-time capacitor currents**. Appending temporary capacitor currents last keeps the persistent inductor indices stable when a positive time step removes capacitor constraint rows. The branch contributes `+I` and `−I` to its two node equations.

At a positive time step, stamp:

```
V1 - V2 - (L / dt) (I - I_old) = 0
```

At startup or a same-time algebraic recomputation, constrain `I = I_old`; use the configured initial current only for a fresh run. Capture prior currents once from the accepted guess before Newton retries or alternate seeds. Do not obtain the “old” current from a changing Newton iterate. Keeping persistent current in `x` may let the existing private-state clone copy it safely without a second mutable memory array; that proposal still needs cancellation, retry and topology tests.

The zero-time current-constraint residual is in amperes, unlike the positive-time inductor row's voltage residual. Preserve the appropriate current/voltage tolerances rather than applying the existing voltage-row tolerance indiscriminately. Update every dimension, branch-index and seed builder consistently, including [grounded startup guesses](../workbench/mixed_startup.go). Current constraints do not provide a guessed node voltage or justify a fabricated ground/leakage path.

The present no-capacitor shortcut takes one algebraic solve, and the adaptive comparison checks capacitor voltages. Both need explicit inductor treatment: compare full/two-half-step currents, relevant node voltages and stored energy, using dimensional tolerances. Pure RL circuits must enter dynamic stepping even with no capacitor. Underdamped LC motion creates an additional hazard: large backward-Euler steps can erase oscillation while full/half results look deceptively similar. Establish and test an oscillation-aware initial/maximum step policy; a small difference between two over-damped numerical answers is insufficient evidence of accuracy. The timestep policy is an unresolved design/validation item, not a claimed bound.

Power accounting must include the inductor's stored-energy change and backward-Euler numerical damping. For an accepted step the latter is `L (I - I_old)² / (2 dt)`. At a same-time solve, instantaneous `V I` participates without a finite energy difference. Preserve the meaning of the existing conservation fields; report finite results and reject out-of-envelope states atomically.

## Operating point and topology

Live initialization with a configured current differs from steady DC. [DC sweeps](../workbench/dc_sweep.go) currently open capacitors in a private plan; their inductor plan must instead impose zero voltage and solve the branch current. A transient inductor directly across an ideal source can be valid while its DC operating point is inconsistent. Apply analysis-specific checks, with explicit diagnostics for indeterminate ideal loops/cutsets and incompatible initial currents. Avoid a blanket static restriction that silently substitutes one analysis for the other.

Review floating islands, parallel/series inductors, capacitor/inductor loops, source/inductor loops, all-zero input, nonzero startup current, singular zero-residual guesses, and switching with/without a recirculation path. An interrupting switch can demand a node voltage beyond the declared envelope. Keep the last accepted state and explain the failure; a flyback diode should produce a separately validated finite recirculation solution. Current limits and finite SPST off resistance remain explicit model properties.

## Integration map

| Area | Current sources to extend | Required result |
|---|---|---|
| Library, values and routing into the network | [graph.go](../workbench/graph.go), [library.go](../workbench/library.go), [model.go](../workbench/model.go), [value_domains.go](../workbench/value_domains.go), [network_compile.go](../workbench/network_compile.go), [limits.go](../workbench/limits.go) | Stable L IDs, named pins, defaults/aliases/assumptions, H/mH/µH text with preserved existing rounding, finite domains, dynamic analysis selection and discoverable bounds |
| Startup/live/Verify and actuation | [mixed.go](../workbench/mixed.go), [mixed_startup.go](../workbench/mixed_startup.go), [initialization.go](../workbench/initialization.go), [mixed_transient.go](../workbench/mixed_transient.go), [verification_types.go](../workbench/verification_types.go), [step_work.go](../workbench/step_work.go), [input_work.go](../workbench/input_work.go), [switch_actuation.go](../workbench/switch_actuation.go) | Correct current memory, immutable staging/cloning, same-time continuity, scenario startup precedence, cancellation and accepted publication |
| Measurements and captured data | [measurement_inspection.go](../workbench/measurement_inspection.go), [display_frame.go](../workbench/display_frame.go), [verification_export.go](../workbench/verification_export.go), [recording_import.go](../workbench/recording_import.go) | Correct signed branch/terminal/net probes, energy, wire injection, CSV units, complete model/result identity checks and retained-sample inspection |
| Persistence and history | [persistence.go](../workbench/persistence.go), [project_recordings.go](../workbench/project_recordings.go), [model.go](../workbench/model.go) | Strict optional initial-current field, deep pointer ownership, saved model parameters/startup intent, undo/recovery, unchanged prior-project and recording hashes when no inductor exists |
| Native and schematic drawing | [drawing.go](../desktop/drawing.go), [component_inspector.go](../desktop/component_inspector.go), [probes.go](../desktop/probes.go), [export.go](../workbench/export.go), [routing.go](../workbench/routing.go) | Consistent coil/pins/caption at every rotation; Fit and SVG/PDF/PNG/preview agree; native placement/value/startup/probe controls route through public commands |
| Discovery/conformance | [commands.go](../workbench/commands.go), [verification_capabilities.go](../workbench/verification_capabilities.go), [native inventory](13-native-command-inventory.md), [conformance runner](../internal/conformance) | Typed schemas, parameter/current-state inventories, errors and jobs; reviewed native bindings and actual four-client execution |

These are integration obligations; the list does not imply every change belongs in those existing files. Keep distinct device/state logic in focused helpers rather than growing the large kernel/view/command builders unnecessarily. Review model/engine/file version policy before changing serialized fields, and preserve old captures' dependency hashes.

## Evidence required before shipping

1. Independent RL step and signed-current decay references, including nonzero/reversed initial current; voltage/current/energy and conservation tolerances stated in SI units.
2. Independent underdamped, critically damped and overdamped RLC references. Include a high-Q/large-output-step case designed to catch artificial numerical loss of ringing, plus default/half/quarter-step refinement and explicit budget refusals.
3. A switched inductive load with a diode return path, using an independent ODE/reference calculation and both accepted event sides. Compare current continuity, stored energy, signed diode/source currents and clamp voltage. A missing-path case must preserve prior physics on failure.
4. Mixed RC/NPN/MOSFET fixtures, same-time source/switch/wiper work, canceled/superseded updates, cold/warm startup and DC operating-point distinctions. Keep existing frozen reference checks passing.
5. Native blank construction/value/startup/probe operation versus agents; physical default-window inspection; rotation/label/pin/route containment in native and exported output. Preserve all existing commands' behavior and identities.
6. Save/fresh-process reopen, edited example origin, recovery/undo, Verify capture/CSV, strict corrupt/foreign/missing-model/result rejection and actual prior-version project/capture reopen with original bytes unchanged.
7. Final lint/import/vet/race, packaged GPU/UI/protected-close smoke, four-client reference and new-device workflows, independent electrical suite and transport-inclusive timing evidence on recorded hardware. Visible-latency, screen-reader, clean distribution, participant and physical-print acceptance remain separately open unless actually performed.

The next implementation work should start with the branch-memory/dimension and strict-model foundations, then complete the integrated workflows and evidence above before making the part available to users. Uncertainty about numerical behavior should produce stronger tests and explicit diagnostic contracts.
