# Offline baseline content admission

October 19, 2026. No native app, capture, input or settings operation occurs.
The frozen ELab /173 app, agent, recorder, SDK and original trials stay unchanged.

The new [validation tool](../../../tools/baseline-admission/README.md) reads an
existing recorder PNG using Go's standard decoder. Pinned LED/RC profiles compare
opaque RGB regions from the previously inspected annotations. A Python worker
helper binds the first complete frame to the tool/profile/trial/unique attempt;
a pure controller gate refuses marker and input callbacks on any invalid receipt.
This is prepared tooling, **not a revised producer already used in native trials**.
Its documented worker/controller integration requires a preserved declaration
before a future native attempt. RC collection remains halted.

| Offline replay | Result |
|---|---:|
| First recorded images | 24 |
| Valid initial LED/RC views admitted | 22 |
| Foreign-app initial views refused (`rc2-02`, `rc2-04`) | 2 |
| Already edited fresh views refused as wrong baselines | 21 |
| Real decisions passed through fake controller callbacks | 45 |
| Controller tests | 28 passed |
| Producer refusal tests | 6 passed |

Every replayed Go region hash agrees with independent Pillow RGB decoding. All
negative decisions enter neither fake marker nor fake input callback. The tests
also cover missing/stale/mismatched receipt bindings, failed analysis, malformed
regions and acknowledgment failure. Missing/malformed/unfinished journals,
symlinks, byte limits, invalid geometry and inconsistent timestamp flags refuse.
Go's bounded profile/PNG parsing, CLI, checksum/size/alpha cases and race checks
pass, alongside gofmt, goimports, vet and Staticcheck. The scope is the new tools
package; the app's previous 650-source quality receipt still matches exactly.

The first replay attempt incorrectly rejected a composition timestamp ahead of
callback receipt. This was a new guard assumption, contrary to the preserved
recorder calibration, and was corrected before the admitted replay. Both timestamp
orderings now have a consistency test; no latency criterion changes. The first
Staticcheck invocation also tried its default shared cache; the final recorded
run uses a project-local cache. Neither issue changes an app or recorded trial.
README deadline wording was clarified after the checks; code hashes are identical
between the checked reports and final artifacts. Historical report hashes stay
as recorded, with the final documentation hash in `artifacts.json`.

`rc2-03` deliberately passes the initial check although it becomes covered later.
Unobserved pixels can also change without failing the selected regions. Admission
does not prove focus, ownership, private input, continued visibility, acceptance,
freshness or timing eligibility. Full-frame visibility/clock/physics analysis
still rejects covered recordings. No failed trial is replaced, no full RC
percentile is created and no overall PRD gate closes.

Raw PNGs stay in the existing ignored local archives with their original tracked
hashes. This folder contains decisions and provenance only. `cases.json` binds
each image to its source journal and archive metadata; `result.json` records the
Go/Pillow comparison, `quality/` holds exact check logs, and `artifacts.json`
records unchanged app/SDK/recorder/prior-proof identities and final tool sources.
The tool binary remains ignored and is pinned by SHA-256 in each decision.

Recheck metadata and fake callback admission:

```sh
python3 -B docs/validation/baseline-content-admission-173/verify.py
```

To repeat actual image decoding, use the replay command in the tool README with
a new output directory, Pillow and the original local archives. Restore missing
archives; never regenerate a trial to reconstruct evidence. Native integration,
desktop isolation, the RC percentile, owner walkthrough and broader PRD gates
remain open.
