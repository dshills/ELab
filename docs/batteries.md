# Batteries: voltage sag and stored charge

`battery` is a two-terminal part (`positive`, `negative`), default 9 V full-charge
voltage, 100 mAh capacity, 1 Ω internal resistance and 100% startup charge.
`component.set` edits full-charge voltage (0.1–12 V). `battery.configure` sets
all three remaining saved startup parameters:

```
{"id":"B1","capacity_ah":0.0002,"internal_resistance_ohm":10,"initial_charge_fraction":0.75}
```

Capacity is 1 µAh–100 Ah; ESR is 0.01–1000 Ω; startup charge is 0–1.
Users and agents call this same guarded, undoable command, including in an
atomic startup `edit.batch`. The inspector uses mAh and percent and converts
to the command's Ah/fraction. Running edits restart; paused edits become stale.
Restart and independent Verify use saved startup charge. Save does not silently
replace startup intent with whatever charge happened to remain at that instant.

## Explicit generic model

A series resistance provides terminal sag, as in the simple source/impedance
model described in the [Analog Devices battery impedance experiment](https://wiki.analog.com/university/courses/alm1k/alm-eis-lab).
ELab additionally uses a deliberately simple linear charge reservoir:

- Q = capacity_Ah × 3600 coulombs; C_equivalent = Q / V_full.
- OCV = V_full × SOC; dSOC/dt = −I_delivered / Q.
- V_terminal = OCV − I_delivered × R_internal.
- Stored energy = Q × V_full × SOC² / 2; ESR heat = I_delivered² × R_internal.

This curve is an educational model choice, not a fitted AA, alkaline or Li-ion
chemistry. It omits plateau/knee chemistry, ageing, temperature, self-discharge,
protection and ratings. External forced charging is an ideal reversible reservoir,
not a charger/protection simulation. Live SOC outside 0–1 is explicitly reported;
no hidden clamp manufactures or deletes stored charge.

Private storage is integrated by the existing coupled Radau solver, with its
unchanged error and work guards. Failed/canceled steps publish no partial discharge.
Battery internal storage/resistance consume existing internal storage, resistor,
part and node slots. Private implementation parts never enter the authored graph,
component browser, resistor/capacitor result inventories or public net inventory.
DC analysis freezes charge at its saved startup value; battery voltage is not an
ideal-source sweep override. Use a schematic switch to disconnect a battery:
ideal-source High/Low controls are unavailable for a battery primary source.

`mixed.batteries` reports SOC, remaining Ah, OCV, terminal voltage, delivered
current, ESR heat and stored energy. `v:B1` / `i:B1` probe its actual ports;
positive current is delivered from its positive terminal. Project files,
kept captures, recovery inspection and schematic/verification exports retain
explicit parameters and `power_model` provenance. Captured charge, voltage,
current, energy and terminal readings are cross-validated on import.

## Examples

**Battery sag and LED fade** (`battery-led`) starts a deliberately small 20 µAh,
120 Ω battery. D1 dims as charge falls; closing S1 adds a load that immediately
sags terminal voltage while stored charge remains continuous. This small capacity
makes discharge visible in seconds, not the hours of a physical battery pack.

**Battery discharge and regulator dropout** (`battery-regulator`) supplies a
5 V regulator from a 9 V, 200 µAh, 10 Ω battery. Output initially regulates, then
falls as battery headroom disappears. Select B1 for startup settings and live
charge, voltage/current and energy; compare the battery/output scope traces.
