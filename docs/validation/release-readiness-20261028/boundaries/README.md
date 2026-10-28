# Item 2: analysis boundaries

The maintained elab-conformance --analysis-boundaries suite constructs parallel
RC circuits through public commands at 3, 18 and 64 simulated parts. It runs 201
points for DC, AC and noise with three selected voltage outputs (AC gain/phase
has six columns), validates every first-output point against independent DC,
RC transfer and Johnson-noise references, saves/reopens all three exact captures,
edits R1, checks all source associations become stale, then saves/reopens again.
All four adapters pass 12 boundary cases; all 27 external processes exit 0 normally.
Rendered user actions start the 64-part noise sweep, return from its viewer and
apply an R1 edit. Existing commands/models/work limits are unchanged.

Initial controller assertions expected three AC columns, omitted downgrading a
fixture without sweeps to document v4, and tried inspector navigation inside the
sweep viewer. Those failures are retained; no production behavior was changed to
satisfy them. The initial failing ordinary-agent test needed forced cleanup;
explicit app.close in the new workflow's deferred cleanup fixes that harness
issue, and final public receipts all show normal exits.

Prism initial/follow-up reviews use Anthropic Claude Opus 5.5, complete without
skipped/truncated input. Initial findings are fixed: operation identity retained
from admission, shape checks before indexing, exactly three captures required
after edit, UI component/count guards, 20ms polling, cross-mode comparison and
negative noise/action evidence tests. A nil-reference low is already prevented
by CompareAnalysisBoundaries rejecting missing results on the first adapter.
Follow-up has no medium/high findings; the remaining length-check ordering low
is fixed with explicit early shape validation. Final race/rendered controls and
lint pass. Shared-command native service checks are not OS-input/rate acceptance.
