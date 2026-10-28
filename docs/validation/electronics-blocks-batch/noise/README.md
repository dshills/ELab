# Noise analysis evidence

Item 3 follows committed blocks and temperature support. Existing sweep services
provide bounded cancellation/source capture/save/import/export/cursors. New noise
parameter declares white model sources and tracks component PSD, input transfer,
RMS quadrature, actual bias and assumptions. No live random waveform or new solver
limits. Unsupported noise mechanisms/devices refuse explicitly. Two examples.

Independent references: resistor divider equivalent parallel resistance, RC
Johnson transfer PSD and analytic finite-band integral/kT/C limit, amplifier
noise gain, terminal shot relation and independent contribution power sum.
Deterministic reruns, zero-transfer undefined input reference, canceled work,
no live mutation, deep ownership, capture forgery refusal and saved project
round trip controls accompany the electrical checks. Ordinary/offscreen UI parity
and all-theme validation use the same public sweep commands.

Initial noise failures exposed an authored-graph fast path that intentionally
returns an empty lowering descriptor for ordinary parts; noise now uses the
physical MNA graph on that fast path. A first fixture used V1 rather than the
NPN template's VCC. Capture integral validation originally compared complete bias
clones, including nil/empty slice representation; it now compares the actual
summary quantities and validates bias through the existing recorded-device rules.
These initial outcomes are preserved and not counted as passing evidence.

Prism initial review is complete: 7/7 Anthropic Claude Opus 5.5 chunks, 21 files,
no skips/truncation, seven medium and nine low observations. Dispositions:

- eaa5287a106854e3: reproduced by ordinary all-theme UI validation; duplicate
  panel readouts removed, retained once in the inspector.
- 1987a62f7a0d1876 / c66a6f27731231ac / ad06ab49c7d0fc89: existing sweep import
  validates probe/point widths before the noise helper, and integrals already
  enforce summary widths by equality. A shared explicit shape guard now protects
  standalone renderer/export callers too; malformed-shape refusal tests added.
- 1e336b1711d40605: PSD is explicitly contributor-major [source][output] in the
  producer, validator and exporter, matching header order. Shared shape guard
  enforces both dimensions; multi-output export regression verifies columns.
- 4205f839f9e923ac: captured sweep import deliberately validates declared models,
  shape, bias/device state and numerical relationships without rerunning source
  experiments. Existing DC/AC imports have the same recorded-evidence contract.
  No claim of recomputed physical certification; the guide states this explicitly.
  Re-solving up to 201 points inside the file validator would change that contract
  and introduce hidden unbounded work. Source density, sums, gain/referral and
  integrals are checked; independent tests qualify the actual producer.
- e573be9fc3b6d9bf: shot test now requires the contributor's presence.
- c29a9143c0324855 / 7fce1f3ecf1b48f5: explicit agent deadline diagnostic and UI
  callback errors returned to the test goroutine.
- cfb9c2bf976c8b08 / f17eb6e649f3be41: command/target guidance now mentions noise.
- 190973ba774a300d: bound tests reset to ascending valid bands and independently
  test lower Start/upper Stop, including loop/noise targets.
- 164e5fbbe5df7dc4: explicit regression enforces IsFrequencySweep(noise)==false
  while IsSpectralSweep(noise)==true. AC gain/phase and noise remain distinct.
- 9ae4319d0bc48eab: graph validation enforces positive resistor values and accepted
  temperature range. At worst alpha=−0.005 and T=150, multiplier is 0.385>0;
  the other corner is 0.665. No valid thermal resistance reaches zero/negative.
- 1fdf02249caa46c0 / dff3dc88cd3762e4: JSON encoding errors checked; both bias and
  contribution deep ownership asserted.

Follow-up Prism completes 2/2 chunks, 11 repair/inventory files, no omissions.
fc2142842ace29ac: actual callers already guard Noise!=nil, but standalone readout
helper now returns nil before its shape guard for nil/non-noise input.
da3da60e1ac09e9b: CSV test resolves the contributor index by ID instead of assuming
index zero. Final direct checks cover these bounded helper/test corrections.

Core noise/AC/DC/sweep race passed 11.699 s; repair electrical/discovery controls
2.845 s, final noise/catalog/capability race 5.326 s. Ordinary/offscreen UI checks
pass 1.173/2.403 s after the duplicated-ID repair; lint passes. An inventory refresh
initially attempted create-only --output against an existing path and refused
without replacing it; final fresh-path generation is explicitly copied and
verified. Source catalog remains 217 commands, 58 physical kinds, now 83 examples.
No unchanged full root gate repeated here; final batch gate follows later items.
