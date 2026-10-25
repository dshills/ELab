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

Observe `snapshot.dc_sweep` / `sweep.inspect`, or wait with `simulation.wait`.
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
parts initially include R/L/C, potentiometers, resistive sensors, static switches,
ideal sources, grounds/labels/junctions, and the generic powered op-amp. Other
parts explicitly refuse; hysteretic/digital state requires transient simulation.
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
