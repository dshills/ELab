# Optocouplers

The **LED / phototransistor optocoupler** has anode/cathode input pins and
collector/emitter output pins. Its shared `optocoupler.configure` command sets
`{"id":"OC1","current_transfer_ratio":0.5}` (0.01–5 A/A). Native CTR entry and
slider show percent, so 50% is 0.5 A/A. This is a saved, undoable startup edit;
running circuits restart, and paused readings wait for Restart/Resume.

The input uses ELab's generic **red LED** law, not a calibrated infrared input
or a named optocoupler. Forward output current is

`CTR × max(LED current, 0) × Vce / hypot(Vce, 0.1 V) + Vce / 1 GΩ`.

For negative Vce only the 1 GΩ dark path remains. The smooth headroom factor
reduces effective CTR near saturation; the inspector labels Vce below 0.3 V as
limited output headroom when illuminated. Effective CTR excludes dark leakage.
CTR is approximately realized with adequate forward Vce and enough load current.
The law is instantaneous: no propagation delay, charge storage, temperature,
ageing, capacitance, breakdown or physical isolation rating is modeled.

Each port balances its own terminal currents. Optical coupling adds control
sensitivity, and creates **no conductive path** across the barrier. Each circuit
island still needs a voltage reference. Ground symbols in the example share a
mathematical zero; they are not an isolation-voltage test. An unreferenced output
is refused rather than grounded through optical coupling. Output supply/load
changes cannot back-feed input LED current through this model.

**Optically isolated LED switch** uses separate input and output supplies.
Close S1 to illuminate D1, select OC1 to inspect both port currents, available
photocurrent, effective CTR, Vce and separate powers; reduce CTR and Restart.
The four plotting actions expose LED voltage/current and output Vce/current.
All pin voltages and signed currents into each pin are available in probe
inventory for agents. Save/reopen, undo/redo, duplicates, batch edits, captures,
SVG/PNG/PDF exports and generic wiring retain authored OC1 identity.

Electrical tests cover the analytic Jacobian, CTR/load lines and saturation,
independent port power, pulse switching, output-to-input isolation, floating
output refusal, inductor startup, capture integrity and project roundtrip.
Native controls are compared against the same public agent command. One opto
reserves a diode and transistor work slot within the existing unchanged budgets.

The [Renesas optocoupler specification reference](https://www.renesas.com/en/products/interface/optoelectronics/explanation-photocoupler-optocoupler-specifications)
explains CTR and saturation context; ELab's illustrative law above is its own
educational model.
