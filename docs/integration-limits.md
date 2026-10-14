# Integration limits and recovery

General-network transient integration retains the original **512 attempts / 100 ms** configured budget for one output sample, including source-corner evaluations. Numerical tolerances, model equations and sampling limits are unchanged. The attempt count includes the rejected admission, so an attempt-cap error reports 513 attempted with a maximum of 512.

Users and agents now receive distinct codes:

| Code | Meaning | Next action |
|---|---|---|
| `integration-attempt-limit` | The sample needed more integration attempts than allowed. | Reduce the output sample interval; Restart a stopped live run or retry paused Step. |
| `integration-time-limit` | Elapsed time exceeded the configured integration budget before another attempt. | Reduce the interval or retry when the computer is less busy; Restart a stopped live run. |

Raw details include attempted count and elapsed milliseconds, with the same configured maxima. Wrapped errors retain their classification through live sampling, waiting/background manual Step and recorded Verify. Other solver failures keep their existing generic codes. These are added error codes within the existing open string schema; protocol/schema versions and command count remain unchanged.

`project.inspect` and mutation snapshots expose `limits.integration_attempts_per_sample` and `limits.integration_wall_ms_per_sample`. Live diagnostics carry the specific code; failed Step/Verify jobs carry it in `error.code`. Shared Circuit guidance explains the cause and recovery without inventing a faulty part. Technical details remain available through the existing `diagnostics.details.set` command/disclosure.

Live failure retains the last accepted sample and marks readings stale. Failed manual Step preserves its original committed time, charge, graph and history; staged progress is not committed physics. Failed Verify retains captured input metadata for inspection, but has no result rows or kept partial recording. Successful smaller-step recovery clears the failed-step guidance. Retrying a known failed key returns its original captured outcome without reapplying it to a newer successful run.

The elapsed-time cap is checked **between integration attempts**. It does not preempt a currently executing solve or scheduler suspension, so a detected overrun can report more than 100 ms. This is bounded cooperative work accounting, not a hard real-time completion promise. A smaller sample that avoids a work-limit failure can still alias a fast waveform; sampling warnings and Fit waveform resolution remain separate.

## Qualification

Deterministic clock/counter tests cover both original boundaries, wrapped classification, generic fallback and live/Step/Verify state preservation. Native tests activate the actual Step callback, inspect readable guidance and recover with a smaller interval. Packaged GPU smoke exercises that rejection, disclosure, schematic rendering and recovery on its owner. [Headless](validation/headless-integration-limits.json) and [native](validation/desktop-integration-limits.json) public scenarios verify limits, real coarse-sample rejection, independent recording failure and retry-key semantics.

The opt-in [pressure audit](validation/integration-budget-pressure.json) reproduces the doubled-capacitance flasher fixture in a separate Go test process with one worker CPU. Idle integration completed six simulated seconds with a final sampled period of 0.98 s, inside the previous fixture tolerance. Eight owned CPU burners caused an elapsed-time rejection after 45 attempts, with 212.097 ms observed at the next check; the last accepted time remained 0.18 s and exact solver memory/canonical physical results were unchanged. All burners were stopped and joined. This demonstrates one synthetic time-limit case; it does not establish the cause of the earlier unclassified race-run failure or universal performance under load.

```sh
ELAB_INTEGRATION_AUDIT_REPORT="$PWD/build/integration-audit-new.json" \
  go test -race ./workbench -run '^$' \
  -bench '^BenchmarkFlasherIntegrationBudgetAudit$' \
  -benchtime=1x -count=1 -cpu=1 -timeout=60s -v
python3 scripts/integration-limit-demo.py --output build/integration-public-new.json
```

The audit report path must be new; failed subcases cannot publish a passing report. Synthetic pressure runs only in that owned test process. Public demos replace only their selected session, and `--connect` must identify an owned development app.

The [release receipt](validation/integration-limits-validation.json) records quality checks, SDK baseline, actual packaged smoke and prior workflow regression scope. Broader capacity, participant, visible-latency and physical-device qualification remain separate.
