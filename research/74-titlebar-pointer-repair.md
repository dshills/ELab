# Native window dragging and pointer repair

October 23, 2026. Follow-up to the Apple-inspired workspace in 0.180.0 /198.

The owner clarified that the mouse problem was in the top bar: the window could
not be moved, and the pointer looked larger and different from the normal macOS
pointer. This repair targets those two problems.

The unified full-content title-bar policy adds an empty toolbar and transparent
full-size content view while explicitly disabling background window dragging.
ELab now requests the default OS window chrome, leaving the macOS title bar
outside the editor content. The artificial 32dp editor inset is removed. The
component library, compact toolbar, three-section inspector and live canvas stay
in their redesigned layout. Dragging uses the normal macOS left-button gesture.

Vellum's desktop driver previously rasterized its cursor SVGs at 32dp, including
theme inversion and shadow. The macOS path now uses GLFW standard native cursors
for arrow, I-beam, pointing hand, crosshair and horizontal/vertical resize shapes.
macOS supplies their artwork and accessibility sizing. Unsupported/custom shapes
continue through the existing raster path. Other desktop platforms retain their
existing cursor behavior.

The framework repair is a separately committed compatibility patch against the
same public Vellum base, pinned in the SDK lock. Each window owns at most six
standard cursor handles; native-main allocation/binding/cleanup, request
retirement and duplicate-hover fast paths remain enforced. Returning to a cached
custom raster explicitly rebinds it after a native shape. The old SDK, release and
running owner app are preserved. No electronics features, persistence format or
shared user/agent commands change.

Regression coverage checks shape mapping and fallback, bounded reuse, allocation
refusal, teardown despite native callback panic, closed-owner refusal, repeated
hover without a native queue roundtrip, and native/raster/native transitions.
The test fixtures replace native backends before touching software-only windows.
Initial fixture cleanup attempted an uninitialized GLFW window; the fixture was
corrected to substitute both native backends, then the full focused cursor and
pointer race suite passed. SDK export initially lacked network access to verify
its dependency/license graph; the exporter completed with the normal pinned
registry access.

ELab 0.180.1 /199 source and package validation is recorded in
[the validation receipt](../docs/validation/native-pointer-199/README.md). Full
source checks, cursor/pointer race tests, native/agent LED/RC parity and ZIP
extraction passed. The isolated app exposed title-bar controls and accepted Search
input. CUA captures returned blank Metal content, so attempted title-bar drags
and native pointer appearance remain visually unverified. Flasher timing hit the
existing 100 ms integration work limit on two attempts; no solver limits or
electronics code changed. The isolated test session shut down cleanly.

The separately launched normal /199 app subsequently rendered the canvas,
library and inspector correctly in a read-only CUA screenshot. It remains open
with a blank document. The /198 owner instance was preserved. Native pointer
appearance and actual drag displacement still need direct confirmation.
