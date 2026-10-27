# Recorded spectrum and harmonic distortion

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
