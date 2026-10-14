# Reference circuit backend performance

Measured October 14, 2026 on Apple M4 Pro (14 logical CPUs, 64 GiB), macOS 26.6.2, Go 1.27.2, packaged ELab 0.42.0 / 57. All four public adapters passed 800 measurements in total: five circuits × fresh/warm sessions × 20 repetitions × four adapters. Four probes were active in each live and Verify fixture.

The packaged app meets the 500 ms backend fresh-result and 2 s Verify targets on these fixtures. Its slowest case p95 was **5.015 ms** and maximum Verify was **48.470 ms**. These are transport-inclusive backend measurements; visible UI feedback remains a separate gate.

| Packaged app circuit | Condition | Command p95 (ms) | Fresh result p95 (ms) | Verify maximum (ms) |
|---|---|---:|---:|---:|
| led | fresh-session | 0.992 | 2.065 | 1.957 |
| led | warm-session | 0.686 | 1.502 | 1.380 |
| rc | fresh-session | 1.070 | 1.070 | 2.585 |
| rc | warm-session | 1.853 | 1.854 | 1.204 |
| npn | fresh-session | 1.304 | 1.939 | 2.606 |
| npn | warm-session | 1.126 | 2.063 | 1.891 |
| npn-pulse | fresh-session | 1.002 | 1.641 | 3.715 |
| npn-pulse | warm-session | 2.358 | 4.454 | 2.960 |
| flasher | fresh-session | 2.257 | 3.999 | 48.470 |
| flasher | warm-session | 2.507 | 5.015 | 46.931 |

Fresh sessions exclude process launch/construction and do not purge OS caches. Warm sessions start after an unmeasured edit/Verify. Each measured edit alternates a resistor value; fresh results must carry the changed run/revision. Verify is 200 ms at 1 ms sampling for RC/pulse NPN/flasher; DC circuits use operating-point verification. The first benchmark is retained as preliminary evidence in `build/backend-performance-20`; the final run retains exact graph definitions in `build/backend-performance-final`.

See the [reproducible method](../docs/performance-qualification.md), [hardware/build receipt](../docs/validation/backend-performance-validation.json), and [all samples and circuit definitions](../docs/validation/backend-performance-conformance.json). Graphs, model IDs, settings, probes and horizons were checked for equality across adapters before publishing the evidence. Performance timings themselves are not required to match.

Full lint/import checks, root/native race tests and vet pass. No runtime changes or bundle replacement were needed. Functional coverage remains 17 implemented / 10 partial; visible feedback, deliberately difficult solves, maximum-capacity timing, accessibility, distribution and participant gates remain open. Schematic output/printing stays on the requested backlog.
