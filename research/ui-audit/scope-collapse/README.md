# Compact scope dock — October 14, 2026

Hide scope in the live or recorded toolbar reclaims schematic space. Show scope restores the plot. Commands, readings, live acquisition and recorded playback remain available while collapsed. The same session preference is exposed as `scope.visibility.set` and `snapshot.scope_visible`.

| Capture | Result |
|---|---|
| [01 Earlier wide candidate](01-expanded.png) | Expanded recorded scope; earlier control placement |
| [02 Rejected compact candidate](02-expanded-compact.png) | Header scrollbar and clipped Theme action; retained honestly |
| [03 Final compact expanded](03-final-expanded-compact.png) | Theme and Hide scope visible; recorded plot, seek/speed, cursors, meters and footer present |
| [04 Final compact collapsed](04-final-collapsed-compact.png) | Full RC circuit visible at unchanged center/zoom, meters retained, sample 3 and Show scope retained |
| [05 Final compact restored](05-final-restored-compact.png) | Sample 3, cursor A at 10 ms and the same trace/measurements restored |

Final captures are 1184 × 854 pixels at the same window size. Expanded and restored views show a smaller schematic viewport, so this does not claim automatic Fit or full circuit visibility while expanded. The final header uses compact actor captions/padding and puts scope visibility inside the dock.

[Core/native command evidence](../../../docs/validation/scope-visibility-validation.json) covers paused invariants, hidden acquisition/playback and actual GPU canvas retirement/remount. Saved images are direct CUA captures, not mockups. Scope resizing/persistent layout and full accessibility acceptance remain open.
