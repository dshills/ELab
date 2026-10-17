# Pilot macOS screen-reader check · prepared, not executed

The 0.123.0 /140 owner-review window is not a disposable test target. Use the separate `ELab-first-version-ui.app` harness with the exact review executable, a fresh private endpoint/recovery directory and session-only appearance. Save test projects under its private directory. Never replace existing user files.

## Approval boundary

Temporarily enabling macOS VoiceOver and its caption panel changes system preferences. The computer-use confirmation policy requires confirmation for an unrequested non-sensitive system-setting change. Obtain permission for both changes and restoration before changing either. A pending question or elapsed time is not approval.

After approval, record the initial VoiceOver/caption states through the actual settings UI. Restore those exact states afterward, including leaving an already-enabled feature enabled. Record any unexpected OS permission request separately rather than treating this approval as a general permission grant.

## Test sequence

1. Activate only the owned harness and confirm blank startup/Search focus. Use VoiceOver navigation to find Search, component list, placement X/Y and the named wiring controls. Caption output must identify the control's name, role, value and enabled/selected state. A raw AX tree alone is insufficient evidence.
2. Build source/resistor/LED/ground with keyboard controls and named terminals. Read the live current through VoiceOver. Change 1 kΩ to 2 kΩ, apply, undo/redo and introduce/repair a missing wire. Confirm that current, paused, updating, stale and incomplete statuses are understandable without color or motion.
3. Open RC, pause/restart/step, reach the independently scrolled scope controls, choose a voltage/current probe and enter/clear cursors. Confirm current bounds/focus and usable names after scrolling; do not mistake a hidden or zero-bounds element for a reachable control.
4. Exercise Save As, Cancel, Open and unsaved-change decisions in native panels. Confirm the chosen filename, prompt options and return of keyboard focus. Preserve the last manual save. Open/close print preview without submitting a physical print job.
5. On the NPN/flasher examples, inspect both device sides and the scope using the same control paths. Record any canvas information that cannot be obtained through named inspection/navigation.

For each step retain the actual VoiceOver caption output, focus sequence, outcome, exact build/window/configuration and a reproduction for any failure. Distinguish a control name in an AX snapshot from an actual screen-reader announcement. If audio cannot be captured, qualify the caption/navigation path only; audio quality, comprehension and human assistive-technology acceptance remain open.

## Cleanup and acceptance

Save private test state, close only the owned harness, restore the prior OS settings and verify the owner-review files/window were not changed. Update the existing first-version issue list rather than starting another product-feature queue. This check does not replace owner feedback, the ten-person study, other hardware/display qualification or visible-latency measurement.
