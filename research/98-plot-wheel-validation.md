# Scope dock scrolling and intentional schematic zoom

October 24, 2026. [Evidence](../docs/validation/plot-scroll-checks/README.md).

The existing owned /206 session handles the two plot regions differently, as
intended. Wheel input over the scope plot scrolls its short dock to reveal the
input toolbar. Reversing that scroll returns its viewport. The complete public
snapshot stays exact, including physical state, graph, values, history and
scope settings. Pixels show the movement despite an unchanged accessibility tree.

Wheel input over the schematic deliberately changes its view from about 44% to
99%. Reverse input restores the prior view within 1e-12; the largest difference
is about 2.84e-14 world units. Only view and revision/actor metadata differ from
the baseline. Circuit, charge, time, history, dirty state and saved fixtures stay
exact. This is a paused single-frame live-scope case, not every plot interaction.

Five relevant application source files and the framework's mesh/frame sources
are identical to those in the old native build. New race-tested regressions
check that the scope forwards a wheel packet only to its named host, restores
that host's temporary scroll eligibility whether initially on or off, and
suppresses unrelated-pane/mesh zoom when no dock exists. Three cases pass in
1.747 s; desktop staticcheck and vet also pass. An initial fixture compile error
chained a setter returning a tree base and is retained separately.

No product behavior, dependency or package changed. The /219 source/package
qualification inputs remain exact; its separate fresh native acceptance list is
unchanged. The current download remains /205. No new launch or confirmation
prompt is queued, and the owned test window remains paused for reuse.
