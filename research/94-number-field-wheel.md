# Wheel scrolling over numeric controls /217

October 23, 2026. Existing mouse behavior; no new components, commands, models
or formats. /205 remains the current download.

The already-running owned /206 session reproduces an accidental edit. With its
recorded playback number field unfocused, a normal downward wheel gesture moves
the scope and leaves the complete public snapshot unchanged. After clicking the
same field, the same gesture changes its displayed rate from `0.1` to
`0.13000001` and stops scrolling. The public snapshot confirms the changed
recorded playback preference; its live circuit, results and measurements remain
unchanged. This check starts no app and does not touch the owner's ordinary app
or projects.

The native /206 SDK's number-field factory and complete upstream spinner source
match the corresponding elab9 sources before this repair. Current-source
retained-widget tests independently reproduce two failures: wheel input changes
live initial voltage and recorded playback speed when focused. Their unfocused
cases scroll without accepted-state changes. These fixtures use production
nodes and callbacks inside a supported scroll host; they do not mount a custom
schematic or qualify a new native event loop.

The managed number-field factory removes the underlying spinner's default wheel
handlers. Wheel input can then propagate to the containing pane. Typing, step
buttons, keyboard increments, validation and public command delivery remain
available. A focused driver regression preserves an unfinished `925` draft and
logical selection, preserves the accepted value and sends no Change callback;
the scroll host moves. The standalone case also preserves text/value. Up and
Down still each submit one deliberate increment. The caret's screen rectangle
is allowed to move with the pane; logical editing state must stay exact.

The two final driver regressions fail against a read-only overlay of the
original factory and pass with the repair in 1.906 s. The production field
regressions fail in 1.527 s and pass with a read-only prospective overlay in
2.724 s, including deliberate keyboard edits. Verified SDK caches are not
edited. The [full framework and application gates pass](../docs/validation/number-field-wheel-217/README.md).
The elab10 export changes three members and preserves all upstream dependency
bytes. Final pinned keyboard/wheel checks pass in 5.984 s. Desktop race takes
189.581 s; mounted race 168.282 s. All 888 source/check inputs match.
The clean /217 package and matching agent also pass circuit Undo/Redo, 42
independent RC readings, saved-capture reopen, dirty-work refusal and normal
headless cleanup. The candidate is held for fresh native acceptance.

The first full framework gate passed its normal/offscreen main driver tests,
but the workbench example could not bind local Unix sockets under the test
sandbox. The failure log is preserved separately; the passing rerun uses the
same unchanged source with authorized local socket access. No graphical app is launched.

The native trial restores the effective playback rate to `0.1`. The original
snapshot stored the unset sentinel `0`, which the UI displayed as its `0.1`
fallback. The public setter requires a positive rate, so restoration stores an
explicit `0.1`; revision counters and last-actor metadata also change. Every
other snapshot field remains exact. Both scroll viewports return to their prior
positions. This is a scoped restoration, not a byte-identical full snapshot.
The test session remains available for further authorized checks.

Initial fixture errors used a nonexistent label constant and omitted the
Simulate inspector selection. They are retained separately from the final
reproductions. Early probe formatting and restoration assertions also confused
the stored default sentinel with its displayed fallback; the final receipt
records the actual difference rather than claiming an exact restoration.

## Native acceptance procedure for the rebuilt candidate

Use a disposable paused RC session with a completed recording. Set recorded
speed explicitly to `0.1` before capturing the baseline, so restoration does
not have to convert an unset sentinel. Preserve project/capture files and the
full public snapshot. Standing launch/desktop authorization is in `AGENTS.md`;
do not ask for it again or leave work waiting on a launch-tool prompt.

1. Reveal the recorded rate in the scope. With another field focused, wheel
   over the rate. Confirm visible scope movement and an exact public snapshot.
2. Click the rate and confirm actual focus from AX. Repeat the wheel gesture.
   The scope must move, the displayed rate must stay `0.1`, and the full public
   snapshot must remain exact.
3. Type an unfinished rate without Return. Confirm its visible text first,
   then wheel over it. Preserve that draft and accepted state while scrolling.
   A no-change AX diff alone cannot prove viewport movement; capture pixels.
4. Cancel the draft, then use explicit Up/Down and step buttons. Each should
   change the rate once by its declared step, within float32 representation
   tolerance. Typing a complete rate and Return must still commit normally.
5. Repeat focused/unfocused wheel checks over live initial voltage in Simulate.
   No settings, run, sample, charge, graph or document change is allowed. This
   case needs no deliberate live-value edit or simulation restart.
6. Restore the explicit rate and original viewports, compare every protected
   snapshot field and preserve captures/files. Record any actor/revision-only
   differences. Close only the owned session when no longer needed.

This procedure is prepared, not executed against the rebuilt candidate. The
old /206 reproduction and current source/offscreen results retain their own
separate scopes.
