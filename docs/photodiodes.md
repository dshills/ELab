# Photodiodes and transimpedance sensing

Place **Photodiode** from Semiconductors or search “light sensor” / “TIA”. The
symbol has incoming light arrows; current is positive anode→cathode. Light drives
photocurrent in the opposite direction. Select it to adjust **Incident light · µW**
with a slider or exact field and inspect terminal current, charging current and
junction energy. Responsivity, dark saturation current and capacitance are saved
configuration fields. These controls work across all themes.

User and agents use the same commands:

```json
{"command":"photodiode.light.set","params":{"id":"PD1","incident_light_uw":20}}
```

```json
{"command":"photodiode.configure","params":{"id":"PD1","parameters":{"version":"elab.generic-photodiode.v1","responsivity_a_per_w":0.5,"incident_light_uw":20,"dark_saturation_current_a":1e-9,"junction_capacitance_f":1e-10}}}
```

Use the chosen transport’s normal actor/revision/retry envelope. Filter
`capabilities.list` for full typed schemas and bounds. Responsivity is 0.01…1 A/W,
incident light 0…1000 µW, saturation dark current 1 fA…1 µA, capacitance 0…1 µF.
All configuration fields are required. Changes to model parameters restart the
experiment; light changes stage atomic `input_work` on a fresh transient, preserving
accepted time, run and capacitor/coil state. Inspect `requested_optical_light`,
then `simulation.wait`. Newer requests supersede pending input; failed/superseded
requests preserve the committed drawing, history and physics. Committed light is
saved/undoable and used by independent Verify. Batch edits configure startup.

The generic 27 °C model has a PN junction with n=1.5, configured saturation current,
fixed 1 GΩ shunt, constant junction capacitor, and Iphoto=responsivity·optical watts.
Iterminal=Is·expm1(V/(1.5 kT/q))+V/1GΩ−Iphoto+C·dV/dt. Above exponent 40, a continuous
linear continuation retains the junction slope and keeps trial evaluations finite.
No series resistance, breakdown, avalanche multiplication, spectral/lux conversion,
voltage-dependent capacitance, transit delay or manufacturer fit is claimed.
The configured dark current is the saturation-law parameter; reverse terminal
leakage also includes the explicit shunt.

Junction capacitance starts uncharged, counts as one capacitor and one primitive
within existing budgets, and remains continuous on a live light change. Insert
external resistance when powering it from an ideal source; an inconsistent direct
source/capacitor loop refuses. Stored energy is C·V²/2 and included once in electrical
energy/capture readings. Terminal power can be negative under photovoltaic operation
because incident light supplies external energy. This is not a calibrated optical
power-conversion or physical-rating model.

**Photodiode light-to-voltage sensor** reverse-biases the detector through 100k Ω.
At 10 µW / 0.5 A/W, current is about 5 µA and the resistor drop about 0.5 V.
Its 10 nF junction exposes the 1 ms electrical response. **Photodiode transimpedance
amplifier** uses split ±5 V rails, a 100k feedback resistor and 100 pF feedback
capacitor. A 20 µW input produces about 1 V; the detector stays near zero bias.
Both examples prepare an optical-power DC sweep. Its `parameter:"light"` endpoints
are plain µW numbers, and each point solves an independent operating point.

AC includes junction capacitance. Noise analysis includes optical/dark shot noise
2q·(Iphoto+|Ijunction|) and shunt Johnson noise, alongside the existing declared
resistor/amplifier sources. This remains an illustrative uncorrelated white model.
Projects, captures and schematic SVG/print metadata retain the detector parameters;
recorded terminal current, charge/energy and optical state are checked on import.

[Hamamatsu’s detector guide](https://hub.hamamatsu.com/us/en/technical-notes/detector-selection/the-wits-guide-to-selecting-a-photodetector.html)
describes the parallel photocurrent/junction/shunt/capacitor equivalent.
[TI’s photodiode amplifier reference](https://www.ti.com/tool/CIRCUIT060018)
illustrates a transimpedance sensing workflow; ELab’s generic example is not a
simulation of TI’s named op-amp or a qualification of its stated bandwidth.
