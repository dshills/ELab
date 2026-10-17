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

The coil extension uses `--inductor-performance-suite --repetitions 20` for four separate families: RL, RLC, NMOS flyback and NPN flyback with an explicit 1 µF base capacitor. All capture 200 ms at 1 ms outputs, including two switching periods. It retains both resistor variants, the exact verified sample step, fresh/warm distributions and the same backend targets. With four adapters the minimum run produces 640 coil measurements. This flag supplements the original five-family suite.

The [0.104 qualification](validation/inductor-release-validation.json) reruns all 800 original-family and 640 coil measurements on the recorded Apple M4 Pro workstation. Maximum case p95 edit-to-fresh is **10.239 ms**; maximum individual Verify is **271.670 ms**. [Complete evidence](validation/inductor-release-conformance.json) shares repeated graphs and sample contexts through an explicitly documented lossless encoding; reconstruction was compared to the full raw report. [The sample audit](validation/inductor-release-sample-audit.json) recomputes distributions and checks complete coverage. These are backend observations with the scope and limits below; they do not establish visible latency.

The unreleased op-amp extension uses `--opamp-performance-suite --repetitions 20` for the follower, inverting amplifier, RC/NPN LED driver and two-buffer RC/MOSFET LED driver. Each captures 200 ms at 1 ms outputs and selects four live/Verify probes. It alternates RL between 10/20 kΩ, RF between 20/22 kΩ, or the mixed input resistor RI between 10/12 kΩ. The same fresh/warm definitions and backend targets apply; four adapters produce 640 measurements. Timing requires matching run/revision identities in both transient and mixed results, the expected amplifier count and complete fixed op-amp model metadata.

Use the development binary for this extension; the qualified 0.105 app predates op-amps:

```sh
GOPROXY=off ./scripts/build.sh
./build/elab-conformance --opamp-suite --opamp-performance-suite --repetitions 20 \
  --artifacts build/opamp-performance-new --native-binary build/elab
```

The numerical suite supplies separate reference/capture and adapter-equality evidence. Timing alone does not qualify model accuracy, physical amplifier bandwidth/stability, maximum capacity or visible response.

The [unreleased op-amp qualification](validation/opamp-performance-validation.json) passes all 640 measurements and 931 numerical reference checks per adapter. Maximum case p95 edit-to-fresh is **11.837 ms**; maximum individual Verify is **604.578 ms**. The [sample audit](validation/opamp-performance/sample-audit.json) recomputes every distribution, graph hash, resistor variant, engine/model/probe association and coverage count. Its reviewer also rejects eight deliberately altered reports. The [complete report](validation/opamp-performance/conformance-report.json.gz) is gzip of the exact original JSON, with a verified lossless round trip; [reproduction instructions](validation/opamp-performance/README.md) describe the retained audit sources. These results use the pinned private source SDK and do not promote the canonical app.

## Measurement boundaries

| Measurement | Boundary and target |
|---|---|
| Command round trip | Just before dispatch to decoded response; includes transport/JSON overhead. This is **not** input-to-visible acknowledgment. |
| Edit to fresh result | Same dispatch start to first public result carrying the new run/electrical revision. If the edit response is stale, includes `simulation.wait`. Conservative backend upper bound; p95 must be ≤500 ms. |
| Verify completion | Before public admission to first observed completed operation, excluding subsequent sample-page reads. Includes polling/transport overhead; maximum must be ≤2 seconds. RC, pulse NPN and flasher use 200 ms at 1 ms sampling; DC cases use a zero-duration operating point. |

Fresh means a newly created owned session with circuit/probes prepared. Launch/construction time is excluded, and OS/filesystem caches are not purged. Warm means one prepared session after an unmeasured edit/Verify, followed by repeated alternating edits. Fresh and warm distributions are reported separately. Percentiles use nearest rank: p50 is rank 10 and p95 rank 19 of 20 sorted observations. The maximum remains visible.

These results qualify the specific reference fixtures on one workstation. They do not establish performance at maximum supported capacity, deliberate difficult-solve responsiveness, cold machine startup, visible frame latency, or universal hardware performance. The visible acknowledgment, accessibility, participant and distribution gates remain open. The original measurement checkpoint retained production 0.42.0 /57 because it changed the harness and evidence; later runtime qualifications are recorded separately in the progress log.
