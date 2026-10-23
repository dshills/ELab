# Using the first version

October 23, 2026. [ELab 0.181.3 /204](../build/releases/ELab-0.181.3-macos-arm64/ELab.app)
is the current unsigned private macOS release. [Handoff and evidence](79-first-version-release-closeout.md).
Your review remains optional and deferred; no review response is needed for
engineering completion, and no owner workflow result is inferred.

The core tasks remain useful for optional feedback:

1. Start a blank circuit, place a DC source, resistor, red LED and ground; connect the loop. Try 1 kΩ and 2 kΩ. Expect approximately 2.917 mA and 1.477 mA.
2. Open RC charge/discharge. Pause, Restart and Step to 10 ms; capacitor voltage is approximately 3.161 V. Switch to discharge and step another 10 ms; approximately 1.163 V remains.
3. Open the capacitor-timed flasher. Select both collector voltages or LED currents, place scope cursors and inspect the timing. Doubling both timing capacitors approximately doubles the period.
4. Use Save As for a separate working file, reopen it and export SVG/PDF/PNG. Choose Letter or A4 and Fit to page for a complete single-page drawing; preview explicit-scale tiling before assembly.

The release includes three editable paused examples, matching agent, dependency notices and rollback instructions. Existing /189 and all prior candidates remain preserved. Generic educational models and private-alpha limits apply. The [earlier /195 walkthrough](35-first-version-review-history-2026-10-22.md) retains its original unobserved owner-review status.
