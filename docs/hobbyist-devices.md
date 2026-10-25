# PWM power, amplifier dynamics, serial registers and sound

These additions share ordinary placement/wiring, configuration, scope, project,
recovery and independent verification commands. The model envelope is
`elab.hobbyist-devices.v1` (`device_model`). Existing op-amps keep their original
model unless dynamics are explicitly enabled. Existing circuit capacity,
integration accuracy, 512-attempt and 100 ms per-output limits remain unchanged.

## Examples

| Examples menu / ID | Experiment |
| --- | --- |
| PWM buck converter / `buck` | Real 12 V switched power stage, freewheel diode, 50 mH coil, 470 µF capacitor and load switch. Select PWM to change frequency/duty; close SLOAD for a second 50 Ω load. |
| Op-amp slew-limited follower / `opamp-slew` | Compare a ±1 V pulse with the follower response; select U1 to change or disable dynamics. |
| Op-amp bandwidth limit / `opamp-bandwidth` | Compare a 1 kHz, 10 mV sine with a follower configured for 1 kHz gain-bandwidth; AC sweeps use the same compensation Jacobian. |
| Serial shift and parallel latch / `shift-register` | SDATA sets data. CLK shifts automatically. Close SLATCH to capture; release before capturing again. SCLEAR clears the shift chain. SOE disables the parallel outputs. Seven segments show bits 0–6; D7 shows bit 7. |
| 555 piezo tone / `555-tone` | The existing 555 drives a capacitive piezo through 220 Ω. Select BZ1 to render, explicitly play, stop or export a short audio preview. |
| Light-sensitive buzzer alarm / `light-alarm` | Lower LDR1 light to activate the comparator/buzzer. Change the reference with P1. Render a new preview after changing the circuit. |

Use `examples.list` and `example.load` with these IDs. Loading is silent.
Paused SPST switch changes take effect at the next accepted sample; Step once
to apply DATA, LATCH, clear, OE or load changes while paused.

## PWM controller and buck stage

`pwm-controller` has OUT, VCC and GND. Its normalized clock has configurable
10–1000 Hz frequency and 0.1–0.9 duty fraction. Rise and fall each occupy 5% of
the period; the high interval includes the rising phase. Its complementary
50 Ω output paths include 1 GΩ leakage, a 1 MΩ supply load and smooth supply
activation from 2–3 V. The internal clock participates in source-corner handling,
integration step guidance and visible sampling advice independently of primary
experiment High/Low mode.

The example uses the existing H-bridge's first output as a high-side power
switch, with explicit freewheel diode and winding resistance. It is **open-loop**:
there is no feedback error amplifier, compensation, current limit or rated
converter IC. Capacitor charge, winding current, switching ripple, startup and
load droop are solved electrically, rather than drawn from an averaged output
formula. The default settles near 4.69 V with roughly 67 mV ripple in the tested
steady interval; changing the timestep/load/parameters changes that measurement.

```json
{"command":"pwm.configure","params":{"id":"PWM","parameters":{"frequency_hz":500,"duty_fraction":0.6}}}
```

Configuration is a saved, undoable electrical startup edit. Running experiments
restart; paused readings remain stale until Restart/Resume. The existing SLOAD
switch command changes load while preserving accepted capacitor voltage/current.
`mixed.pwm_controllers` reports clock phase, output voltage, three into-device
terminal currents and power. `v:ID` measures OUT−GND; pin-current probes use
`i:ID.out`, `i:ID.vcc`, `i:ID.gnd`.

## Optional op-amp dynamics

`opamp.dynamics.set` retains the existing five-pin op-amp and its fixed v1 output
stage. When enabled, one private 1 nF compensation capacitor/node is counted
within ordinary resource limits. With A0=100000 and ωp=2π·GBW/A0, the unconstrained
compensation current is:

```
C · slew · tanh(ωp · (A0·(V+input−V−input) + supply_midpoint − Vpole) / slew)
```

Smooth rail gating prevents outward integrator windup near the rails. The charged
pole controls the existing finite-resistance, current-limited output stage.
The output target stays within the present rails even if stored compensation
voltage outlasts a falling supply. Compensation supply current, stored energy and dissipation participate in KCL
and power accounting. Small-signal AC includes its explicit capacitor/Jacobian;
AC does not predict large-signal slew distortion.

```json
{"command":"opamp.dynamics.set","params":{"id":"U1","enabled":true,"parameters":{"gain_bandwidth_hz":1000,"slew_v_per_s":1000}}}
```

GBW is 100 Hz–1 MHz and slew is 100–10000000 V/s (1000 V/s = 0.001 V/µs).
Disabling removes dynamics, restoring the original memoryless law. Both numeric
fields remain required and validated when disabling. Configuration edits restart
as above. `mixed.opamp_dynamics` publishes pole voltage relative to V−,
compensation current and stored energy. The fixed base-model envelope is not
silently changed for old projects. This single-pole educational augmentation
omits silicon fitting, multiple poles, offset, noise, thermal behavior and ratings.

## Eight-bit serial register

