# Inspector form polish

Inspect, Simulate and Document use aligned 34-point fields and actions,
four-point caption/editor spacing, eight-point spacing within sections and
twenty-point separation between sections. Existing section headings receive a
consistent semibold treatment; groups without a visible heading use their
existing accessible group name. Help paragraphs retain their muted treatment.
Paper and Orientation receive visible captions. Warning/destructive colors and
native focus/filled-editor underlines are preserved.

The presentation helper groups existing captions with editors before applying
the existing rail width policy. Leaf IDs, callbacks, command/revision captures,
control-state keys and validation behavior are retained. The themed control
skin runs only inside the right inspector body. Text/numeric editors reserve
their native underline geometry; selectors/actions reserve their side borders,
so their visible right edges align.

The twelve PNGs and `final-paint.txt` cover all three pages in Light, Dark,
System and Apollo, mounting the full production inspector including its
scrolling form. Checks measure 248-point visible field widths, 34-point heights,
aligned captions, caption/control and following-content gaps, and actual bottom
border paint. Mounting and repainting preserve circuit and simulation state.
These are manual scene-adapter software captures, not native GPU screenshots.

`inventory-review.json` records identical command, alias and route coverage:
all 404 native source sites are unchanged except for line numbers. One new
production presentation source is added. The existing Apollo selection test is
updated to expect ordinary field text rather than action-accent text.

`focused.txt` passes editing, value application, output-page, validation,
focused-field reveal, scroll preservation, theme and layout regressions with
the race detector in 44.281 seconds. The full desktop suite including offscreen
tests passes in 378.850 seconds (`desktop-race.txt`). Lint, vet, root inventory
tests and checked-in inventory verification pass.

Clean-source **0.182.6 /230**, built from `f3a6f82b`, is open at
`build/ELab-0.182.6-230-Inspector/ELab.app`. `build-result.json` records clean
provenance, executable hashes, strict ad hoc signature verification and the
unchanged preserved /205 release archive. Its build-time launch flag predates
the native checks here.

Native captures cover the selected R1 component form, simulation controls and
Document settings in Apollo, plus Document in Light, Dark and System (resolved
Light). Wheel scrolling reveals the lower Document sections while navigation
remains fixed; returning to Inspect restores its start. The app is left paused
with R1 selected in Apollo. Existing sessions and projects are preserved.

Some intermediate native captures omit custom canvas content while retaining
the inspector form. Complete final/theme captures are retained separately;
tooltips outside the inspector are preserved as observed. Comprehensive stable
capture, accessibility and distribution acceptance remain outside this change.
