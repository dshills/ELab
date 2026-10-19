# Visible-latency recorder preparation · October 19, 2026

**Offline checks passed.** The standalone [Swift recorder](../../../tools/frame-probe/README.md) compiled in Swift 6 with warnings as errors and passed pure checks in 0.899 s. This compilation/pure-check invocation observed no window, performed no capture, operated no input and requested no OS permission. The subsequent [authorized runtime calibration](../frame-probe-calibration/README.md) is recorded separately. Eligible latency observations remain **0**; the gate decision is **null**. The utility is outside ELab's runtime and sealed review package.

[Checkpoint](checkpoint.json) pins the three source files, compiled development tool, installed SDK header, compiler output and current /165 review identity. [Check output](check-stdout.txt) records the successful direct invocation; stderr is empty. [Producer source](record-preparation.py.txt) preserves how those observations were recorded. The first unsuccessful pure path check is [retained separately](initial-location-check.txt): Foundation abbreviated existing `/private/tmp` to `/tmp`, causing the original prefix comparison to reject an allowed child. POSIX ancestor resolution corrected that check. Its wrapper's final status did not establish the failed check's exit code, so that earlier exit remains unknown.

The final pure checks cover argument rejection, marker syntax and precise ticks, Mach-time arithmetic, physical temporary-path boundaries, application/symlink exclusion, refusal to reuse a directory and refusal to overwrite a marker acknowledgment. They do not exercise ScreenCaptureKit, stream teardown, callback limits, visibility, image encoding or permission behavior. The macOS 14 deployment setting is not OS compatibility evidence.

CUA's documented snapshot/input APIs were inspected and expose no event/frame timestamps. Its computer-use rules require an explicit user request before another capture technology is used. The owner subsequently authorized the recorder method; the [decision record](../../../research/ui-audit/first-version/private-alpha-owner-decisions.json) retains the exact reply. Any required OS privacy permission and a safe isolated target session are separate prerequisites; this tool never requests permission itself. The owner separately [deferred ordinary default-startup/clean-account qualification](../../../research/ui-audit/first-version/private-alpha-owner-decisions.json) for the private alpha. That decision does not authorize this recorder or changing shared preferences.

The installed `SCStream.h` header documents `SCStreamFrameInfoDisplayTime` in the Mach clock domain; the checkpoint retains the header digest. Apple's [display-time reference](https://developer.apple.com/documentation/screencapturekit/scstreamframeinfo/displaytime) and [capture guide](https://developer.apple.com/documentation/screencapturekit/capturing-screen-content-in-macos) are linked for review. The browser returned JavaScript shells and could not fetch their Markdown variants; the inspected local SDK header is the retained semantic evidence. The nominal calibration checks timestamp arithmetic; physical clock interpretation remains unvalidated. Marker-to-frame intervals precede input and must not be labeled exact event latency. Window-content capture and receipt-time on-screen membership do not prove unobscured display visibility at the frame timestamp.

The initial nominal capture is complete, but input ordering and a causal/visibility annotation protocol are still needed before collecting the PRD's reference cohorts. No p95, first-visible-sample timing, human-study result, accessibility acceptance, clean-account pass or owner walkthrough acceptance follows from compilation.

Read-only integrity check:

```sh
python3 -B research/ui-audit/first-version/verify-frame-probe-preparation.py
```

This checks retained bytes and preparation facts; it neither records UI nor reruns the recorder.
