# Recorded-page discovery mismatch and source repair

October 19, 2026. Frozen /174 accepts explicit `null` offsets and limits as
recorded-page defaults, but its ordinary and detailed discovery advertise only
integers. **PAGE-NULL-SCHEMA-01** is a confirmed discovery-accuracy defect.
The source repair advertises the existing nullable inputs, adds the detailed
offset default and clarifies the descriptions. Required recording identity,
decoder, paging algorithm, limits, output schemas, models and numerical code
are unchanged. Clean replacement packaging remains separate.

## Original /174 evidence

One owned packaged headless process loads the preserved synthetic RC project
from the native recovery test. Its live graph is 30 kΩ, while its independent
stored recording remains the original 10 kΩ, 209-row, two-probe capture.
Three explicit-null query forms return exactly the ordinary 100-row default
page, preserving the full before/after snapshot. Both catalog forms nevertheless
declare `type: integer` for `offset` and `limit`: four metadata mismatches.

Twenty-nine preceding pure cases cover discovery, captured metadata, omitted/
zero/null defaults, exact 100/100/9 and 200/9 paging, final row/EOF, invalid
bounds/types/overflow/identity and unknown fields. The collector then incorrectly
expects a duplicated JSON field to produce a command-level `invalid-params`
response. The transport correctly rejects the malformed request earlier with
uncorrelated `invalid-json`, empty ID and revision zero. The collector stops;
that request has no post-query purity observation. The source project remains
unchanged, and the agent completes joined shutdown and exits 0. The collector
is not reported as an overall pass.

`original/` retains exact raw requests/responses, original ordinary/detailed
schemas, fixture and snapshots, partial report, producer, error log and
production source copies. The mismatch does not depend on interpreting the
collector's malformed-request expectation as an application defect.

## Repair and checks

Only `operation.samples` input discovery changes. Ordinary catalog input types
become `integer | null`; detailed discovery inherits those types, states that
omitted/null offset means zero, and that omitted/null/zero limit means 100,
otherwise 1–200. Output offsets/counts remain integers. The decoder's accepted
values and the query's validation/publication behavior do not change.

The new regression exercises actual multi-page null/zero defaults and complete
snapshot purity while checking both catalog forms, the required ID type and
the resolved local numeric output schema. The exact final test fails with the
two original production metadata files supplied through a temporary Go overlay.
It passes with the repair; four related discovery/default tests pass under the
race detector. Initial test-fixture mistakes are retained separately: a duration
pointer compile error and failure to resolve the local output-schema reference.
Neither is an original product failure. The final overlay removes that ambiguity.

One separately built development agent then passes all thirty public cases,
including the corrected duplicate-field transport response and a following
snapshot resynchronization. All 95 requests are retained, with no retry or hidden
mutation. Full project state stays unchanged across each query; complete edge
rows and SI probe order are exact. The ordinary catalog's 164 entries differ
only in the two intended input-type properties; every output schema and local
schema registry is unchanged. Normal joined shutdown and process exit are 0.
This is development-source evidence, not a clean release or native UI run.

The source preparation's initial Python keyword typo occurs before its module
executes and launches no agent. Its ensuing missing-script log and the corrected
preparation source are preserved; no product change is attributed to that error.

`source/` retains the corrected producer, catalog projection, source/fixture
pins, raw protocol/snapshots and focused logs. Full source quality and the final
checkpoint must be read with their own recorded status; no passing source or
package gate is inferred from the protocol report alone. No existing app,
owner project/settings, printer, network account or privacy grant is changed.
AGENT-02 remains partial outside this finite contract slice.

## Coil validation method

The initial default full gate fails in the existing `npn-coil` functional fixture:
its race-instrumented in-process solver exceeds the unchanged 100 ms integration
wall limit (114.420 ms, 100 attempts of a 512-attempt maximum). A separate exact
isolated race rerun passes; both observations remain archived. Neither result
establishes a product performance percentile.

The full coil fixture now uses the existing `ordinaryAgentFactory` method already
used by the capacity fixture. The controller/report assertions remain under race;
the owned external solver runs ordinarily with the production deadline enabled.
All four families, three changed runs each, 200 ms captures, 1 ms sampling, four
probes, model/run identities and exact assertions stay unchanged. Core numerical
race coverage is retained separately. All four focused fixtures pass in 6.798 s.
The test adapter is the only fourth Go change; no solver bound is relaxed.

The 650-file source pin sets distinguish the original development probe/failed
gate from the final gate with this test-only method correction. The initial
pin-generation helper's wrong JSON-container lookup exits before modifying a
file or starting a test; the corrected pin set is retained. These preparation
errors are not product failures.

## Final source gate

The final default `GOPROXY=off ./scripts/check.sh` exits 0 in 359.541 s:
SDK admission, imports/format/Staticcheck, requirement-ledger consistency,
54 Python tests, both Go modules' race/vet checks, and selected mounted offscreen
controls/vet pass. Its exact 650 Go hashes are archived and checked before this
checkpoint. The earlier full-gate failure remains a failure, with its unmodified
log and original test source. This passing gate follows the test-adapter repair;
it does not retroactively qualify that earlier source configuration.

Read-only replay: `python3 -B docs/validation/operation-page-contract-174/verify.py`.
Clean replacement packaging is the next separate gate. No new physical UI,
latency, owner walkthrough or broader MVP completion is established.
