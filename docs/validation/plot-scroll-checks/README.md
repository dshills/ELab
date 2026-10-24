# Scope wheel forwarding and schematic zoom

October 24, 2026. [Receipt](report.json), [raw producers/state/logs](raw.tar.gz),
[member hashes](raw-manifest.json), [native captures](../../../research/ui-audit/plot-scroll-checks/README.md),
[notes](../../../research/98-plot-wheel-validation.md).

The existing owned /206 window scrolls the short scope dock when the wheel is
over the plot, revealing its controls. Scrolling back returns the viewport;
the complete public snapshot stays exact. Schematic wheel zoom deliberately
changes presentation only, from 44% to 99%; reverse zoom restores the view within
1e-12. Its graph, values, paused physical state, history and document stay exact.
Revision/actor metadata change, and tiny floating-point view differences remain.

The five relevant app sources and framework mesh/frame sources are identical to
the old native build. Three focused race-test cases exercise named-host routing,
temporary scroll-ability restoration and the undocked no-zoom/no-other-pane path.
They pass in 1.747 s; desktop staticcheck and vet pass. The first test fixture
mistakenly chained SetName, which returns a tree base; its compile failure is
retained separately and no product defect is inferred.

No product, SDK or package changes for this check. All 889 earlier qualification
inputs match; only a new regression file is added. No full gate or replacement
build is repeated solely for this evidence. /219 remains held for its separately
listed native repairs, and /205 stays the download. The owned paused pot and its
previously saved RC capture remain available, with both files unchanged.
