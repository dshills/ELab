# All components mouse scrolling and scrollbar repair

October 23, 2026. Maintenance scope: existing component browser and mouse input.

The /201 app could navigate All components with the keyboard, but wheel events
and scrollbar grabs could reach the wrong control. Native button and wheel
callbacks polled GLFW's current hardware position. A queued or window-targeted
event can carry a different location. Cursor movement already used event
coordinates, so a drag could start on the wrong widget before its movement
arrived correctly.

The pinned Vellum compatibility source now reads the matching current AppKit
button/wheel event on the native main thread. It checks the exact NSWindow and
eligible event types, converts content coordinates to top-left logical points,
and retains existing finite/range checks and one DPR conversion. GLFW virtual
coordinates remain authoritative in disabled mode. Missing/unrelated events
fall back to the existing cursor query. The guarded schematic-canvas bridge and
all shared commands remain unchanged.

A second issue appeared when wheeling directly over the scrollbar: the slider
handler inverted a delta the driver had already normalized, making its direction
differ from the containing frame. Removing that redundant inversion makes the
rows and scrollbar agree. Regression tests compare both paths for horizontal
and vertical movement, positive/negative deltas and bounds.

Both repairs are reproducible native dependency stages with input/output hashes,
new test sources and a complete patch from the public Vellum base. No verified
SDK cache or sibling Vellum checkout was edited. The active archive is elab4;
the elab3 archive was a local preview preserved under the ignored build folder.

The desktop checks use CUA and clean empty documents. /201 failures were captured
before changes. The first repaired preview visibly scrolled to the last row and
back with the wheel, and the thumb could be dragged down and back up. Coordinate
clicks selected Resistor and focused Search; typing mosfet and Return produced
two results. A stale/resized intermediate capture is retained separately from
the refreshed reverse-drag confirmation. The later scrollbar-wheel correction
requires its own final packaged-app check; preview evidence does not stand in
for that check.

Final artifact identity, test results and observations belong in
[the /202 validation receipt](../docs/validation/component-scroll-202/README.md).
There are no new component kinds, circuit-model changes, file-format changes or
agent commands in this repair.
