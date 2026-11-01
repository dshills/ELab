# Web parity milestones — 2026-11-01

This record tracks implementation and qualification on `exp-web`. Prism receipts
record the configured provider/model, coverage, token allowance and findings.
The local Prism build allows 32,768 output tokens, a 65,536-token final retry,
and 15 minutes per Wafer request for reasoning responses.

## 1. Spectrum

Dedicated live/recorded source selection, capture/channel selection, native FFT
window/grid/fundamental/harmonics settings, live hold/off controls, ranked peaks,
exact frequency and plot cursors, cursor deltas, FFT/THD/assumptions and recorded
exports. Source identity prevents a previous capture or live result from
impersonating the selected data. All measurements use native results.

Validation: frontend syntax and 17 tests; race-enabled web FFT regression covering
THD, exact peak/cursor bins, export and independence from live controls. Browser
RC capture: 200 ms, 10 Hz fundamental, 5 Hz bins, native peaks at 10/30/50/70/90 Hz;
A/B bins at 10/20 Hz and a 10 Hz delta. Live controls exercised separately.

Prism receipts and finding resolutions are retained alongside each milestone.

## Remaining sequence

2. Dedicated sweep methods, retained results, exact cursors and native analyses.
3. Recording comparison, edge timing and step response.
4. Combined automated/browser validation and qualification of the running app.

The qualified native alpha and the owner's persistent project library remain
preserved. Browser tests use disposable storage on port 8789.
