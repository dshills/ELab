# Web feature parity and qualification

Status: local macOS web candidate, 2026-11-01. The shared native workbench remains
the source of numerical behavior. This work does not publish or replace the
qualified native alpha.

## Feature coverage

| Native feature family | Web implementation |
| --- | --- |
| Parts and examples | All 61 kinds and 88 examples, including four-MOSFET H-bridge; native geometry, terminals and routes |
| Drawing | Place, wire, junctions/net labels, move, rotate, duplicate, delete, undo/redo, keyboard placement/movement, pan/zoom/fit |
| Circuit blocks | Create/copy/paste, published values, expand/collapse, release; native commands in Tools |
| Models | Native component configuration forms for thermal/BJT/op-amp, JFET, thyristor, photodiode, motor, transformer, battery, optocoupler, current source/load, speaker, PWM and sound |
| Transient controls | Pause/resume/restart, sample or arbitrary step, sampling fit, speed, live source modes, startup charge/current, passive losses and waveform timings |
| Live controls | Switch, momentary/bouncing button hold/release/click, potentiometer, sensor/light/temperature and motor load |
| Measurements | Accepted readings, stale-state explanations, native wire colors/current cues, pinned readings, LED and seven-segment indicators, IC labels |
| Scope | Direct live/captured source picker, trace chooser, precise time windows and cursors, A/B values and deltas, exact current sample markers/readings, DC points and recorded reference overlay |
| Recording analysis | Direct capture/progress/cancel/keep, playback/scrubber/seek, speed/RMS/statistics; dedicated comparison/reference overlay, threshold/transition timing with complete pages, and step-response/settling controls; import/export |
| Spectrum | Dedicated live/recorded source and capture/channel selection, FFT window/grid and fundamental/harmonics, hold/off, native ranked peaks and A/B frequency cursors/deltas, THD and recorded exports |
| Sweeps | Dedicated DC/AC/noise/loop methods, multi-probe/range/grid controls, retained results/progress/cancel/keep, native captured channels, exact index/keyboard/plot cursors, native analyses/assumptions and pinned exports |
| Tolerances | Seeded native Monte Carlo settings, progress, trials/statistics and full JSON/CSV export |
| Projects | Native `.elab` import/export, kept recordings/sweeps/models, named local library and Save a copy |
| Durability | Atomic checkpoints and named revisions; restart/lost-cookie recovery, retained previous saves, reversible Trash, stale-editor/concurrent-save protection |
| Audio | Independent native WAV render; explicit browser playback/stop; native WAV download and invalidation |
| Output | SVG, visible scope CSV, capture/sweep/spectrum/tolerance JSON/CSV, WAV; macOS PDF/PNG; browser print preview |
| Workspace | Resizable panels, narrow-screen toggles, large schematic labels, stronger contrast, Light/Dark/System/Apollo themes and reduced motion |

## Platform adaptations and limits

- The Go server must run locally. There is no WASM-only runtime, cloud account,
  hosted authentication or remote collaboration layer.
- Browser file pickers, downloads, media controls and Print / Save PDF replace OS
  file associations, native audio playback and native printer discovery/job control.
  Native socket/MCP session discovery remains part of the native/agent hosts.
- PDF/PNG downloads currently require macOS CoreGraphics. Browser SVG printing is
  the portable route. Physical printer output has not been submitted or qualified.
- Browser panels, drawing gestures and viewport are browser presentation state;
  native host-only UI commands are omitted from its command menu. Documents refit
  on open/resize. Model values, exact captured data and supported saved preferences
  use native serialization.
- macOS is the exercised local target. The Unix storage lock supports macOS/Linux;
  other platforms use a conservative exclusive marker and are not crash-restart
  qualified. Linux CI is configured but was not executed remotely in this task.
- The original load-sensitive 555 timeout is fixed in the shared solver: 100 ms
  of worker CPU, 512 attempts and a separate 5 s wall backstop. Synthetic CPU-load
  checks preserve exact samples, including the live web clock; see the
  [solver qualification](../docs/validation/solver-cpu-budget-20261101/README.md).
  A busy computer can still reduce playback speed; visible latency under arbitrary
  system load is not qualified.
- Results tables cap displayed arrays at 200 rows; downloads retain complete data.
  Numerical/device/size limits are the same native educational-model limits.

## Verification

