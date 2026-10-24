# Workspace tab presentation

The owner's October 24 screenshot showed the drawing modes and inspector
sections as unrelated boxes with incomplete lower edges. This change groups
Select/Wire/Pan into a segmented rail and presents Inspect/Simulate/Document
as equal-width, flat tabs with a continuous baseline and accent underline.
Light, Dark, System and Apollo retain their respective palettes and typefaces.

The old drawing toolbar allowed 32 content units while its buttons used 18
units of vertical padding. Normal quiet buttons also reserved shadow space.
The replacement uses Vellum's flat toolbar appearance and explicit content
sizes, allowing complete outlines and readable labels. Inspector tab cells
reserve the actual button height, gap and two-unit indicator. Selected drawing
segments compensate for their border width so choosing another mode does not
move the labels.

## Evidence

- `before-apollo.png` and `before-border.txt`: the original production view
  fails the painted lower-border probe. The Esc button has zero central columns
  containing its expected lower border. Rectangle containment alone passed
  because the accessibility bounds were themselves clipped.
- `light.png`, `dark.png`, `system.png`, `apollo.png`: final mounted renderings
  of the real production toolbars, with the production schematic inset and
  inspector width. This reduced probe excludes the custom schematic renderer.
- `focused.txt`: race-enabled mounted tests pass in all four themes, checking
  visible control bounds, painted inspector indicators, Apollo's lower border,
  physical bottom-edge clicks, arrow navigation and Return activation through
  the nested groups. Agent commands select the same modes and sections;
  navigation preserves circuit, history and simulation state.
- `inventory-review.json` and `uiaudit-race.txt`: the production source inventory
  adds the presentation helper; all 404 native source sites are unchanged except
  for source line numbers. Public commands, aliases and route coverage remain
  identical. The checked-in inventory is regenerated and verified.
- `lint.txt`: goimports and staticcheck pass for the root and desktop modules.
- `desktop-vet.txt`: vet passes for the desktop module including offscreen tests.

The full race-enabled desktop suite, including all mounted offscreen tests,
reported one failure in 362.900 seconds: the existing inspector descriptor test
still required the old accent-filled background. See
`broad-race-old-selection-assertion.txt`. Its assertion was updated to verify
the new active label and underline colors, and the inactive colors of every
other tab, for both user and agent selection. This changes only the test;
production code remained identical to the full run. `affected-race.txt` records
the subsequent rerun of that test, all mounted inspector tests and request
guidance checks. The broad run was not repeated after this assertion update.

The initial reduced test retained focus on an excluded Resume button; that
fixture issue was corrected before the original lower-border failure above.
An intermediate design still reserved normal button shadows, and an initial
32-unit inspector cell clipped its indicator. The final test explicitly checks
the painted indicator as well as its bounds so those defects cannot pass on
rectangle checks alone.

This is a presentation repair. It introduces no agent commands or electronics
features. Native captures and package provenance are recorded separately after
the candidate is built; these mounted checks do not establish full native
accessibility or distribution acceptance.
