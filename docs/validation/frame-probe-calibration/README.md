# Owned-window recorder calibration · October 19, 2026

**Nominal capture works; visible latency remains unmeasured.** After the owner authorized the recorder method, the exact qualified /165 executable ran in its existing isolated smoke mode. Screen access was already available; no OS permission was requested. The two-second recording received **103 complete timestamped frames**, stored **18 distinct 1276×1146 PNGs** (3,698,345 bytes), acknowledged one marker and stopped normally. The app completed its smoke session and exited 0, its endpoint disappeared, and all sealed review files verified unchanged. All three owned test PIDs are absent.

The [checkpoint](checkpoint.json), [passing controller report](passing/report.json), [capture report](passing/capture/report.json) and [frame journal](passing/capture/frames.jsonl.gz) retain the observations. The [controller source](calibrate.py.txt) matches the passing run and the second attempt's readiness/cleanup logic. It does not reproduce the first attempt's original immediate-window assertion. Runtime capture output had a 0700 directory and 0600 files; these observed modes are retained in the checkpoint because Git does not preserve private permission bits. Raw text logs/journal are compressed losslessly. Disposable endpoint descriptors from stopped failed attempts are omitted from this repository copy; hashes/reasons are recorded without their authentication data. Original temporary artifacts remain local.

The display times increase strictly and precede receipt times. Their conversion using the retained 125/3 Mach timebase agrees with recorded presentation seconds within 6.985e-10 s. First-to-last display span is **2.1311245 s**, illustrating that the requested two-second sleep is not an exact stream interval. These arithmetic observations do not calibrate physical pixels, prove exact frame delivery frequency or establish an input event clock.

The marker acknowledgment follows six frames, and 97 subsequent frames include a marker interval. **No test input was operated.** The existing smoke harness changed synthetic circuits itself; this is not a user-edit trial. The first and last retained images were inspected and show the synthetic LED/refusal and flasher/scope states. CUA also observed the owned smoke window independently. Window-content capture does not establish unobscured display visibility at the recorded timestamp. Every frame remains `eligible_for_latency: false`; eligible observations are **0** and the latency-gate decision is **null**.

Two failures are preserved:

- [First readiness attempt](first-readiness/report.json): the local endpoint existed before an on-screen window did. The original controller refused the empty result and terminated its owned native process. No recording started; its native exit code was not retained. A later process check confirms absence. The corrected controller waits for exactly one verified owned window and refuses ambiguity.
- [Filter startup attempt](filter-crash/report.json): the recorder aborted with signal 6 before opening a journal. The [selected crash stack](filter-crash-stack-excerpt.json) identifies `SCContentFilter.init(desktopIndependentWindow:)` and the graphics-initialization assertion; unrelated diagnostic metadata is omitted. The controller terminated only its owned app (−15). Initializing AppKit before constructing the filter resolves this startup failure in the successful retry. This is an instrumentation failure, not an ELab simulation defect.

Normal capture/PNG writing, timestamp serialization, marker acknowledgment and teardown have one bounded observation. Input-to-marker ordering with CUA, causal acknowledgment/result frames, pixel scale against the display, physical clock interpretation, occlusion, idle/blank/dropped frames, stream errors and resource-limit paths still require calibration. At least 20 eligible observations for each LED, RC and resistor-loaded NPN reference remain outstanding. No p95 or responsiveness pass follows from this run.

Smoke mode uses private recovery/appearance directories and disables window-geometry saving. It does not qualify ordinary startup or the [clean-account gate deferred for this private alpha](../../../research/ui-audit/first-version/private-alpha-owner-decisions.json). No owner work, app version, SDK, device model, public command or sealed review was changed.

Read-only verification:

```sh
python3 -B research/ui-audit/first-version/verify-frame-probe-calibration.py
```
