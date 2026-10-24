# PNP and broader bipolar circuit validation

See [the implementation contract](../../bipolar-transistors.md) for model,
commands, signed measurements, examples, limits and file compatibility.

Independent scalar Ebers–Moll/load-line checks cover high-side PNP switching,
loaded emitter followers, complementary sourcing/sinking, emitter feedback,
common-base and diode-connected devices. Finite differences check all three
terminal Jacobian rows with each terminal grounded in turn. An independent RK4
ODE checks emitter capacitor timing for both polarities. Coupled coil/capacitor
polarity symmetry preserves current/power balance and the integration attempt
bound; race-instrumented numerical execution is not a latency measurement.

The external stdio replay verifies all three shipped examples, capture, live
stepping, signed PNP switching, independent follower references, save/reopen and
SVG/PNG/PDF export. Maximum follower voltage reference error is below 0.1 nV in
these pinned, idealized fixtures. This is equation/reference verification, not
hardware measurement or an independent SPICE run. The retained PNGs were visually
inspected for symbols, labels and complete output. The last supply-caption layout
adjustment has a separate focused regression.

The full root race suite passes. Its original sandboxed attempt could not bind
local sockets; a normal authorized socket-capable run passes transport and
conformance. An initially stale catalog-size expectation and missing pinned VBE
syntax rule were corrected. The final focused checks include every library probe
with present/removed owners and saved selected transistor measurements.

Native callback tests construct the three circuits from blank through placement,
value, rotation and named-wire controls, then compare the shared agent results.
Menus retain dirty-work protection; stepping and paused edits retain their guards.
Lint, root/desktop vet and 67 Python helper tests pass. The native command inventory
still has 166 public commands and 404 source sites; the three new native example
aliases target the existing example.load command. No new public command path.

Final full desktop offscreen/race and native candidate observations are recorded
in the completion checkpoint below when available. The preserved /205 release
archive is not replaced by this development candidate.
