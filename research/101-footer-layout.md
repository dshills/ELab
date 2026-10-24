# Footer layout repair

October 24, 2026. The /220 native captures show the long document summary
wrapping below the fixed-height footer. The mounted production footer reproduces
the problem with a long legal filename in Light/Dark at 1040 and 1440 units.
Its label consumes the row's 23-pixel height and clips the wrapped content.

The footer now shows the component and wire counts. The document title and
history retain their existing roles; the complete previous summary is available
through the footer's hover/accessibility help. No document, simulation, command,
history or persistence behavior changes.

Four mounted regressions fail before and pass after the change with race
checking. The compact label is 18 pixels high and fits the existing row in both
themes. The fixture mounts the real footer and root style without the custom
schematic renderer, which the manual driver deliberately does not admit.
These focused captures do not establish full-window native acceptance.

The command inventory review finds exactly one changed source hash and shifted
line numbers; all 404 command/callback sites, 134 source routes and 166 public
commands remain identical. The initial full gate correctly refuses the stale
ledger. Its output is retained; the reviewed ledger is updated before rerunning.
Full quality and clean package/native qualification follow.

The separate Save shortcut observation remains unqualified: the native Edit >
Select All action works, but CUA Command-A does not select the filename. The
dispatch cause and ordinary hardware-keyboard behavior have not been established.
No speculative responder change is included in this layout repair.
