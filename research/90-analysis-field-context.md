# Analysis entry ownership /213

October 23, 2026. A bounded repair to existing verification, DC sweep and
threshold-timing controls. No commands, devices, models or formats are added.

## Reproduction

Capture a production field callback, let an agent advance the editor context,
then deliver malformed text through the old callback. Five fields install the
old `oops` text into the newer form: verification duration and sample step,
sweep point count, and threshold interval start and end. The accepted service
snapshot stays unchanged, but a refusal now belongs to the wrong view and can
obstruct the next form action.

Shared time normalization can fail before command freshness checks; the sweep
point-count parser also runs locally. Therefore an `invalid-params` result is
insufficient evidence that the callback owns the current form. The existing
request-help guard prevents stale specialized advice, but does not by itself
protect these private refusal stores.

## Repair

One field-context predicate compares the callback's command revision and document
generation with the current service. Verification, sweep and timing controls
use it before retaining a rejected entry. A stale local sweep-count parser
reports changed settings instead of blaming the newer form's accepted count.

An already retained, current refusal remains editable after unrelated agent
work. It is not globally cleared whenever the command revision changes. Existing
field-value/document pruning, pending-action settlement, valid continuations,
parser error precedence and public-command behavior remain intact.

## Evidence

Nine production field cases cover verification time fields, sweep limits/count
and timing levels/intervals. The three prior form handlers, loaded with a Go build overlay,
reproduce the five visible `oops` replacements. The other four stale cases
already refuse correctly. Nine complementary cases preserve current refusals
through later unrelated work and apply a fresh correction exactly once, without
changing the agent's draft, physics, document or circuit history.

The original focused run fails in 3.333 s; the refined visible-field baseline
fails in 4.308 s. Final focused race checks, including existing verification,
sweep and timing settlement/continuation tests, pass in 12.048 s. Every semantic
native command site is unchanged; only four source hashes and line locations
move. Inventory stays at 166 commands, 134 routes, 404 sites and 77 sources.

The [full quality gate passes](../docs/validation/analysis-field-context-213/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted race and
offscreen vet. Desktop race takes 172.817 s; mounted race 153.785 s. All
880 source/check inputs remain exact. Native event-queue acceptance remains separate
from production-callback, offscreen and headless evidence. The current /205
download, previous candidates and the running owned native session are preserved.
Standing ELab/desktop authorization remains in `AGENTS.md`; no new launch prompt
or permission request is queued.


The [clean /213 matching agent](../docs/validation/analysis-field-context-213/candidate-report.json)
passes nine current invalid entries, three stale valid requests and three fresh
form corrections, preserving the agent's draft. A 41-row RC capture passes
123 independent scalar references and brackets the analytic 3 V crossing.
Workers join and the process exits 0. The signed bundle, captured inputs,
previous candidate and /205 download remain exact. Private native refusal
behavior remains a separate acceptance check; the candidate is held.
