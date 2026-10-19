# Stable input-session preflight · October 19, 2026

**The protected session runs, but this launch route cannot support the planned CUA input/capture trial.** Two exact-/165 sessions use private recovery/appearance paths and a sandbox denying reads/writes to the current account's shared `Library/Cogent Core` directory. Before either launch, a non-destructive canary confirms read and write-only-open denial (errno 1); no truncating or creating open is attempted. Existing geometry file hashes and modes match before/after both sessions. Both native processes close normally through the shared public `app.close` command, endpoints disappear and sealed review files verify unchanged.

AppKit finds each exact native PID/executable but reports a **null bundle identifier**. CUA cannot bind the first session by the qualified app path (`Running application not found: com.dshills.elab`), and the recorder's exact-target check refuses it. No recorder guard is relaxed. The second control changes only the test profile's outbound-network rule; its bundle identifier is also null. Outbound-IP denial alone does not explain the observation. The root cause is not established, and this is not evidence that the ordinary app's UI or simulation is broken.

[Checkpoint](checkpoint.json) retains the scopes and results. [First registration observation](network-denied/registration-observation.json) is explicitly transcribed from the completed diagnostic tool result; the [control's raw registration output](network-default/registration-stdout.txt.gz) is retained losslessly. The [small diagnostic source](target-diagnostic.swift.txt) reads only the provided owned PID's AppKit registration fields. It was compiled in Swift 6 with warnings as errors and exited 0 for both observations; it neither captures images nor operates input.

The executed [network-denied controller](session-network-denied.py.txt) and [default-network control](session-network-default.py.txt) are retained with their profiles and normal-close reports. They seed the LED through public commands and prepare a separate exact-agent 2 kΩ reference. Native baseline/final graphs and readings stay at 1 kΩ /2.916796577 mA; the unused reference predicts 1.476723549 mA. **No test edit was operated and no frame was recorded.** That prepared result is not a user/agent input-parity or timing observation. There is no new eligible cohort: count **0**, latency decision **null**.

Only bounded evidence files are copied here; private endpoint credentials and recovery-store internals are omitted. Full synthetic temporary artifacts remain local. Raw logs/large snapshots are compressed losslessly. The session reports preserve the expected framework permission-denial logs rather than classifying them as an ELab defect. All matching native test processes were absent after cleanup; no forced termination was used.

This narrows the next action: another input-timing attempt needs a documented launch/isolation method that preserves both target identity and owner settings, verified before input. Do not repeatedly launch the same failing setup, loosen recorder ownership checks, or infer timing from API solves. The earlier [nominal smoke capture](../frame-probe-calibration/README.md) remains valid in its limited scope. The owner’s [default-startup/clean-account deferral](../../../research/ui-audit/first-version/private-alpha-owner-decisions.json) remains in force; this preflight is not a clean-account pass or permission to change shared settings. Continue other bounded release work while timing remains unqualified.

Read-only verification:

```sh
python3 -B research/ui-audit/first-version/verify-stable-frame-input-preflight.py
```
