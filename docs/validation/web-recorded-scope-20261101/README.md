# Web recorded-scope qualification — 2026-11-01

This change brings the native recorded-viewer controls into the web Scope. It
uses existing guarded workbench commands and exact saved rows. No solver,
numerical limits, native application package or published alpha was changed.

## Delivered workflow

- Live/completed-recording source picker; explicit kept membership.
- Play/pause, stored-row stepping, keyboard/pointer scrubber, exact index and
  engineering-unit time seek, playback speed and draft reset.
- Native captured values and current-row plot marker, including DC operating
  points; DC playback and out-of-range step controls are disabled.
- Trace selection, fixed time interval/reset, exact A/B cursors, sample
  statistics, and separate time-weighted interval RMS/AC RMS/mean/peak-to-peak.
- Capture source mode/startup override, unit-aware duration/step, DC defaults,
  retained admission errors, progress and cancellation.
- Fixed requested plot margins, single-point rendering, recorded sampling labels,
  live-versus-captured inspector guidance and source-owned form guards.

## Automated checks

- `go test -race ./exp-web -count=1`: passed. The added HTTP regressions verify
  exact row stepping/time seeking, precise window boundaries, captured trace and
  cursor changes, interval measurements, playback completion/replay, unchanged
  live circuit and saved rows, stale request refusal, and one-point DC data.
- `go vet ./exp-web`: passed.
- `node scripts/check-web.mjs`: all module syntax checks and 13 tests passed.
  New cases cover source/generation/server-instance identity, engineering-unit
  requests, fractional indices preserved for validation, fixed/empty plot margins,
  optional capture startup overrides and native DC probe defaults.
- `git diff --check`: passed.

See `go-race.txt` and `frontend-checks.txt` for the final command output.

## Browser checks

Used an owned server at localhost:8789 with disposable storage under
`build/web-parity-20261101`. The user's persistent workspace on port 8788 was
preserved. No physical printing, external review or publication was performed.

1. Recorded RC data, selected it from Scope, stepped one row and sought `20 ms`.
   The captured capacitor value was 4.3233 V; the paused live value stayed separate.
2. Played the capture, sought index zero, then used ArrowRight on the scrubber.
   The displayed position became exactly sample 1 at 1 ms.
3. Hid the captured source trace, applied 10.1–20.9 ms, and observed ten exact
   capacitor samples. The plot preserved both requested margins.
4. Applied 10–20 ms time-weighted measurement and cursor A at 15 ms. The native
   results reported RMS 3.8508 V, AC RMS 333.18 mV and mean 3.8363 V.
5. Changed Scope source from a second tab. The original settings dialog showed
   its source-change warning and disabled every form action.
6. Entered an invalid capture duration. The dialog retained the draft and native
   parsing guidance. Corrected it to `100 ms`, selected Charge and completed a
   101-row capture using the project sample interval.
7. Recorded the LED DC example without live probes. Native default measurements
   appeared as single points; previous/play/next were disabled. The footer labels
   this as a DC operating point rather than an invented elapsed interval.
8. Checked Scope and interval controls at 600 × 900, then restored the temporary
   browser viewport. Browser console error log was empty during final checks.

Screenshots: [recorded Scope](recorded-scope.jpg), [interval measurements](interval-measurements.jpg),
[narrow Scope](narrow-scope.jpg), [DC capture](dc-capture.jpg).

## Internal review

Reviewed source identity and retained membership; draft ownership across native
selection/document/server changes; guard capture before queued actions; rejected
input preservation; DC and end-of-recording boundaries; cursor/plot interval
mapping; and numerical provenance. Fixed review findings: stale plot/probe guards,
capture completion overriding later selections, misleading live-reading captions,
recorded plots inheriting live sample labels, DC cursor clicks requesting
nonexistent times, compressed plot layout, and success feedback styled as errors.

Scope data remains the native snapshot. Plot point reduction is presentation
only; seeking, cursor readings and interval measurements use stored native data.
Advanced comparison/timing/step-response and spectrum/sweep workflows still use
Tools, as documented in `exp-web/PARITY.md`.
