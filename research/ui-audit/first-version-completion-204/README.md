# First-version /204 keyboard and compact UI evidence

Unmodified CUA captures of the exact extracted 0.181.3 /204 native executable.
The [keyboard receipt](../../../docs/validation/first-version-completion-204/keyboard-report.json)
and [release receipt](../../../docs/validation/first-version-completion-204/report.json)
identify the package/source and precise scope. No owner project was opened.

1. A blank circuit was populated using keyboard search/list selection and exact
placement, then four terminal menus connected it. Live LED current is 2.917 mA.
This precedes Fit, so the canvas still uses its initial 100% zoom.
2. After applying 2 kΩ, entering 0 and activating Apply retains the 1.477 mA result
and existing 2 kΩ circuit. The draft stays visible and generic refusal is displayed.
3. Corrected 2 kΩ, Undo to 1 kΩ and Redo to 2 kΩ restore the expected 1.477 mA.
4. Native keyboard Save/Open reopens LED-keyboard.elab cleanly with four components
and four wires. Actual saved JSON and a fresh packaged-agent reopen independently
confirm its values and terminal pairs. Capture may precede the final repaint;
CUA state records in the raw archive retain the accepted title/current/clean state.
5. Compact 1053×802 Dark view after reopening; drawing and Inspect controls readable.
6. Compact Dark Simulate controls are reachable.
7. Compact Dark Document export/print/page controls are reachable.

The Light keyboard captures and Dark checks use the default test window with
shared geometry access denied; this is not a clean OS-account installation.
The test app runs with appearance/recovery disabled and is closed after checks;
its endpoint disappears, PID is absent and the saved project remains. The first
wrapper-ID attachment was delayed and returned application-not-found; the already
running extracted app was then attached directly. No duplicate launch was needed.

8. Light header/browser/Inspect at the enforced 1053×762 minimum.
9. Dark at the same enforced minimum, with all six Common rows visible.
10. Original 1221×969 size and Light theme restored; clean blank document,
empty search and disabled Undo/Redo are unchanged. No circuit edit was made.
11. Dark working-size layout at 1221×969, before restoring Light again.
12. Original 1221×969 reference before the minimum-size check.

For 8–12 the CUA binding selected the existing /204 release window, rather than
the separately running helper with the same bundle identifier. Both executable
paths are verified. Only window size/theme were temporarily changed on the
existing window; the disposable helper's private marker and endpoint were used
for its independent safe shutdown. This distinction is explicit in the
[minimum layout receipt](../../../docs/validation/first-version-completion-204/minimum-layout-report.json).
