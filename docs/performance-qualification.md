# Backend performance qualification

`elab-conformance --performance-suite --repetitions 20` adds repeated performance measurements to the ordinary public reference workflows. It covers the LED, ideal-pulse RC, DC NPN, pulse-driven NPN and mixed capacitor/NPN flasher through in-process, stdio, headless socket and optional packaged-native socket adapters. It creates and closes owned sessions without connecting to an existing user app.

```sh
GOPROXY=off ./scripts/build.sh
./build/elab-conformance --performance-suite --repetitions 20 \
  --artifacts build/backend-performance-new \
  --native-binary build/ELab.app/Contents/MacOS/ELab
```

Use an ordinary build, a new output directory and no concurrent heavy test/build workload. The flag accepts 20–100 repetitions per circuit/condition; the minimum run records 800 measurements across four adapters. Timing is not compared for numerical equivalence. A missed backend target fails the command, with individual target outcomes retained in `report.json`.

Each repetition changes a resistor value while running and requires an increased electrical revision/run identity and a non-stale result for that exact identity. Four live probes and four Verify probes are selected. Verify operates on paused captured state and must leave the live physics unchanged. Completed records are excluded from project retention, allowing the existing bounded store to evict them. Timing includes normal command parsing, validation, transport and snapshot serialization.

The report preserves each duration, model ID, engine version, settings, graph hash, run/revision, probe selection and verified horizon. `circuits_by_sha256` stores the exact graphs for both resistor values; hashes use Go's `json.Marshal(workbench.Graph)` representation. The [workstation receipt](validation/backend-performance-validation.json) pins the OS/hardware, source and binary identities; [measurement evidence](validation/backend-performance-conformance.json) retains all samples and circuit definitions. Raw process logs remain in the original build artifacts.

## Measurement boundaries

| Measurement | Boundary and target |
|---|---|
| Command round trip | Just before dispatch to decoded response; includes transport/JSON overhead. This is **not** input-to-visible acknowledgment. |
| Edit to fresh result | Same dispatch start to first public result carrying the new run/electrical revision. If the edit response is stale, includes `simulation.wait`. Conservative backend upper bound; p95 must be ≤500 ms. |
| Verify completion | Before public admission to first observed completed operation, excluding subsequent sample-page reads. Includes polling/transport overhead; maximum must be ≤2 seconds. RC, pulse NPN and flasher use 200 ms at 1 ms sampling; DC cases use a zero-duration operating point. |

Fresh means a newly created owned session with circuit/probes prepared. Launch/construction time is excluded, and OS/filesystem caches are not purged. Warm means one prepared session after an unmeasured edit/Verify, followed by repeated alternating edits. Fresh and warm distributions are reported separately. Percentiles use nearest rank: p50 is rank 10 and p95 rank 19 of 20 sorted observations. The maximum remains visible.

These results qualify the specific reference fixtures on one workstation. They do not establish performance at maximum supported capacity, deliberate difficult-solve responsiveness, cold machine startup, visible frame latency, or universal hardware performance. The visible acknowledgment, accessibility, participant and distribution gates remain open. The production app stays at 0.42.0 / 57 because this checkpoint changes the harness and evidence, not its runtime.
