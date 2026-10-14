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

Network initialization, NPN phase initialization, manual multi-sample Step, input/wiper algebraic solves and some file/recovery work remain synchronous. This release does not satisfy the whole slow-work/nonblocking requirement. The physical envelope remains 50 total drawing objects and 12 non-ground network nodes; the original 50 simulated-component / 100-node target is unchanged. Mounted-driver gesture invalidation, complete reference-circuit visible p95, screen-reader, clean-installation and participant gates remain open.

Run `python3 scripts/live-responsiveness-demo.py --output REPORT --artifacts DIRECTORY` after building. Use `--connect` only for an owned development endpoint: it replaces that session's project. The packaged development launch uses absolute endpoint/recovery paths.
