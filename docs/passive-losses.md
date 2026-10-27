# Capacitor and inductor losses

Ideal behavior remains the default. Select a capacitor or standalone inductor
and enable **Include electrical losses**. Capacitors offer series ESR and optional
leakage; inductors offer winding resistance. Try **Capacitor ESR and leakage**
and **Coil winding loss and damping** in Examples. Disable losses to restore the
original ideal model. Every setting is saved, undoable and shared with agents.

`passive.configure` requires `id`, `enabled` and complete `parameters`:

```json
{"id":"C1","enabled":true,"parameters":{"version":"elab.passive-loss.v1","esr_ohm":10,"leakage_resistance_ohm":1000000,"winding_resistance_ohm":0}}
```

For an inductor set ESR/leakage to zero and specify `winding_resistance_ohm`.
Unused fields must be zero and parameters remain required when disabling.
ESR/winding resistance: 0 off, or 0.001–100000 Ω. Leakage resistance: 0 off, or
1000–1e12 Ω. Inspect detailed discovery for the schemas and ordinary revision,
document and retry guards. No new components or storage slots are added.

The capacitor equivalent is series ESR with ideal storage, plus a leakage
resistor across the **external terminals**. This is an explicitly lumped model,
not a fitted dielectric-leakage network. With charging current Ic:

- Vterminal = Vstored + ESR × Ic.
- Iterminal = Ic + Vterminal / Rleak (second term zero when off).
- Stored energy = C × Vstored² / 2.
- ESR heat = ESR × Ic²; leakage heat = Vterminal² / Rleak.

Initial voltage applies to stored voltage. Terminal voltage can jump across ESR
while stored voltage stays continuous. Leakage lowers the steady charge in a
resistively driven supply. DC opens storage but retains the leakage shunt. AC
uses `Y = 1/Rleak + jωC/(1+jωC×ESR)` about the configured bias.

Inductor terminal voltage is `L×dI/dt + Rw×I`, with heat `Rw×I²` and unchanged
magnetic energy `L×I²/2`. Source/part power and storage accounting include each
loss once. Read `mixed.capacitors[].losses` / `mixed.inductors[].losses` for
storage voltage/current and each heat component. Ordinary scope probes report
external terminal voltage and **total** current. The inspector shows both views;
Verify captures and validates their model-bearing metadata. SVG/printed parts
retain configured parameters. Existing project format remains v4.

The internal ESR storage node obeys the existing ±30 V absolute node envelope.
Error, integration-work, source and part limits stay unchanged. ESR supplies a
finite startup path, so a direct ideal-source/capacitor connection is admitted
only when series ESR is nonzero. Leakage alone does not remove the ideal voltage
constraint. These are constant electrical loss parameters: no dielectric
absorption, skin effect, magnetic saturation, temperature evolution, physical
polarity, manufacturer fit or safe component-rating prediction is implied.
