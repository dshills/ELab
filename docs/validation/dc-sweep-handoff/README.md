# DC sweep Run: unfinished field handoff

**One Run now captures its own range and point-count drafts before starting the shared sweep operation.** Six original-production retained-widget cases fail before repair: pointer activation commits the draft, then refuses Run at the old revision; direct accessibility-style activation starts a sweep using old settings. The three controls are Start, Stop and Points. Original native files match the /171 source commit; that source correspondence is separate from a new run of the copied /171 binary.

The repair settles all three fields through the existing SDK FIFO barrier and retains only their exact accepted settings and revision/document context. A later snapshot proves freshness but cannot supply replacement settings. Run dispatches the same public `sweep.start` with those settings once settlement completes. Duplicate callbacks cannot start a second job. Intervening agent/user commands, selection, document replacement, unrelated sweep controls, refusal and settlement/admission errors preserve their guards.

Invalid text cannot fall back to previous accepted settings merely because another field succeeds. Correction can Run on its first attempt, including correction back to the original point count. Private text belongs to the configured target/parameter; a new target with the same displayed point count retires the old draft. Only the owner's exact accepted viewport response can continue its settings handoff through a feedback-driven layout change.

Thirty focused race-tested cases pass in 8.443 s: 12 real retained-widget pointer/activation/invalid/correction/held-agent/target cases and 18 deferred/interruption/rejection/viewport cases. The six valid field/activation variants compare their full completed curves with an independent public agent service, while preserving the live graph, run/electrical identity and result. These use production bound callbacks and actual private SDK controls. They are not new physical macOS typing, IME, screen-reader or CUA pointer acceptance.

Two test-fixture failures remain explicit: a test-only minimum size conflicted with production maximum bounds, and a helper designed for successful UI actions treated intended refusals as fatal. Removing the size override and directly invoking the captured production callbacks for expected refusal cases repairs the fixtures; neither failure is used as evidence of a product defect. The six original-source failures and their regression source are retained separately.

The checked-in native inventory now has 72 production files, 397 sites, 164 public commands and 132 source-routed IDs, with the same public/alias/route sets. Factoring settlement changes the static categories from 198 literal/173 review sites to 197 literal/174 review sites; those categories are not counts of untested capabilities. The standard mounted-control gate now includes this sweep family. No public command/schema, solver, model, saved format or SDK changes.

The final default `./scripts/check.sh` passes in **231.526 s**, without serialization overrides. It includes lint/static checks, all 54 Python tests, root/native race and vet, and the expanded mounted-control race/vet gate. All 650 Go source hashes remain unchanged throughout the run. `checkpoint.json` binds the report and logs. This source checkpoint does not qualify a replacement package or actual OS input. The /171 review contains the original sweep bindings and remains preserved; clean replacement qualification is separate. The source-only matrix optimization also remains outside /171. Owner walkthrough and broader PRD gates remain open; no promotion or complete parity acceptance is inferred.

Read-only source/evidence verification:

```sh
python3 -B docs/validation/dc-sweep-handoff/verify.py
```

Actual focused tests run from `desktop` with the pinned SDK prepared and repository-local Go cache:

```sh
go test -race -tags=offscreen -run '^(TestMountedDCSweep|TestDCSweep(Deferred|Settlement|InvalidField|OwnViewport))' -count=1 ./...
```

`quality.py.txt` retains the full gate producer. All proof fixtures are synthetic; no owner app, project, settings, printer, recording or OS privacy grant is operated by these source tests.
