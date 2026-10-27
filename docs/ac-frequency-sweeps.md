# AC frequency sweeps and Bode plots

Open **Sweeps** and choose an ideal source's **AC frequency /1 V excitation**
target. Choose a frequency range (0.01 Hz–1 GHz, 2–201 inclusive points), usually
logarithmic spacing, and 1–3 differential voltage outputs. Run AC sweep, then
switch between gain in dB and unwrapped phase in degrees. Click/drag an exact
sampled point for both readings. The half-power estimate and sampled peak are
shown above the plot. A frequency sweep is a captured analysis; it never moves
the live clock or changes capacitor voltage, winding current or edit history.

The existing shared commands handle the entire workflow:

```
sweep.start {"settings":{"component_id":"V1","parameter":"frequency","spacing":"logarithmic","start":"1 Hz","stop":"100 kHz","points":201,"probe_ids":["v:C1"]}}
```

Observe `snapshot.dc_sweep` / `sweep.inspect` until its operation is terminal,
or watch `events.poll`. `simulation.wait` covers live/manual/project work and
does not wait for independent sweep operations.
`sweep.trace.select` accepts captured `gain:v:C1` / `phase:v:C1` IDs.
`sweep.cursor.set` selects an exact point or clears with explicit null.
`sweep.cancel` cancels active work; partial results are not published. Existing
30-second cooperative sweep deadline, eight retained captures, network limits
and transport bounds remain unchanged. `sweep.settings.set` sets the next draft.
The same native controls dispatch these commands. Keep/exclude captures, export
JSON/CSV, import JSON and project save/reopen use the existing sweep workflow.
The legacy `dc_sweep` snapshot key remains for compatibility; the capture format
is `elab.ac-sweep` version 1, engine `elab.small-signal-mna.v1`.

A DC operating point establishes bias (sine offset, pulse low, configured DC).
The selected ideal voltage source has normalized 1 V AC excitation; all other
ideal voltage sources are AC grounded. AC amplitude is independent of configured
waveform amplitude. The solver uses the analytic real device Jacobian plus jωC
and −jωL branch terms, with a complex linear solve at each frequency. Supported
parts include R/L/C, potentiometers, resistive sensors, static switches,
ideal sources, grounds/labels/junctions, transformers, the generic powered
op-amp, NPN/PNP BJTs, N/P-channel MOSFETs, LEDs and silicon/Schottky/Zener
diodes. Other parts explicitly refuse; hysteretic/digital state requires
transient simulation. Semiconductor tangent conductances use the solved DC
bias, not the live transient vector or the capacitor's initial charge. At the
exact piecewise Zener knee the blocking-side derivative is used.
The op-amp model has no intrinsic bandwidth/slew rate: external feedback RC sets
filter response, and large-signal clipping needs transient simulation.

Gain is differential Vout / selected source's differential voltage. Exact
complex values are retained. Phase unwrap follows requested point order.
Transfer magnitudes below 1e−30 V/V plot at −600 dB; exact zero has undefined
phase (`phase_defined=false`, displayed as zero) and is identified at the cursor.
Half-power means sampled peak minus 10 log10(2), interpolated in log frequency
between adjacent dB samples. This is an estimate, not an exact pole calculation.
The reference changes if the sweep fails to include the passband/peak: expand
the range and increase point density. Peaks are explicitly **sampled** peaks.

Examples **RC filter Bode plot**, **RLC resonance Bode plot**, and **Op-amp
low-pass Bode plot** load prepared ranges. RC tests compare every complex point
with 1/(1+jωRC) and the ~159.15 Hz cutoff. RLC tests compare the complete transfer
function and both half-power crossings; op-amp tests compare an inverting RC
feedback low-pass. Tests also cover source capture, conservation residuals,
strict complex/gain/phase/band consistency, cancellation, no live-state mutation,
saved reopen, native/agent parity, and existing DC sweep behavior.

The [Analog Devices Bode reference](https://www.analog.com/en/resources/technical-articles/2021/03/29/17/10/how-to-generate-a-bode-plot-with-ltspice.html)
provides the frequency/gain/phase context. ELab's implementation and supported
model domain are described above; it does not invoke LTspice.

## Discrete amplifier frequency response

Load `example.load {"id":"discrete-class-ab"}` or choose the all-discrete
class-AB example in the browser. Its default sweep selects VIN, 1 Hz–100 kHz,
201 logarithmic points and load voltage `v:RL`. Run AC sweep to view gain and
phase without changing the running amplifier. The same `sweep.settings.set`,
`sweep.start`, cursors, exports and project commands serve users and agents.

The low-frequency gain is about 10.9 V/V. C1's external compensation shapes the
frequency response; v1 generic transistors and diodes still have no internal
junction capacitance, charge storage or physical bandwidth. Small-signal results
cannot establish clipping, crossover THD or physical stability. Sweep metadata
records these assumptions. Existing RC/op-amp capture formats remain readable.

Independent tests compare BJT follower and MOSFET common-source complex transfer
functions with emitter/drain capacitors; diode tangent resistances include
forward silicon/Schottky/LED behavior and reverse Zener breakdown. The amplifier's
low-frequency response is checked against central DC perturbations. Captured
results reopen exactly, negative-input DC bias works with authored storage
opened, and native controls match agent commands while preserving live state.
