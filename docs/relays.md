# SPDT relay

`relay` has separate `coil1`, `coil2`, `common`, `no` and `nc` terminals.
Its value is coil inductance (default 100 mH), editable with the ordinary
value field or `component.set`. Winding resistance is fixed at 100 Ω.
The coil starts at zero current, energizes the contacts at |I| ≥30 mA,
releases at |I| ≤10 mA, and retains its state between those limits.
The existing inductor integration and event localization preserve coil
current through switching. Relays share the existing four-inductor limit.

NO/NC paths have 1 Ω closed resistance and 1 GΩ open resistance. Contacts
are electrically isolated from the coil; each circuit needs its own external
reference. Unused NO/NC terminals are permitted. This generic educational
model has no mechanical delay, contact bounce, arcing, wear, saturation,
thermal protection or physical voltage/current rating. The fixed
`control_model` parameters are saved in projects and captures.

**Pushbutton transistor relay** (`relay-switch`) drives a 5 V coil through
an NPN transistor, using a momentary button and base resistor. The contacts
switch a separate 9 V LED load. **Relay coil and flyback protection**
(`relay-flasher`) uses a finite-edge pulse source to show repeated operation.
Both include a diode across the coil: its cathode connects to coil positive.
Watch coil current, transistor collector voltage, flyback diode current and
the contact LED. The [Omron relay reference](https://www.ia.omron.com/support/guide/17/further_information.html)
describes relay operation; the examples are not fitted models of an Omron
part. A diode provides a recirculation path when the driver opens. Release
occurs when the simulated decaying coil current crosses dropout, rather
than immediately on a source edge.

The Inspector reports contact state, coil current/voltage, stored energy
and dissipated power. `v:K1` measures coil1−coil2; `i:K1` measures current
entering coil1. Named terminal voltages and currents expose both isolated
banks. `mixed.controls` reports coil energy ½LI² and winding/contact loss;
raw pin currents enter the device in coil1, coil2, common, NO, NC order.

All ordinary placement, wiring, value editing, simulation, plotting,
verification, save/load and export commands work for both UI and agents.
The button example also uses `button.click` and `button.set`. Captures
validate model, contact guards, energy and terminal current consistency.
Restart/reopen establish zero coil current and released contacts. DC sweeps
are refused because pickup/dropout require history.

Electrical tests compare the coil trajectory against the independent RL
exponential, check pickup timing, hysteresis, contact isolation, KCL, power
and stored energy, and exercise both protected driver examples through
pickup, release and flyback, including capture and project round trips.
