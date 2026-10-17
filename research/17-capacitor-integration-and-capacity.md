# Capacitor integration and public capacity qualification

The next electronics task after amplifier output is capacity through the ordinary user/agent API. Core tests already covered 100 nodes and 64 passive parts. They did not qualify a full-size multi-capacitor nonlinear load through external adapters.

## Reproduction and implementation

A fresh construction using the prior `da4d03b` agent produced a 64-part /60-node four-stage NMOS LED filter. Its first 1 ms output failed the unchanged wall budget. A controlled-clock regression on the original integration source fails at attempt 513 after roughly 119 µs, distinguishing the attempt collapse from scheduler pressure. Overlapping fixture positions and an incorrect snapshot field name were setup errors, not product defects.

The repaired capacitor path uses two-stage SDIRK2 with full/two-half comparison. Model laws, local tolerances, sample grids, source corners and work limits remain. The final implicit stage is algebraically consistent. No hidden agent path or public method-selection requirement is introduced. [Derivation, limits and primary reference](../docs/capacitor-integration.md).

New engine IDs distinguish the method. Historical backward-Euler captures are accepted only for their original analysis family, remain unchanged on export, and must match their final-result engine. Exact RC, NPN phase and inductor methods are unchanged. Earlier timings remain historical; they cannot substitute for current-method measurements.

## Qualification sequence

1. Retain original wall/attempt failures and the unchanged circuit.
2. Check stage stability/order, failed-stage preservation, full-size attempts/conservation and genuine older captures.
3. Independently freeze four-stage RC values using 80-digit Decimal spectral evaluation; check the Go matrix exponential at twelve times.
4. Construct the 100-node and both 64-part fixtures publicly, run four probes and complete edge/refinement captures, save and reopen.
5. Exercise node 101 and part 65 while preserving editing, prior recordings and accepted physics. Paused lifecycle and unsupported compilation are separate fields.
6. Recompute report coverage/references and reject changed graphs, limits, models, nodes, rows, boundaries and reopen outcomes.
7. Run normal build, lint/Python/root/native race/vet, packaged smoke and all four adapters on frozen source. Requalify affected backend timing separately from visible latency.

`--capacity-suite` adds CLI qualification coverage, not a product command: construction, simulation, probes, files and lifecycle already use the shared dispatcher. The canonical app and previous candidates stay preserved. Promotion, repeated maximum-capacity timing, complete native input/accessibility, distribution and hardware require their own evidence.

## Review findings

The first complete native/four-adapter run passed, but code review found that SDIRK stage history still fed the old backward-Euler damping readout. A failing-before test measured a fictitious 1.1989054932412604 µW. Final-stage capacitor current remains unchanged; instantaneous `V × I` now supplies power balance and fresh SDIRK2/Radau captures require zero damping. Negative/nonfinite damping refuses for every admitted method; genuine backward-Euler captures preserve their nonnegative historical reading. All 615 coarse rows, 405 fine rows and eighteen observations match the pre-fix capture exactly apart from damping/power-residual fields. The final source is qualified again rather than treating preliminary timing as current evidence.

A subsequent full race run exceeded the 100 ms deadline in the existing 120 Ω NPN/coil fixture. Five ordinary runs of the unchanged production solver passed with roughly 12 ms maximum per output and 161 attempts at that load. The numerical work regression now injects a clock while retaining the production attempt ceiling; ordinary public inductor timing separately keeps the real wall deadline. Neither runtime limits nor device/integration equations were changed to address instrumented timing. The first failed race log remains with the evidence.

Three genuine captures produced by the preserved pre-change binary also pass public stdio import, recorded seek, exact export, keep/save and fresh project reopen. Each retains its older engine and full metadata, both exported copies match original JSON bytes, the source files stay intact, and the live RC circuit is preserved. The first helper incorrectly indexed an optional mixed-result field on the optimized RC snapshot; the corrected reader compares absent fields as absent. A subsequent exclusive-directory refusal was also a helper setup error, not a product failure.

## Qualified result

The frozen ordinary app/agent candidate passes 45 packaged native smoke stages and all four public adapters. Each adapter passes 6,899 numerical references, including 4,900 capacity checks: **27,596 total /19,600 capacity**. Capacity captures retain 4,080 rows across adapters, plus refinement comparisons, and all 24 fresh valid/excess reopens retain exact recorded JSON. The retained archive contains sixty capacity files and twelve genuine-legacy project/recording files; every member hash is verified.

Reference, powered-amplifier and coil timing suites retain **2,080 measurements** (20 fresh/20 warm per family/adapter). All measured backend targets pass on the recorded Apple M4 Pro workstation. Maximum case p95 edit-to-fresh is **12.53275 ms**; maximum Verify is **243.860667 ms**. This does not measure visible feedback or maximum-capacity p95, and OS caches were not purged.

Final report review adds a verifier-only rejection for negative maximum KCL residuals, demonstrated by a failing-before self-comparison regression. All seventeen damaged categories must refuse even when both reports carry the same corruption. The final checker recomputes all retained adapters from raw and gzip reports and runs a fresh three-case public capacity workflow. The app, agent and native inventory binaries remain byte-identical to those measured; only the conformance CLI changes. Both source indexes and the exact two-file verifier delta are retained. Auxiliary readers’ initial live/Verify-grid assumption was corrected to the documented per-template settings; no measured data or production equation changed.

See [receipt](../docs/validation/capacitor-capacity-validation.json) and [raw evidence/reproduction](../docs/validation/capacitor-capacity/README.md). Canonical/preserved bundles remain unchanged. Functional P0 remains **17 implemented /10 partial**; broader topology/hardware, repeated capacity timing, complete input/accessibility, visible latency, clean distribution, physical print and participant gates stay open.
