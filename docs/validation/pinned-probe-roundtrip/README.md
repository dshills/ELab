# PIN-REOPEN-01: valid advanced pins block reopening and recovery

The /177 file reader rejects supported MOSFET and powered op-amp pin IDs that
ordinary `measurement.pin.add` accepts. Save completes and marks the document
clean, but a fresh Open fails with `invalid-document`. Deleted-source copies
also fail. The data is still in the saved file; this is a reader validation
defect, not evidence that file bytes were lost or a physical circuit changed.

The old syntax list predates gate/drain/source and the powered op-amp terminals,
plus the gate-source, input-difference and supply voltage probes. The repair
gets physical terminal names from the existing library and admits the three
known voltage differences. A present owner still requires an exact match to its
available probes. Unknown suffixes, wrong owner/type, invalid prefixes, malformed
IDs and current versions of the voltage-only differences remain refused. No
command/schema/model, writer, format or SDK change is needed.

The regression checks every available library probe with its owner present and
removed. Six real NMOS/PMOS/follower/inverting groups exercise all 44 advanced
probe definitions through save/reopen, deletion/reopen, undo and fresh readings.
Two controlled recovery cases preserve the prior manual save. Final tests fail
against the original production reader: six successfully saved groups cannot be
decoded and both recovery catalogs expose no valid candidate. Corrected source
passes the focused pinned-reading race suite. Existing strict validation tests
remain in that suite and the full quality gate.

Final default quality passes in **366.232 s** on the same 652 application Go
inputs: format/import checks, Staticcheck, all 65 Python tests, root/desktop race
tests and vet, and the registered mounted offscreen checks. This is a check-run
duration, not visible-performance acceptance.

The actual /177 agent creates six synthetic present-source files and six
deleted-source files through ordinary commands. All twelve fresh Open attempts
refuse; each refusal preserves the empty incoming circuit/document/history.
The corrected development agent opens **those exact same twelve files**, with
44 live pin readings and 44 missing-source definitions respectively. All input
file hashes remain unchanged. The original 18 owned processes and final twelve
processes close normally across **236 raw requests**. These are headless
public-command checks, not new native input or automatic-recovery observations.

`original/` retains the saved bytes, creator/reader transcripts and states;
`final/` retains exact-file reopen states/transcripts. `sources/` archives the
original/final reader, final tests and exact public producer as text. Checks and
the full source-pin record remain in `checks/` and `quality.json`. The existing
offline runtime profile denies development files, outbound IP and shared window
geometry. All fixtures are private and synthetic; owner files/settings and the
canonical/review/ZIP/SDK pins remain preserved.

Run `python3 -B docs/validation/pinned-probe-roundtrip/verify.py` for read-only
integrity/protocol/state/file replay with the recorded build artifacts present.
The verifier starts no app and refuses optimized Python. Historical current-root
source pins intentionally stop matching after legitimate later changes.

This is a source repair. A clean replacement package and exact copied
native/agent pin-file qualification remain the next release step. /177 remains
affected and preserved; the owner walkthrough, broader file/environment,
accessibility, performance, participant and full-MVP gates remain open.
