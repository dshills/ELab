# Finite pulse full-phase review · /179

Eight owned 10 kΩ/1 µF RC circuits exercise finite pulses with no high/low hold,
negative/constant levels, minimum-length phases and decimal cycle boundaries.
The exact copied **0.162.0 /179** headless agent runs the same public commands
used by the UI. **No additional defect is reproduced** in these cases.

| Configuration | Captured rows | Maximum capacitor error |
|---|---:|---:|
| Rise occupies the full 7 ms high phase | 71 | 27.026 µV |
| Fall occupies the full 13 ms low phase | 71 | 60.687 µV |
| Both phases are ramps: 7/13 ms triangle | 68 | 45.213 µV |
| −3/−1 V triangle, 20/30 ms phases | 158 | 23.411 µV |
| 1 µs high/rise, 999 µs low/fall | 609 | 0.0341 µV |
| 999 µs high/rise, 1 µs low/fall | 608 | 0.0308 µV |
| Constant 2.5 V, configured full-phase ramps | 62 | 20.670 µV |
| 6/4 ms triangle, partial final output step | 38 | 21.069 µV |

The precollection declaration retains the existing **3 mV** RC bound, 1 nV
source bound and corresponding 0.3 µA branch-current bound. The worst measured
errors are **60.687 µV /0.421 pV /6.069 nA** respectively. These are eight simple
fixtures, not universal device/topology or performance guarantees.

The independent reference constructs four piecewise-linear forcing segments
with Decimal boundary arithmetic. Zero-length holds are omitted. For a segment
starting at source `u`, slope `m`, duration `h` and capacitor voltage `v`, it uses
the exact first-order convolution, evaluated with `expm1`:

`v_next = v + (u−v)(1−exp(−h/τ)) + m(h−τ(1−exp(−h/τ)))`, with `τ=0.01 s`.

Every one of **1,685 captured rows** is checked against that reference for source
voltage, capacitor voltage and both signed branch currents. Live scope and Verify
match **5,968 retained channel values** at identical stored timestamps; full
final mixed results also match with only independent run/electrical identities
excluded. All **49 unique configured corner events** are present, including
coincident rise/high and fall/period boundaries. Constant levels produce no
invented corner events.

The two minimum-phase cases deliberately use **5 µs output steps** so three
cycles fit the existing 1,000-output-step Verify limit. Their 1 µs phases remain
inside the existing model bounds; no solver/work/sample limit changes. Their
four-samples-per-ramp target is 0.25 µs, below the supported 1 µs output minimum.
The app keeps the unresolved/minimum warning, rather than claiming the output
grid resolves that target. Full-rise/full-fall cases also keep their ordinary
sampling warnings. Internal integration, corner samples and output grid are
distinct. The live history correctly retains only 512 rows in the fast cases;
the captured records keep all 609/608 rows.

All eight projects keep their original capture. A fresh process reopens each
project paused at time zero, preserves waveform/graph/settings and exports the
kept recording **byte-identically**. Original manual project bytes remain intact.
All **16 owned processes** end with completed, joined, recovery-finalized,
exit-ready public shutdowns and exit 0. No failed collection or cleanup occurs.

All **654 application Go inputs**, the sealed /179 bundle, SDK and canonical
/120 remain unchanged. The agent runs under the already-qualified copied-runtime
policy denying development files, outbound IP and shared window settings. No
native window/input, display recording, privacy grant, printer job, owner
project, rebuild or release/version change occurs. Full Go suites are not
repeated for data-only validation of unchanged source.

`raw/` retains complete observations, captures, projects and reopened exports as
lossless gzip copies. `manifest.json` checks stored/original sizes and hashes;
`producers/` retains the collector and independent reference as text. These
producers preserve their historical private output paths; they are not an
unattended rerun recipe. Replay without starting an app or reading owner settings:

```sh
python3 -B docs/validation/finite-phase-review-179/verify.py
```

The initial read-only replay lacked the scripts import path for its shared
comparator. Adding that path fixes replay without recollecting or changing any
observation; the setup note is retained.

The read-only replay checks all equations, live/Verify parity, corner events,
sampling warnings, byte-identical kept captures, lifecycle and source/artifact
pins. Optimized Python refuses. Owner walkthrough stays **Not tried yet**; native,
reader/IME, full agent contracts, environment, performance and participant gates
retain their existing scopes. P0 counts remain 17 implemented /10 partial.
