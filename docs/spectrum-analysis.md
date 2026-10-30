# Scope, live spectrum and recorded harmonic distortion

The signal dock has **Scope / Spectrum** tabs. Scope retains its live and recorded
waveform controls. Spectrum puts the existing FFT plot in the dock, with **Live**
and **Recorded** source choices. Both tabs and all settings use the same shared
commands for the desktop and AI agents.

Both tabs remain available for DC and empty circuits, including when the dock is
collapsed. Use **Show scope** to expand it. When no time-based data exists, an
explanation points to transient examples and **Simulate → Recorded runs**. A DC
operating point alone is not a waveform. Selecting a tab does not change the
circuit, advance simulation time, or start a recording. Recorded analysis stays
independent of the current circuit.

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

## Peak markers and frequency/amplitude cursors

In either **Live** or **Recorded** mode, **Peaks** toggles P1–P5 markers.
They identify the five strongest non-DC local maxima above the display floor
and within 60 dB of the strongest candidate. Equal plateaus and equal-amplitude
peaks choose the lower frequency. These labels report sampled FFT bins;
they do not interpolate a more precise frequency. Peaks remain inspectable by
agents when their plot markers are hidden.

Select **A** or **B**, then click or drag the spectrum to place that cursor.
Alternatively enter its frequency in the A/B fields, using numeric Hz or unit
text such as `100 Hz`, `2 kHz` or `250 mHz`. **Peak → cursor** places the selected
cursor on P1. Clear A/B removes a cursor; an empty frequency field also clears it.
Cursors snap to the nearest actual bin, with midpoint ties choosing the lower
frequency. Each crosshair shows that bin's frequency, peak amplitude in the
selected probe's unit (V or A), and dB relative to one unit. The horizontal
line follows the bin's amplitude; it is not an independent amplitude threshold.
A cursor outside the visible dB scale is labeled **off** without drawing a
misleading horizontal line on the plot edge.

Readouts include signed B−A differences in Hz, peak amplitude and dB. Bins at
or below the −300 dB display floor cannot support an exact dB difference, so
that difference is explicitly unavailable. This floor is not simulated noise.
The plot scales small voltage/current signals as well as signals above one unit.
The controls and readout rows keep fixed allocations as frames and values change.

Live cursors retain the requested frequency while new frames arrive, then snap
again to the current frame's bins. Thus frequency resolution changes can change
the measured bin. Hold freezes the frame and readings. Changing the probe,
circuit/run, actuation, scope selection/window or FFT settings retires old live
anchors. Recorded cursors belong to one exact result; changing its settings or
analyzing a different result retires them. The two sources have independent A/B
positions and active cursors. Marker visibility is shared between them.

### Shared measurement commands

```json
{"command":"spectrum.markers.set","params":{"visible":true}}
{"command":"spectrum.cursor.set","params":{"source":"live","cursor":"a","frequency_hz":"100 Hz"}}
{"command":"spectrum.cursor.set","params":{"source":"live","cursor":"b","peak":1}}
{"command":"spectrum.cursor.set","params":{"source":"recorded","cursor":"b","select":true}}
{"command":"spectrum.cursor.set","params":{"source":"recorded","cursor":"a","clear":true}}
```

These are command/parameter fragments: transports also require the normal
version, request/retry identity and command/document revision guards. A cursor
command requires `source` (`live`/`recorded`), `cursor` (`a`/`b`) and exactly one
of `frequency_hz`, `peak` (current rank 1–5), `clear:true` or `select:true`.
Frequency must be finite, nonnegative and at most the current Nyquist frequency.
Placement requires a current result; selecting or clearing is allowed without
one. Optional `result_id` pins an exact current frame and returns `conflict`
if it has retired. Omit it to target the latest valid live frame.

`project.inspect` and measurement events expose `spectrum.measurements` for
Recorded and `spectrum.live.measurements` for Live: source/result ID, unit,
visibility, active cursor, ranked peaks, requested A/B frequencies, actual bin
readings, validity/reason and available differences. Results and readings are
detached snapshot data. Discover the complete input contract with detailed
`capabilities.list`; the compact catalog uses shared schema references.

Markers and cursors are session-only. They do not change FFT bins/THD, physics,
acquisition, simulation time, Undo history or the document dirty flag. Document
replacement resets them. Existing spectrum JSON/CSV exports continue to contain
the captured FFT result, rather than a mutable cursor view.
