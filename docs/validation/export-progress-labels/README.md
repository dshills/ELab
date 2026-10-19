# Captured export progress wording

October 19, 2026. Source correction for `UI-JOB-LABEL-01` from the
[background-work review](../../../research/40-background-job-coverage.md).

PDF, PNG and sweep exports previously fell into the native file-progress panel's
Open category. The panel consequently said **Project open / Cancel project open**
and described an Open operation's edit-supersession behavior. These jobs actually
export captured data. The corrected command switch groups all six export routes,
giving them **Project export / Cancel project export** and the existing captured
export explanation. Save, imports, recovery and Open retain their classifications.

Only `desktop/project_work.go` changes among the 650 application Go inputs.
Shared command IDs/parameters, cancellation callbacks/targets, publication guards,
document guards, solver/models, file format and SDK remain unchanged. No new
behavior tests are added for this bounded presentation correction. Existing
quality checks supply integration coverage; no new native screenshot or physical
Cancel observation is claimed.

The first full quality attempt correctly refuses the stale native source ledger
after 16.187 s. Its original ledger, source, logs and result remain. Review of the
candidate finds one changed source hash and one unchanged Cancel binding moving
from line 60 to 61; all non-line site fields, public IDs, aliases/routes and totals
remain identical: 72 files, 397 sites, 132 routed /164 public IDs. The reviewed
ledger is updated rather than bypassing its guard.

Final `./scripts/check.sh` passes in **231.978 s**, including PRD coverage, imports/
Staticcheck, 54 Python tests, both modules' race/vet checks and mounted-control
race/vet checks. Desktop race and mounted stages take 158.185/43.228 s. Source
hashes stay fixed during both attempts. Normal existing test caching remains
enabled; no solver budgets or quality flags are relaxed.

The signed /173 review, agent, SDK/recorders and canonical /120 bytes remain
unchanged. This correction is **source-qualified, not packaged or promoted**.
The next step is one clean replacement review app/agent with copied-runtime
qualification. Owner acceptance, visible responsiveness and broader gates remain
open. Earlier /173 source-pinned verifiers require their original source revision;
their frozen results are not claims about this changed working tree. The original
changed source and native ledger are preserved here for comparison.

Read-only current-source/artifact/quality replay:

```sh
python3 -B docs/validation/export-progress-labels/verify.py
```