- `go test -race ./exp-web -count=1`: browser HTTP integration coverage, including
  RC values, H-bridge direction reversal, native file/recording round-trip, model
  settings, stale guards, origin/path boundaries, atomic autosave/recovery, shared
  library lost-update refusal, lost-cookie discovery, storage failure/rescue,
  AC complex-reference values and kept curve recovery, WAV bytes and macOS PDF/PNG.
- `go test -race ./... -count=1 -timeout=30m`: all packages passed across the initial
  run and the targeted rerun of socket-dependent packages. The sandbox initially
  refused Unix socket binds; `internal/agentmcp`, `internal/conformance`,
  `internal/printspool` and `transport` passed when rerun outside that restriction.
  Native core race tests completed successfully (about 320 seconds).
- `go vet ./...`, internal staticcheck for web/core and goimports checks: passed.
- `node scripts/check-web.mjs`: all JS/module syntax checks and nine frontend
  settings/schema tests passed. Quality-runner Python tests passed.
- Browser checks: named save/library/Trash/restore; applying 40 °C thermal settings;
  acknowledged 2.2 µF edit surviving reload and an actual forced process crash;
  automatic reconnection; native Monte Carlo completion (16 trials); inspector and Simulate controls
  accessible at 600 × 900; native WAV
  loaded into browser media and explicitly played to its 20 ms end with no media
  error. Browser print preview loaded the complete native SVG without submitting
  a physical job. Earlier editor/RC/H-bridge/recording/spectrum/sweep checks remain covered
  by the HTTP tests and the initial browser comparison work.

### Recorded-scope workflow, 2026-11-01

The dedicated Scope now exposes native playback, sample index/time seeking,
recording membership, captured trace selection, interval/cursor entry and
time-weighted measurements. Capture accepts native time quantities and source
modes, retains refused entries, and supports the zero-duration DC operating point.
Viewing an existing recording no longer silently changes its kept membership;
new captures retain the existing automatic keep behavior.

`go test -race ./exp-web`, `go vet ./exp-web` and all 13 frontend tests passed.
Browser checks exercised RC recording/playback, keyboard scrubbing, exact time
entry, trace/window/cursor changes, interval RMS, invalid-capture correction,
single-point DC rendering and disabled DC playback, source changes in another tab,
and a 600 × 900 layout. See [evidence and internal review](../docs/validation/web-recorded-scope-20261101/README.md).

### Spectrum workflow, 2026-11-01

The dedicated Spectrum now exposes live and recorded FFT settings, native peaks,
exact frequency/plot cursors, distortion details and recorded exports. It keeps
captured results separate from live frames and refuses stale source edits. All
17 frontend tests and the race-enabled native FFT HTTP regression passed. Prism
completed with full coverage; accepted findings were fixed and internally checked.
See [evidence and finding resolutions](../docs/validation/web-parity-completion-20261101/spectrum-review.md).

### Sweep workflow, 2026-11-01

Dedicated sweep workflows passed all 19 frontend tests and the full web race
suite. HTTP tests exercise all four native methods; browser checks cover DC/AC,
retained results, exact cursors and correction of refused entries. Prism completed
with full coverage. See [review and fixes](../docs/validation/web-parity-completion-20261101/sweeps-review.md).

### Recording-analysis workflow, 2026-11-01

Scope now exposes captured comparison, threshold/transition timing and step
response directly. All 23 frontend tests, race-enabled web regressions, combined
shared race qualification and vet passed. Browser checks include native comparison
values, exact timing pages, RC settling, omitted-zero/false settings, stale-source
refusal and a 600 × 900 layout. Prism completed with full coverage; accepted
refinements were internally verified. See [review and evidence](../docs/validation/web-parity-completion-20261101/recording-review.md).

The dedicated spectrum, sweep and recording-analysis workflow milestones are
complete. Platform adaptations and numerical-model limits above remain part of
this local macOS candidate's scope.

## Internal review

Reviewed storage publication ordering, ownership/bounds of native captures,
concurrent workspace locking, copy-on-recovery, cookie/session lifetime, native
command routing, file/origin boundaries, schema resolution, DOM form submission,
stale media cleanup and native/UI result provenance. Fixed issues found during
review and browser checks: shared-library isolation, lost-update races, nullable
schema references and empty defaults, nested-array reindexing, model-ID form
submission shadowing, stale result replacement, audio snapshot field routing,
paused storage-error visibility and read-only example previews clearing project
association, and RC charge/discharge buttons using network-only mode IDs. Initial qualification used internal review. The later dedicated workflow milestones
were reviewed with Prism at the owner’s explicit request; their complete receipts
and finding resolutions are retained in the qualification record.
