# LED label clearance

LED IDs remain upright and clear their own body, outward light arrows and terminal wire stubs at all four rotations. The native canvas and SVG output use one shared position; PDF, PNG and print preview inherit the SVG position. Fit includes the new label bounds, including long IDs. Existing rotate, Fit, save and export commands work the same for users and agents.

At 0° the label sits beyond the arrows on the right. At 90° it sits above the horizontal symbol; at 180° it sits beside the body on the right, away from the arrows; at 270° it sits below the horizontal symbol. Rendering fonts retain their existing styles. No graph, connectivity, model, numerical, saved-format or command identity changes are required.

Before this repair, the export's default label began amid the arrow tips; horizontal rotations could put its background over a terminal stub. The native canvas used a separate label above the component, which could crowd a vertical stub. Regression geometry encloses the larger native symbol, both sets of emission arrows and 24-unit terminal stubs. Native mesh checks require actual ID glyphs at the shared positions, and Fit checks preserve graph/physics/document state while containing long labels.

The public reference `scripts/led-labels-demo.py` exports a four-rotation incomplete drawing, reopens a previous-release 30 V LED project and recreates its 24 V version, then exports every guided example. SVG/PNG/PDF exports must preserve graph, physics, history and document state. Optional native `--keep-open` leaves the owned rotation drawing for visual inspection; the caller must then close that owned session explicitly.

This is clearance from an LED's own geometry, not automatic global label placement. A neighboring wire, component, source annotation or other label can still need manual rearrangement. Full typography/layout/accessibility and physical-printer acceptance remain separate work.

Native inspection of the four-rotation grid and the complete live 24 V circuit confirms the visible label clearance. Direct PNG review also covers the flasher, NMOS, PMOS and LED-fade layouts. At the 0.95.0 checkpoint the flasher still had separate RB resistor-label crossings at neighboring LED pins. The [0.96.0 template spacing repair](flasher-layout.md) clears those bias labels; general neighboring-object avoidance remains open.

The oversized native smoke fixture now explicitly scrolls its offscreen scope host into view before awaiting positioned scrollbar geometry. Its strict wheel movement/no-zoom assertions remain. The viewport wait transfers completed immutable samples through a channel, avoiding shared mutable locals when owner callbacks finish after context cancellation. Neither harness change alters the product’s circuit commands or model.
