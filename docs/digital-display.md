# Seven-segment display and BCD decoder

The Logic category has **BCD decoder** (`bcd-decoder`) and **Seven-segment
display** (`seven-segment`). The display is seven independent red LED branches
sharing a **common cathode**. Connect `common` to the return reference. Each
segment `a` through `g` needs its own external current-limiting resistor.

The powered decoder has inputs `b0`–`b3` (least significant first), outputs
`a`–`g`, `vcc` and `gnd`. Connect every input and both power pins. Outputs may
be unloaded. Inputs 0–9 produce the conventional digit; 10–15 blank all
outputs. This basic combinational device has no latch, lamp test or decimal
point. The [TI CD4511B reference](https://www.ti.com/product/CD4511B) provides
background on BCD/common-cathode driving; ELab does not claim that IC's pinout,
latch/control inputs or physical output ratings.

**555-driven digit counter** (`display-counter`) joins the existing real 555
astable timing circuit to a decade counter, decoder, seven 1 kΩ resistors and
display. It starts at zero and advances on subsequent rising clock edges,
wrapping after nine. Changing RB or C1 changes the counting period. Inspect
C1 voltage, timer output, counter bits or any segment current on the scope.

The fixed `elab.generic-seven-segment.v1` model embeds the existing red LED and
powered-logic models. Each LED uses the same series-resistance diode law as an
individual red LED. Each decoder output uses the declared load-dependent
50 Ω logic paths, 1 GΩ bleeds, 1 MΩ supply load and smooth supply enable.
Intermediate BCD input levels interpolate the complete truth polynomial;
these analog outputs are not a valid digit. Zero/reversed decoder supply
turns off active drivers. There is no hardware current limit, delay, heating
or calibrated light output. The display is one seven-branch package within
the existing combined part/node/work limits, not seven separately placed LEDs.

Visible segments follow **accepted segment current**, with a fixed 0.1 mA
illustrative visibility threshold. An unwired or unpowered segment cannot
light from a decoder's numeric input alone. Unknown/partial patterns are
reported separately from digits. Stale electrical readings do not show fresh
segment illumination. Schematic exports show the wiring symbol and all seven
segment outlines rather than claiming a saved live illumination state.

## Users and agents

All ordinary drawing, simulation, probe, capture, save/reopen and export
commands work unchanged. Use `parts.list`, `examples.list`, and
`example.load {"id":"display-counter"}`. Native menus and plot controls use
the same commands. Both parts have no numeric value field.

Snapshots, document/recovery inspection and verification metadata include
`display_model`; mixed results include `displays`. Each device publishes kind,
model, digit (`-1` for non-digit), state, seven segment voltages/currents, seven
`lit` flags, power and 13 pin currents. Raw current order is
`b0,b1,b2,b3,a,b,c,d,e,f,g,vcc,gnd/common`; absent pins are zero. Raw currents
enter the device. Decoder segment current probes report delivered current;
display segment probes enter the LEDs. Decoder `lit` flags mean sufficient
current is being delivered, not a guarantee about an attached display.
Named voltage probes use the circuit's common reference; segment-result
voltages are relative to each device's local GND/common cathode.

Saved captures require complete fixed arrays and model/identity/segment
consistency against their captured terminal voltages. Projects retain model
provenance and kept recordings; reopening restarts the live clock and counter
from a fresh startup. Existing document versions remain readable.
