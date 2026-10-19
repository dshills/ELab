# Cropped display frame probe

Separate macOS qualification instrumentation; it is not part of ELab or the review ZIP. The owner authorized tightly cropped display recordings for calibration and LED/RC/NPN timing trials, up to five seconds each, local only and without audio. [Exact scope and decision](../../research/36-visible-latency-method.md). That approval does not grant new OS privacy access or authorize other apps' input. The frozen [window-only recorder](../frame-probe/README.md) and its bytes remain unchanged.

The tool requires a pinned owned ELab PID, window ID, executable path/hash, explicit display ID and display-local crop. It checks the target against AppKit, ScreenCaptureKit and CoreGraphics metadata, requires the whole window to fit one display, and compares the display's independent backing scale. It never follows a moving target. A wrong, empty, fractional, off-display or oversized crop is refused before a stream is created. Apple documents that an unspecified source rectangle can capture the entire display; this tool has no such fallback. [Apple sourceRect reference](https://developer.apple.com/documentation/screencapturekit/scstreamconfiguration/sourcerect). Installed SDK headers specify source rectangles in logical points and destination rectangles in pixels.

Display composition excludes no windows or applications. Another window/notification inside the crop can appear; that risk was disclosed and authorized. Mouse cursors are omitted. Audio and microphone are disabled, and the tool never requests Screen Recording access, opens settings, monitors input or injects events. All input stays in CUA. Discovery returns only metadata for the independently verified owned target and its containing display.

## Offline checks

```sh
sh tools/display-frame-probe/check.sh
xcrun swift-format lint --strict tools/display-frame-probe/*.swift
```

Swift 6 warnings are errors. Checks cover negative display origins/native scale, malformed crops/options, geometry/owner/mode changes, missing/nonfinite frame metadata, rescaling, clocks, native PNG encoding/deduplication, deadlines, limits, first-error retention and file/marker collisions. They create only synthetic pixels and temporary files, with no UI, window enumeration, privacy preflight or capture call. Synthetic CoreImage encoding needs macOS image-surface access. Compiler deployment target macOS 14 does not establish supported-OS acceptance.

[Offline proof](../../docs/validation/display-frame-probe-offline/README.md) records 53 passing checks. An initial Swift literal syntax failure and restricted execution's IOSurface/image assertion failure remain recorded. The latter is not a controlled causal diagnosis. Actual crop/scale/visibility calibration and all timing cohorts remain pending.

## Authorized runtime sequence

Use a fresh, hash/PID-verified protected ELab session with synthetic data, shared-geometry and development-file/network denial. Obtain the owned window ID independently. Never guess an ID or select the first running app/window.

```sh
# Placeholders: do not execute against an unverified or owner session.
build/display-frame-probe-tools/display-frame-probe --inspect-target \
  --pid OWNED_PID --window OWNED_WINDOW \
  --executable /absolute/verified/ELab --sha256 PINNED_SHA256

build/display-frame-probe-tools/display-frame-probe --capture-display \
  --pid OWNED_PID --window OWNED_WINDOW \
  --executable /absolute/verified/ELab --sha256 PINNED_SHA256 \
  --display VERIFIED_DISPLAY --crop VERIFIED_X,Y,WIDTH,HEIGHT \
  --output /private/tmp/new-private-output --seconds 3
```

`--preflight` reads existing access only. If it returns unavailable, stop; a new privacy grant is a separate action-time decision. Runtime output uses new 0700 temporary directories, exclusive 0600 files, ≤4096×4096 output pixels, ≤300 inspected callbacks and ≤64 MiB unique PNG bytes. The fixed Mach deadline starts before stream admission. Callbacks after that deadline are counted and ignored before metadata/pixels are inspected or stored; asynchronous server stop can acknowledge later. This is an output-recording bound, not a promise of synchronous OS cancellation.

Complete frames must carry matching crop/content/scale metadata, expected buffer dimensions, a positive nonregressing display timestamp no later than receipt, and unchanged independent target geometry at receipt. Rejected frames retain diagnostic metadata without image output; the first failure invalidates the trial. Missing frames, stream failures, zero complete frames and changed post-stop geometry cannot produce success. Receipt checks do not prove past window visibility or physical scanout. Real pixel alignment and an owned-occluder comparison against the frozen window stream must qualify the method before any cohort.

A full marker acknowledgment is written after journal synchronization and precedes CUA input. Every frame remains `eligible_for_latency=false`; reports retain `eligible_latency_observations=0`, null gate outcome and uncalibrated visibility/pixels. The tool does not identify causal app frames, compute p95 or pass a target. External analysis must retain every attempted trial and apply the original PRD thresholds and predeclared cohorts. No supported-alpha, physical-clock, typing/IME/accessibility or owner acceptance is inferred.
