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
The full quality gate passes lint/staticcheck, 67 Python tests, root/desktop
race tests and vet, mounted race tests and offscreen vet. Desktop race takes
177.010 s; mounted race takes 167.844 s. All 891 frozen source/check inputs match.

The clean 0.181.20 /221 package is built from committed source. An ordinary
update of the existing owned test app keeps its identity, launcher and sandbox;
normal CUA launch succeeds without another permission question. Actual 1053×802
blank and long-filename RC views show the complete footer in Light and Dark.
Native accessibility help retains the full summary. The outside inset matches
the root surface, and the theme switch preserves the complete circuit, viewport,
divider, transient and recorded state. The original source RGB constants are not
used as native screenshot colors because the capture is color managed.

All 1,001 imported stored rows exactly match /220, and 2,002 independent voltage
references pass. The old fixture and /205 ZIP remain unchanged. The candidate
bundle remains untouched; the native tests use a code-equivalent copy packaged
under the existing test identity. The test app is left clean, paused and Light.
[Complete evidence](../docs/validation/footer-layout-221/README.md).

Four of eighteen accumulated native checks now have scoped evidence, including
three unaffected checks carried forward from /220. Fourteen remain open, so
/221 is held and does not replace the /205 download.

The separate Save shortcut observation remains unqualified: the native Edit >
Select All action works, but CUA Command-A does not select the filename. The
dispatch cause and ordinary hardware-keyboard behavior have not been established.
No speculative responder change is included in this layout repair.
