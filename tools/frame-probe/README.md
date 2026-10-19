# Window frame probe — qualification instrumentation

This standalone macOS qualification utility is **not part of ELab's runtime or release bundle**. It is intended to collect calibration evidence for the existing 100 ms visible acknowledgment and 500 ms first-valid-sample targets. It does not calculate a p95, classify a frame as eligible, or pass a latency gate. Offline checks and one authorized window-capture calibration are recorded separately; neither supplies eligible latency observations.

CUA's documented native observation API returns snapshots without input/frame timestamps and waits before observation. It is suitable for functional verification, not the missing timestamped cohort. Its computer-use rules require an explicit user request before another technology is used for computer interaction. Compiling this tool and running its pure checks do not read UI; **running capture requires separate human authorization for this method and the isolated target session**. No such authorization is inferred from this document or a CLI flag.

The owner authorized this recorder method on October 19. That authorization does not grant OS privacy access or permit changes to shared preferences. `--preflight` reads existing Screen Recording access without requesting it, enumerating windows or recording; unlike `--self-test`, it is an OS access check.

A [protected non-smoke launch preflight](../../docs/validation/latency-launch-preflight-170/README.md) now preserves exact /170 identity, one owned window and CUA binding: a temporary LaunchServices launcher explicitly execs into the verified sandbox before starting ELab. The test closes normally with unchanged shared geometry. This clears that launch/binding prerequisite only; it supplies no input, recording or latency observation. Use fresh paths/PID/window and repeat the guards for any subsequent session; never reuse the stopped binding.

The subsequent [one-LED input/frame calibration](../../docs/validation/marker-input-calibration-170/README.md) validates a complete marker acknowledgment before one CUA Apply action and identifies the resulting schematic/readout image. Arm the recorder in the context with existing access, then start its short stream from a filesystem signal issued by the already-running CUA controller; a model/tool round trip consumed the earlier recording window. The Node-host recorder preflight returned access unavailable, so that context was not used for capture and no permission was requested. These observations do not qualify visibility, physical clock/pixels, fault paths or a latency cohort.

## Check without recording

```sh
sh tools/frame-probe/check.sh
build/frame-probe-tools/frame-probe --help
```

The check compiles with Swift 6 and warnings as errors, then exercises argument rejection, marker IDs, Mach-time conversion, precise tick serialization and refusal to overwrite an acknowledgment or existing output directory. It does not query windows, preflight/request screen permission, capture images, monitor input, open settings or operate ELab. An arm64/macOS 14 deployment target is a compiler setting, not a claim of tested OS support.

## Capture contract — do not execute without authorization

After the owner provides an isolated test session and specifically authorizes this recorder, use an independently owned ELab instance with synthetic test circuits. Obtain the exact PID from an owned test launch, then use `--target-windows --pid PID --executable /path/ELab --sha256 HASH` to report only that verified process’s on-screen window IDs. Wait for a window to become ready; refuse ambiguous targets. Never select the first app/window or guess an ID. Require its exact executable path and SHA-256 from the qualified review.

```sh
# Placeholders only; this is not a command to run during preparation.
build/frame-probe-tools/frame-probe --capture \
  --pid OWNED_PID --window OWNED_WINDOW_ID \
  --executable /absolute/path/to/ELab.app/Contents/MacOS/ELab \
  --sha256 QUALIFIED_BINARY_SHA256 \
  --output /private/tmp/new-private-capture-directory --seconds 2
```

The intended guards require the `com.dshills.elab` bundle ID, exact process/executable hash, matching on-screen window and a new output directory. Screen access is preflighted; if unavailable, the tool fails **without requesting it or opening settings**. Any OS privacy grant is a separate action-time decision. No software installation, signing operation or change to ELab preferences is performed by the recorder. The ordinary-startup [account/isolation prerequisite](../../docs/validation/startup-handoff-preflight-165/README.md) remains open.

Only a window-specific ScreenCaptureKit filter is configured. Audio/microphone/child-window capture is disabled, and the tool has no input injection or keystroke listener. Selecting a target requires reading OS shareable-window/process metadata; metadata for unrelated windows is not written. A frame records only whether the target appears in the current on-screen window list. Screen Recording permission itself may be broader than this filter, so it is not a privacy permission to grant silently.

