# Visible latency: measurement decision

October 19, 2026. Status: software clock check completed; a broader capture method is proposed, not authorized or executed.

The PRD retains two different targets: input event → visible acknowledgment at p95 ≤100 ms, and accepted edit → first valid updated sample at p95 ≤500 ms for LED, RC and resistor-loaded NPN. Backend timing and a draft appearing in an editor cannot establish either target. No target, reference circuit or acceptance rule changes here.

## What the current recorder can prove

The [latest nominal check](../docs/validation/frame-probe-sample-runtime/README.md) binds one owned /171 window, a before-input marker, one CUA Apply, accepted public physics and timestamped window-content images. Its 186.354 ms marker-to-fresh-frame interval is not an eligible observation.

The missing visibility evidence is structural. Apple's single-window filter supplies the window's content even when other windows cover it. Therefore receiving a complete frame and finding the target in an on-screen list cannot establish that a person could see those pixels at its display timestamp. Repeating this capture mode cannot resolve that gap. [Apple WWDC22, single-window and display-filter discussion](https://developer.apple.com/videos/play/wwdc2022/10155/).

`displayTime` is documented in the installed ScreenCaptureKit SDK as a Mach absolute timestamp for the Window Server's frame event. CoreMedia's installed `CMSync.h` documents its host clock and conversion functions in that domain. A new [clock-only runtime check](../docs/validation/frame-probe-clock-check/README.md) compares 200 host-clock readings with surrounding Mach readings, using exact host-time brackets. All pass. Four conversions have at most one reconstructed tick of loss and less than 1 ns observed seconds error. This supports the software clock-domain interpretation on this execution; it does not measure physical display scanout or visibility.

The first check incorrectly required reconstructed integer ticks to fall strictly inside raw Mach brackets. It failed: converting ticks to integer nanoseconds and back can floor twice. The corrected check compares the captured host `CMTime` against the two converted bracket endpoints directly, with no arbitrary time allowance. The failed reports remain evidence; all out-of-raw-bracket observations remain recorded in the passing report.

Pixel mapping also needs independent calibration. A content rectangle, content scale, point/pixel scale and output dimensions describe different transforms. The next method must retain all of them numerically and validate alignment against known owned-window pixels. Matching PNG dimensions alone is insufficient. [Apple content-filter geometry reference](https://developer.apple.com/documentation/screencapturekit/sccontentfilter/contentrect), [frame metadata sample](https://developer.apple.com/documentation/screencapturekit/capturing-screen-content-in-macos).

## Proposed next action: cropped display-composition capture

Request authorization for a separate ScreenCaptureKit display-composition mode, cropped **before output** to the independently verified owned ELab test window's rectangle. Keep the current window-only recorder and its evidence intact. The crop would be fixed for a trial and stay entirely within one display. No whole-display image output, audio/microphone, global input listener, alternative input injector, upload or privacy-permission request is proposed.

This differs from the authorized window-only filter: another app's window or notification overlapping that rectangle could appear in the locally saved images. A moving/hidden/replaced target must invalidate the trial; cropping does not remove that privacy risk. This broader set of possible pixels requires separate owner authorization before any such capture. Existing Screen Recording access is not consent to use the new scope.

The proposed bounds are 0.1–5 seconds per trial, ≤60 requested frames/s, ≤300 callbacks, ≤4096×4096 output pixels and ≤64 MiB unique image bytes, matching the existing instrumentation caps. Use new private temporary directories with 0700/0600 modes and no overwrites. Apply only CUA input to an independently launched, hash/PID/window-verified synthetic ELab session whose sandbox denies shared geometry and repository/toolchain/network access. Any newly needed macOS privacy grant remains a separate action-time decision.

Implement and check these guards before executing the new capture:

1. Validate finite, nonempty, integral geometry; reject partial/off-display/cross-display targets, scale changes, movement, ambiguous identity, oversized output, missing metadata and any unspecified/empty crop. No fallback to an uncropped display or another window.
2. Calibrate crop origin and scale against multiple known points in the owned window, including edge markers; reject clipping, rescaling or coordinate-space ambiguity. Test a deliberately wrong crop without storing pixels outside the approved region.
3. Test an owned occluder: the composition must show the occlusion while the existing window-only stream still contains the target. This demonstrates why the two observations have different visibility meaning. No other application needs to be operated.
4. Retain source/configuration hashes, crop/display identities, timestamps, complete/incomplete statuses and every attempted trial. Reject mismatched clock/geometry, absent images or stream errors. Do not equate a callback gap with proof of a missed display update.
5. Repeat the marker → acknowledged CUA action → causal acknowledgment/fresh-image sequence, preserving explicit stale labels and matching public agent physics. Public API state alone cannot identify the visible transition.

These are proposed checks, not existing test results. Authorization would permit preparing and executing this method within those bounds; it would not qualify its output automatically.

## How a later timing decision would work

Predeclare at least 20 trials per LED/RC/NPN reference on one recorded build/OS/device/model configuration. Record every attempted trial and reason for any unavailable measurement. Do not repeat until a favorable timing appears or silently remove slow/inconclusive samples.

An acknowledged marker precedes the CUA input and accepted edit. A validated causal composition frame after that marker could therefore supply a conservative upper bound for either interval. A bound below the target can support a pass within that method's validated scope; a bound above it is inconclusive about the product because recorder/controller overhead is included. It is not proof of a product failure. Actual event/acceptance timestamps would be preferable if a supported native input path can supply them without new monitoring access.

Use a declared percentile convention and retain raw values; do not report a p95 from only successful or selected trials. Difficult-solve navigation/cancel/save/undo remains separate required coverage. Photons, other display modes, monitor changes, human typing/IME, accessibility, owner acceptance and the hobbyist study are not inferred from screen-capture measurements.

If the broader capture method is declined, retain window-only captures as functional/render evidence and keep visible latency unqualified. Independent release work can continue; the private-alpha environment deferral does not waive this PRD target.
