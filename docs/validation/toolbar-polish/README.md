# Consistent toolbar controls

The document header, drawing utilities, simulation transport, live and recorded
scope toolbars, sweep and print preview share one presentation pass. Buttons
have 30-point targets, eight-point gaps and a consistent outline. Icon-only
document actions are square. Active modes use accent text and outline with a
neutral surface; disabled actions remain muted. Native keyboard-focus styling
and all existing control IDs, callbacks, field ownership and command guards
remain intact. Inspector tabs and the drawing-mode rail keep their established
presentation. Pause/Resume have equal widths, and subtle document separators
make the existing action groups easier to distinguish.

Toolbar rows are 40 points tall, with no vertical padding inside the constrained
row. This reserves room for controls, focus and horizontal overflow without
allowing following content to paint over them. Time/numeric inputs keep their
minimum widths. The toolbar background is explicitly taken from the session
palette; omitting it on the new simulation group exposed a light default in
Apollo during the initial test and was corrected.

## Checks

- `light.png`, `dark.png`, `system.png`, `apollo.png`: mounted production header,
  drawing and complete live scope controls, with only custom graph rendering
  replaced by a blank layout placeholder. These are manual software-rendered
  scene-adapter captures, not native GPU screenshots.
- `final-mounted.txt`: four-theme measurements verify actual button/input
  heights and containment, and painted lower outlines rather than relying on
  potentially clipped accessibility rectangles. Native focus is tested with
  its own outline color. Narrow-toolbar End navigation scrolls the final action
  into view, and a physical click activates Restart.
- `inventory-review.json` and `uiaudit-race.txt`: all 404 native source command
  sites, public commands, aliases and route coverage remain unchanged apart
  from source line numbers. The inventory adds one presentation source file.

`focused.txt` passes the final four-theme toolbar, existing tabs, live cursor,
recorded numeric-control, footer and theme checks with the race detector. It
additionally exercises physical and agent cursor selection, verifies the
selected outline in captured pixels, and measures equal Pause/Resume targets.
`lint.txt` and `vet.txt` pass. `desktop-race.txt` records the full desktop suite,
including offscreen tests, passing in 371.814 seconds. That run compiled before
the final retention of legacy minimum input widths and added transition
assertions; those adjustments are covered by the final focused run.

## Native app

Clean-source **0.182.5 /229**, built from `ab00881a`, is open at
`build/ELab-0.182.5-229-Toolbar/ELab.app`, with a paused RC example and Apollo
selected. `build-result.json` records clean executable provenance and strict
ad hoc signature verification; the preserved /205 archive is unchanged. Its
build-time launch flag predates these native checks.

`native-apollo.jpg`, `native-light.jpg`, `native-dark.jpg` and
`native-system.jpg` show the actual document, drawing and live scope toolbars.
The native example menu and Pause action work, with full control outlines and
the existing inspector underline visible. System resolves to Light. A component
library tooltip appears outside the toolbar area in the Light/Dark/System
captures. Vellum retains the platform keyboard-focus accent independently of
the app's selected-mode styling. Older sessions and user projects are preserved.

These checks do not establish full distribution or accessibility acceptance.
