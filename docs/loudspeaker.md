# Loudspeaker electrical load and explicit preview

Open **Amplifier driving a loudspeaker** and select SP1. The powered follower
uses a 400 Hz, 50 mV peak input, small enough for the generic op-amp's declared
current limit. Inspect speaker terminal voltage/current, signed absorbed power,
coil heat, motional loss and stored energy. Change resonance or coil parameters,
then compare the live scope and Bode response. No audio plays when a project
opens, edits, runs, records or sweeps.

The inspector and agents share `speaker.configure`:

```json
{"id":"SP1","parameters":{"version":"elab.loudspeaker-impedance.v1","coil_resistance_ohm":6,"coil_inductance_h":0.0005,"resonance_hz":100,"motional_resistance_ohm":30,"motional_inductance_h":0.02}}
```

Placement, wiring, duplicate, delete, batch edits, undo/redo and saving use the
ordinary component commands. Detailed discovery includes every field, version,
bound and unit. Defaults are explicit and saved. The speaker uses two private
inductors, one capacitor, two resistors and two nodes within the existing storage,
part, resistor and node budgets. Initial energy is zero. Simulation accuracy,
work limits and current conventions are unchanged.

The passive electrical equivalent is a series coil R/L followed by parallel
motional R/L/C. The derived capacitance is
`Cm = 1 / ((2*pi*resonance_hz)^2 * motional_inductance_h)` and impedance is
`Rc + j*w*Lc + 1/(1/Rm + 1/(j*w*Lm) + j*w*Cm)`. This captures a resonant impedance
rise and high-frequency coil inductance. It does not convert these equivalent
parameters to cone displacement, calibrated sound pressure, enclosure/radiation
response, heating, excursion, frequency-dependent losses or a manufacturer part.
The general lumped electrical/mechanical modeling approach is described in
[COMSOL's loudspeaker reference](https://doc.comsol.com/6.4/doc/com.comsol.help.models.aco.lumped_loudspeaker_driver/lumped_loudspeaker_driver.html)
and [Klippel's impedance overview](https://klippel.de/know-how/measurements/transducer-parameters/electrical-impedance.html).
ELab's deliberately smaller constant-parameter equivalent is independently tested;
it does not reproduce those products' acoustic models.

For sound, explicitly **Render preview**, then **Play**, **Stop** or **Export WAV**.
Agents use the existing `audio.render`, `audio.read`, `audio.play`, `audio.stop`
and `export.start` with `kind: "audio-wav"` and the exact preview ID. Preview
runs independently, never advancing or changing the live experiment. Speaker
preview uses terminal voltage, a 100 Hz high-pass, peak normalization above a 1 µV signal floor, selected
gain and short fades; metadata identifies this signal. The floor suppresses numerical residue in the preview only; it does not change the electrical simulation. It illustrates the
waveform rather than predicting loudness or speaker acoustics. The existing
12 kHz mono, 20–250 ms duration, gain ≤25%, cancellation and source-identity guards
remain. Piezo/buzzer preview behavior is unchanged.

Recordings include model-bearing parameters, internal dynamic state and verified
terminal/energy readings. Captured source hashes and project round trips retain
them. SVG metadata and printed parts descriptions include profile/speaker intent.

The powered-follower example and independent passive reference circuits are
qualified for transient and AC operation. Combining the original 1 nF,
eight-BJT class-AB startup with the speaker's much slower LC states currently
can exceed the unchanged 100 ms integration budget. Startup is admitted, but
that combination is not qualified for progressing waveforms. The regression
checks bounded refusal and preservation of the accepted state; it is not counted
as a successful all-discrete speaker simulation. A 10 nF control also reached
the guard. No solver budget, precision, or preserved amplifier example was
changed to conceal this limitation.
