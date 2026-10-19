# Cropped display recorder: offline guard checks

October 19, 2026. A separate explicit display-composition recorder is implemented; its actual display/crop/visibility calibration and timing cohorts are pending. The owner has already authorized the scope in [the measurement decision](../../../research/36-visible-latency-method.md). No capture, window enumeration, UI input, privacy preflight or permission grant occurs in these offline checks.

The source requires the exact owned ELab process/window/executable/hash and explicit display-local crop. Independent CG/SC window/display bounds and AppKit backing scale must agree. Missing/wrong/off-display/fractional/oversized crops cannot fall back to a display capture. The crop stays fixed. Complete frames require matching numeric content/screen/scale metadata and native dimensions, valid nonregressing display timestamps, and unchanged target geometry at receipt. Bad frame metadata is retained without image output. Nonfinite metadata is serialized as diagnostic text rather than breaking the journal.

The fixed Mach output deadline begins before stream admission. Samples delivered after it are counted and ignored before metadata/pixels are inspected or stored. Server stop acknowledgment is asynchronous and is not claimed to complete within that output bound. Recording is limited to 0.1–5 seconds, 300 inspected callbacks, 4096×4096 pixels and 64 MiB of unique PNG data. Audio/microphone and cursor output are disabled. Streams exclude no windows/applications; overlapping pixels can appear within the authorized crop. No global input observation/injection or OS permission request is added.

Swift 6 compilation with warnings as errors and strict Swift formatting pass. **53 offline checks** pass: 46 geometry/options/clock/metadata/PNG/deadline/storage/first-error cases and seven journal/marker filesystem collisions. Synthetic native 4×3 PNGs decode with unchanged dimensions/digests, deduplicate without dropping frame rows and retain 0600 permissions. Timestamp regression preserves the earlier image and invalidates the later frame. Nonfinite/missing metadata, geometry changes, existing image bytes and first stream-error retention have explicit counterexamples.

The first compile log retains invalid trailing-dot Swift numeric literals in the new fixture. A subsequent restricted execution logs IOSurface errors and fails its valid-image assertion. These failures are retained; the latter is not a controlled causal diagnosis or an actual capture failure. Final source checks run with macOS image-surface access and succeed. The new binary hash is bound in `checkpoint.json`; the former window recorder and all of its source hashes remain unchanged.

All 650 Go hashes still match the default source gate qualified for /173. The 400-entry sealed review, 401-file ZIP, SDK and canonical app remain intact. No new app build, model, command, format or SDK change is made. This instrumentation is outside the application runtime and review bundle.

The tool always reports zero eligible observations, null latency outcome and uncalibrated pixels/visibility. Receipt checks alone do not prove past display visibility. Runtime wrong-crop refusal, independent multi-point/edge alignment, an owned-occluder comparison with the frozen window stream, and acknowledged CUA causal-frame calibration remain required before predeclared LED/RC/NPN cohorts. Photons, other hardware/modes, typing/IME/accessibility, owner acceptance and the actual study are not inferred.

Read-only verification:

```sh
python3 -B docs/validation/display-frame-probe-offline/verify.py
```

Actual offline checks, with access to macOS image surfaces:

```sh
sh tools/display-frame-probe/check.sh
xcrun swift-format lint --strict tools/display-frame-probe/*.swift
```

[Tool contract](../../../tools/display-frame-probe/README.md), [qualified /173](../sweep-review-173/README.md). Apple documents the full-display default for an unspecified crop in its [sourceRect reference](https://developer.apple.com/documentation/screencapturekit/scstreamconfiguration/sourcerect); installed SDK headers distinguish logical source points from destination pixels. Actual API geometry is not accepted until runtime calibration establishes it.
