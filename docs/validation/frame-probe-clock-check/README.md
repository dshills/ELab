# CoreMedia host/Mach clock-domain check

**All 200 CoreMedia host-clock readings lie within their surrounding converted Mach brackets.** Four fixed/current-tick conversions round-trip within one tick and have at most 0.667 ns observed error in seconds. Swift 6 warnings-as-errors compilation, help and invalid-argument checks pass. No app, window, capture, image, privacy or input API is used; the standalone utility imports only CoreMedia, Darwin and Foundation. The calibrated recorder is unchanged.

This resolves a software representation question, not the visible-latency gate. Installed `CMSync.h` documents CoreMedia's host clock in the Mach domain; ScreenCaptureKit's `SCStream.h` describes frame display timestamps in that domain. The header hashes and exact toolchain/OS information are retained without copying the SDK. These runtime checks are local consistency evidence, not physical display/scanout calibration or a guarantee for another OS/device.

The initial raw-tick assertion failed. A diagnostic revision retains 55/200 reconstructed ticks outside raw brackets. Its one-tick example is consistent with flooring twice across 125/3 ns Mach ticks and integer nanoseconds. The passing revision compares `CMTime` endpoints with `CMTimeCompare`, avoiding that lossy reconstruction. It records the raw-tick test as well: **106/200** remain outside in that execution; they are not discarded or relabeled as directly bracketed ticks. Python's independent rational-arithmetic replay verifies the native comparisons. No arbitrary time allowance is added to the bracket comparison, and neither the 100 ms nor 500 ms product target changes.

Both failed versions, their source/logs and the passing report are retained. A replay check rejects a one-second shifted host reading and a forged latency-eligibility flag. The helper's conversion check separately permits at most one reconstructed tick and <10 ns seconds error; the observed error is smaller. This tolerance is not used for the bracket comparison or the product timing thresholds.

Apple's single-window filter can still return target pixels while another window covers them. Thus a shared software clock is insufficient for visible latency. [Apple's filter explanation](https://developer.apple.com/videos/play/wwdc2022/10155/) supports the [separate method decision](../../../research/36-visible-latency-method.md): a tightly cropped display-composition capture could address that visibility gap, but can include overlapping other-app content and requires new authorization before use. No composition capture or wider permission is performed here.

This is instrumentation outside the app/agent runtime. Current Go sources, /171 app/agent/ZIP, SDK and the calibrated recorder bytes are unchanged. There are **0 eligible latency observations**, no p95 decision, no new package or promotion. Pixel mapping, composition visibility, actual capture delivery/fault behavior and fixed LED/RC/NPN cohorts remain open. Do not infer full physical-clock qualification from this check.

Re-run the actual clock-only check:

```sh
sh tools/frame-probe-clock/check.sh
```

Replay retained evidence and exact rational comparisons, without invoking native clock or capture APIs:

```sh
python3 -B docs/validation/frame-probe-clock-check/verify.py
```