`shift-register` has DATA, CLK, LATCH, RESET_N, OE_N, Q0–Q7, SERIAL, VCC and GND.
It is a functional 74HC595-style device, **not a manufacturer model or package
pinout**. Sixteen private D memories reuse the existing hybrid engine's edge rules.
Internal stage data uses ideal logic rails with explicit 1 GΩ loading; only
public Q/SERIAL pins have the finite 50 Ω output paths. An inverter maps clear.
Internal parts/nodes count against existing bounds; one register adds 17 solver
parts and one private clear node. Two can cascade within
bounds in a suitably small circuit; larger drawings may legitimately refuse.

- Rising CLK shifts DATA into bit 0; bit 7 appears at SERIAL.
- Rising LATCH captures the pre-edge shift byte into parallel storage.
- RESET_N low clears only the shift chain, not parallel storage.
- OE_N low enables Q0–Q7. High disables the active paths without losing data.
- SERIAL has a separate output buffer unaffected by OE_N.
- Simultaneous common-clock cascades/latches sample pre-edge data.
- Startup and power loss clear both banks; supply above the generic 12 V bound
  refuses. These are ELab generic model bounds, not 74HC595 ratings.

Inputs use the existing 30/70% local-supply thresholds. Active output paths are
50 Ω. Disabled parallel outputs retain 1 GΩ paths to their selected rails, so a
floating pin can have a voltage even though its delivered current is negligible.
No propagation, setup/hold, metastability or output-protection claims are made.
`mixed.shift_registers` exposes both bytes, edge memories, nine output voltages,
all 16 pin currents and power. Ordinary pin-voltage/current probes, wiring,
switch/source, Save and Verify commands are the complete control interface.

## Piezo, active buzzer and explicit audio

Both sounders have POSITIVE and NEGATIVE terminals. `piezo` uses explicit saved
capacitance (1 nF–1 µF, default 20 nF) in parallel with 10 MΩ leakage. Insert
series resistance before an ideal source; redundant source/capacitor startup
constraints refuse rather than creating impulses. `active-buzzer` uses a passive
1 kΩ enabled DC load, 10 MΩ leakage and smooth activation from 2–3 V. Its acoustic
oscillator is illustrative; electrical oscillator ripple is omitted.

```json
{"command":"sound.configure","params":{"id":"BZ1","parameters":{"capacitance_f":2e-8,"frequency_hz":1000}}}
```

Both fields are saved and required. Piezo uses capacitance; active buzzer uses
frequency (100–4000 Hz). The other field is retained without effect for that kind.
`mixed.sounders` reports terminal voltage/current, explicit stored energy,
dissipation and active state. Ordinary voltage/current probes are available.

**No audio plays when placing, loading, simulating or rendering.**

1. `audio.render` captures an independent experiment, not the editor's live state:
   `{"id":"BZ1","start_s":0.02,"duration_s":0.25,"gain":0.1}`.
2. Observe `snapshot.audio_preview` or `audio.read {"id":"audio-1"}` until
   status is `completed`; `failed`/`canceled` includes the reason.
3. `audio.play {"id":"audio-1"}` explicitly schedules that exact current preview
   once. `audio.stop` with the same ID stops playback/cancels rendering.
4. `audio.read` returns owned WAV bytes as base64. Export with `export.start`:
   `{"kind":"audio-wav","id":"audio-1","path":"/absolute/path/tone.wav"}`.
   Inspect the returned project job for completion, byte count and digest.

These snippets show command/params; mutation envelopes still need the ordinary
version, request ID, retry key and revision guards. Detailed `capabilities.list`
for each command describes bounds and completion. UI buttons use these routes.

Preview start is 0–1 s; duration 20–250 ms; gain 0–0.25. Start/duration
round to the nearest 12 kHz sample; metadata reports the actual sample grid. Rendering is one bounded
asynchronous job (30 s cooperative deadline), 12 kHz mono/16-bit PCM, at most
3000 output samples plus 12000 pre-roll steps. Existing per-output simulation
budgets remain. The passive piezo preview is terminal voltage /5 with a 100 Hz
DC-blocking filter; the active buzzer uses its configured sinusoid gated by
terminal voltage. Both fade at their edges and limit level. This is **not** a
mechanical resonance, room, calibrated SPL or physical audibility prediction.

macOS playback uses AVAudioPlayer. Other builds can render/read/export WAV.
Playback completion reports scheduling, not speaker audibility. Stop cancels
owned playback; source edits/replacement and Close stop it too. Stale previews
remain readable, with source_current=false, but cannot play or export as current.
No audio data is persisted into projects. Offline rendering leaves live clock,
physics, history and document dirty state unchanged.

## References and model scope

The design references explain the underlying concepts; numerical values above
are explicitly ELab choices, not copied hardware specifications:

- [Analog Devices AN-140 — switching power-supply concepts](https://www.analog.com/en/resources/app-notes/an-140.html).
- [Analog Devices MT-033 — op-amp gain and bandwidth](https://www.analog.com/media/en/training-seminars/tutorials/MT-033.pdf).
- [TI SN74HC595 datasheet — separate shift/storage clocks, clear and OE](https://www.ti.com/lit/ds/symlink/sn74hc595.pdf).
- [Murata — externally driven and self-driven piezo circuits](https://www.murata.com/en-global/products/sound/library/basic/circuit).
