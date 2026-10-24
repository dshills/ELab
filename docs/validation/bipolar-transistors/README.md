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

## Completion checkpoint

The full desktop suite with offscreen paint/input tests and the race detector
passes in 421.816 seconds. The full root race suite passes, including workbench
(89.290 s), conformance (116.810 s) and socket transport (1.380 s). These runs
cover the electrical implementation; the subsequent supply-caption-only move
passes final focused electrical/example checks. Lint, vet and Python checks pass.

Native /232 shows the working Apollo PNP example with an illuminated LED,
negative base/collector/emitter readings and synchronized input/LED/VBE/collector
traces. The native check revealed ambiguous duplicate Plot voltage captions.
They now explicitly identify VBE, VCE and collector voltage/current. Selected
readings now precede the model notes. This presentation-only follow-up retains
all control IDs, shared commands and guards; focused user/agent, flasher, NPN,
inspector form across four themes, tab and symbol regressions pass in 43.466 s.
The owned /232 session was closed normally for its replacement; other sessions
and user projects were preserved. No original electrical reference was rewritten.

Clean-source **0.183.1 /233**, built from `c53a4802`, passes strict ad hoc
signature verification. Its packaged external agent passes the three-example
stdio replay again; `stdio-report.json` and `build-result.json` describe the
finished candidate. The actual-part inspector ordering/label test also passes
in 7.639 seconds. Native observations confirm follower and complementary
waveforms, PNP sinking under negative drive, and explicit PNP LED Low/High plus
Step transitions. Low gives LED 2.860 mA and signed collector -2.85999 mA; High
gives cutoff. The PNP LED example is left running in Auto with Q1 selected and
Apollo active. The owned /232 test session was closed normally; existing user
sessions and projects remain intact. The /205 release archive is unchanged.

Some immediate AX observations preceded queued native updates, so fresh state
was used before dependent actions. This is not a general one-click latency
acceptance result. Some CUA captures omit unchanged text; native capture,
comprehensive accessibility and distribution qualification remain separate.
