# SCRs and TRIACs

Place **SCR thyristor** or **TRIAC** under Semiconductors. SCR pins are anode,
cathode and gate; TRIAC pins are MT2, MT1 and gate. Gate return is cathode/MT1.
Use ordinary sources, buttons and wires to trigger them. Select the device to
inspect main/gate currents, power, conduction state and saved parameters.

Both user controls and agents use `thyristor.configure` with complete parameters:

```json
{"id":"TR1","parameters":{"version":"elab.generic-thyristor.v1","gate_trigger_current_a":0.0005,"latching_current_a":0.005,"holding_current_a":0.002,"on_resistance_ohm":1,"forward_drop_v":0.8,"gate_resistance_ohm":100}}
```

The normal transport actor/revision/retry envelope applies. All fields are required;
configuration is a saved, undoable startup edit that restarts from blocking state.
Trigger current is 10 µA…10 mA, latching current 0.1 mA…1 A, holding current
10 µA…50 mA **strictly below latching current**, on resistance 0.1…100 Ω, forward
drop 0…2 V and gate resistance 10…10000 Ω. Filter `capabilities.list` for numeric
schemas and the relation. Main voltage is bounded to ±24 V and gate voltage to
±12 V within the existing node/source/work envelope.

The main path blocks through 1 GΩ. With a sufficient gate signal and suitable
main polarity/voltage it enters **gate-supported** conduction. An SCR conducts
forward; a TRIAC conducts either direction. Above the configured forward drop,
current is (|Vmain|−Vdrop)/Ron with the appropriate sign, plus blocking bleed.
The gate uses a 0.6 V threshold, configured resistance and 1 GΩ bleed. SCR gate
current is unidirectional; the educational TRIAC gate is symmetric in all four
trigger quadrants, without real quadrant sensitivity differences.

**Latching checks actual solved load current**, after tentative conduction, rather
than dividing the prior blocking voltage by Ron. A load below Il can conduct while
the gate is driven but cannot stay latched after its removal. Once latched, the
same-direction main current can fall below Il and still hold until it drops below
Ih. Current reversal commutates the path; direction memory prevents a skipped
zero-current sample from retaining an old-polarity latch. Recovery time, breakover,
dv/dt firing, charge storage, heating and manufacturer models are omitted.

The existing hybrid solver brackets transitions, publishes left/right trace values
at each event and keeps proposed memory separate from accepted states. Gate-pulse
resolution still depends on the existing waveform and integration work/accuracy
limits; an impractical large request refuses rather than increase those limits.
DC/AC/noise sweeps refuse the stateful model; use transient Verify and recorded
measurements. Storage and suitable external commutation/flyback paths remain
necessary for reactive loads. No mains control or hardware/snubber qualification
is implied by these low-voltage examples.

**SCR latching LED alarm**: press or Click TRIGGER, release it, and see SCR1 remain
latched. Open RESET and advance a step to interrupt main current. The existing
switch command saves its desired setting immediately and applies it on the next
accepted output step; reopening the switch does not bypass that contract. Close
RESET and trigger again. Scope main/gate current and main voltage.

**Low-voltage TRIAC phase control**: an 8 V peak / 50 Hz source drives a 100 Ω load.
A rectified 100 Hz gate waveform triggers both half-cycles near their peaks.
Changing VG phase shifts firing time through the existing sine-source command.
Compare RL voltage/power and main current; the path releases before each current
zero because the holding-current threshold is nonzero. This uses a symmetric
four-quadrant generic TRIAC, including positive-gate negative-main triggering.

Capture/export/save/reopen retain the source model/config, separate gate/main pin
currents, latching and polarity state. Imports check model identity, valid memory,
local guards, current and power relationships; they do not reconstruct unrecorded
physical history. SVG/printing metadata retains the saved device settings.

[ST AN303: latching current](https://www.st.com/resource/en/application_note/an303-thyristors-and-triacs-latching-current-stmicroelectronics.pdf)
explains the load-current requirement after gate removal and illustrates how slow
inductive current rise can prevent latching. ELab models this distinction with
explicit illustrative parameters rather than a named ST part.
