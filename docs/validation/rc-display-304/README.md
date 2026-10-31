# RC workspace · build 304

In /303 the RC schematic used the command revision to cache its mesh, so a
solver tick could update the inspector without refreshing probe badges. Mesh
preparation could also take a newer snapshot than the inspector. /304 binds
both schematic and scope preparation to the snapshot used to construct each
workspace frame and invalidates the schematic on sample revisions. Drag previews
change copies of the affected objects, and a trace without a point at the
current sample time is omitted from the live badges.

The [running RC example](live-rc.jpg) shows 743 mV at C1, 0.743 V in Inspect,
and the plot and inspector both at 0.069 s. Native clicks on Charge and Discharge
drove the capacitor toward 5 V and back to zero with the advanced controls
closed. Clicking Wire entered wiring mode and revealed Cancel. The test
returned the example to Auto and Select.

Regression tests exercise two real RC ticks without a command revision change:
the live inspector text, prepared schematic geometry and scope endpoint must
match the same accepted sample even when rendering is delayed. The mounted
tests cover scope layout, toolbar rendering and Wire activation in all four
themes. No simulation model or public command changed.
