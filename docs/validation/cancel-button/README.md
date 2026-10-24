# Cancel drawing control

Renamed the visible Esc action to Cancel. Its tooltip explains unfinished wire
and component placement cancellation, return to Select, preservation of existing
components/wires, and the Escape keyboard shortcut. The public command remains
`drawing.cancel` with the existing activation and focus behavior.

Mounted measurement reproduced a three-point vertical offset between the drawing
mode buttons and toolbar actions. A row matching the mode rail's inset/outline
now aligns Cancel, DC sweep and Fit with Select/Wire/Pan. Each visible button is
30 points tall; the eight-point gap after the mode rail is retained.

The four PNGs are offscreen software captures of the production toolbar and full
inspector rail. Race-enabled regression tests verify aligned bounds across Light,
Dark, System and Apollo, painted bottom outlines, physical lower-edge Cancel
activation, mode/tab keyboard navigation, Escape handling, and toolbar overflow.
Focused desktop tests pass in 11.423 seconds. Lint, vet and native command ledger
tests pass. The source inventory retains all 166 public commands and 404 native
sites; differences are the Cancel display text and line numbers only.
