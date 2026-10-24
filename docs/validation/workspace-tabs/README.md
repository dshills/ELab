# Workspace tab presentation

The owner's October 24 screenshot showed the drawing modes and inspector
sections as unrelated boxes with incomplete lower edges. This change groups
Select/Wire/Pan into a segmented rail and presents Inspect/Simulate/Document
as equal-width, flat tabs with a continuous baseline and accent underline.
Light, Dark, System and Apollo retain their respective palettes and typefaces.

The old drawing toolbar allowed 32 content units while its buttons used 18
units of vertical padding. Normal quiet buttons also reserved shadow space.
The replacement uses Vellum's flat toolbar appearance and explicit sizes,
allowing complete outlines and readable labels. Inspector tab cells reserve
the actual button height, gap and two-unit indicator. Selected drawing segments
compensate for their border width so choosing another mode does not move the
labels. The mode rail takes its intrinsic width, including padding and outline,
so Esc remains separated from it.

The first native candidate (/227) revealed a second layout issue: vertical
padding inside the constrained inspector toolbar let the following scrolling
form paint over the indicator. Moving the top inset into a separate sibling
fixes the allocation without changing the tab labels' positions.

## Evidence

- `before-apollo.png` and `before-border.txt`: the original production view
  fails the painted lower-border probe. The Esc button has zero central columns
  containing its expected lower border. Rectangle containment alone passed
  because the accessibility bounds were themselves clipped.
- `light.png`, `dark.png`, `system.png`, `apollo.png`: final mounted renderings
  of the production toolbars and complete inspector rail, including its
  scrolling form. The probe retains the production schematic inset and
  inspector width, but excludes the custom schematic renderer. It enables the
  scene rendering adapter with the manual software fallback, not a GPU device.
- `native-spacing-final.txt`: race-enabled mounted tests pass in all four themes, checking
  visible control bounds, painted inspector indicators, Apollo's lower border,
  physical bottom-edge clicks, arrow navigation and Return activation through
  the nested groups. Agent commands select the same modes and sections;
  navigation preserves circuit, history and simulation state. Esc must also
  remain separated from the drawing-mode rail.
- `focused.txt`: the earlier reduced header-only test passed but could not
  detect the overlapping sibling. `native-before-spacing-system.jpg` and
  `native-before-spacing-apollo.jpg` show /227's missing underline.
  `full-rail.txt` reproduces that native issue in the expanded regression test:
  indicator bounds exist but the scrolling form covers the painted pixels.
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
production code initially remained identical to the full run. `affected-race.txt`
records the first corrected rerun. The later native spacing repair above changes
production layout; `final-affected-race.txt` records the subsequent rerun of that
test, all mounted inspector tests and request guidance checks. The broad run was
not repeated after the assertion update and spacing repair.

The initial reduced test retained focus on an excluded Resume button; that
fixture issue was corrected before the original lower-border failure above.
An intermediate design still reserved normal button shadows, and an initial
32-unit inspector cell clipped its indicator. The final test explicitly checks
the painted indicator as well as its bounds so those defects cannot pass on
rectangle checks alone.

## Native confirmation and package

Clean-source candidate **0.182.4 /228**, built from `81323ce2`, is open at
`build/ELab-0.182.4-228-Tabs/ELab.app`. `build-result.json` records executable
hashes, clean Git provenance, strict ad hoc signature verification and the
unchanged preserved /205 archive. The build-time `graphical_app_launched:false`
precedes the native checks recorded here and in `report.json`.

`native-apollo.jpg`, `native-light.jpg`, `native-dark.jpg` and `native-system.jpg`
confirm complete outlines, the separated Esc control, and visible selected
inspector underlines in the actual app. System resolves to Light on this
desktop. Native actions selected Wire and all three inspector sections; the
app is left in Select/Inspect with Apollo selected. The empty superseded /227
test app was closed normally; older sessions were preserved.

One back-to-back automated mode/section click met the existing command revision
guard; the section action succeeded after observing a fresh frame. The
intermediate `native-simulate-partial.jpg` shows the correctly moved underline
but omits some unchanged labels, consistent with the prior native capture
limitation. The complete final theme frames are retained separately. No stable
capture or comprehensive native acceptance is claimed.

This is a presentation repair. It introduces no agent commands or electronics
features. These checks do not establish full native accessibility or
distribution acceptance.
