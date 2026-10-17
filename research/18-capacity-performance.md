# Repeated timing at circuit capacity

This continuation measures the published 100-node and 64-part envelope through the existing public user/agent commands. It extends the preceding [capacitor/capacity qualification](17-capacitor-integration-and-capacity.md), whose maximum-capacity timing gate remained open.

## Protocol

Three fixtures retain their topology and counts: fifty potentiometers at 100 nodes, a 63-resistor chain at 64 parts/64 nodes, and the 64-part/60-node four-stage RC/NMOS LED filter. Each uses four live/Verify probes, 100 ms high/low source phases and a complete 200 ms capture at 1 ms outputs. P1 alternates 10/20 kΩ, R1 alternates 1/2 kΩ, and the LED load RL alternates 1/2 kΩ. These are real electrical edits. The insulated gate filter itself is unchanged; the load-line reference uses the actual load resistance.

The optional `--capacity-performance-suite` runs twenty fresh and twenty warm samples per fixture and adapter. Fresh means a new owned session with construction outside the timed edit; OS caches are not purged and process launch is excluded. Warm includes one unmeasured edit/Verify first. Measurements retain full graph/model/settings/probe/identity associations and all 205 rows, including both sides of pulse edges. Numerical references and capture statistics are checked after the measured completion timestamp, so report review is outside the Verify timing.

Backend targets remain p95 edit-to-fresh ≤500 ms and maximum 200 ms Verify ≤2 s. Command roundtrip is transport-inclusive and does not measure visible acknowledgment. Missing/stale/wrong-run outputs and no-op edits are refused. The report reader reconstructs exact variants, requires complete coverage, and recomputes references, summaries and target flags instead of trusting aggregate success.

## Review and setup

The initial ordinary in-process run produced 120 genuine samples and passed all backend targets. Its complete report is frozen as a regression fixture. Review adds thirty corruption cases and missing/stale live-state checks. Two failing-before cases exposed missing finite final-time and sample-statistic consistency checks; both are repaired. The first mutation helper decoded into a shared nonnil pointer, contaminating its own reference. It now decodes into a fresh value and verifies that the original report still passes after all mutations; this was a test-helper defect.

The runtime/device/native sources remain unchanged in this slice. Rebuilding after the preceding commit changes Go VCS build metadata, so the new binary hashes are recorded and a new private candidate is packaged and qualified. Earlier canonical and candidate app bundles are preserved. Functional P0 remains 17 implemented /10 partial; this engineering protocol does not close visible latency, general topology/hardware, full native input/accessibility, distribution, physical printing or participant gates.


The first full race run hit the production wall guard inside the instrumented full-size solver after fourteen attempts. The ordinary 120-sample run had already passed. The functional regression now builds an ordinary owned agent and exercises the same public socket commands from the instrumented controller; no fixture, duration, grid or production limit is reduced. The core still has controlled-clock numerical race coverage, and the report reader/corruption tests remain instrumented. This separates machine timing from instrumentation without skipping the full-size workflow. The agent's process, private endpoint and logs are test-owned, and its normal close is required.

## Final result

All four adapters pass **480 measurements**, retaining **98,400 rows /393,600 timed row-reference checks** and **27,596 accompanying numerical checks**. Every repeated same-variant row and normalized final physical result is identical across sessions and adapters. The frozen ordinary candidate passes 45 native smoke stages and all current quality checks, including 22 Python tests, root/native race and vet.

| Fixture | Maximum case p95 edit-to-fresh | Maximum Verify |
|---|---:|---:|
| capacity-100-nodes | 23.375 ms | 81.404 ms |
| capacity-64-parts | 18.941 ms | 34.407 ms |
| capacity-rc-mosfet | 18.598 ms | 470.214 ms |

The maximum case p95 is **23.375084 ms**, and maximum Verify is **470.2145 ms** on the recorded Apple M4 Pro workstation. All named-case backend targets pass. Full quality retains the instrumented timeout and test-helper setup errors separately. [Receipt](../docs/validation/capacity-performance-validation.json) and [complete evidence/reproduction](../docs/validation/capacity-performance/README.md) preserve the exact sources, binaries, metadata, rows and limitations.

This closes the repeated-timing gap for these three boundary fixtures on this workstation. Simultaneous maxima, all allowed topologies/device combinations, other hardware, visible latency, complete input/accessibility, clean distribution, physical printing and participant acceptance remain unqualified. The canonical 0.105 app and preserved 0.104 app stay unchanged; the 0.108 candidate is private. P0 remains **17 implemented /10 partial**.
