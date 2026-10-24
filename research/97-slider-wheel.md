# Scrolling over potentiometers and other controls

October 23–24, 2026. Mouse behavior in existing controls; no new feature, component,
model, command or format. [Native baseline](ui-audit/slider-wheel-219/README.md).

In the already-running owned /206 session, wheel input over an ordinary time
field and a probe selector moves the Simulate inspector without changing the
full public snapshot. Dragging the inspector thumb returns to its first
controls. Dragging the scope thumb reveals the complete playback toolbar and
returns to its prior viewport. Pixels demonstrate movement even though the
accessibility tree is unchanged. This checks those paths, not every control or
native accessibility requirement.

A potentiometer slider behaves differently: wheel input over its track, without
clicking or focusing it, changes 50% to 100%, marks the circuit edited and adds
an Adjust wiper history entry. The accepted model readings change as well.
One native Undo restores the graph; an explicit Restart refreshes the paused
stale result at time zero. Capacitor voltage and time remain continuous during
the actuation; instantaneous current can change. The previous RC graph and its
complete captured run are saved separately and reopen exactly in a fresh
headless agent. The reused test session stays on the disposable pot fixture,
fresh and paused at 50%; revisions/run/history differ. No whole-state restoration
or owner-project modification is claimed.

The original managed slider factory and complete upstream slider source match
between the native /206 SDK and the current elab10 pin. Two current production
wiper fixtures reproduce accidental accepted-state changes with and without
focus. Four driver cases also reproduce the problem in standalone and scrolling
hosts. Initial working-directory/import errors and an overly strict native
command-count assertion are retained as harness mistakes, not app defects.

The narrow repair removes wheel handlers from the managed value-slider factory,
matching the number-field policy. Wheel input reaches its enclosing pane;
standalone sliders ignore it. Scrollbars retain their existing handlers because
they are created directly by the underlying frame. Pointer clicks/dragging and
arrow-key adjustments remain deliberate edits, using the same public wiper
command and history contract.

Four focused driver regressions pass in 0.595 s and two production-control cases
pass with a read-only prospective overlay in 0.710 s. Accepted state stays exact,
the scroll host moves, and deliberate keys retain a single shared history entry.
The driver also preserves one pointer-click commit. Verified SDK caches are not
edited. The [elab11 pin review and full framework/application gates pass](../docs/validation/slider-wheel-219/README.md).
All 8,435 SDK members match; upstream dependencies/notices/bootstrap bytes are
unchanged. Desktop race takes 190.584 s, mounted race 169.917 s, and all
889 frozen source/check inputs match. The clean /219 package and matching agent pass 90 numerical checks: 84
captured waveform readings, two piecewise RC voltage endpoints and four
instantaneous current/power relations. Deliberate wiper changes preserve charge
and time; graph Undo/Redo, immutable captured data, exact dirty-work refusal
and saved reopen pass. All owned headless workers join and the process exits 0.
The app bundle remains intact. The candidate is held for native acceptance. The /205 download stays preserved; changed-candidate native acceptance
remains separate from this old native reproduction.
