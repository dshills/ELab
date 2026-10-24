# Slider wheel input /219

October 24, 2026. [Receipt](report.json), [source inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json),
[native baseline](../../../research/ui-audit/slider-wheel-219/README.md),
[repair notes](../../../research/97-slider-wheel.md).

The owned old /206 native slider changes its potentiometer from 50% to 100%
on wheel input without a click. Model readings and dirty/history state change.
Its original managed slider factory and complete upstream slider match the
current source before repair. Two production-control and four driver cases fail.

The managed value-slider factory drops its wheel handler. Wheel input scrolls
the containing pane without accepted-state changes, while deliberate pointer
clicks/dragging and arrow keys still edit. Framework scrollbars are constructed
directly by their frames and keep their existing handlers. ELab has one managed
value-slider node. No production desktop Go source or backend source changes.

All 8,435 export members match the manifest; only the factory, new driver test
and generated provenance differ from elab10. All upstream dependency bytes,
notices and bootstrap helpers remain unchanged. The full SDK root/driver normal
and offscreen race/vet gates and 30 preparation tests pass. Prior supplemental
full upstream Core race failures remain recorded separately, without a new pass.

The full app gate passes lint/staticcheck, 67 Python tests, root/desktop race and
vet, configured mounted race and offscreen vet. Desktop race takes 190.584 s;
mounted race 169.917 s. All 889 frozen source/check inputs match. Original
fixture/import/cwd/native-count mistakes and the restricted local-socket refusal
are retained separately from genuine control regressions.

Native inspector/text/selector/thumb checks preserve the complete RC snapshot.
The prior RC graph and complete captured run are saved and freshly reopened.
The reused session is left on the clean paused pot at 50%, with explicit fresh
readings; its history/run/revision changes are recorded. This old-native proof
does not qualify the changed candidate. /218 and /205 stay preserved; no launch
or unattended permission question is queued. Clean packaging follows.
