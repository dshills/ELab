# Exclusive recorder journal and offline filesystem faults

**A pre-existing journal is now refused without changing its bytes or mode.** The original collector uses an overwrite-capable `FileManager.createFile` call. A constructor regression writes a sentinel journal, invokes that actual original collector and fails with `Existing frame journal was overwritten or admitted`. It runs only self-test code, never a capture/window/privacy path. The outer command's fresh-directory guard protected the earlier recorded trials; this counterexample is an internal collector-boundary defect, not evidence that historical recordings were lost.

The collector now opens its journal with `O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC`, mode 0600, then transfers ownership to `FileHandle`. Seven offline filesystem/marker cases pass under Swift 6 warnings as errors:

1. Existing journal refused with bytes and permissions preserved.
2. Existing directory destination refused and left intact.
3. Symlink destination refused with its canary target preserved.
4. Dangling symlink refused without creating its target.
5. New journal has 0600 permissions; repeated finalization returns the same empty report.
6. A second collector cannot truncate real marker evidence while the first collector/timer remains active.
7. An existing acknowledgment survives the attempted write; the collector retains its error and does not count an accepted marker.

The last two cases exercise the real serial marker queue and filesystem handshaking. They do not feed sample buffers or query the Window Server. Existing argument, output-directory, marker-name, timebase, tick-serialization and exclusive-acknowledgment checks also pass. `sh tools/frame-probe/check.sh` compiles both Swift source files and executes these tests. No window, screen capture, input, Screen Recording preflight/request, app session or system setting is operated in this work.

The repaired development recorder hash is `677c5e117c40ca8f60b5c54e43176324ce4c10f799cbcf8f56813f4532d4cbaa`. The previously calibrated `6a861e7…` executable is preserved byte-identically at `build/frame-probe-journal-checks/frame-probe-calibrated`. Its original [174-frame input/image evidence](../marker-input-calibration-170/README.md) still replays, and all 64 retained evidence files verify. All 1,651 sealed /170 review entries and canonical /120 bytes remain unchanged. ELab, its SDK, native bindings, numerical models and project format are unchanged.

The new recorder has **not** been runtime-calibrated. Historical source/recorder admission guards retain their original hashes; they must reject this different tool rather than silently inherit prior calibration. This fixes one evidence-preservation fault boundary. Stream errors, frame/image budgets, blank/idle/dropped-frame behavior, physical clock/pixels, unobscured visibility and cohorts remain open. There are **0 eligible latency observations** and no p95 decision or owner acceptance.

`checkpoint.json` binds final source hashes, original regression/error, successful checks and readonly preservation results. Raw logs/source are compressed losslessly; no app/recorder executable is added to Git. Repeat the current offline tests with the command above; the original regression source is retained only as a historical counterexample.
