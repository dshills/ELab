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

## Milestone sequence

2. Dedicated sweep methods, retained results, exact cursors and native analyses —
   completed in `b54a53f3`.
3. Recording comparison, edge timing and step response — completed in `d9e367bb`.
4. Combined automated/browser validation and qualification of the running app —
   evidence below.

The qualified native alpha and the owner's persistent project library remain
preserved. Browser tests use disposable storage on port 8789.

## 2. Sweeps

Dedicated DC/component, AC gain/phase, white-noise and loop-response methods use
native target inventories and defaults. Measurement selection supports native
multiple-output limits and omission for automatic selection. Retained results,
progress/cancellation, project membership, captured trace selection, exact stored
point indices and keyboard/plot cursors are directly accessible. Draft settings
never relabel previously captured results. Native statistics, integrated noise,
AC bands, loop margins and source assumptions are displayed without recomputing
numerical analyses in JavaScript. Exports pin the selected captured operation.

Validation: 19 frontend tests; race-enabled HTTP checks of all four methods,
converted captured channel IDs, exact point selection/clear, JSON/CSV exports and
independence from live components/transient state. Browser checks cover DC/AC,
retained navigation, native defaults, exact index and End-key selection, and
captured assumptions. Full web race suite passed. See [Prism findings and fixes](sweeps-review.md) and
[the complete review receipt](sweeps-prism.json).

## 3. Recording analysis

The dedicated Scope now opens comparison, threshold timing and step-response
workflows. Forms bind to the selected capture, document/server identity and
acknowledged command revision. Stale sources disable editing; the “Use current
settings” action explicitly refreshes drafts. Acknowledged local analysis changes can
continue without accepting an intervening external edit. Refused inputs remain
editable with native validation reasons.

Comparison reports native A/B readings, B − A, exact sample provenance and
optional whole-record statistics/reference overlay. Timing uses captured probe
units, optional transition levels, direction/interval and complete paged crossing
brackets with native bounds. Step response accepts supplied reference levels,
tolerance and continuous hold, and shows native overshoot/settling or its reason
for unavailability. All three analyses can be cleared explicitly.

Validation: all 23 frontend tests at qualification time and syntax checks passed; race-enabled HTTP
coverage verifies comparison/overlay, timing pages and brackets, RC settling,
clear commands, stale-source refusal and clean-document/live-data immutability.
Embedded module routes are checked for JavaScript content and no-store caching.
Browser checks covered unit-error correction, timing pagination, A/B values,
false-preference round trips, 39.1 ms RC settling, cross-tab source protection and
refresh, and a 600 × 900 layout without page overflow. No console errors.
Prism completed with full coverage. See [findings and fixes](recording-review.md),
[the review receipt](recording-prism.json), [frontend results](checks/frontend.txt),
and [HTTP race results](checks/web-race.txt). The frontend runner is
`scripts/check-web.mjs`; HTTP tests are in `exp-web/recording_analysis_test.go`.

## 4. Combined qualification

Qualified code: `d9e367bb273e7cccefbb162b494145eee54f237d` on `exp-web`.
The embedded macOS/arm64 build was compiled from that clean source, using Go
1.27.3 with CGO. Its SHA-256 is
`107e9ecd3e05882a8d76728864c3013d0bae41a52bb4e534e44666093276d2e4`.
See the [machine-readable qualification](qualification.json) and
[embedded build information](checks/build-info.txt).

The complete combined race suite passed (workbench 330.473 s), followed by the
final web race check (12.201 s) after the small presentation/test-diagnostic
review refinements. The native runtime was unchanged by those refinements.
All-package vet passed, with the final web package rechecked. All 23 frontend
tests and module syntax checks passed; the Python quality runner passed 104 tests.
The requirement and command inventories verify source coverage, and do not
establish product acceptance: the requirement audit still records ten partial
items. Logs are in [checks](checks/).

The running embedded build was checked at `http://localhost:8788/`:
Scope, Spectrum and Sweep load and the local engine is connected, with no browser
console errors. [Its screenshot](qualified-web.png) shows the default RC example.
The deeper workflows were checked with disposable storage on port 8789; that
owned test server was closed normally.
The owner's default project library and qualified native alpha remain preserved.

Qualification is for the local macOS candidate in the in-app browser. A local Go
server is required. Remote Linux CI, other browser/platform combinations and
physical printer jobs were not exercised in this qualification. Native OS flows
use the browser adaptations listed in [PARITY.md](../../../exp-web/PARITY.md).

Prism's larger output budget is committed separately in its own repository as
`3807dd6`: 32,768 initial output tokens, 65,536 on the final unsplittable retry,
and a 900-second Wafer request timeout. Review/provider race tests and vet passed;
completed live reviews confirmed full coverage. Configured provider/model settings were retained; the output allowance and
Wafer HTTP timeout changed as documented.

The [qualification-record review](qualification-prism.json) completed with full
coverage and four low findings. The local URL is recorded as text; the Prism
budget receipt now names its repository and full commit. Per-check source states
and log timestamps distinguish the combined run from final package/frontend
rechecks; [the source comparison](checks/shared-runtime-identity.json) establishes
unchanged production Go code/dependencies. Go timings use the same fields and
explicitly identify the measured package, rather than claiming whole-command
elapsed time. These documentation refinements were internally checked for valid
JSON, links, commit identities and binary checksum; no runtime code changed.
