# Small-signal noise analysis

Open **Simulate → Frequency, noise & DC sweeps**. Select a voltage source’s
**White noise / input reference** target, choose 1–3 voltage output probes and an
ascending positive frequency band (0.01 Hz…1 GHz, 2–201 samples). Log spacing is
usually useful. **Run noise analysis** produces output density in V/√Hz, component
RMS contributions and total/input-referred RMS. The live drawing/time are unchanged.
The source defines input transfer; its configured offset/low/DC value establishes
bias. It does not become a random source in the live simulation.

The same user and agent commands configure, run, inspect, cancel, select, measure,
save and export this analysis. Example command parameters:

```json
{"component_id":"V1","parameter":"noise","start":"1 Hz","stop":"100 kHz","points":201,"spacing":"logarithmic","probe_ids":["v:C1"]}
```

Send these with `sweep.settings.set`, then `sweep.start {}`; or use
`sweep.start {"settings": ...}`. Observe `snapshot.dc_sweep` / `sweep.inspect`.
Results carry model `elab.small-signal-white-noise.v1`, the source graph/models,
DC bias, source density/units, transferred contribution PSD at every sample,
input transfer, input/output integrals and assumptions. Keep curves in projects,
reopen or import/export `elab.noise-sweep` JSON. CSV contains output/input density,
input-valid flags and individual output contribution PSDs. Viewer cursors select
stored samples. Captured data is inspectable recorded evidence, not an external
certification that the source was solved.

Open **RC filter thermal noise** for a prepared filter experiment or **Amplifier
noise budget** to compare resistor and op-amp contributions. The plotted output
is total voltage density; the inspector lists per-source RMS budgets, while
JSON/CSV expose the full component spectra. Increase points or narrow the band
to check integration convergence before comparing a physical circuit.

Declared independent, one-sided white mechanisms:

- Resistors: Norton PSD 4kT/R; 300.15 K unless their temperature option is enabled.
- Silicon/Schottky/LED: effective terminal shot PSD 2q|I|. Internal series-resistor
  partition and diode correlations are omitted; this is an explicit approximation.
- NPN/PNP: separate base-emitter and collector-emitter shot sources, 2q|IB| and
  2q|IC| using conductive currents. Saturation/reverse behavior is approximate.
- JFET/MOS: channel PSD (8/3)kT|gm| at 300.15 K; gate/bleed noise omitted.
- Op-amp: illustrative input white voltage 10 nV/√Hz, plus independent 1 pA/√Hz
  currents at each input returned to V−. Voltage noise perturbs the model’s
  differential control, including its optional compensation pole. These constants
  do not describe a named physical op-amp.

Ideal sources, capacitors and inductors contribute no noise. Lossy storage,
transformers, loudspeakers, switches/sensors, current devices, Zener avalanche
and other unlisted devices refuse this analysis rather than receive an invented
noise floor. Flicker/excess noise, avalanche multiplication, source correlations,
quantum corrections and manufacturer fits are absent. No transient random noise,
audio hiss or THD+N measurement is claimed.

An adjoint complex solve transfers each source through the same electrical
Jacobian and storage admittances used for small-signal AC. Independent powers
add; density is the square root of total PSD. RMS is sqrt of the trapezoidal
integral of **linear PSD versus Hz**, including on log-spaced grids. It is a
sampled quadrature estimate, not an exact continuous-band integral. Input-referred
density divides by the selected source’s voltage-transfer magnitude; it is
unavailable below 1e−30 V/V. Input RMS is unavailable if any sample is undefined.
The reported adjoint residual is a linear-system diagnostic, not electrical KCL
or an AC power-conservation certificate. Existing operation, deadline, resource
and solver limits are unchanged.

[TI’s noise-analysis report](https://www.ti.com/lit/an/slva043b/slva043b.pdf)
describes independent noise powers, Johnson/shot sources and bandwidth integration.
[Analog Devices’ LTspice noise guide](https://www.analog.com/en/resources/technical-articles/ltspice-integrating-noise-over-a-bandwidth.html)
illustrates the distinction between spectral density and integrated band RMS.
