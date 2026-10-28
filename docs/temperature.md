# Temperature and self-heating

Select a resistor, silicon diode, NPN or PNP transistor and enable **Temperature**
in Inspect. Ambient is an imposed component temperature unless **Include
self-heating** is checked. Settings are saved, undoable and share the exact
`thermal.configure` command with CLI, JSON, MCP and built-in agents. Other diode
families, LEDs, MOSFETs and JFETs retain their original models.

```json
{"command":"thermal.configure","params":{"id":"R1","enabled":true,"parameters":{"version":"elab.electrothermal.v1","ambient_c":27,"self_heating":true,"thermal_resistance_k_per_w":100,"thermal_capacitance_j_per_k":0.001,"resistance_coefficient_per_k":0}}}
```

Use the command envelope/actor/revision fields required by the selected transport.
`capabilities.list` with this command returns the complete typed parameter schema.
All six parameter fields are required, even when disabling the model. `enabled:
false` restores the original exact law. Generic junction constants refer to 27 °C;
optional transistor profile gain, saturation current, Early voltage and constant
junction capacitances can coexist with temperature settings.

For a resistor, R(T)=Rref·[1+α(T−27)]. The coefficient is per kelvin; the inspector
shows ppm/K. Ambient is −40…125 °C, α is ±0.005/K (zero on junction devices),
Rth is 0.1…2000 K/W and Cth is 0.001…10 J/K. Junction thermal voltage is kT/q.
The declared saturation scaling is (T/Tref)^(3/n)·exp[1.11 eV/(n k)·(1/Tref−1/T)],
with absolute temperatures, n=1 for BJT transport and n=1.7 for the silicon diode.
Fixed β/Early parameters remain illustrative; this is not a temperature-fitted
manufacturer SPICE model. The diode retains its original series resistance.

Self-heating solves Cth·dT/dt=Pconductive−(T−ambient)/Rth together with the
electrical network. Startup is at ambient. Accepted junction temperatures outside
−40…150 °C refuse, preserving the prior accepted state. A bounded continuation
helps nonlinear trial iterations; it does not clip an accepted over-temperature
solution. One thermal RC consumes one node, one capacitor, one resistor and two
primitives from existing budgets. Accuracy, work and capacity limits are unchanged.

Live/agent `mixed.thermal` readings report temperature, conductive heat, heat to
ambient, stored sensible heat, storage rate and thermal balance. Sensible heat is
Cth·(T−ambient), with ambient as the reference; cooling below ambient can make it
negative. This is a separate thermal domain, not electrical capacitor energy.
Without self-heating, an imposed bath removes conductive heat and no thermal
storage is reported. Captures retain and validate the thermal state and model.

The ordinary DC sweep target list includes **Ambient temperature** on enabled
parts. Endpoints are plain °C numbers, including negative values with linear
spacing. Each point solves its own steady operating point; it is not a heat ramp
or replay of the live trajectory. Self-heating sweeps solve thermal equilibrium.
AC analysis linearizes the coupled electrical/thermal operating point, including
its thermal storage; high-frequency behavior remains limited by declared models.

Open **Resistor self-heating**: 5 V across 100 Ω produces 250 mW. Rth=100 K/W and
Cth=0.001 J/K yield a 25 K rise with a 100 ms time constant. Set a positive α and
observe current reduction. **Class-AB temperature and bias drift** starts Q6/Q7
at equal imposed temperatures; changing only one exposes mismatch through the
original feedback circuit. It initially disables self-heating to isolate this
experiment. Neither example predicts a physical package rating, mutual thermal
coupling, thermal runaway protection or safe hardware operation.

The [Nexperia BJT handbook](https://assets.nexperia.com/documents/brochure/nexperia_BJT_Handbook_V2_240425_lowres.pdf)
describes thermal resistance/capacitance analogies. The declared junction scaling
follows the form documented by [MathWorks SPICE Diode](https://www.mathworks.com/help/simscape-electrical/ref/spicediode.html),
using explicit educational EG/XTI parameters rather than a named physical part.
