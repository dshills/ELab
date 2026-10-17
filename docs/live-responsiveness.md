# Live network responsiveness

General-network live samples (including the RC/NPN flasher and waveform networks) now compute outside the shared service lock. Each sample captures independent capacitor memory, solution vectors, result flags and trace storage. The compiled network and retained sample frames are immutable. Publication requires the same run, applied-state/trace identities and actuation revision, with a live uncanceled clock. Old results and failures cannot overwrite pause, restart, close or a newer switch/wiper/input state. Presentation edits can continue and survive a valid sample publication.

Each accepted network sample advances the session revision. Repeated Resume on an already running simulation preserves its clock; defensive clock replacement cancels the previous worker. Shutdown tracks live workers and waits outside the mutex. Existing numerical models, tolerances and sample work bounds remain unchanged.

## Command admission during a live clock

`expected_revision` continues to guard all published session state. The additive `expected_command_revision` option instead guards admitted mutations, including other actors, while ignoring background publication. At least one is required with a retry key; if both are supplied, both must match. Discovery advertises the options in `mutation_preconditions`. Original successful retry outcomes remain protected and independently copied. Use the full-state guard when a decision depends on an exact physical sample or background result state.

Native controls/file dispatch use the command guard. They refuse intervening command conflicts without automatically refreshing the precondition and resubmitting the same action. Original actor/exact-tip undo guards and document-generation/file guards still apply. Earlier clients using full-state preconditions remain supported; they must handle clock conflicts explicitly. The shared Python client offers opt-in command guards so demos can still connect to older owned builds.

A first actual-flasher benchmark exposed repeated clock conflicts under the old one-retry helper. The command guard addresses that case without weakening other-actor detection. Core tests cover actual clock publication, strict combined/full-state refusal, actor-command conflict, successful deduplicated retry and missing-guard refusal.

## Evidence and measured envelope

[Controlled blocked-kernel race tests](validation/blocked-live-network.json) hold one live sample at its numerical boundary, then run the real kernel or return an obsolete failure. Navigation, drawing cancel, save, geometry undo, pause, restart, close and switch queue remain callable. Assertions verify discarded work cannot change restarted/closed/paused state, valid samples preserve presentation edits, and repeated Resume creates no extra clock. These prove lock availability; they are not a visible-latency benchmark.

The actual flasher workload makes 107 public calls (pan, save, inspect, geometry move/undo and pause) while nonlinear live integration progresses, with four scope traces. A separate 200 ms Verify runs while live time is paused, then its saved experiment reopens fresh and clean. Reports retain raw timings and reference configuration: Apple M4 Pro / Mac16,11, 14 logical CPUs, 64 GiB RAM, macOS 26.6.2, Go 1.27.2 arm64; workload binaries are not race-instrumented.

| Transport | Backend API p95 | Maximum API call | Verify 200 ms wall time |
|---|---:|---:|---:|
| [Headless stdio](validation/headless-live-responsiveness.json) | 11.44 ms | 13.86 ms | 52.16 ms |
| [Packaged desktop socket](validation/desktop-live-responsiveness.json) | 10.81 ms | 14.68 ms | 40.97 ms |

The fixture has 12 drawing objects, eight nets, 5 ms output steps and 0.5 simulated seconds per wall second. Live endpoints may differ because scheduling is real; independent Verify captures identical configured startup. Timing is a point measurement on this workstation, includes transport/serialization, and excludes input-to-pixel latency.

## Remaining work

Network and NPN phase initialization now run outside the lock; input/wiper algebraic solves now use [background input work](input-work.md), while some file/recovery work remains synchronous. This release does not satisfy the whole slow-work/nonblocking requirement. The expanded [capacity envelope](circuit-capacity.md) supports 64 simulated parts and 100 nodes including ground; visible-latency validation remains separate. Mounted-driver gesture invalidation, complete reference-circuit visible p95, screen-reader, clean-installation and participant gates remain open.

Run `python3 scripts/live-responsiveness-demo.py --output REPORT --artifacts DIRECTORY` after building. Use `--connect` only for an owned development endpoint: it replaces that session's project. The packaged development launch uses absolute endpoint/recovery paths.

Manual multi-sample Step is now addressed separately by [background manual work](manual-step-work.md); the broader responsiveness gate remains partial.

Project Open now also reads/validates outside the lock with [observable project work](project-work.md). Save/export/manual recovery are separate remaining paths.

Save serialization, destination checks, disk publication and recovery cleanup now also run outside the lock via [captured save work](project-save.md). Export/manual recovery remain separate paths.

SVG/live CSV/recorded JSON+CSV export encoding and disk work now run outside the lock via [captured exports](project-export.md). Manual recovery/import/shutdown remain separate paths.

Recorded import now also uses [background project work](project-import.md). Manual recovery and some shutdown calls remain separate paths.

Manual recovery scan/checkpoint now also run outside the lock through [shared jobs](project-recovery-work.md). Other recovery/shutdown paths remain explicit remaining work.

## Real capacity-work command spans

The unchanged 0.124.0 /141 executable was measured through a private native endpoint on the same Apple M4 Pro configuration above. The frozen capacity RC/MOSFET fixture has **64 simulated parts, 65 drawing objects, 70 wires, 60 nets and four capacitors**. Twenty trials requested one-second /1,000-output-sample manual advances. Pan, drawing cancel, save, geometry undo and exact work cancellation were bracketed by actual same-ID/run numerical progress; Pause was measured separately after the real live clock advanced. No kernel was replaced or artificially blocked.

| Shared command | Eligible backend spans | Request → response p95 | Maximum |
|---|---:|---:|---:|
| `view.pan` | 20 | 4.87 ms | 5.60 ms |
| `drawing.cancel` | 20 | 4.41 ms | 5.06 ms |
| `project.save` | 20 | 27.62 ms | 28.71 ms |
| `edit.undo` | 20 | 5.58 ms | 6.25 ms |
| `simulation.work.cancel` | 20 | 5.04 ms | 5.37 ms |
| `simulation.pause` | 20 | 4.63 ms | 11.05 ms |

Cancellation discarded **8–82 staged samples** per trial while preserving committed physics and graph. This is not a full-advance completion benchmark. The saved capacity circuit freshly reopens paused/clean at time zero with the exact graph and no staged job. [Raw native observations](validation/capacity-responsiveness-native.json), [one-trial headless pilot](validation/capacity-responsiveness-pilot.json), [reopen/native state witness](validation/capacity-responsiveness-states.json.gz), [configuration, identities and limits](../research/ui-audit/first-version/capacity-responsiveness-checkpoint.json).

Run `python3 scripts/capacity-responsiveness-demo.py --connect OWNED_ENDPOINT --artifacts NEW_DIRECTORY`; the default is 20 trials. It replaces only the designated test session and leaves it paused for inspection; close that owned host afterward. With no endpoint its headless session closes on return. Timing includes IPC/JSON, validation and any command waiting. Ineligible spans stay in the report; a percentile requires at least 20 eligible spans. Five regression cases reject queued/unstarted, terminal, replaced, regressed or committed-state-changing work as overlap witnesses. All 43 Python tooling tests pass; compiled Go source and the review package are unchanged, so their full quality gate is reused.

This establishes specific **backend** behavior during real capacity work. It does not time individual Newton iterations, native main-loop acknowledgment, first visible samples or frame presentation. Foreground/occlusion was not established, and default scope selection does not reproduce the separate four-probe flasher condition. One topology does not qualify all difficult/ill-conditioned cases. The PRD's visible 100/500 ms p95 and broader usability gates remain open.
