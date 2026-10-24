# Number fields and wheel scrolling /217

October 23, 2026. [Receipt](report.json), [inputs](source-inputs.json),
[repair notes](../../../research/94-number-field-wheel.md),
[old native baseline](../../../research/ui-audit/number-field-wheel-217/README.md).

A focused number field treated wheel movement as numeric editing, consuming the
pane's scroll. The owned /206 native session changed displayed recorded speed
from 0.1 to 0.13000001; its unfocused control scrolled without changing the full
public snapshot. Current production live-initial-voltage and recorded-speed
nodes reproduce two focused failures; both unfocused cases pass.

The managed factory removes the Spinner's default wheel handlers. Driver tests
preserve the accepted 25 and unfinished `925` text/selection, submit no Change
and move the parent pane. Standalone fields ignore the wheel. Up/Down still
submit one explicit increment each. Both final driver cases fail on a read-only
original-factory overlay and pass with the repair in 1.906 s. Application cases
pass prospectively in 2.724 s; final pinned keyboard/wheel checks take 5.984 s.
The supported-control fixtures do not mount a custom schematic or AppKit loop.

SDK source `2a0f783acb7f2ef20a6003c3277309ef6490be3c` is clean. Its 30 preparation tests and
framework root/driver normal/offscreen race and vet checks pass. All 8,434 export
members match the manifest; only controls.go, its new test and the generated
provenance note change. Upstream Core/typesetting, notices and all six bootstrap
scripts match elab9. Prior full upstream Core race failures remain recorded in
/216; no broad upstream test pass is inferred from these narrower gates.

The full application gate passes lint/staticcheck, 67 Python tests, root/desktop
race and vet, mounted race and offscreen vet. Desktop race takes 189.581 s;
mounted race 168.282 s. All 888 source/check inputs match. All 126 non-test
desktop Go files, backend sources and the 79-source command inventory are
unchanged: 166 commands, 134 routes, 404 sites. [Raw evidence](raw.tar.gz),
[member hashes](raw-manifest.json). Initial label/inspector fixture errors and
probe/default-sentinel mistakes are kept separately from genuine failures.

The old native test restores the effective rate as explicit 0.1; its original
stored default was 0. Revision/actor metadata also differ, while every other
snapshot field and both viewport positions are preserved. This is a scoped
restoration, not exact equality. New candidate native acceptance is still
separate. /205, /216, user projects and the existing owned session are preserved;
no graphical launch or unattended launch-approval prompt is queued.

The [clean /217 candidate and matching agent](candidate-report.json) pass
packaging integrity, circuit Undo/Redo, 42 independent RC readings, saved-capture
reopen, dirty-work refusal and normal headless cleanup. The candidate is held
for [fresh native acceptance](native-acceptance-pending.json); /205 remains current.
