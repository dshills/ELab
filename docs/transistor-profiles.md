# Configurable transistor profiles

Select an NPN or PNP and enable **Use configurable educational profile** in the
inspector. Adjust forward/reverse gain, transport saturation current, Early
voltage, and base–emitter/base–collector capacitance. Disable the profile to
restore the exact generic-v1 law. Try the **Transistor gain, Early effect and
capacitance** example, then compare the emitter reading and a VIN→emitter Bode
sweep. All edits are saved, undoable, and use the same command for users/agents.

`transistor.profile.set` accepts `id`, `enabled`, and a complete `parameters`:

```json
{"id":"Q1","enabled":true,"parameters":{"version":"elab.bipolar-profile.v1","forward_beta":100,"reverse_beta":1,"saturation_current_a":1e-14,"early_voltage_v":100,"base_emitter_capacitance_f":2e-11,"base_collector_capacitance_f":5e-12}}
```

Detailed discovery exposes the schema, units, bounds and effects. Parameters
remain required and validated when disabling. Revision/generation/retry guards,
atomic edit batches, duplicate, undo, redo, recording and project serialization
use the ordinary shared paths. Captured per-transistor model parameters and
profile checksums identify each profile separately from the generic fallback.

The versioned law is reciprocal Ebers–Moll at fixed 27 °C plus an illustrative
forward Early term `IS*max(expm1(VBE/VT),0)*max(VCE,0)/VA` (PNP mirrors signs).
VA=0 disables that term; otherwise VA is 1–1000 V. CBE and CBC are **constant
linear capacitors**, not voltage-dependent depletion or diffusion charge. Each
nonzero capacitance consumes one of the existing four storage slots and one
simulated part; no solver budget or accuracy limit increases. Large discrete
amplifiers can use zero junction capacitance to stay within that storage budget.

Junction capacitors initialize at the solved bias instead of imposing redundant
ideal startup voltage constraints. This is a biased experiment, not a prediction
of zero-energy hardware power-on. Transient terminal currents include charging
current; readings expose junction energy, individual capacitor currents and
conductive heat separately from signed terminal power. AC includes jωC and the
same analytic conductive Jacobian. Original generic devices have no intrinsic
capacitance and retain their original numerical law.

These profiles deliberately have no manufacturer identity. A partial subset of
a vendor SPICE card would omit high-injection rolloff, series resistance,
voltage-dependent charge, diffusion storage and other fitted laws, so it is not
presented as a validated named physical transistor. Breakdown, temperature,
noise, ratings and physical build certification remain outside this model.
