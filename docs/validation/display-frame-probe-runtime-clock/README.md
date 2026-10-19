# Cropped recorder: native timestamps and nominal output

October 19, 2026. The owner-authorized display crop now produces nominal raw images on the qualified /173 app. Full edge-marker, owned-occluder, causal-input and timing-cohort calibration remains pending. **Zero eligible latency observations; no p95 decision.** No application/Go/SDK/model/format change or new app build is made.

A first protected session fails before recording because a copied diagnostic helper lacks executable mode. It closes normally, with no capture. The second session independently establishes the owned process, one window, display 3 and the entire 1053×802 window rectangle at global/display-local (2034,303) on the 5120×1440, 1× display. A deliberately smaller crop entirely inside that rectangle is refused before output creation or stream admission.

That session's three-second capture then rejects its first complete frame before encoding any image. The supplied display timestamp is **8.978333 ms ahead of callback receipt**. Independent rational replay finds that its presentation timestamp equals the converted Mach display time exactly. The old collector's `displayed <= received` assumption causes refusal. This observation does not establish the scheduling cause, another clock epoch, physical presentation or an application defect. [Apple's displayTime reference](https://developer.apple.com/documentation/screencapturekit/scstreamframeinfo/displaytime) defines a window-server frame time; it does not establish our callback-ordering assumption.

The raw collector now preserves future-at-receipt values and their original receipt times. It requires exact agreement with CoreMedia's host-time conversion of the supplied display ticks, and retains explicit lead diagnostics. This is a representation check for the selected configuration, not clock/visibility acceptance. Positive/nonregressing timestamps, independent target/frame geometry, scales, dimensions, first-error and storage guards remain. A prospective frame beyond the predeclared presentation interval keeps a diagnostic row without pixels; it cannot become an observation. Receipt callbacks after the fixed deadline remain uninspected. There is no arbitrary future-time allowance or rewritten timestamp, and every frame stays ineligible.

**55 offline checks** pass with Swift 6 warnings as errors and strict formatting. They include a preserved future timestamp, a mismatched presentation clock, nonregression and a prospective end-boundary row with no image. The original 53-check source checkpoint, its exact six Swift snapshots and initial binary are preserved. Its live-source verifier correctly rejects changed code; this new proof binds the current source and preserved prior bytes rather than weakening that guard.

A third fresh protected session uses recorder `a8cec4e` and the exact same /173 app bytes. Its three-second output interval yields **164 complete timestamped frames and 115 unique PNGs**, recorder exit 0, no collector/close error and unchanged post-stop geometry. All 164 presentation values equal the independent nanosecond host conversion; **143 display times are ahead of receipt**, by 0.176792–11.614292 ms. These are timestamp diagnostics, not product input latencies. No edit or acknowledged input marker occurs. CUA only raises the owned window.

Every original PNG is retained losslessly in `nominal/frames.tar.xz`: 15,623,335 original bytes, 115 separate identities/modes, compressed to about 2 MB. No image was edited or selected away. First/last images are directly viewable; the full journal retains repeated frames. All changes between unique images lie within a 16×16 cursor-shaped region at (521,396)–(537,412). The stream requests `showsCursor=false`; effective cursor suppression is not established. The five sampled static UI features best align at zero offset against an independent CUA window screenshot. This supports nominal alignment only; it is not the required edge-marker/scale/occlusion calibration. Image-analysis source and results are retained.

The displayed 1 kΩ LED circuit and initial/final live graph/results match a separate protected packaged agent. The older producer's unused 2 kΩ expectation file remains historical setup data, not an applied edit. All three sessions close through completed worker cleanup, `open -W` returns 0, their owned PIDs/endpoints disappear and no forced cleanup occurs. Native process exit status remains unavailable under LaunchServices. Shared geometry, sealed/copy/canonical apps and SDK remain unchanged; /173's package verifier still passes.

Future callback ordering makes receipt time unsuitable as a substitute for visible display time. Actual timestamp/visibility interpretation, owned edge markers/occluder, acknowledged causal edits and predeclared LED/RC/NPN cohorts remain required. Photons, frame-drop rates, other hardware/modes, physical typing/IME/accessibility, owner acceptance and the actual study are not inferred. None of these runs changes a PRD target or promotes the app.

Read-only source/data/byte replay, including three rejected evidence mutations:

```sh
python3 -B docs/validation/display-frame-probe-runtime-clock/verify.py
```

[Measurement decision](../../../research/36-visible-latency-method.md), [tool](../../../tools/display-frame-probe/README.md), [qualified /173](../sweep-review-173/README.md).
