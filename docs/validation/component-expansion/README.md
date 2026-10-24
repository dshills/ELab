# Hobbyist component expansion validation

Eight components and nine examples implement the five requested groups:
[display/BCD](../../digital-display.md), [momentary/bouncing buttons](../../momentary-buttons.md),
[comparators](../../comparators.md), [relays](../../relays.md), and
[linear/LDO regulators](../../voltage-regulators.md). The documents specify
pin names, equations, limits, probes and shared UI/agent behavior.

Focused electrical and native command checks pass for each phase. The
packaged stdio replays exercise every example, independent expected values,
conservation, captures, kept-record save/reopen and SVG/PNG/PDF export.
The display replay observes all ten digits, wrapping, and 13 increments.
The control replay checks raw/debounced counts 5/1, reset, 50 comparator
states, protected relay pickup/release/flyback, three supply loads and forty
independently inverted regulator load lines. See the JSON receipts and
reproducible `scripts/display-counter-demo.py` / `scripts/controls-demo.py`.

The [native check receipt](native-checks.json) and [screenshots](ui) record
mouse/keyboard operation of /242. Source review afterward corrected relay
energy in transient totals; the physical coil energy/current were already
correct. A distinct LDO glyph fixes the duplicate browser symbol found by
the broad suite.

The first broad root run (`root-race.txt`) hit the existing 100 ms wall-clock
integration guard in a flasher test while native checks were also running.
The first broad desktop run (`desktop-race.txt`) found the duplicate LDO icon
and an existing scope test that did not await asynchronous Step completion.
Its fixture now restarts at zero and awaits Step; the exact picosecond endpoint
and scope behavior pass three repeated focused race runs. No simulation
limits/tolerances were changed. Isolated full regression receipts are pending.

Generic models are functional teaching models, not fitted IC/package ratings,
independent SPICE validation or hardware measurements. The qualified /205
unsigned private-alpha archive remains preserved; these are development
candidates, not a new distribution qualification.
