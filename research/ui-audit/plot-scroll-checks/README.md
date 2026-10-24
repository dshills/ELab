# Native plot wheel checks

The existing owned /206 test window is reused; no app is launched.

1. [Before](01-before.png): short scope dock at its lower scroll position.
2. [Wheel over the scope plot](02-scope-wheel-up.png): its playback/input toolbar appears.
3. [Reverse scope scroll](03-scope-wheel-restored.png): prior dock position returns.
4. [Schematic wheel zoom](04-schematic-zoom.png): 44%→99%, with accepted physics unchanged.
5. [Reverse schematic zoom](05-schematic-restored.png): prior view within 1e-12.

The complete public snapshot is exact across the scope-wheel pair. Schematic
zoom changes only presentation and revision/actor metadata. All protected
physical, graph, history, document and scope state remains exact. The paused
single-frame plot is the observed case; no exhaustive gesture or changed-build
acceptance is claimed. [Receipt](../../../docs/validation/plot-scroll-checks/report.json).
