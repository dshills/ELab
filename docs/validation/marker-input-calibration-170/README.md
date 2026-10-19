# One LED marker → CUA Apply → causal image calibration

**The coordinated sequence works once; visible latency remains unqualified.** The exact protected /170 app starts with R1 = 1 kΩ and 2.916796577 mA. CUA types private `2k`; a public inspection confirms accepted physics remain at 1 kΩ. The prepared controller starts the armed recorder, reads and validates the complete `led_apply_terminal01` acknowledgment, then clicks Apply once through CUA. It observes 1.477 mA afterward. No global input monitor or alternative input injector is used.

The three-second recorder exits 0 with **174 complete timestamped frames**, four unique **1053×802 PNGs**, one marker and no collector/command error. All four images were inspected. Frames 1–9 still show schematic 1 kΩ /2.917 mA, despite private text `2k`. Frames 10–11 show a dimmed Apply button with the same old graph/readout; that is not an accepted-result image. **Frame 12** first shows both schematic **2 kΩ** and **1.477 mA**, along with 2.047 V and 3.02 mW. The remaining alternate image changes caret visibility while retaining those values. The pixel annotations are explicit manual observations, not an OCR classifier.

The marker-to-frame-12 interval is **153.012708 ms** using the retained 125/3 Mach timebase. It is **not** an input-event or accepted-edit latency, a qualifying upper bound, a performance pass/failure, or a p95. Display times increase strictly and precede receipt times; presentation/Mach arithmetic is internally consistent. Scale metadata is 1 and the content rectangle matches the PNG dimensions; physical pixel/clock calibration is still open. On-screen-stack membership at receipt does not prove unobscured visibility at the earlier display time. Every frame remains ineligible: **0 eligible observations**, latency decision **null**.

Public after-state matches the separate packaged-agent 2 kΩ graph and complete LED result exactly: **1.476723549 mA**, with one electrical revision and one run increment. The prior manual `baseline.elab` remains byte-identical; a separate `final.elab` is saved. Public close reports completed recovery/worker cleanup and exit readiness. `open -W` exits 0; the native endpoint and all four owned native/recorder PIDs are absent. **Native process exit status is unavailable**, since LaunchServices owns it. Shared geometry, all 1,651 sealed review entries, copied/canonical app and SDK remain unchanged; 312 runtime inputs outside conformance tooling still match /170. No app rebuild, version increment, user project, system setting or privacy grant change.

Two failed orchestration attempts remain separate:

- **Node-host attempt:** recorder preflight reports Screen Recording access unavailable without requesting it. The controller also throws because the Node host has no global `process` object. No image or Apply occurs. Its child exit status was not retained; stderr and later PID absence are retained. The replacement controller uses ordered events/wall timestamps for sequencing only.
- **Expired terminal recording:** existing access is available and the recorder exits 0 after 173 callbacks/172 complete frames/two PNGs. The marker request arrives after the completed report (original filesystem times retained); there is no acknowledgment or Apply. This is a coordination failure, not a recording or app failure. The successful worker waits for a CUA start signal before opening its three-second stream, removing the model/tool round trip from that interval. The controller then validates the full acknowledgment before input. No privacy access was newly granted.

`capture/` retains original successful images, marker and journal; failed records/images live under `attempt-node/` and `attempt-terminal-expired/`. Sources, exact controller sequence, before/after/reference snapshots, saves, cleanup and postflight are bound by `checkpoint.json`. Large text is losslessly compressed; endpoint descriptors, credentials and recovery internals are excluded. The producer retains its readiness status alongside explicit terminal cleanup flags. All bindings/PIDs/windows/endpoints here are stopped and must not be reused.

Read-only replay recomputes journal/image/timing/state checks against the retained manual annotations:

```sh
python3 docs/validation/marker-input-calibration-170/analyze.py \
  --artifacts docs/validation/marker-input-calibration-170 --verify
```

Remaining: unobscured-display visibility, physical clock/pixel interpretation, stream-error/idle/blank/dropped-frame and budget calibration, then at least 20 eligible LED/RC/resistor-loaded-NPN observations plus difficult-solve interactions. This calibrates one causal sequence and does not close those gates, ordinary/default-startup qualification, the owner walkthrough, accessibility/IME or broader release acceptance.
