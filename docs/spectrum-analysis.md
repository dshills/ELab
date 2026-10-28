# Scope, live spectrum and recorded harmonic distortion

The signal dock has **Scope / Spectrum** tabs. Scope retains its live and recorded
waveform controls. Spectrum puts the existing FFT plot in the dock, with **Live**
and **Recorded** source choices. Both tabs and all settings use the same shared
commands for the desktop and AI agents.

## Live spectrum

Opening Spectrum for the first time enables live FFT with a Hann window and an
automatic sample count. Choose one of the same selected voltage/current scope
probes, or **First scope probe**. Add/remove channels through the existing Scope
probes controls. A removed channel stays visibly unavailable until reassigned or
restored; it never silently selects a different signal.

Live analysis copies accepted points inside the scope window and reuses the
recorded FFT engine. It does not run another electrical solve or advance time.
One lazy service worker checks for changed samples every 250 ms (at most four
updates/second), computes outside the service mutex, then publishes only if the
source document, circuit/run, actuation, probe selection/window and settings still
match. Unchanged paused data is not repeatedly analyzed. Restart, electrical edits and
replacement documents cannot borrow an old frame.

**Hold** freezes the derived spectrum while simulation continues. Pause stops
simulation normally. Source changes invalidate even a held result. **Recorded**
stops live publication and exposes recording/probe choices and **Analyze**;
advanced interval and export controls remain in Simulate → Spectrum & distortion.
The original recorded settings/result remain independent of live analysis.

Live retains the existing 512-point scope history. An end-exclusive FFT needs
N intervals, so live uses automatic or 32/64/128/256 samples; at least 33 distinct
accepted points are needed. The analyser never invents a sample grid finer than
the recorded spacing. Resolution, Nyquist frequency and frame start/end are
reported for the actual analyzed grid. The original recorded analyser still
supports up to 4096 samples. Mean removal, window scaling and THD requirements
are unchanged.

Tabs, selectors, numeric fields, plot area and two readout rows keep fixed
allocation as results arrive. Only plot content/readout text changes. Short docks
retain their existing scroll viewport. Scope cursor input is ignored while the
Spectrum tab is showing.

### Live commands

- `scope.view.set {"view":"scope"}` or `{"view":"spectrum"}` selects the dock.
  The first Spectrum selection enables live defaults if no live draft exists.
- `spectrum.live.set {"enabled":true,"hold":false,"probe_id":"v:C1","window":"hann","samples":0,"fundamental_hz":0,"harmonics":5}`
  replaces live settings; `enabled` is required. Empty probe means first selected
  available scope trace. Window defaults to Hann, samples to automatic and
  harmonics to five. Frequency accepts numeric Hz or unit text such as `10 Hz`.
- `project.inspect`/events expose `scope_view` and `spectrum.live`: settings,
  status/reason, run, analyzed start/end and a detached FFT result. Publication
  changes the state revision, not the command revision; ordinary mutation guards
  and retry identities still apply. Snapshot reads do not compute FFTs.

Live data/settings are session-only and clear on document replacement. A live
frame is not a Verify recording, and recorded export/page commands retain their
existing captured-result target. To keep/export an analysis, make a Verify capture
and use Recorded. No component, solver, sample-history or integration limit changes.

## Recorded spectrum

After a completed Verify capture, the Inspector's **Spectrum & distortion**
section analyzes any captured voltage or current probe. Choose the recording,
probe, start/end times, window, automatic or explicit sample count, fundamental
frequency and harmonic count, then **Analyze recorded spectrum**. The plot shows
peak magnitude in dB relative to one probe unit versus Hz. Harmonic readouts use
peak V/A, not RMS. No noise floor, THD+N or physical audio-accuracy claim is made.

Try **Examples → Amplifiers & filters → Amplifier clipping and harmonic
distortion** (`amplifier-distortion`). Verify 200 ms at 500 µs, capturing
`v:VIN.positive` and `v:U1`. Choose `v:U1`, a 10 Hz fundamental, rectangular or
Hann window and five harmonics. The 3 V peak input overdrives the gain-of-two
stage. Reduce VIN to 0.5 V and compare. The all-discrete amplifier also works:
capture 400 ms at 500 µs, then analyze the steady 200–400 ms interval at 10 Hz.

The analyzer removes the unweighted mean and linearly resamples recorded data
onto a uniform, end-exclusive grid. It never chooses a grid finer than the
widest recorded sample gap. Automatic samples choose a power of two up to 4096;
explicit choices are 32–4096. Duplicate edge timestamps use the accepted right
side. Resampling can attenuate harmonics: use adequate original capture sampling
and compare a finer capture when accuracy matters. Window coherent gain corrects
peak scaling. Zero/faint bins have a −300 dB display floor, not simulated noise.

THD is `100*sqrt(sum(harmonic_peak², h=2..H))/fundamental_peak`. It needs a
declared fundamental, at least two complete integer cycles inside the interval,
and all requested harmonics strictly below Nyquist. Noncoherent or inadequate
intervals still produce FFT bins with an explicit THD-unavailable reason. A
steady waveform and adequate capture resolution remain the user's responsibility.

## Shared commands and saved results

- `spectrum.settings.set` configures `id` (completed recording), `probe_id`,
  `start_s`, `end_s` (0 means capture end), `fundamental_hz` (0 leaves THD unset),
  `window` (`rectangular`/`hann`), `samples` (0 automatic), `harmonics` (0 means 5).
- `spectrum.analyze` takes the same parameters. `{}` uses current settings;
  a nonempty object fully replaces settings with the declared defaults.
- `project.inspect` exposes `spectrum.settings`, captured `available` probes
  and the result: all frequency bins, harmonics, mean, THD or reason, actual
  sample grid, assumptions, source-recording SHA-256 and exact result ID.
- `spectrum.export {"id":"spectrum-…","path":"/absolute/spectrum.json","format":"json"}`
  exports that exact current result; CSV is also supported. Discover the full
  ID from `snapshot.spectrum.result.id`. An intervening different analysis
  retires the old result identity, so a stale export cannot borrow newer data.

Times accept numeric seconds or text such as `200 ms`; frequency accepts Hz,
kHz or mHz quantities. Mutations use ordinary request/retry/revision/document
preconditions. Detailed `capabilities.list` describes bounds and file completion.
Native controls call these same commands with their displayed context.

Settings and the derived plot are session-only. Export JSON/CSV to keep the
result; include the original Verify recording when saving the `.elab` project.
Reopening that recording and rerunning the same settings reproduces the bins
and THD. The spectrum export's source hash identifies canonical captured metadata
and rows; it is provenance, not hardware validation. Analysis/settings do not
change the live circuit, time, physical state or Undo history. Document replacement
clears the spectrum draft/result. Electrical simulation and analysis never play
sound automatically.
