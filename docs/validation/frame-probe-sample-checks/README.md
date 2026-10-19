# Recorder sample, storage and error checks

Eight new offline checks pass alongside all seven existing journal/marker cases. The recorder is qualification instrumentation, outside ELab's app and agent runtime. No screen capture, window query, privacy preflight/request, input, app launch or shared preference change occurs in this work.

The real collector's sample handler and PNG storage are factored into queue-confined helpers. Production still obtains its receipt-time window metadata at the same point; tests supply explicit metadata and synthetic CoreMedia buffers. Complete 4×3 pixel images are encoded with the production CoreImage/ImageIO path and decoded independently to verify their dimensions and hashes. Repeated pixels share one file while keeping every frame row; changed pixels produce a different image. Files retain 0600 permissions and every synthetic row remains ineligible.

The new checks exercise:

1. Idle, blank, suspended, started and stopped statuses retain rows without PNGs.
2. Missing status, missing/zero display time and missing image cannot invent complete images.
3. Complete-image encoding, decoding, digest/deduplication and explicit receipt metadata.
4. Sample/error callbacks after finalization cannot change the journal or returned report.
5. An existing image path survives the refused write; no successful image row is invented.
6. Callback 301 is refused at the actual 300-callback limit; a secondary error cannot hide the budget failure.
7. The actual 64 MiB unique-byte boundary accepts an at-limit duplicate and refuses the next image without changing prior bytes. The boundary is primed with a synthetic byte payload, explicitly **not a valid or captured PNG**; a real synthetic image then exercises the handler's failure path.
8. The first queued stream error stops admission and survives a secondary error.

The stream delegate previously assigned its error unconditionally. That could hide an earlier frame/storage failure. It now ignores secondary failures and callbacks after finalization. A comparison variant restores only that prior assignment policy in the current factored source; it fails the new frame-limit/first-cause regression. This is a targeted policy comparison, **not a replay of the original executable**. The original source and variant are retained for inspection. Successful image storage, queue ownership, marker protocol, clock arithmetic, duration/pixel limits, frame/byte limits and output schemas are otherwise unchanged; self-test JSON adds the case names.

`sh tools/frame-probe/check.sh` compiles all three Swift files in Swift 6 mode with warnings as errors and passes all 15 fault cases plus the existing argument/marker/timebase checks. The initial restricted-shell attempt failed with CoreVideo surface diagnostics and `Complete image accounting failed`; it is preserved as a failure, not an inferred pass. The final check runs with macOS image-surface access and no capture/privacy API. No new OS permission was requested. The compiler reports Apple Swift 6.4; the arm64/macOS 14 deployment setting does not qualify that OS release.

The new recorder hash is `dcf06ebefa9ec76002f22f0e67564682b1c429af2f19fc208c504c669ec505e7`. The previously nominally calibrated `677c5e…` recorder is preserved byte-identically, and its 175-frame data still replay. Historical source-admission guards must reject the new bytes rather than inherit the old runtime qualification. **Fresh nominal runtime calibration remains required.** Actual OS stream termination/delivery, callback gaps/dropped frames, physical clock/pixels, unobscured visibility and LED/RC/NPN cohorts remain open. There are **0 eligible latency observations**, no p95 decision and no broader responsiveness claim.

No Go application source, SDK, model, command or project format changes. The existing 647-source quality checkpoint still verifies; the /171 review manifest, its ZIP and canonical /120 bytes are unchanged. No app build, package, promotion or owner acceptance occurs.

`checkpoint.json` binds the sources, compiled recorder hashes, passing/failing logs and preservation checks. Text/source variants are compressed losslessly. Read-only evidence verification:

```sh
python3 -B docs/validation/frame-probe-sample-checks/verify.py
```

That verifies retained data and current instrumentation sources, without rerunning synthetic encoding or reading any UI. Re-run `sh tools/frame-probe/check.sh` in a context with macOS image-surface access for actual offline execution. Its synthetic files are removed on completion; no screenshot is produced.