Duration is bounded to 0.1–5 s, at up to 60 requested frames/s, 300 callbacks, a 4096×4096 native-pixel limit, and 64 MiB of unique PNGs. Identical PNG bytes share one file, while every received frame keeps its journal entry. Hitting a frame/image budget is an incomplete recording, not a successful truncated cohort. Output must be a new subdirectory of `/private/tmp` or the current OS temporary directory, outside all app bundles; existing destinations are refused. It is created with mode 0700, with 0600 output files. No network or telemetry code is used. These are source-level intended controls; one short nominal capture has been calibrated; limits, fault paths and visibility remain unqualified.

## Timing and marker data

For each callback, `frames.jsonl` records frame status, Window Server display time in raw Mach ticks, receipt time, presentation time when finite, scale/content-rectangle metadata, and the PNG/hash when a complete timestamped image is available. Raw ticks are decimal strings to preserve precision outside Swift. The timebase ratio is retained in `report.json`.

Apple documents `.displayTime` as the Window Server's frame-display time; [API reference](https://developer.apple.com/documentation/screencapturekit/scstreamframeinfo/displaytime). The installed SDK header was also inspected. The chosen window filter and complete-frame handling follow the public [ScreenCaptureKit API](https://developer.apple.com/documentation/screencapturekit/capturing-screen-content-in-macos). These properties are a starting point for calibration, not proof of physical display timing or unobscured visibility.

An optional **before-input marker** uses filesystem handshaking, avoiding a global input monitor:

1. The authorized CUA controller atomically replaces `marker.request` in the private output directory with a new ASCII ID (letters/digits/underscore/hyphen, 1–32 bytes, no newline).
2. The recorder journals its own Mach timestamp, flushes that journal and writes `marker-ID.json` without overwriting an earlier acknowledgment.
3. The controller must read/parse the complete acknowledgment, verify its ID, then invoke the intended input **through CUA**. The controller retains its action/acknowledgment sequence as evidence. Never treat file existence alone as acknowledgment.
4. The controller observes state afterward under the ordinary CUA rules. It must not send another edit until the trial's intended accepted value/run and visible result have been identified.

This timestamp precedes the input; it is **not the input event or command-acceptance timestamp**. `since_acknowledged_marker_ms` is only a marker-to-frame interval. It could provide a conservative latency bound if ordering, causal frame identity, common clock, visibility and capture completeness are validated. Do not relabel it as exact latency or infer a product failure when recorder/controller overhead makes a bound too large. No such bound is presently accepted as a qualifying observation.

Bind and prepare the input controller before starting a short recording; do not spend its bounded duration waiting for a model/tool round trip to choose an action. The recording/marker/input sequence still needs an authorized calibration before it is used for a cohort.

## Work still required before a cohort

- The owner authorized the alternate capture method; keep a safely isolated ELab test session and handle any new OS permission separately.
- Extend the [initial calibration](../../docs/validation/frame-probe-calibration/README.md) beyond its exact window/hash, nominal PNG/timestamp/marker and normal-teardown observations: validate pixel scale/clock interpretation, stream errors, dropped/idle/blank frames and output limits.
- A visibility protocol: this captures **window contents**, which can differ from an unobscured display composition. On-screen membership sampled at receipt does not establish visibility at the earlier display timestamp. Occlusion, window movement, display changes and missing timestamp/image cases remain ineligible until accounted for.
- An outcome/annotation protocol tied to accepted edits and specific visible acknowledgment/first-valid-result frames. Use changing component/readout values, not a private text draft that could be visible before Apply.
- At least 20 eligible observations per LED, RC and resistor-loaded NPN reference, with locked build/models/device and raw evidence. Flasher/difficult-solve behavior remains separate stress coverage.

The recorder always emits `eligible_latency_observations: 0` and `latency_gate_passed: null`. Prepared source, compilation and pure tests do not close any visible-latency, clean-account, accessibility, distribution or owner-acceptance gate. [Offline evidence](../../docs/validation/frame-probe-preparation/README.md), [authorized calibration](../../docs/validation/frame-probe-calibration/README.md).
