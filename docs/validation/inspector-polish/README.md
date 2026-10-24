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
updated to expect ordinary field text rather than action-accent text. Further
editing/scrolling regression results and native package checks are recorded
after completion; comprehensive accessibility/distribution acceptance remains
outside this cosmetic change.
